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
		MinTestCoverage:          80,
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
				Criteria:   []Criterion{{ID: "ac-1", Then: "ac1"}, {ID: "ac-2", Then: "ac2"}},
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
	for _, tag := range []string{`"spec"`, `"status"`, `"tasks"`, `"id"`, `"title"`, `"criteria"`, `"done"`, `"tddEnabled"`, `"important"`} {
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

func TestTask_NilExec_OmittedFromJSON(t *testing.T) {
	p := Progress{
		Spec:   "s",
		Status: "draft",
		Tasks:  []Task{{ID: "task-1", Title: "t", Exec: nil}},
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	raw := string(data)
	if strings.Contains(raw, `"exec"`) {
		t.Errorf("expected no exec key in JSON when Exec is nil, got: %s", raw)
	}
}

func TestExecState_ZeroValue_OmitsOptionalFields(t *testing.T) {
	e := ExecState{}

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	raw := string(data)
	for _, key := range []string{`"tddCycle"`, `"refactorRounds"`, `"refactorApplied"`, `"planApproved"`, `"planAttempts"`, `"planFeedback"`, `"plan"`, `"worktree"`} {
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

func TestExecState_Plan_RoundTrip(t *testing.T) {
	original := ExecState{
		Plan: &TaskPlan{
			TaskID:       "task-1",
			TouchedFiles: []string{"a.go"},
		},
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	raw := string(data)
	if !strings.Contains(raw, `"plan"`) {
		t.Errorf("JSON missing plan key in: %s", raw)
	}
	var got ExecState
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if got.Plan == nil {
		t.Fatalf("Plan nil after round-trip")
	}
	if got.Plan.TaskID != "task-1" {
		t.Errorf("TaskID = %q, want task-1", got.Plan.TaskID)
	}
	if len(got.Plan.TouchedFiles) != 1 || got.Plan.TouchedFiles[0] != "a.go" {
		t.Errorf("TouchedFiles = %v, want [a.go]", got.Plan.TouchedFiles)
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
		ID:        "t1",
		Title:     "some task",
		EdgeCases: []string{"ec1", "ec2"},
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

func TestCriterion_JSONRoundTrip(t *testing.T) {
	original := Criterion{
		ID:    "ac-1",
		Given: "a running service",
		When:  "a request arrives",
		Then:  "a response is returned",
		Raw:   "Given a running service When a request arrives Then a response is returned",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	raw := string(data)
	for _, tag := range []string{`"id"`, `"given"`, `"when"`, `"then"`, `"raw"`} {
		if !strings.Contains(raw, tag) {
			t.Errorf("JSON missing tag %s in: %s", tag, raw)
		}
	}

	var got Criterion
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if got.ID != original.ID {
		t.Errorf("ID: got %q, want %q", got.ID, original.ID)
	}
	if got.Given != original.Given {
		t.Errorf("Given: got %q, want %q", got.Given, original.Given)
	}
	if got.When != original.When {
		t.Errorf("When: got %q, want %q", got.When, original.When)
	}
	if got.Then != original.Then {
		t.Errorf("Then: got %q, want %q", got.Then, original.Then)
	}
	if got.Raw != original.Raw {
		t.Errorf("Raw: got %q, want %q", got.Raw, original.Raw)
	}
}

func TestCriterion_JSONOmitsEmptyOptionalFields(t *testing.T) {
	c := Criterion{
		ID:   "ac-2",
		Then: "something happens",
	}

	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	raw := string(data)
	if !strings.Contains(raw, `"then"`) {
		t.Errorf("JSON must always contain then key, got: %s", raw)
	}
	for _, tag := range []string{`"given"`, `"when"`, `"raw"`} {
		if strings.Contains(raw, tag) {
			t.Errorf("JSON must omit %s when empty, got: %s", tag, raw)
		}
	}
}

func TestTask_Criteria_JSONKey(t *testing.T) {
	task := Task{
		ID:    "task-1",
		Title: "some task",
		Criteria: []Criterion{
			{ID: "ac-1", Then: "it works"},
		},
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	raw := string(data)
	if !strings.Contains(raw, `"criteria"`) {
		t.Errorf("JSON must contain criteria key, got: %s", raw)
	}
	if strings.Contains(raw, `"ac"`) {
		t.Errorf("JSON must not contain ac key, got: %s", raw)
	}
}

func TestTask_Criteria_RoundTrip(t *testing.T) {
	original := Task{
		ID:    "task-1",
		Title: "some task",
		Criteria: []Criterion{
			{ID: "ac-1", Then: "first thing works"},
			{ID: "ac-2", Given: "precondition", Then: "second thing works"},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var got Task
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(got.Criteria) != len(original.Criteria) {
		t.Fatalf("Criteria length: got %d, want %d", len(got.Criteria), len(original.Criteria))
	}
	if got.Criteria[0].ID != "ac-1" {
		t.Errorf("Criteria[0].ID: got %q, want ac-1", got.Criteria[0].ID)
	}
	if got.Criteria[1].Given != "precondition" {
		t.Errorf("Criteria[1].Given: got %q, want precondition", got.Criteria[1].Given)
	}
}

func TestAssignCriterionIDs_FreshTask(t *testing.T) {
	task := &Task{
		ID:    "task-1",
		Title: "fresh task",
		Criteria: []Criterion{
			{Then: "first thing works"},
			{Then: "second thing works"},
			{Then: "third thing works"},
		},
	}

	AssignCriterionIDs(task)

	if task.Criteria[0].ID != "ac-1" {
		t.Errorf("Criteria[0].ID: got %q, want ac-1", task.Criteria[0].ID)
	}
	if task.Criteria[1].ID != "ac-2" {
		t.Errorf("Criteria[1].ID: got %q, want ac-2", task.Criteria[1].ID)
	}
	if task.Criteria[2].ID != "ac-3" {
		t.Errorf("Criteria[2].ID: got %q, want ac-3", task.Criteria[2].ID)
	}
}

func TestAssignCriterionIDs_Idempotent(t *testing.T) {
	task := &Task{
		ID:    "task-1",
		Title: "idempotent task",
		Criteria: []Criterion{
			{Then: "first thing works"},
			{Then: "second thing works"},
		},
	}

	AssignCriterionIDs(task)
	firstID := task.Criteria[0].ID
	secondID := task.Criteria[1].ID

	AssignCriterionIDs(task)

	if task.Criteria[0].ID != firstID {
		t.Errorf("Criteria[0].ID changed on second run: got %q, want %q", task.Criteria[0].ID, firstID)
	}
	if task.Criteria[1].ID != secondID {
		t.Errorf("Criteria[1].ID changed on second run: got %q, want %q", task.Criteria[1].ID, secondID)
	}
}

func TestAssignCriterionIDs_RefineCollision(t *testing.T) {
	task := &Task{
		ID:    "task-1",
		Title: "refine task",
		Criteria: []Criterion{
			{ID: "ac-1", Then: "existing first"},
			{ID: "ac-3", Then: "existing third"},
			{Then: "new criterion added after refine"},
		},
	}

	AssignCriterionIDs(task)

	if task.Criteria[0].ID != "ac-1" {
		t.Errorf("Criteria[0].ID must remain ac-1, got %q", task.Criteria[0].ID)
	}
	if task.Criteria[1].ID != "ac-3" {
		t.Errorf("Criteria[1].ID must remain ac-3, got %q", task.Criteria[1].ID)
	}
	if task.Criteria[2].ID != "ac-4" {
		t.Errorf("Criteria[2].ID must be ac-4 (max+1), got %q", task.Criteria[2].ID)
	}
}

func TestTraceEntry_JSONTags(t *testing.T) {
	entry := TraceEntry{
		FunctionName: "TestFoo_Bar",
		TaskID:       "task-1",
		CriterionIDs: []string{"ac-1"},
		EC:           []string{"EC-3"},
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	raw := string(data)
	for _, tag := range []string{`"functionName"`, `"taskId"`, `"criterionIds"`, `"ec"`} {
		if !strings.Contains(raw, tag) {
			t.Errorf("JSON missing tag %s in: %s", tag, raw)
		}
	}
}

func TestTraceEntry_JSONRoundTrip(t *testing.T) {
	original := TraceEntry{
		FunctionName: "TestDoSomething",
		TaskID:       "task-2",
		CriterionIDs: []string{"must handle nil input"},
		EC:           []string{"EC-1"},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var got TraceEntry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if got.FunctionName != original.FunctionName {
		t.Errorf("FunctionName: got %q, want %q", got.FunctionName, original.FunctionName)
	}
	if got.TaskID != original.TaskID {
		t.Errorf("TaskID: got %q, want %q", got.TaskID, original.TaskID)
	}
	if !reflect.DeepEqual(got.CriterionIDs, original.CriterionIDs) {
		t.Errorf("CriterionIDs: got %q, want %q", got.CriterionIDs, original.CriterionIDs)
	}
	if !reflect.DeepEqual(got.EC, original.EC) {
		t.Errorf("EC: got %q, want %q", got.EC, original.EC)
	}
}

func TestTraceability_IsStruct(t *testing.T) {
	tr := Traceability{
		Entries: map[string][]TraceEntry{
			"task-1": {
				{FunctionName: "TestAlpha", TaskID: "task-1", CriterionIDs: []string{"ac-1"}, EC: nil},
			},
		},
		Coverage: map[string]map[string]float64{
			"task-1": {"internal/spec/model.go": 80},
		},
	}

	if tr.Entries == nil {
		t.Fatal("Entries must not be nil after explicit init")
	}
	if tr.Coverage == nil {
		t.Fatal("Coverage must not be nil after explicit init")
	}
}

func TestTraceability_JSONRoundTrip(t *testing.T) {
	original := Traceability{
		Entries: map[string][]TraceEntry{
			"task-1": {
				{FunctionName: "TestAlpha", TaskID: "task-1", CriterionIDs: []string{"ac-1"}, EC: nil},
				{FunctionName: "TestBeta", TaskID: "task-1", CriterionIDs: []string{"ac-2"}, EC: []string{"EC-1"}},
			},
			"task-2": {
				{FunctionName: "TestGamma", TaskID: "task-2", CriterionIDs: []string{"ac-1"}, EC: []string{"EC-2"}},
			},
		},
		Coverage: map[string]map[string]float64{
			"task-1": {"internal/spec/model.go": 90},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var got Traceability
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(got.Entries) != len(original.Entries) {
		t.Fatalf("Entries length: got %d, want %d", len(got.Entries), len(original.Entries))
	}

	for taskID, entries := range original.Entries {
		gotEntries, ok := got.Entries[taskID]
		if !ok {
			t.Errorf("missing key %q in round-tripped Entries", taskID)
			continue
		}
		if len(gotEntries) != len(entries) {
			t.Errorf("task %q: entry count got %d, want %d", taskID, len(gotEntries), len(entries))
			continue
		}
		for i, e := range entries {
			if gotEntries[i].FunctionName != e.FunctionName {
				t.Errorf("task %q[%d].FunctionName: got %q, want %q", taskID, i, gotEntries[i].FunctionName, e.FunctionName)
			}
			if gotEntries[i].TaskID != e.TaskID {
				t.Errorf("task %q[%d].TaskID: got %q, want %q", taskID, i, gotEntries[i].TaskID, e.TaskID)
			}
			if !reflect.DeepEqual(gotEntries[i].CriterionIDs, e.CriterionIDs) {
				t.Errorf("task %q[%d].CriterionIDs: got %q, want %q", taskID, i, gotEntries[i].CriterionIDs, e.CriterionIDs)
			}
			if !reflect.DeepEqual(gotEntries[i].EC, e.EC) {
				t.Errorf("task %q[%d].EC: got %q, want %q", taskID, i, gotEntries[i].EC, e.EC)
			}
		}
	}

	if len(got.Coverage) != len(original.Coverage) {
		t.Fatalf("Coverage length: got %d, want %d", len(got.Coverage), len(original.Coverage))
	}
	for k, v := range original.Coverage {
		gotV, ok := got.Coverage[k]
		if !ok {
			t.Errorf("Coverage missing key %q", k)
			continue
		}
		if !reflect.DeepEqual(gotV, v) {
			t.Errorf("Coverage[%q]: got %v, want %v", k, gotV, v)
		}
	}
}

func TestTraceability_JSONTags(t *testing.T) {
	tr := Traceability{
		Entries:  map[string][]TraceEntry{},
		Coverage: map[string]map[string]float64{"task-1": {"src/main.go": 50}},
	}

	data, err := json.Marshal(tr)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	raw := string(data)
	if !strings.Contains(raw, `"entries"`) {
		t.Errorf("JSON missing tag %q in: %s", "entries", raw)
	}
	if !strings.Contains(raw, `"coverage"`) {
		t.Errorf("JSON missing tag %q in: %s", "coverage", raw)
	}
}

func TestTraceability_CoverageOmittedWhenEmpty(t *testing.T) {
	tr := Traceability{
		Entries:  map[string][]TraceEntry{},
		Coverage: nil,
	}

	data, err := json.Marshal(tr)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	raw := string(data)
	if strings.Contains(raw, `"coverage"`) {
		t.Errorf("coverage should be omitted when nil, got: %s", raw)
	}
}

func TestTraceability_EmptyEntries(t *testing.T) {
	tr := Traceability{
		Entries:  map[string][]TraceEntry{},
		Coverage: map[string]map[string]float64{},
	}

	data, err := json.Marshal(tr)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var got Traceability
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(got.Entries) != 0 {
		t.Errorf("expected empty Entries, got length %d", len(got.Entries))
	}
}

func TestTraceEntry_CriterionIDs_JSONKey(t *testing.T) {
	entry := TraceEntry{
		FunctionName: "TestSomething",
		TaskID:       "task-1",
		CriterionIDs: []string{"ac-1", "ac-2"},
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	raw := string(data)
	if !strings.Contains(raw, `"criterionIds"`) {
		t.Errorf("JSON must contain criterionIds key, got: %s", raw)
	}
	if strings.Contains(raw, `"ac"`) {
		t.Errorf("JSON must not contain ac key, got: %s", raw)
	}
}

func TestTraceEntry_CriterionIDs_RoundTrip(t *testing.T) {
	original := TraceEntry{
		FunctionName: "TestDoWork",
		TaskID:       "task-2",
		CriterionIDs: []string{"ac-1", "ac-3"},
		EC:           []string{"EC-1"},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var got TraceEntry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if got.FunctionName != original.FunctionName {
		t.Errorf("FunctionName: got %q, want %q", got.FunctionName, original.FunctionName)
	}
	if got.TaskID != original.TaskID {
		t.Errorf("TaskID: got %q, want %q", got.TaskID, original.TaskID)
	}
	if !reflect.DeepEqual(got.CriterionIDs, original.CriterionIDs) {
		t.Errorf("CriterionIDs: got %v, want %v", got.CriterionIDs, original.CriterionIDs)
	}
	if !reflect.DeepEqual(got.EC, original.EC) {
		t.Errorf("EC: got %v, want %v", got.EC, original.EC)
	}
}

func TestDefaultSettings_MinTestCoverage(t *testing.T) {
	got := DefaultSettings()
	if got.MinTestCoverage != 80 {
		t.Fatalf("DefaultSettings().MinTestCoverage = %d, want 80", got.MinTestCoverage)
	}
}

func TestSettings_MinTestCoverage_JSONRoundTrip(t *testing.T) {
	s := Settings{
		TDDEnabled:               true,
		SkipVerifierEnabled:      false,
		ImportantTaskGateEnabled: false,
		MinTestCoverage:          75,
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	if !strings.Contains(string(data), `"minTestCoverage"`) {
		t.Fatalf("JSON missing minTestCoverage key, got: %s", data)
	}

	var got Settings
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if got.MinTestCoverage != 75 {
		t.Fatalf("round-trip MinTestCoverage = %d, want 75", got.MinTestCoverage)
	}
}

func TestSettings_MinTestCoverage_OmittedUsesDefault(t *testing.T) {
	raw := `{"tddEnabled":true,"skipVerifierEnabled":false,"importantTaskGateEnabled":false}`

	base := DefaultSettings()
	if err := json.Unmarshal([]byte(raw), &base); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if base.MinTestCoverage != 80 {
		t.Fatalf("MinTestCoverage after unmarshal of omitted field = %d, want 80", base.MinTestCoverage)
	}
}

func TestSettings_MinTestCoverage_ExplicitZeroRoundTrips(t *testing.T) {
	s := Settings{
		TDDEnabled:               true,
		SkipVerifierEnabled:      false,
		ImportantTaskGateEnabled: false,
		MinTestCoverage:          0,
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var got Settings
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if got.MinTestCoverage != 0 {
		t.Fatalf("round-trip MinTestCoverage = %d, want 0", got.MinTestCoverage)
	}
}

func TestSettings_ClampCoverage(t *testing.T) {
	cases := []struct {
		in   int
		want int
	}{
		{-5, 0},
		{0, 0},
		{80, 80},
		{100, 100},
		{150, 100},
	}
	for _, c := range cases {
		s := Settings{MinTestCoverage: c.in}
		s.ClampCoverage()
		if s.MinTestCoverage != c.want {
			t.Errorf("ClampCoverage(%d) = %d, want %d", c.in, s.MinTestCoverage, c.want)
		}
	}
}

func TestExecState_LastModifiedFiles_JSONRoundTrip(t *testing.T) {
	es := ExecState{
		LastModifiedFiles: []string{"a.go", "b.go"},
	}
	data, err := json.Marshal(es)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if !strings.Contains(string(data), `"lastModifiedFiles"`) {
		t.Fatalf("JSON missing lastModifiedFiles key, got: %s", data)
	}
	var got ExecState
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(got.LastModifiedFiles, es.LastModifiedFiles) {
		t.Fatalf("round-trip LastModifiedFiles = %v, want %v", got.LastModifiedFiles, es.LastModifiedFiles)
	}
}

func TestExecState_LastCoverage_JSONRoundTrip(t *testing.T) {
	es := ExecState{
		LastCoverage: map[string]float64{"task-1": 92, "task-2": 85},
	}

	data, err := json.Marshal(es)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	if !strings.Contains(string(data), `"lastCoverage"`) {
		t.Fatalf("JSON missing lastCoverage key, got: %s", data)
	}

	var got ExecState
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if !reflect.DeepEqual(got.LastCoverage, es.LastCoverage) {
		t.Fatalf("round-trip LastCoverage = %v, want %v", got.LastCoverage, es.LastCoverage)
	}
}

func TestExecState_LastCoverage_OmitEmptyWhenNil(t *testing.T) {
	es := ExecState{TDDCycle: "red"}

	data, err := json.Marshal(es)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	if strings.Contains(string(data), `"lastCoverage"`) {
		t.Fatalf("expected lastCoverage to be omitted when nil, got: %s", data)
	}
}

func TestTask_HasNoCoverageField(t *testing.T) {
	task := Task{}
	rt := reflect.TypeOf(task)

	for i := 0; i < rt.NumField(); i++ {
		name := strings.ToLower(rt.Field(i).Name)
		if strings.Contains(name, "coverage") {
			t.Fatalf("Task struct must not have a coverage-related field, found: %s", rt.Field(i).Name)
		}
	}
}
