package spec

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestPhaseInitial(t *testing.T) {
	if PhaseInitial != "spec-settings" {
		t.Fatalf("expected PhaseInitial to be %q, got %q", "spec-settings", PhaseInitial)
	}
}

func TestDefaultSettings(t *testing.T) {
	got := DefaultSettings()
	want := Settings{
		TDDEnabled:               true,
		SkipVerifierEnabled:      false,
		ImportantTaskGateEnabled: false,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DefaultSettings() = %+v, want %+v", got, want)
	}
}

func TestState_JSONTags(t *testing.T) {
	now := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	original := State{
		Version: 1,
		Slug:    "x",
		Phase:   PhaseInitial,
		Answers: map[string][]Answer{
			"q1": {{Key: "k1", Value: "v1"}},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	raw := string(data)
	for _, tag := range []string{`"version"`, `"slug"`, `"phase"`, `"answers"`, `"createdAt"`, `"updatedAt"`, `"key"`, `"value"`} {
		if !strings.Contains(raw, tag) {
			t.Errorf("JSON missing tag %s in: %s", tag, raw)
		}
	}

	var roundTripped State
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if roundTripped.Version != original.Version {
		t.Errorf("Version mismatch: got %d, want %d", roundTripped.Version, original.Version)
	}
	if roundTripped.Slug != original.Slug {
		t.Errorf("Slug mismatch: got %q, want %q", roundTripped.Slug, original.Slug)
	}
	if roundTripped.Phase != original.Phase {
		t.Errorf("Phase mismatch: got %q, want %q", roundTripped.Phase, original.Phase)
	}
	if !reflect.DeepEqual(roundTripped.Answers, original.Answers) {
		t.Errorf("Answers mismatch: got %+v, want %+v", roundTripped.Answers, original.Answers)
	}
	if !roundTripped.CreatedAt.Equal(original.CreatedAt) {
		t.Errorf("CreatedAt mismatch: got %v, want %v", roundTripped.CreatedAt, original.CreatedAt)
	}
	if !roundTripped.UpdatedAt.Equal(original.UpdatedAt) {
		t.Errorf("UpdatedAt mismatch: got %v, want %v", roundTripped.UpdatedAt, original.UpdatedAt)
	}
}

func TestSettings_JSONTags(t *testing.T) {
	s := DefaultSettings()
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	raw := string(data)
	for _, tag := range []string{`"tddEnabled"`, `"skipVerifierEnabled"`, `"importantTaskGateEnabled"`} {
		if !strings.Contains(raw, tag) {
			t.Errorf("JSON missing tag %s in: %s", tag, raw)
		}
	}

	if !strings.Contains(raw, `"tddEnabled":true`) {
		t.Errorf("expected tddEnabled to be true in: %s", raw)
	}
	if !strings.Contains(raw, `"skipVerifierEnabled":false`) {
		t.Errorf("expected skipVerifierEnabled to be false in: %s", raw)
	}
	if !strings.Contains(raw, `"importantTaskGateEnabled":false`) {
		t.Errorf("expected importantTaskGateEnabled to be false in: %s", raw)
	}
}

func TestProgress_JSONTags(t *testing.T) {
	now := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	original := Progress{
		Spec:   "my-spec",
		Status: "draft",
		Tasks: []Task{
			{
				ID:         "t1",
				Title:      "first task",
				AC:         []string{"ac1", "ac2"},
				Done:       false,
				TDDEnabled: true,
				Important:  false,
			},
		},
		UpdatedAt: now,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	raw := string(data)
	for _, tag := range []string{`"spec"`, `"status"`, `"tasks"`, `"id"`, `"title"`, `"ac"`, `"done"`, `"tddEnabled"`, `"important"`} {
		if !strings.Contains(raw, tag) {
			t.Errorf("JSON missing tag %s in: %s", tag, raw)
		}
	}

	var roundTripped Progress
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if roundTripped.Spec != original.Spec {
		t.Errorf("Spec mismatch: got %q, want %q", roundTripped.Spec, original.Spec)
	}
	if roundTripped.Status != original.Status {
		t.Errorf("Status mismatch: got %q, want %q", roundTripped.Status, original.Status)
	}
	if !reflect.DeepEqual(roundTripped.Tasks, original.Tasks) {
		t.Errorf("Tasks mismatch: got %+v, want %+v", roundTripped.Tasks, original.Tasks)
	}
}

func TestProgress_EmptyTasksSerializesAsArray(t *testing.T) {
	p := Progress{
		Spec:   "s",
		Status: "draft",
		Tasks:  []Task{},
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	raw := string(data)
	if !strings.Contains(raw, `"tasks":[]`) {
		t.Errorf("expected tasks to serialize as [] but got: %s", raw)
	}
	if strings.Contains(raw, `"tasks":null`) {
		t.Errorf("tasks must not serialize as null, got: %s", raw)
	}
}

func TestProgress_NilExecution_OmittedFromJSON(t *testing.T) {
	p := Progress{
		Spec:      "s",
		Status:    "draft",
		Tasks:     []Task{},
		Execution: nil,
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	raw := string(data)
	if strings.Contains(raw, `"execution"`) {
		t.Errorf("expected no execution key in JSON when Execution is nil, got: %s", raw)
	}
}

func TestExecState_OnlyIteration_OmitsOptionalFields(t *testing.T) {
	e := ExecState{
		Iteration: 5,
	}

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	raw := string(data)
	if !strings.Contains(raw, `"iteration"`) {
		t.Errorf("expected iteration key in JSON, got: %s", raw)
	}
	for _, key := range []string{`"tddCycle"`, `"refactorRounds"`, `"refactorApplied"`, `"approvedPlans"`, `"planAttempts"`, `"planFeedback"`} {
		if strings.Contains(raw, key) {
			t.Errorf("expected key %s to be omitted when zero, got: %s", key, raw)
		}
	}
}

func TestTaskPlan_JSONRoundTrip(t *testing.T) {
	original := TaskPlan{
		TaskID:         "task-2",
		Assumptions:    []string{"existing tests cover happy path"},
		TouchedFiles:   []string{"internal/foo/bar.go", "internal/foo/bar_test.go"},
		DesignPatterns: []string{"strategy"},
		BestPractices:  []string{"single responsibility"},
		Approach:       "Implement X by extending Y",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	raw := string(data)
	for _, tag := range []string{`"taskId"`, `"assumptions"`, `"touchedFiles"`, `"designPatterns"`, `"bestPractices"`, `"approach"`} {
		if !strings.Contains(raw, tag) {
			t.Errorf("JSON missing tag %s in: %s", tag, raw)
		}
	}

	var got TaskPlan
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if got.TaskID != original.TaskID {
		t.Errorf("TaskID mismatch: got %q, want %q", got.TaskID, original.TaskID)
	}
	if len(got.Assumptions) != len(original.Assumptions) || got.Assumptions[0] != original.Assumptions[0] {
		t.Errorf("Assumptions mismatch: got %v, want %v", got.Assumptions, original.Assumptions)
	}
	if len(got.TouchedFiles) != len(original.TouchedFiles) {
		t.Errorf("TouchedFiles mismatch: got %v, want %v", got.TouchedFiles, original.TouchedFiles)
	}
	if len(got.DesignPatterns) != len(original.DesignPatterns) || got.DesignPatterns[0] != original.DesignPatterns[0] {
		t.Errorf("DesignPatterns mismatch: got %v, want %v", got.DesignPatterns, original.DesignPatterns)
	}
	if len(got.BestPractices) != len(original.BestPractices) || got.BestPractices[0] != original.BestPractices[0] {
		t.Errorf("BestPractices mismatch: got %v, want %v", got.BestPractices, original.BestPractices)
	}
	if got.Approach != original.Approach {
		t.Errorf("Approach mismatch: got %q, want %q", got.Approach, original.Approach)
	}
}

func TestExecState_TaskPlans_RoundTrip(t *testing.T) {
	original := ExecState{
		TaskPlans: map[string]TaskPlan{
			"task-1": {
				TaskID:       "task-1",
				TouchedFiles: []string{"a.go"},
			},
		},
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	raw := string(data)
	if !strings.Contains(raw, `"taskPlans"`) {
		t.Errorf("JSON missing taskPlans key in: %s", raw)
	}
	var got ExecState
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	plan, ok := got.TaskPlans["task-1"]
	if !ok {
		t.Fatalf("task-1 not found in TaskPlans after round-trip")
	}
	if plan.TaskID != "task-1" {
		t.Errorf("TaskID = %q, want task-1", plan.TaskID)
	}
	if len(plan.TouchedFiles) != 1 || plan.TouchedFiles[0] != "a.go" {
		t.Errorf("TouchedFiles = %v, want [a.go]", plan.TouchedFiles)
	}
}

func TestExecState_LastFailedACs_LastUncoveredEC(t *testing.T) {
	original := ExecState{
		LastFailedACs:   []string{"ac-1", "ac-2"},
		LastUncoveredEC: []string{"ec-1"},
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	raw := string(data)
	if !strings.Contains(raw, `"lastFailedACs"`) {
		t.Errorf("JSON missing lastFailedACs key in: %s", raw)
	}
	if !strings.Contains(raw, `"lastUncoveredEC"`) {
		t.Errorf("JSON missing lastUncoveredEC key in: %s", raw)
	}
	var got ExecState
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(got.LastFailedACs, original.LastFailedACs) {
		t.Errorf("LastFailedACs = %v, want %v", got.LastFailedACs, original.LastFailedACs)
	}
	if !reflect.DeepEqual(got.LastUncoveredEC, original.LastUncoveredEC) {
		t.Errorf("LastUncoveredEC = %v, want %v", got.LastUncoveredEC, original.LastUncoveredEC)
	}
}

func TestTask_EdgeCases_RoundTrip(t *testing.T) {
	original := Task{
		ID:         "t1",
		Title:      "some task",
		EdgeCases:  []string{"ec1", "ec2"},
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	raw := string(data)
	if !strings.Contains(raw, `"edgeCases"`) {
		t.Errorf("JSON missing edgeCases key in: %s", raw)
	}
	var got Task
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(got.EdgeCases, original.EdgeCases) {
		t.Errorf("EdgeCases = %v, want %v", got.EdgeCases, original.EdgeCases)
	}
}

func TestExecState_OmitemptyZeroValues(t *testing.T) {
	e := ExecState{}
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	raw := string(data)
	for _, key := range []string{`"taskPlans"`, `"lastFailedACs"`, `"lastUncoveredEC"`} {
		if strings.Contains(raw, key) {
			t.Errorf("expected key %s to be omitted when zero, got: %s", key, raw)
		}
	}
}

func TestState_EmptyAnswersMap(t *testing.T) {
	s := State{
		Version: 1,
		Slug:    "y",
		Phase:   PhaseInitial,
		Answers: map[string][]Answer{},
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	raw := string(data)
	if !strings.Contains(raw, `"answers":{}`) {
		t.Errorf("expected answers to serialize as {} but got: %s", raw)
	}
	if strings.Contains(raw, `"answers":null`) {
		t.Errorf("answers must not serialize as null, got: %s", raw)
	}
}
