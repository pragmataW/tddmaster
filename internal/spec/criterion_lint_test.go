package spec

import (
	"encoding/json"
	"testing"
)

func makeTask(id string, criteria []Criterion) Task {
	return Task{ID: id, Criteria: criteria}
}

func makeCriterion(id, when, then, raw string) Criterion {
	return Criterion{ID: id, When: when, Then: then, Raw: raw}
}

func TestLintCriteria_EmptyThen_ReturnsBlockUntestable(t *testing.T) {
	task := makeTask("task-2", []Criterion{
		makeCriterion("ac-1", "user calls endpoint", "", ""),
	})
	findings := LintCriteria(task)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	f := findings[0]
	if f.Severity != "block" {
		t.Errorf("expected Severity block, got %q", f.Severity)
	}
	if f.Category != "untestable" {
		t.Errorf("expected Category untestable, got %q", f.Category)
	}
	if f.AcID != "ac-1" {
		t.Errorf("expected AcID ac-1, got %q", f.AcID)
	}
	if f.TaskID != "task-2" {
		t.Errorf("expected TaskID task-2, got %q", f.TaskID)
	}
	if f.Source != "linter" {
		t.Errorf("expected Source linter, got %q", f.Source)
	}
}

func TestLintCriteria_WhitespaceThen_ReturnsBlockUntestable(t *testing.T) {
	task := makeTask("task-2", []Criterion{
		makeCriterion("ac-2", "user calls endpoint", "   ", ""),
	})
	findings := LintCriteria(task)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Severity != "block" {
		t.Errorf("expected Severity block, got %q", findings[0].Severity)
	}
	if findings[0].Category != "untestable" {
		t.Errorf("expected Category untestable, got %q", findings[0].Category)
	}
}

func TestLintCriteria_NonEmptyThen_NotFlaggedBySemanticGuess(t *testing.T) {
	cases := []string{
		"system works",
		"it handles properly",
		"data is saved correctly",
		"it behaves as expected",
	}
	for _, then := range cases {
		t.Run(then, func(t *testing.T) {
			task := makeTask("task-2", []Criterion{
				makeCriterion("ac-1", "when something", then, ""),
			})
			findings := LintCriteria(task)
			if len(findings) != 0 {
				t.Fatalf("then %q: linter must not guess testability, expected 0 findings, got %d", then, len(findings))
			}
		})
	}
}

func TestLintCriteria_EmptyWhenAndEmptyRaw_ReturnsWarnWeakCriterion(t *testing.T) {
	task := makeTask("task-2", []Criterion{
		makeCriterion("ac-1", "", "returns 200", ""),
	})
	findings := LintCriteria(task)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Severity != "warn" {
		t.Errorf("expected Severity warn, got %q", findings[0].Severity)
	}
	if findings[0].Category != "weak-criterion" {
		t.Errorf("expected Category weak-criterion, got %q", findings[0].Category)
	}
	if findings[0].AcID != "ac-1" {
		t.Errorf("expected AcID ac-1, got %q", findings[0].AcID)
	}
	if findings[0].Source != "linter" {
		t.Errorf("expected Source linter, got %q", findings[0].Source)
	}
}

func TestLintCriteria_EmptyWhenButNonEmptyRaw_NoWeakCriterionFinding(t *testing.T) {
	task := makeTask("task-2", []Criterion{
		makeCriterion("ac-1", "", "returns 200", "raw text here"),
	})
	findings := LintCriteria(task)
	for _, f := range findings {
		if f.Category == "weak-criterion" {
			t.Errorf("expected no weak-criterion finding when Raw is non-empty, got one")
		}
	}
}

func TestLintCriteria_EmptyThenButNonEmptyRaw_NoUntestable(t *testing.T) {
	task := makeTask("task-2", []Criterion{
		makeCriterion("ac-1", "user calls endpoint", "", "user calls endpoint, expects 200 with body"),
	})
	findings := LintCriteria(task)
	for _, f := range findings {
		if f.Category == "untestable" {
			t.Errorf("expected no untestable finding when Raw is non-empty, got one")
		}
	}
}

func TestLintCriteria_CleanCriterion_NoFindings(t *testing.T) {
	task := makeTask("task-2", []Criterion{
		makeCriterion("ac-1", "user sends DELETE /resource/1", "returns 404 with body {error:not found}", ""),
	})
	findings := LintCriteria(task)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for clean criterion, got %d", len(findings))
	}
}

func TestLintCriteria_MultipleCriteria_FindingsHaveCorrectAcID(t *testing.T) {
	task := makeTask("task-3", []Criterion{
		makeCriterion("ac-1", "user calls endpoint", "", ""),
		makeCriterion("ac-2", "user sends request", "returns 200", ""),
		makeCriterion("ac-3", "", "returns 200", ""),
	})
	findings := LintCriteria(task)

	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}

	findingByAcID := map[string]Finding{}
	for _, f := range findings {
		findingByAcID[f.AcID] = f
	}

	f1, ok := findingByAcID["ac-1"]
	if !ok {
		t.Fatal("expected finding for ac-1")
	}
	if f1.Severity != "block" || f1.Category != "untestable" {
		t.Errorf("ac-1: expected block/untestable, got %q/%q", f1.Severity, f1.Category)
	}
	if f1.TaskID != "task-3" {
		t.Errorf("ac-1: expected TaskID task-3, got %q", f1.TaskID)
	}

	f3, ok := findingByAcID["ac-3"]
	if !ok {
		t.Fatal("expected finding for ac-3")
	}
	if f3.Severity != "warn" || f3.Category != "weak-criterion" {
		t.Errorf("ac-3: expected warn/weak-criterion, got %q/%q", f3.Severity, f3.Category)
	}

	if _, exists := findingByAcID["ac-2"]; exists {
		t.Error("ac-2 is clean and should produce no finding")
	}
}

func TestLintCriteria_ReturnTypeIsFindingSlice(t *testing.T) {
	task := makeTask("task-2", []Criterion{
		makeCriterion("ac-1", "when x", "", ""),
	})
	var findings []Finding = LintCriteria(task)
	if findings == nil {
		t.Fatal("expected non-nil []Finding return")
	}
	f := findings[0]
	_ = f.Severity
	_ = f.Category
	_ = f.TaskID
	_ = f.AcID
	_ = f.Detail
	_ = f.Suggestion
	_ = f.Source
}

func TestLintCriteria_OnlyEmptyThenBlocks_NonEmptyPasses(t *testing.T) {
	task := makeTask("task-2", []Criterion{
		makeCriterion("ac-1", "when x", "works", ""),
	})
	findings := LintCriteria(task)
	if len(findings) != 0 {
		t.Fatalf("non-empty Then must produce no linter finding, got %d", len(findings))
	}

	task2 := makeTask("task-2", []Criterion{
		makeCriterion("ac-1", "when x", "", ""),
	})
	findings2 := LintCriteria(task2)
	if len(findings2) != 1 || findings2[0].Severity != "block" {
		t.Errorf("expected block for empty Then, got %v", findings2)
	}
}

func TestLintCriteria_FindingJsonTags(t *testing.T) {
	f := Finding{
		Severity:   "block",
		Category:   "untestable",
		TaskID:     "task-2",
		AcID:       "ac-1",
		Detail:     "Then is empty",
		Suggestion: "add a concrete outcome",
		Source:     "linter",
	}
	data, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	var out Finding
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if out.Severity != f.Severity {
		t.Errorf("Severity mismatch after round-trip: got %q", out.Severity)
	}
	if out.Category != f.Category {
		t.Errorf("Category mismatch after round-trip: got %q", out.Category)
	}
	if out.TaskID != f.TaskID {
		t.Errorf("TaskID mismatch after round-trip: got %q", out.TaskID)
	}
	if out.AcID != f.AcID {
		t.Errorf("AcID mismatch after round-trip: got %q", out.AcID)
	}
	if out.Source != f.Source {
		t.Errorf("Source mismatch after round-trip: got %q", out.Source)
	}
}
