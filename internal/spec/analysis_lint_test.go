package spec

import (
	"testing"
)

func TestBuildLint_ReturnsFindingSlice(t *testing.T) {
	tasks := []Task{
		{
			ID: "task-1",
			Criteria: []Criterion{
				{ID: "ac-1", Given: "given", When: "when", Then: "then"},
			},
		},
	}
	result := BuildLint(tasks)
	var _ []Finding = result
}

func TestBuildLint_TaskWithZeroCriteria_BlockTaskNoAC(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Criteria: nil},
	}
	findings := BuildLint(tasks)
	found := false
	for _, f := range findings {
		if f.Severity == "block" && f.Category == "task-no-ac" && f.TaskID == "task-1" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected block/task-no-ac finding for task-1, got %+v", findings)
	}
}

func TestBuildLint_ExactDuplicateCriteria_WarnDuplicate(t *testing.T) {
	tasks := []Task{
		{
			ID: "task-1",
			Criteria: []Criterion{
				{ID: "ac-1", Given: "a user", When: "submits form", Then: "record is saved"},
				{ID: "ac-2", Given: "a user", When: "submits form", Then: "record is saved"},
			},
		},
	}
	findings := BuildLint(tasks)
	found := false
	for _, f := range findings {
		if f.Severity == "warn" && f.Category == "duplicate" && f.Source == "linter" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warn/duplicate finding for identical criteria, got %+v", findings)
	}
}

func TestBuildLint_DelegatesUntestableToCriterionLint(t *testing.T) {
	tasks := []Task{
		{
			ID: "task-1",
			Criteria: []Criterion{
				{ID: "ac-1", Given: "given", When: "when", Then: ""},
			},
		},
	}
	findings := BuildLint(tasks)
	found := false
	for _, f := range findings {
		if f.Category == "untestable" && f.TaskID == "task-1" && f.AcID == "ac-1" && f.Source == "linter" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected untestable finding from LintCriteria delegation, got %+v", findings)
	}
}

func TestBuildLint_NoSemanticCategories(t *testing.T) {
	tasks := []Task{
		{
			ID: "task-1",
			Criteria: []Criterion{
				{ID: "ac-1", Given: "state A", When: "action X", Then: "output Y"},
				{ID: "ac-2", Given: "state A", When: "action X", Then: "output Z"},
			},
		},
	}
	findings := BuildLint(tasks)
	forbidden := map[string]bool{
		"contradiction":      true,
		"scope-gap":          true,
		"out-of-scope-leak":  true,
		"order":              true,
		"semantic-duplicate": true,
	}
	for _, f := range findings {
		if forbidden[f.Category] {
			t.Errorf("BuildLint must not emit semantic category %q; got finding %+v", f.Category, f)
		}
	}
}

func TestBuildLint_EmptyTasks_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("BuildLint panicked on empty input: %v", r)
		}
	}()
	nilResult := BuildLint(nil)
	emptyResult := BuildLint([]Task{})
	if len(nilResult) != 0 {
		t.Errorf("expected empty result for nil input, got %+v", nilResult)
	}
	if len(emptyResult) != 0 {
		t.Errorf("expected empty result for empty slice, got %+v", emptyResult)
	}
}

func TestBuildLint_DistinctCriteria_NoDuplicateFinding(t *testing.T) {
	tasks := []Task{
		{
			ID: "task-1",
			Criteria: []Criterion{
				{ID: "ac-1", Given: "a user", When: "submits form", Then: "record is saved"},
				{ID: "ac-2", Given: "a user", When: "submits form", Then: "error is shown"},
			},
		},
	}
	findings := BuildLint(tasks)
	for _, f := range findings {
		if f.Category == "duplicate" {
			t.Errorf("expected no duplicate finding for distinct criteria, got %+v", f)
		}
	}
}

func TestBuildLint_SelfDependency_BlockDepSelf(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Criteria: []Criterion{{ID: "ac-1", Given: "g", When: "w", Then: "t"}}, DependsOn: []string{"task-1"}},
	}
	findings := BuildLint(tasks)
	found := false
	for _, f := range findings {
		if f.Severity == "block" && f.Category == "dep-self" && f.TaskID == "task-1" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected block/dep-self finding for task-1, got %+v", findings)
	}
}

func TestBuildLint_UnknownDependency_BlockDepUnknown(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Criteria: []Criterion{{ID: "ac-1", Given: "g", When: "w", Then: "t"}}, DependsOn: []string{"task-9"}},
	}
	findings := BuildLint(tasks)
	found := false
	for _, f := range findings {
		if f.Severity == "block" && f.Category == "dep-unknown" && f.TaskID == "task-1" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected block/dep-unknown finding for task-1, got %+v", findings)
	}
}

func TestBuildLint_DependencyCycle_BlockDepCycle(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Criteria: []Criterion{{ID: "ac-1", Given: "g", When: "w", Then: "t"}}, DependsOn: []string{"task-2"}},
		{ID: "task-2", Criteria: []Criterion{{ID: "ac-1", Given: "g", When: "w", Then: "u"}}, DependsOn: []string{"task-1"}},
	}
	findings := BuildLint(tasks)
	found := false
	for _, f := range findings {
		if f.Severity == "block" && f.Category == "dep-cycle" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected block/dep-cycle finding, got %+v", findings)
	}
}

func TestBuildLint_ValidDependencies_NoDepFindings(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Criteria: []Criterion{{ID: "ac-1", Given: "g", When: "w", Then: "t"}}},
		{ID: "task-2", Criteria: []Criterion{{ID: "ac-1", Given: "g", When: "w", Then: "u"}}, DependsOn: []string{"task-1"}},
	}
	findings := BuildLint(tasks)
	for _, f := range findings {
		if f.Category == "dep-self" || f.Category == "dep-unknown" || f.Category == "dep-cycle" {
			t.Errorf("expected no dependency findings for a valid DAG, got %+v", f)
		}
	}
}
