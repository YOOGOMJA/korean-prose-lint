package tests

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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
