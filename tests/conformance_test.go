package tests

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

type finding struct {
	Path      string `json:"path"`
	RuleID    string `json:"rule_id"`
	Severity  string `json:"severity"`
	Message   string `json:"message,omitempty"`
	Line      int    `json:"line"`
	SpanStart int    `json:"span_start"`
	SpanEnd   int    `json:"span_end"`
	Match     string `json:"match"`
}

type expectationCase struct {
	Input    string    `json:"input"`
	Findings []finding `json:"findings"`
}

type expectationManifest struct {
	Cases []expectationCase `json:"cases"`
}

type ruleCatalog struct {
	ContractVersion string             `json:"contract_version"`
	Rules           []ruleCatalogEntry `json:"rules"`
}

type ruleCatalogEntry struct {
	ID              string   `json:"id"`
	Summary         string   `json:"summary"`
	Rationale       string   `json:"rationale"`
	Scope           []string `json:"scope"`
	DefaultSeverity string   `json:"default_severity"`
	Exceptions      []string `json:"exceptions"`
	FixSafety       string   `json:"fix_safety"`
}

var ruleIDPattern = regexp.MustCompile(`^KoreanProse\.[A-Z][A-Za-z0-9]*$`)

type valeAlert struct {
	Check    string `json:"Check"`
	Severity string `json:"Severity"`
	Message  string `json:"Message"`
	Line     int    `json:"Line"`
	Span     [2]int `json:"Span"`
	Match    string `json:"Match"`
}

func normalizeValeAlerts(data []byte, basePath string) ([]finding, error) {
	var alertsByPath map[string][]valeAlert
	if err := json.Unmarshal(data, &alertsByPath); err != nil {
		return nil, fmt.Errorf("decode Vale alerts: %w", err)
	}

	var findings []finding
	for alertPath, alerts := range alertsByPath {
		if !filepath.IsAbs(alertPath) {
			alertPath = filepath.Join(basePath, alertPath)
		}
		relativePath, err := filepath.Rel(basePath, alertPath)
		if err != nil {
			return nil, fmt.Errorf("make Vale alert path relative: %w", err)
		}
		for _, alert := range alerts {
			item := finding{
				Path:      filepath.ToSlash(relativePath),
				RuleID:    alert.Check,
				Severity:  alert.Severity,
				Message:   alert.Message,
				Line:      alert.Line,
				SpanStart: alert.Span[0],
				SpanEnd:   alert.Span[1],
				Match:     alert.Match,
			}
			if err := validateFindingValue(item); err != nil {
				return nil, fmt.Errorf("normalize Vale alert: %w", err)
			}
			findings = append(findings, item)
		}
	}
	return findings, nil
}

func normalizeValeAlertsForInput(data []byte, repositoryRoot, input string) ([]finding, error) {
	findings, err := normalizeValeAlerts(data, repositoryRoot)
	if err != nil {
		return nil, err
	}
	wantPath := filepath.ToSlash(filepath.Clean(input))
	for index := range findings {
		if findings[index].Path != wantPath {
			return nil, fmt.Errorf("Vale alert path %q does not resolve requested input %q", findings[index].Path, wantPath)
		}
		findings[index].Path = wantPath
	}
	return findings, nil
}

type comparableFinding struct {
	Path      string
	RuleID    string
	Severity  string
	Line      int
	SpanStart int
	SpanEnd   int
	Match     string
}

func compareFindings(want, got []finding) error {
	wantComparable := comparableFindings(want)
	gotComparable := comparableFindings(got)
	slices.SortFunc(wantComparable, compareFinding)
	slices.SortFunc(gotComparable, compareFinding)

	if !slices.Equal(wantComparable, gotComparable) {
		return fmt.Errorf("findings differ:\nwant: %#v\n got: %#v", wantComparable, gotComparable)
	}
	return nil
}

func comparableFindings(findings []finding) []comparableFinding {
	comparable := make([]comparableFinding, 0, len(findings))
	for _, item := range findings {
		comparable = append(comparable, comparableFinding{
			Path:      item.Path,
			RuleID:    item.RuleID,
			Severity:  item.Severity,
			Line:      item.Line,
			SpanStart: item.SpanStart,
			SpanEnd:   item.SpanEnd,
			Match:     item.Match,
		})
	}
	return comparable
}

func compareFinding(left, right comparableFinding) int {
	if order := cmp.Compare(left.Path, right.Path); order != 0 {
		return order
	}
	if order := cmp.Compare(left.Line, right.Line); order != 0 {
		return order
	}
	if order := cmp.Compare(left.SpanStart, right.SpanStart); order != 0 {
		return order
	}
	if order := cmp.Compare(left.SpanEnd, right.SpanEnd); order != 0 {
		return order
	}
	if order := cmp.Compare(left.RuleID, right.RuleID); order != 0 {
		return order
	}
	if order := cmp.Compare(left.Severity, right.Severity); order != 0 {
		return order
	}
	return cmp.Compare(left.Match, right.Match)
}

func parseExpectationManifest(data []byte) (expectationManifest, error) {
	var manifest expectationManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return expectationManifest{}, fmt.Errorf("decode expectation manifest: %w", err)
	}
	for caseIndex := range manifest.Cases {
		for findingIndex := range manifest.Cases[caseIndex].Findings {
			manifest.Cases[caseIndex].Findings[findingIndex].Path = manifest.Cases[caseIndex].Input
		}
	}
	return manifest, nil
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func loadRuleCatalog(t *testing.T, root string) ruleCatalog {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "rules.json"))
	if err != nil {
		t.Fatal(err)
	}
	var catalog ruleCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		t.Fatalf("decode rule catalog: %v", err)
	}
	validateRuleCatalog(t, catalog)
	return catalog
}

func loadExpectationManifest(t *testing.T, path string) expectationManifest {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := parseExpectationManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Cases) == 0 {
		t.Fatal("expectation manifest has no cases")
	}
	return manifest
}

func validateRuleCatalog(t *testing.T, catalog ruleCatalog) {
	t.Helper()
	if err := validateRuleCatalogValue(catalog); err != nil {
		t.Fatal(err)
	}
}

func validateRuleCatalogValue(catalog ruleCatalog) error {
	if catalog.ContractVersion != "0.1" {
		return fmt.Errorf("contract_version = %q, want 0.1", catalog.ContractVersion)
	}
	if len(catalog.Rules) == 0 {
		return fmt.Errorf("rule catalog has no rules")
	}
	validScope := map[string]bool{"prose": true, "heading": true}
	validSeverity := map[string]bool{"suggestion": true, "warning": true, "error": true}
	validFixSafety := map[string]bool{"none": true, "review": true, "safe": true}
	seenRuleIDs := make(map[string]bool, len(catalog.Rules))
	for index, rule := range catalog.Rules {
		if rule.ID == "" || rule.Summary == "" || rule.Rationale == "" || len(rule.Scope) == 0 || rule.DefaultSeverity == "" || len(rule.Exceptions) == 0 || rule.FixSafety == "" {
			return fmt.Errorf("rules[%d] is missing a required field: %#v", index, rule)
		}
		if !ruleIDPattern.MatchString(rule.ID) {
			return fmt.Errorf("rules[%d].id = %q, want KoreanProse.<PascalCaseRuleName>", index, rule.ID)
		}
		if seenRuleIDs[rule.ID] {
			return fmt.Errorf("rules[%d].id = %q is duplicated", index, rule.ID)
		}
		seenRuleIDs[rule.ID] = true
		for _, scope := range rule.Scope {
			if !validScope[scope] {
				return fmt.Errorf("rules[%d].scope contains unsupported %q", index, scope)
			}
		}
		if !validSeverity[rule.DefaultSeverity] {
			return fmt.Errorf("rules[%d].default_severity = %q", index, rule.DefaultSeverity)
		}
		if !validFixSafety[rule.FixSafety] {
			return fmt.Errorf("rules[%d].fix_safety = %q", index, rule.FixSafety)
		}
	}
	return nil
}

func validateInputPath(repositoryRoot, input string) error {
	if filepath.IsAbs(input) {
		return fmt.Errorf("input path must be relative to repository root: %q", input)
	}
	root, err := filepath.EvalSymlinks(repositoryRoot)
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	inputPath, err := filepath.EvalSymlinks(filepath.Join(root, input))
	if err != nil {
		return fmt.Errorf("resolve input %q: %w", input, err)
	}
	relativePath, err := filepath.Rel(root, inputPath)
	if err != nil {
		return fmt.Errorf("make input path relative: %w", err)
	}
	if relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return fmt.Errorf("input path escapes repository root: %q", input)
	}
	info, err := os.Stat(inputPath)
	if err != nil {
		return fmt.Errorf("stat input %q: %w", input, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("input path is not a regular file: %q", input)
	}
	return nil
}

func valeBinary(t *testing.T) string {
	t.Helper()
	if path := os.Getenv("VALE_BIN"); path != "" {
		return path
	}
	path, err := exec.LookPath("vale")
	if err != nil {
		t.Fatal("Vale backend unavailable: set VALE_BIN or install Vale 3.17.1")
	}
	return path
}

func verifyValeVersion(t *testing.T, valePath string) {
	t.Helper()
	output, err := exec.Command(valePath, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("run Vale version: %v: %s", err, output)
	}
	if err := validateValeVersionOutput(string(output)); err != nil {
		t.Fatalf("unsupported Vale version: %s", output)
	}
}

func validateValeVersionOutput(output string) error {
	if strings.TrimSpace(output) != "vale version 3.17.1" {
		return fmt.Errorf("unsupported Vale version: %q", strings.TrimSpace(output))
	}
	return nil
}

func validateStyleRuleFiles(root string, catalog ruleCatalog) error {
	want := make(map[string]bool, len(catalog.Rules))
	for _, rule := range catalog.Rules {
		want[strings.TrimPrefix(rule.ID, "KoreanProse.")] = true
	}

	entries, err := os.ReadDir(filepath.Join(root, "KoreanProse"))
	if err != nil {
		return fmt.Errorf("read style directory: %w", err)
	}
	got := make(map[string]bool, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yml" {
			continue
		}
		got[strings.TrimSuffix(entry.Name(), ".yml")] = true
	}

	for name := range want {
		if !got[name] {
			return fmt.Errorf("catalog rule %q has no KoreanProse/%s.yml", name, name)
		}
	}
	for name := range got {
		if !want[name] {
			return fmt.Errorf("KoreanProse/%s.yml has no rule catalog entry", name)
		}
	}
	return nil
}

func validateV01RuleIDs(catalog ruleCatalog) error {
	want := map[string]bool{
		"KoreanProse.DoubleSpace":         true,
		"KoreanProse.SentenceSpacing":     true,
		"KoreanProse.RepeatedPunctuation": true,
		"KoreanProse.RedundantExpression": true,
	}
	got := make(map[string]bool, len(catalog.Rules))
	for _, rule := range catalog.Rules {
		got[rule.ID] = true
	}
	for ruleID := range want {
		if !got[ruleID] {
			return fmt.Errorf("v0.1 rule catalog is missing %q", ruleID)
		}
	}
	for ruleID := range got {
		if !want[ruleID] {
			return fmt.Errorf("v0.1 rule catalog contains unexpected %q", ruleID)
		}
	}
	return nil
}

func validateFixtureCaseCoverage(root, ruleName string, manifest expectationManifest) error {
	directory := filepath.Join(root, "fixtures", ruleName)
	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("read fixture directory: %w", err)
	}
	fixturePaths := make(map[string]bool)
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
			fixturePaths[filepath.ToSlash(filepath.Join("fixtures", ruleName, entry.Name()))] = true
		}
	}
	casePaths := make(map[string]bool, len(manifest.Cases))
	for index, testCase := range manifest.Cases {
		if casePaths[testCase.Input] {
			return fmt.Errorf("cases[%d].input duplicates %q", index, testCase.Input)
		}
		casePaths[testCase.Input] = true
	}
	for path := range fixturePaths {
		if !casePaths[path] {
			return fmt.Errorf("fixture %q has no expectation manifest case", path)
		}
	}
	for path := range casePaths {
		if !fixturePaths[path] {
			return fmt.Errorf("expectation manifest input %q is not a fixture Markdown file", path)
		}
	}
	return nil
}

func validateFindingValue(item finding) error {
	if item.Path == "" || item.RuleID == "" || item.Message == "" || item.Match == "" {
		return fmt.Errorf("finding has an empty required field: %#v", item)
	}
	if item.Severity != "suggestion" && item.Severity != "warning" && item.Severity != "error" {
		return fmt.Errorf("finding severity is unsupported: %q", item.Severity)
	}
	if item.Line < 1 || item.SpanStart < 1 || item.SpanEnd < item.SpanStart {
		return fmt.Errorf("finding has invalid 1-based inclusive coordinates: %#v", item)
	}
	return nil
}

func validateFindingSource(root, input string, item finding) error {
	data, err := os.ReadFile(filepath.Join(root, input))
	if err != nil {
		return fmt.Errorf("read finding input: %w", err)
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if item.Line > len(lines) {
		return fmt.Errorf("finding line %d is outside %q", item.Line, input)
	}
	line := []rune(lines[item.Line-1])
	if item.SpanEnd > len(line) {
		return fmt.Errorf("finding span %d-%d is outside line %d", item.SpanStart, item.SpanEnd, item.Line)
	}
	matched := string(line[item.SpanStart-1 : item.SpanEnd])
	if matched != item.Match {
		return fmt.Errorf("finding match %q differs from source slice %q", item.Match, matched)
	}
	return nil
}

func runVale(t *testing.T, root, configPath, input string) []finding {
	t.Helper()
	if err := validateInputPath(root, input); err != nil {
		t.Fatalf("input error: %v", err)
	}
	valePath := valeBinary(t)
	verifyValeVersion(t, valePath)
	command := exec.Command(valePath, "--config="+configPath, "--no-global", "--no-exit", "--output=JSON", input)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("Vale backend error: %v: %s", err, output)
	}
	findings, err := normalizeValeAlertsForInput(output, root, input)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range findings {
		if err := validateFindingSource(root, input, item); err != nil {
			t.Fatalf("normalization error: %v", err)
		}
	}
	return findings
}

func TestNormalizeValeAlerts(t *testing.T) {
	raw, err := os.ReadFile("testdata/vale-alerts.json")
	if err != nil {
		t.Fatal(err)
	}

	got, err := normalizeValeAlerts(raw, "/workspace")
	if err != nil {
		t.Fatal(err)
	}
	want := []finding{
		{
			Path:      "docs/예시.md",
			RuleID:    "KoreanProse.DoubleSpace",
			Severity:  "warning",
			Message:   "연속된 공백을 하나로 줄여 보세요.",
			Line:      2,
			SpanStart: 4,
			SpanEnd:   5,
			Match:     "  ",
		},
		{
			Path:      "docs/예시.md",
			RuleID:    "KoreanProse.RepeatedPunctuation",
			Severity:  "suggestion",
			Message:   "반복된 문장 부호를 하나로 줄여 보세요.",
			Line:      4,
			SpanStart: 8,
			SpanEnd:   9,
			Match:     "!!",
		},
	}

	if err := compareFindings(want, got); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("got %d findings, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Message != want[i].Message {
			t.Fatalf("finding %d message = %q, want %q", i, got[i].Message, want[i].Message)
		}
	}
}

func TestNormalizeValeAlertsRebasesRequestedInput(t *testing.T) {
	raw := []byte(`{
  "fixtures/DoubleSpace/invalid.md": [
    {
      "Check": "KoreanProse.DoubleSpace",
      "Severity": "warning",
      "Message": "message",
      "Line": 1,
      "Span": [5, 6],
      "Match": "  "
    }
  ]
}`)
	root := repositoryRoot(t)
	findings, err := normalizeValeAlertsForInput(raw, root, "fixtures/DoubleSpace/invalid.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Path != "fixtures/DoubleSpace/invalid.md" {
		t.Fatalf("unexpected normalized findings: %#v", findings)
	}

	if _, err := normalizeValeAlertsForInput(raw, root, "fixtures/DoubleSpace/valid.md"); err == nil {
		t.Fatal("accepted a Vale alert path for a different requested input")
	}
}

func TestCompareFindingsIgnoresOrder(t *testing.T) {
	first := finding{Path: "a.md", RuleID: "KoreanProse.DoubleSpace", Severity: "warning", Line: 1, SpanStart: 2, SpanEnd: 3, Match: "  "}
	second := finding{Path: "b.md", RuleID: "KoreanProse.RepeatedPunctuation", Severity: "suggestion", Line: 2, SpanStart: 4, SpanEnd: 5, Match: "??"}

	if err := compareFindings([]finding{first, second}, []finding{second, first}); err != nil {
		t.Fatal(err)
	}
}

func TestCompareFindingsKeepsDuplicates(t *testing.T) {
	want := finding{Path: "a.md", RuleID: "KoreanProse.DoubleSpace", Severity: "warning", Line: 1, SpanStart: 2, SpanEnd: 3, Match: "  "}
	if err := compareFindings([]finding{want}, []finding{want, want}); err == nil {
		t.Fatal("compareFindings accepted an unexpected duplicate")
	}
}

func TestCompareFindingsIgnoresMessage(t *testing.T) {
	want := finding{Path: "예시.md", RuleID: "KoreanProse.DoubleSpace", Severity: "warning", Message: "원래 문구", Line: 1, SpanStart: 2, SpanEnd: 3, Match: "한글"}
	got := want
	got.Message = "바뀐 문구"

	if err := compareFindings([]finding{want}, []finding{got}); err != nil {
		t.Fatal(err)
	}
}

func TestCompareFindingsIncludesContractFields(t *testing.T) {
	base := finding{Path: "a.md", RuleID: "KoreanProse.DoubleSpace", Severity: "warning", Line: 1, SpanStart: 2, SpanEnd: 3, Match: "  "}
	tests := map[string]func(*finding){
		"path":       func(item *finding) { item.Path = "b.md" },
		"rule ID":    func(item *finding) { item.RuleID = "KoreanProse.RepeatedPunctuation" },
		"severity":   func(item *finding) { item.Severity = "suggestion" },
		"line":       func(item *finding) { item.Line++ },
		"span start": func(item *finding) { item.SpanStart++ },
		"span end":   func(item *finding) { item.SpanEnd++ },
		"match":      func(item *finding) { item.Match = "   " },
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			changed := base
			mutate(&changed)
			if err := compareFindings([]finding{base}, []finding{changed}); err == nil {
				t.Fatalf("compareFindings ignored changed %s", name)
			}
		})
	}
}

func TestParseExpectationManifest(t *testing.T) {
	raw := []byte(`{
  "cases": [
    {
      "input": "fixtures/DoubleSpace/invalid.md",
      "findings": [
        {
          "rule_id": "KoreanProse.DoubleSpace",
          "severity": "warning",
          "line": 1,
          "span_start": 3,
          "span_end": 4,
          "match": "  "
        }
      ]
    }
  ]
}`)

	manifest, err := parseExpectationManifest(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Cases) != 1 || len(manifest.Cases[0].Findings) != 1 {
		t.Fatalf("unexpected manifest: %#v", manifest)
	}
	if got := manifest.Cases[0].Findings[0].Path; got != "fixtures/DoubleSpace/invalid.md" {
		t.Fatalf("finding path = %q, want case input", got)
	}
}

func TestValidateInputPathRejectsMissingFixture(t *testing.T) {
	root := repositoryRoot(t)
	if err := validateInputPath(root, "fixtures/DoubleSpace/does-not-exist.md"); err == nil {
		t.Fatal("accepted a missing fixture that Vale would report as exit 0 with no findings")
	}
}

func TestValidateRuleCatalogRejectsInvalidRuleIDs(t *testing.T) {
	validRule := ruleCatalogEntry{
		ID:              "KoreanProse.DoubleSpace",
		Summary:         "summary",
		Rationale:       "rationale",
		Scope:           []string{"prose"},
		DefaultSeverity: "warning",
		Exceptions:      []string{"exception"},
		FixSafety:       "none",
	}

	invalidFormat := validRule
	invalidFormat.ID = "KoreanProse.double-space"
	if err := validateRuleCatalogValue(ruleCatalog{ContractVersion: "0.1", Rules: []ruleCatalogEntry{invalidFormat}}); err == nil {
		t.Fatal("accepted a non-PascalCase rule ID")
	}

	if err := validateRuleCatalogValue(ruleCatalog{ContractVersion: "0.1", Rules: []ruleCatalogEntry{validRule, validRule}}); err == nil {
		t.Fatal("accepted duplicate rule IDs")
	}
}

func TestValidateValeVersionRejectsPrefixCollision(t *testing.T) {
	if err := validateValeVersionOutput("vale version 3.17.10\n"); err == nil {
		t.Fatal("accepted Vale 3.17.10 as pinned Vale 3.17.1")
	}
}

func TestValidateStyleRuleFilesRejectsOrphanRule(t *testing.T) {
	root := t.TempDir()
	styleDirectory := filepath.Join(root, "KoreanProse")
	if err := os.Mkdir(styleDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"DoubleSpace.yml", "OrphanRule.yml"} {
		file, err := os.Create(filepath.Join(styleDirectory, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
	catalog := ruleCatalog{Rules: []ruleCatalogEntry{{ID: "KoreanProse.DoubleSpace"}}}
	if err := validateStyleRuleFiles(root, catalog); err == nil {
		t.Fatal("accepted a style rule absent from the rule catalog")
	}
}

func TestValidateV01RuleIDsRejectsMissingRule(t *testing.T) {
	catalog := ruleCatalog{ContractVersion: "0.1", Rules: []ruleCatalogEntry{{ID: "KoreanProse.DoubleSpace"}}}
	if err := validateV01RuleIDs(catalog); err == nil {
		t.Fatal("accepted an incomplete v0.1 rule set")
	}
}

func TestValidateFixtureCaseCoverageRejectsOrphanFixture(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "fixtures", "DoubleSpace")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"valid.md", "invalid.md", "orphan.md"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte("fixture"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	manifest := expectationManifest{Cases: []expectationCase{
		{Input: "fixtures/DoubleSpace/valid.md"},
		{Input: "fixtures/DoubleSpace/invalid.md"},
	}}
	if err := validateFixtureCaseCoverage(root, "DoubleSpace", manifest); err == nil {
		t.Fatal("accepted a fixture absent from the expectation manifest")
	}
}

func TestValidateFindingValueRejectsInvalidFields(t *testing.T) {
	valid := finding{Path: "input.md", RuleID: "KoreanProse.DoubleSpace", Severity: "warning", Message: "message", Line: 1, SpanStart: 1, SpanEnd: 2, Match: "  "}
	tests := map[string]func(*finding){
		"empty path":       func(item *finding) { item.Path = "" },
		"empty rule ID":    func(item *finding) { item.RuleID = "" },
		"invalid severity": func(item *finding) { item.Severity = "fatal" },
		"empty message":    func(item *finding) { item.Message = "" },
		"zero line":        func(item *finding) { item.Line = 0 },
		"zero span start":  func(item *finding) { item.SpanStart = 0 },
		"reversed span":    func(item *finding) { item.SpanEnd = 0 },
		"empty match":      func(item *finding) { item.Match = "" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			invalid := valid
			mutate(&invalid)
			if err := validateFindingValue(invalid); err == nil {
				t.Fatalf("accepted finding with %s", name)
			}
		})
	}
}

func TestValidateFindingSourceChecksUnicodeSpan(t *testing.T) {
	root := t.TempDir()
	input := "input.md"
	if err := os.WriteFile(filepath.Join(root, input), []byte("한글  문장\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	valid := finding{Path: input, RuleID: "KoreanProse.DoubleSpace", Severity: "warning", Message: "message", Line: 1, SpanStart: 3, SpanEnd: 4, Match: "  "}
	if err := validateFindingSource(root, input, valid); err != nil {
		t.Fatalf("rejected matching Unicode source span: %v", err)
	}
	invalid := valid
	invalid.Match = "한글"
	if err := validateFindingSource(root, input, invalid); err == nil {
		t.Fatal("accepted match that differs from the inclusive source span")
	}
}

func TestValeConformance(t *testing.T) {
	root := repositoryRoot(t)
	catalog := loadRuleCatalog(t, root)

	for _, rule := range catalog.Rules {
		rule := rule
		ruleName := strings.TrimPrefix(rule.ID, "KoreanProse.")
		t.Run(ruleName, func(t *testing.T) {
			manifest := loadExpectationManifest(t, filepath.Join(root, "fixtures", ruleName, "expected.json"))

			configPath := filepath.ToSlash(filepath.Join("tests", "vale", ruleName, ".vale.ini"))
			for _, testCase := range manifest.Cases {
				testCase := testCase
				t.Run(filepath.Base(testCase.Input), func(t *testing.T) {
					got := runVale(t, root, configPath, testCase.Input)
					for _, item := range got {
						if item.RuleID != rule.ID {
							t.Fatalf("isolated config returned %q, want %q", item.RuleID, rule.ID)
						}
					}
					if err := compareFindings(testCase.Findings, got); err != nil {
						t.Fatal(err)
					}
				})
			}
		})
	}
}

func TestRuleCatalogCoverage(t *testing.T) {
	root := repositoryRoot(t)
	catalog := loadRuleCatalog(t, root)
	if err := validateV01RuleIDs(catalog); err != nil {
		t.Fatal(err)
	}
	if err := validateStyleRuleFiles(root, catalog); err != nil {
		t.Fatal(err)
	}

	for _, rule := range catalog.Rules {
		rule := rule
		ruleName := strings.TrimPrefix(rule.ID, "KoreanProse.")
		t.Run(ruleName, func(t *testing.T) {
			requiredFiles := []string{
				filepath.Join("KoreanProse", ruleName+".yml"),
				filepath.Join("fixtures", ruleName, "valid.md"),
				filepath.Join("fixtures", ruleName, "invalid.md"),
				filepath.Join("fixtures", ruleName, "expected.json"),
				filepath.Join("tests", "vale", ruleName, ".vale.ini"),
			}
			for _, requiredFile := range requiredFiles {
				info, err := os.Stat(filepath.Join(root, requiredFile))
				if err != nil {
					t.Fatalf("required file %s: %v", filepath.ToSlash(requiredFile), err)
				}
				if !info.Mode().IsRegular() {
					t.Fatalf("required path is not a regular file: %s", filepath.ToSlash(requiredFile))
				}
			}

			manifest := loadExpectationManifest(t, filepath.Join(root, "fixtures", ruleName, "expected.json"))
			if err := validateFixtureCaseCoverage(root, ruleName, manifest); err != nil {
				t.Fatal(err)
			}
			fixturePrefix := filepath.ToSlash(filepath.Join("fixtures", ruleName)) + "/"
			for caseIndex, testCase := range manifest.Cases {
				if !strings.HasPrefix(testCase.Input, fixturePrefix) {
					t.Fatalf("cases[%d].input = %q, want path under %s", caseIndex, testCase.Input, fixturePrefix)
				}
				if err := validateInputPath(root, testCase.Input); err != nil {
					t.Fatalf("cases[%d].input: %v", caseIndex, err)
				}
				for findingIndex, item := range testCase.Findings {
					if item.RuleID != rule.ID {
						t.Fatalf("cases[%d].findings[%d].rule_id = %q, want %q", caseIndex, findingIndex, item.RuleID, rule.ID)
					}
					if item.Severity != rule.DefaultSeverity {
						t.Fatalf("cases[%d].findings[%d].severity = %q, want %q", caseIndex, findingIndex, item.Severity, rule.DefaultSeverity)
					}
				}
			}
		})
	}
}

func TestAllRulesTogether(t *testing.T) {
	root := repositoryRoot(t)
	manifest := loadExpectationManifest(t, filepath.Join(root, "tests", "integration", "expected.json"))
	if len(manifest.Cases) != 1 {
		t.Fatalf("integration manifest has %d cases, want 1", len(manifest.Cases))
	}
	testCase := manifest.Cases[0]
	got := runVale(t, root, ".vale.ini", testCase.Input)
	if err := compareFindings(testCase.Findings, got); err != nil {
		t.Fatal(err)
	}
}
