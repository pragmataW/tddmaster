package execution

import (
	"testing"

	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/spec"
	"github.com/pragmataW/tddmaster/internal/state"
	"github.com/stretchr/testify/assert"
)

type stubRenderer struct{}

func (stubRenderer) C(sub string) string                  { return "tddmaster " + sub }
func (stubRenderer) CS(sub string, _ *string) string      { return "tddmaster " + sub }

func boolPtr(b bool) *bool       { return &b }
func stringPtr(s string) *string { return &s }

func baseImportantState(taskID string) state.StateFile {
	specName := "demo"
	imp := true
	return state.StateFile{
		Spec:  &specName,
		Phase: state.PhaseExecuting,
		OverrideTasks: []state.SpecTask{
			{ID: taskID, Title: "Add password reset", Important: &imp},
		},
		Execution: state.ExecutionState{
			Iteration: 0,
		},
	}
}

func importantParsedSpec(taskID string) *spec.ParsedSpec {
	return &spec.ParsedSpec{
		Name:  "demo",
		Tasks: []spec.ParsedTask{{ID: taskID, Title: "Add password reset"}},
	}
}

func TestImportantGate_BlocksExecutionUntilPlanApproved(t *testing.T) {
	st := baseImportantState("task-1")
	out := Compile(stubRenderer{}, st, nil, nil, 50, importantParsedSpec("task-1"), true)

	assert.NotNil(t, out.ImportantTaskGate, "gate block must be present when task important and no plan approved")
	if out.ImportantTaskGate != nil {
		assert.Equal(t, "task-1", out.ImportantTaskGate.TaskID)
		assert.Equal(t, "tddmaster-planner", out.ImportantTaskGate.DelegateAgent)
		assert.Equal(t, "planning", out.ImportantTaskGate.Phase)
		assert.Equal(t, []string{"accept", "revise", "reject"}, out.ImportantTaskGate.UserReviewOptions)
		assert.Contains(t, out.Instruction, "IMPORTANT TASK GATE")
	}
}

func TestImportantGate_Disabled_NoBlock(t *testing.T) {
	st := baseImportantState("task-1")
	out := Compile(stubRenderer{}, st, nil, nil, 50, importantParsedSpec("task-1"), false)

	assert.Nil(t, out.ImportantTaskGate, "gate must not fire when manifest gate is disabled")
	assert.NotContains(t, out.Instruction, "IMPORTANT TASK GATE")
}

func TestImportantGate_NonImportantTask_NoBlock(t *testing.T) {
	specName := "demo"
	st := state.StateFile{
		Spec:  &specName,
		Phase: state.PhaseExecuting,
		OverrideTasks: []state.SpecTask{
			{ID: "task-1", Title: "Routine bump"}, // Important nil
		},
	}
	out := Compile(stubRenderer{}, st, nil, nil, 50, importantParsedSpec("task-1"), true)

	assert.Nil(t, out.ImportantTaskGate, "gate must not fire on non-important tasks")
}

func TestImportantGate_PlanAlreadyApproved_NoBlock(t *testing.T) {
	st := baseImportantState("task-1")
	st.Execution.ApprovedImportantPlans = []string{"task-1"}

	out := Compile(stubRenderer{}, st, nil, nil, 50, importantParsedSpec("task-1"), true)

	assert.Nil(t, out.ImportantTaskGate, "approved plan must clear the gate")
}

func TestImportantGate_RetryReusesPriorFeedback(t *testing.T) {
	st := baseImportantState("task-1")
	st.Execution.PendingPlanAttempts = map[string]int{"task-1": 1}
	st.Execution.LastPlanFeedback = map[string]string{"task-1": "missing rate-limit detail"}

	out := Compile(stubRenderer{}, st, nil, nil, 50, importantParsedSpec("task-1"), true)

	if assert.NotNil(t, out.ImportantTaskGate) {
		assert.Equal(t, "review", out.ImportantTaskGate.Phase, "after at least one attempt the gate is in review phase")
		assert.Equal(t, 1, out.ImportantTaskGate.AttemptCount)
		if assert.NotNil(t, out.ImportantTaskGate.PriorFeedback) {
			assert.Equal(t, "missing rate-limit detail", *out.ImportantTaskGate.PriorFeedback)
		}
		assert.Contains(t, out.Instruction, "missing rate-limit detail")
	}
}

func TestImportantGate_PlanSchemaCarriesAssumptions(t *testing.T) {
	st := baseImportantState("task-1")
	out := Compile(stubRenderer{}, st, nil, nil, 50, importantParsedSpec("task-1"), true)

	if assert.NotNil(t, out.ImportantTaskGate) {
		assert.Contains(t, out.ImportantTaskGate.PlanSchema.Fields, "assumptions")
		assert.Contains(t, out.ImportantTaskGate.PlanSchema.Fields, "touchedFiles")
		assert.Contains(t, out.ImportantTaskGate.PlanSchema.Fields, "designPatterns")
		assert.Contains(t, out.ImportantTaskGate.PlanSchema.Fields, "bestPractices")
		assert.Contains(t, out.ImportantTaskGate.PlanSchema.Fields, "approach")
	}
}

func TestImportantGate_TaskBlockStillPresent(t *testing.T) {
	st := baseImportantState("task-1")
	out := Compile(stubRenderer{}, st, nil, nil, 50, importantParsedSpec("task-1"), true)

	// The task block (id/title) must remain so the orchestrator can pass scope
	// to tddmaster-planner.
	if assert.NotNil(t, out.Task) {
		assert.Equal(t, "task-1", out.Task.ID)
	}
}

// Silence unused warnings for helpers reserved for future expansion.
var (
	_ = stringPtr
	_ = boolPtr
	_ = model.DefaultImportantPlanShape
)
