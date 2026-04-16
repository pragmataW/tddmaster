package cmd

import (
	"testing"

	"github.com/pragmataW/tddmaster/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStructuredReport_RecognizesExecutorShape(t *testing.T) {
	report, ok := parseStructuredReport(`{"completed":["ac-1"],"remaining":[]}`)
	require.True(t, ok, "executor report with completed/remaining must be recognized as structured")
	_, hasCompleted := report["completed"]
	assert.True(t, hasCompleted)
}

func TestParseStructuredReport_RecognizesFlatVerifierShape(t *testing.T) {
	report, ok := parseStructuredReport(`{"passed":true,"phase":"red","output":"ok"}`)
	require.True(t, ok)
	assert.Equal(t, true, report["passed"])
}

func TestParseStructuredReport_RecognizesWrappedVerifierShape(t *testing.T) {
	_, ok := parseStructuredReport(`{"tddVerification":{"passed":true,"phase":"red"}}`)
	assert.True(t, ok, "tddVerification wrapper must count as a structured report")
}

func TestParseStructuredReport_RejectsFreeformAnswer(t *testing.T) {
	_, ok := parseStructuredReport(`"I did the thing"`)
	assert.False(t, ok, "bare string must NOT be treated as a structured report")
}

func TestParseStructuredReport_RejectsEmptyObject(t *testing.T) {
	_, ok := parseStructuredReport(`{"somethingElse":"value"}`)
	assert.False(t, ok, "JSON object without known keys must not trigger structured routing")
}

func TestExtractVerifierPayload_FlatPassed(t *testing.T) {
	report := map[string]interface{}{"passed": true, "output": "ok"}
	payload, ok := extractVerifierPayload(report)
	require.True(t, ok)
	assert.Equal(t, true, payload["passed"])
}

func TestExtractVerifierPayload_WrappedTddVerification(t *testing.T) {
	report := map[string]interface{}{
		"tddVerification": map[string]interface{}{
			"passed": false,
			"phase":  "red",
		},
	}
	payload, ok := extractVerifierPayload(report)
	require.True(t, ok, "nested tddVerification.passed must be unwrapped")
	assert.Equal(t, false, payload["passed"])
	assert.Equal(t, "red", payload["phase"])
}

func TestExtractVerifierPayload_WrappedVerificationAlias(t *testing.T) {
	report := map[string]interface{}{
		"verification": map[string]interface{}{"passed": true},
	}
	payload, ok := extractVerifierPayload(report)
	require.True(t, ok, "`verification` alias must also be unwrapped")
	assert.Equal(t, true, payload["passed"])
}

func TestExtractVerifierPayload_ExecutorReportReturnsFalse(t *testing.T) {
	report := map[string]interface{}{"completed": []interface{}{"ac-1"}}
	_, ok := extractVerifierPayload(report)
	assert.False(t, ok, "executor-shaped report must NOT be unwrapped as verifier")
}

func TestExtractVerifierPayload_WrappedWithoutPassedIgnored(t *testing.T) {
	report := map[string]interface{}{
		"tddVerification": map[string]interface{}{"phase": "red"},
	}
	_, ok := extractVerifierPayload(report)
	assert.False(t, ok, "wrapper without a `passed` field must not be treated as verifier")
}

// =============================================================================
// Refinement payload parsing
// =============================================================================

func TestRefinementPayload_StringRefinement(t *testing.T) {
	parsed := map[string]interface{}{"refinement": "rename task-3 to handler.go"}
	payload := refinementPayload(parsed)
	require.NotNil(t, payload)
	// Free-form text is now returned as notes (not text).
	assert.Equal(t, "rename task-3 to handler.go", payload["notes"])
}

func TestRefinementPayload_StructuredRefinement(t *testing.T) {
	parsed := map[string]interface{}{
		"refinement": map[string]interface{}{
			"add":   []interface{}{"A", "B"},
			"notes": "split into smaller steps",
		},
	}
	payload := refinementPayload(parsed)
	require.NotNil(t, payload)
	add, ok := payload["add"].([]interface{})
	require.True(t, ok)
	assert.Len(t, add, 2)
}

func TestRefinementPayload_TopLevelAdd(t *testing.T) {
	parsed := map[string]interface{}{
		"add": []interface{}{"new task"},
	}
	payload := refinementPayload(parsed)
	require.NotNil(t, payload, "top-level `add` must be recognized as a refinement payload")
	assert.Contains(t, payload, "add")
}

func TestRefinementPayload_TopLevelLegacyTasks(t *testing.T) {
	parsed := map[string]interface{}{
		"tasks": []interface{}{"task-1: X"},
	}
	payload := refinementPayload(parsed)
	require.NotNil(t, payload, "top-level legacy `tasks` must be recognized as a refinement payload")
	assert.Contains(t, payload, "tasks")
}

func TestRefinementPayload_EmptyStringIgnored(t *testing.T) {
	parsed := map[string]interface{}{"refinement": "   "}
	payload := refinementPayload(parsed)
	assert.Nil(t, payload, "blank refinement text must not create a payload")
}

func TestRefinementPayload_UnrelatedJSONIgnored(t *testing.T) {
	parsed := map[string]interface{}{"somethingElse": "value"}
	assert.Nil(t, refinementPayload(parsed))
}

func TestRefinementPayload_FreeformStringIsNotes(t *testing.T) {
	// Free-form text refinement must be returned as notes (not expand into tasks).
	parsed := map[string]interface{}{"refinement": "please rename the handler file"}
	payload := refinementPayload(parsed)
	require.NotNil(t, payload)
	_, hasTasks := payload["tasks"]
	_, hasAdd := payload["add"]
	assert.False(t, hasTasks, "free-form refinement must not emit a tasks override")
	assert.False(t, hasAdd, "free-form refinement must not emit an add verb")
	assert.Equal(t, "please rename the handler file", payload["notes"])
}

func TestRefinementPayload_StringWithTaskListRoutesToTasks(t *testing.T) {
	parsed := map[string]interface{}{
		"refinement": "task-1: Title VO | task-2: TodoItem Aggregate | task-3: Repository | task-4: go.mod setup",
	}
	payload := refinementPayload(parsed)
	require.NotNil(t, payload)
	tasks, ok := payload["tasks"].([]interface{})
	require.True(t, ok, "structured task list must become a tasks override")
	assert.Len(t, tasks, 4)
	assert.Equal(t, "Title VO", tasks[0])
	assert.Equal(t, "go.mod setup", tasks[3])
	_, hasNotes := payload["notes"]
	assert.False(t, hasNotes, "structured task list must NOT fall through to notes")
}

func TestRefinementPayload_StringWithNewlineSeparatedTasks(t *testing.T) {
	parsed := map[string]interface{}{"refinement": "task-1: A\ntask-2: B"}
	payload := refinementPayload(parsed)
	require.NotNil(t, payload)
	tasks, ok := payload["tasks"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, []interface{}{"A", "B"}, tasks)
}

func TestRefinementPayload_MixedStringStaysAsNotes(t *testing.T) {
	parsed := map[string]interface{}{
		"refinement": "task-1: A | please also rename the handler",
	}
	payload := refinementPayload(parsed)
	require.NotNil(t, payload)
	assert.Equal(t, "task-1: A | please also rename the handler", payload["notes"])
	_, hasTasks := payload["tasks"]
	assert.False(t, hasTasks)
}

func TestRefinementPayload_CompressedTaskListRoutesToTasks(t *testing.T) {
	// "task-N: title" sınırlarıyla sıkıştırılmış format — nokta+boşluk ayraçlı.
	input := "task-1: Initialize Go module. Files: go.mod, main.go. task-2: Implement TodoItem entity. task-3: Write unit tests. REMOVE task-1 (duplicate)."
	parsed := map[string]interface{}{"refinement": input}
	payload := refinementPayload(parsed)
	require.NotNil(t, payload)
	tasks, ok := payload["tasks"].([]interface{})
	require.True(t, ok, "compressed task list must become a tasks override")
	assert.Len(t, tasks, 3)
	assert.Equal(t, "Initialize Go module. Files: go.mod, main.go", tasks[0])
	assert.Equal(t, "Implement TodoItem entity", tasks[1])
	assert.Equal(t, "Write unit tests", tasks[2])
}

func TestRefinementPayload_CompressedListLeadingProseStaysNotes(t *testing.T) {
	// "please" directivePrefixRe ile eşleşmiyor → leading prose tespit edilip notes'a düşer.
	parsed := map[string]interface{}{"refinement": "please rename: task-1: A task-2: B"}
	payload := refinementPayload(parsed)
	require.NotNil(t, payload)
	assert.NotNil(t, payload["notes"], "leading prose must stay as notes")
	_, hasTasks := payload["tasks"]
	assert.False(t, hasTasks)
}

func TestRefinementPayload_DirectivePrefixedPipeList(t *testing.T) {
	parsed := map[string]interface{}{
		"refinement": "REPLACE all tasks with: task-1: A | task-2: B | task-3: C",
	}
	payload := refinementPayload(parsed)
	require.NotNil(t, payload)
	tasks, ok := payload["tasks"].([]interface{})
	require.True(t, ok, "directive-prefixed pipe list must become tasks")
	assert.Equal(t, []interface{}{"A", "B", "C"}, tasks)
}

func TestRefinementPayload_NumberedSlugTaskList(t *testing.T) {
	input := "1. task-setup: Initialize module\n2. task-entity: Implement TodoItem\n3. task-handlers: HTTP handlers"
	parsed := map[string]interface{}{"refinement": input}
	payload := refinementPayload(parsed)
	require.NotNil(t, payload)
	tasks, ok := payload["tasks"].([]interface{})
	require.True(t, ok, "numbered + slug task IDs must become tasks")
	assert.Len(t, tasks, 3)
	assert.Equal(t, "Initialize module", tasks[0])
	assert.Equal(t, "HTTP handlers", tasks[2])
}

func TestRefinementPayload_DashListWithSlugIDs(t *testing.T) {
	input := "- task-foo: do foo\n- task-bar: do bar"
	parsed := map[string]interface{}{"refinement": input}
	payload := refinementPayload(parsed)
	require.NotNil(t, payload)
	tasks, ok := payload["tasks"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, []interface{}{"do foo", "do bar"}, tasks)
}

func TestRefinementPayload_TopLevelNotesWithTaskListRoutesToTasks(t *testing.T) {
	// Self-review sonrası model refined task'ları yanlışlıkla {"notes": "..."}
	// olarak gönderebiliyor. Bu blob notes'a düşmemeli — tasks override olmalı.
	parsed := map[string]interface{}{
		"notes": "task-1: Init Go module | task-2: TDD RED entity tests | task-3: TDD GREEN entity",
	}
	payload := refinementPayload(parsed)
	require.NotNil(t, payload)
	tasks, ok := payload["tasks"].([]interface{})
	require.True(t, ok, "top-level notes with a task list must become tasks override")
	assert.Equal(t, []interface{}{"Init Go module", "TDD RED entity tests", "TDD GREEN entity"}, tasks)
	_, hasNotes := payload["notes"]
	assert.False(t, hasNotes, "task list must NOT persist as a note blob")
}

func TestRefinementPayload_TopLevelFreeformNotesStayAsNotes(t *testing.T) {
	// Task pattern yoksa notes aynen kalmalı — regresyon testi.
	parsed := map[string]interface{}{"notes": "Tests should cover edge cases"}
	payload := refinementPayload(parsed)
	require.NotNil(t, payload)
	assert.Equal(t, "Tests should cover edge cases", payload["notes"])
	_, hasTasks := payload["tasks"]
	assert.False(t, hasTasks)
}

func TestRefinementPayload_StructuredRefinementNotesWithTaskListRoutes(t *testing.T) {
	// Model {"refinement": {"notes": "task-1: ..."}} yollarsa da redirect olmalı.
	parsed := map[string]interface{}{
		"refinement": map[string]interface{}{
			"notes": "task-1: A | task-2: B",
		},
	}
	payload := refinementPayload(parsed)
	require.NotNil(t, payload)
	tasks, ok := payload["tasks"].([]interface{})
	require.True(t, ok, "refinement.notes with a task list must become tasks override")
	assert.Equal(t, []interface{}{"A", "B"}, tasks)
}

func TestRefinementPayload_StructuredRefinementWithBothVerbsAndFreeformNotes(t *testing.T) {
	// refinement objesinde hem add hem serbest notes varsa notes serbest kalır,
	// verbs de korunur.
	parsed := map[string]interface{}{
		"refinement": map[string]interface{}{
			"add":   []interface{}{"New task"},
			"notes": "Audit trail updated",
		},
	}
	payload := refinementPayload(parsed)
	require.NotNil(t, payload)
	add, ok := payload["add"].([]interface{})
	require.True(t, ok)
	assert.Len(t, add, 1)
	assert.Equal(t, "Audit trail updated", payload["notes"])
}

func TestRefinementPayload_EmptyTopLevelNotesIgnored(t *testing.T) {
	// Boş notes task'a çevrilmez, ama "notes" key'i var olduğu için top-level
	// range'den geçer — parseRefinementVerbs downstream actionable değil diye
	// error verir (mevcut davranış).
	parsed := map[string]interface{}{"notes": "   "}
	payload := refinementPayload(parsed)
	require.NotNil(t, payload)
	_, hasTasks := payload["tasks"]
	assert.False(t, hasTasks, "blank notes must not fabricate a tasks override")
}

func TestRefinementText_PrefersTextOverNotes(t *testing.T) {
	payload := map[string]interface{}{"text": "primary", "notes": "fallback"}
	assert.Equal(t, "primary", refinementText(payload))
}

func TestRefinementText_FallsBackToNotes(t *testing.T) {
	payload := map[string]interface{}{"notes": "only notes"}
	assert.Equal(t, "only notes", refinementText(payload))
}

func TestRefinementText_NoTextFields(t *testing.T) {
	payload := map[string]interface{}{"tasks": []interface{}{"a"}}
	assert.Equal(t, "", refinementText(payload))
}

func TestReadStringArray_FiltersBlanks(t *testing.T) {
	payload := map[string]interface{}{
		"tasks": []interface{}{"task-1", "  ", "task-2", ""},
	}
	out := readStringArray(payload, "tasks")
	assert.Equal(t, []string{"task-1", "task-2"}, out)
}

func TestReadStringArray_MissingKeyReturnsNil(t *testing.T) {
	payload := map[string]interface{}{}
	assert.Nil(t, readStringArray(payload, "tasks"))
}

// =============================================================================
// parseRefinementVerbs — structured verb parsing
// =============================================================================

func TestParseRefinementVerbs_AddVerb(t *testing.T) {
	payload := map[string]interface{}{
		"add": []interface{}{"new task A", "new task B"},
	}
	v, err := parseRefinementVerbs(payload)
	require.NoError(t, err)
	assert.Equal(t, []string{"new task A", "new task B"}, v.Add)
}

func TestParseRefinementVerbs_RemoveVerb(t *testing.T) {
	payload := map[string]interface{}{
		"remove": []interface{}{"task-3", "task-5"},
	}
	v, err := parseRefinementVerbs(payload)
	require.NoError(t, err)
	assert.Equal(t, []string{"task-3", "task-5"}, v.Remove)
}

func TestParseRefinementVerbs_UpdateVerb(t *testing.T) {
	payload := map[string]interface{}{
		"update": map[string]interface{}{"task-1": "new title"},
	}
	v, err := parseRefinementVerbs(payload)
	require.NoError(t, err)
	assert.Equal(t, "new title", v.Update["task-1"])
}

func TestParseRefinementVerbs_NotesOnly(t *testing.T) {
	payload := map[string]interface{}{"notes": "just a note"}
	v, err := parseRefinementVerbs(payload)
	require.NoError(t, err)
	assert.Equal(t, "just a note", v.Notes)
}

func TestParseRefinementVerbs_EmptyPayloadReturnsError(t *testing.T) {
	payload := map[string]interface{}{}
	_, err := parseRefinementVerbs(payload)
	assert.Error(t, err, "empty payload must return error")
}

func TestParseRefinementVerbs_LegacyTasksFormat(t *testing.T) {
	payload := map[string]interface{}{
		"tasks": []interface{}{"task A", "task B"},
	}
	v, err := parseRefinementVerbs(payload)
	require.NoError(t, err)
	assert.Equal(t, []string{"task A", "task B"}, v.Add)
	assert.Equal(t, []string{legacyReplaceAllSentinel}, v.Remove, "legacy format must set replace sentinel")
}

// =============================================================================
// applyTaskRefinement — task list mutation
// =============================================================================

func TestApplyTaskRefinement_AddVerb_AppendsWithIncrementedID(t *testing.T) {
	current := []state.SpecTask{
		{ID: "task-1", Title: "existing"},
		{ID: "task-2", Title: "also existing"},
	}
	verbs := refinementVerbs{Add: []string{"brand new task"}}
	tasks, _, err := applyTaskRefinement(current, verbs, nil)
	require.NoError(t, err)
	require.Len(t, tasks, 3)
	assert.Equal(t, "task-3", tasks[2].ID)
	assert.Equal(t, "brand new task", tasks[2].Title)
}

func TestApplyTaskRefinement_RemoveVerb_RemovesFromCompletedToo(t *testing.T) {
	current := []state.SpecTask{
		{ID: "task-1", Title: "keep"},
		{ID: "task-2", Title: "remove me"},
	}
	completed := []string{"task-2"}
	verbs := refinementVerbs{Remove: []string{"task-2"}}
	tasks, newCompleted, err := applyTaskRefinement(current, verbs, completed)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "task-1", tasks[0].ID)
	assert.NotContains(t, newCompleted, "task-2", "removed task must be cleaned from completed list")
}

func TestApplyTaskRefinement_UpdateVerb_PreservesCompletedStatus(t *testing.T) {
	current := []state.SpecTask{
		{ID: "task-1", Title: "old title", Completed: false},
	}
	completed := []string{"task-1"}
	verbs := refinementVerbs{Update: map[string]string{"task-1": "new title"}}
	tasks, newCompleted, err := applyTaskRefinement(current, verbs, completed)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, "new title", tasks[0].Title)
	assert.Contains(t, newCompleted, "task-1", "updated task ID must remain in completed list")
}

func TestApplyTaskRefinement_UpdateVerb_UnknownIDReturnsError(t *testing.T) {
	current := []state.SpecTask{{ID: "task-1", Title: "exists"}}
	verbs := refinementVerbs{Update: map[string]string{"task-99": "ghost"}}
	_, _, err := applyTaskRefinement(current, verbs, nil)
	assert.Error(t, err)
}

func TestApplyTaskRefinement_LegacyFullReplace(t *testing.T) {
	current := []state.SpecTask{
		{ID: "task-1", Title: "old A"},
		{ID: "task-2", Title: "old B"},
	}
	completed := []string{"task-1"}
	verbs := refinementVerbs{
		Add:    []string{"new A", "new B"},
		Remove: []string{legacyReplaceAllSentinel},
	}
	tasks, newCompleted, err := applyTaskRefinement(current, verbs, completed)
	require.NoError(t, err)
	assert.Len(t, tasks, 2)
	assert.Equal(t, "task-1", tasks[0].ID, "legacy replace renumbers from 1")
	assert.Equal(t, "new A", tasks[0].Title)
	assert.Nil(t, newCompleted, "completed list must be cleared on full replace")
}

func TestApplyTaskRefinement_AddSkipsBlanks(t *testing.T) {
	current := []state.SpecTask{{ID: "task-1", Title: "existing"}}
	verbs := refinementVerbs{Add: []string{"", "   ", "valid task"}}
	tasks, _, err := applyTaskRefinement(current, verbs, nil)
	require.NoError(t, err)
	assert.Len(t, tasks, 2, "blank add entries must be skipped")
}
