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
			findings = append(findings, finding{
				Path:      filepath.ToSlash(relativePath),
				RuleID:    alert.Check,
				Severity:  alert.Severity,
				Message:   alert.Message,
				Line:      alert.Line,
				SpanStart: alert.Span[0],
				SpanEnd:   alert.Span[1],
				Match:     alert.Match,
			})
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
	if !strings.Contains(string(output), "vale version 3.17.1") {
		t.Fatalf("unsupported Vale version: %s", output)
	}
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

func TestValeConformance(t *testing.T) {
	root := repositoryRoot(t)
	catalog := loadRuleCatalog(t, root)

	for _, rule := range catalog.Rules {
		rule := rule
		ruleName := strings.TrimPrefix(rule.ID, "KoreanProse.")
		t.Run(ruleName, func(t *testing.T) {
			manifestData, err := os.ReadFile(filepath.Join(root, "fixtures", ruleName, "expected.json"))
			if err != nil {
				t.Fatal(err)
			}
			manifest, err := parseExpectationManifest(manifestData)
			if err != nil {
				t.Fatal(err)
			}
			if len(manifest.Cases) == 0 {
				t.Fatal("expectation manifest has no cases")
			}

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
