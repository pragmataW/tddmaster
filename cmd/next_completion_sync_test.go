package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pragmataW/tddmaster/internal/state"
)

// applyExecutorReport must set OverrideTasks[i].Completed=true for every
// completed task ID so that subsequent spec.md + progress.json regenerations
// render the task as done (not pending).
func TestApplyExecutorReport_MarksOverrideTaskCompleted(t *testing.T) {
	st := state.StateFile{
		Phase: state.PhaseExecuting,
		OverrideTasks: []state.SpecTask{
			{ID: "task-1", Title: "first", Completed: false},
			{ID: "task-2", Title: "second", Completed: false},
			{ID: "task-3", Title: "third", Completed: false},
		},
	}
	report := map[string]interface{}{
		"completed": []interface{}{"task-1", "task-3"},
		"remaining": []interface{}{"task-2"},
	}

	out, err := applyExecutorReport(st, nil, report)
	require.NoError(t, err)

	assert.True(t, out.OverrideTasks[0].Completed, "task-1 must be marked completed")
	assert.False(t, out.OverrideTasks[1].Completed, "task-2 must remain pending")
	assert.True(t, out.OverrideTasks[2].Completed, "task-3 must be marked completed")

	// Execution.CompletedTasks must also still receive the IDs (legacy slice).
	assert.Contains(t, out.Execution.CompletedTasks, "task-1")
	assert.Contains(t, out.Execution.CompletedTasks, "task-3")
}

// A completed ID with no matching OverrideTask must not panic and must still
// be appended to Execution.CompletedTasks.
func TestApplyExecutorReport_NonExistingTaskID_Graceful(t *testing.T) {
	st := state.StateFile{
		Phase:         state.PhaseExecuting,
		OverrideTasks: []state.SpecTask{{ID: "task-1", Title: "first"}},
	}
	report := map[string]interface{}{
		"completed": []interface{}{"task-99"},
	}

	out, err := applyExecutorReport(st, nil, report)
	require.NoError(t, err)
	assert.False(t, out.OverrideTasks[0].Completed)
	assert.Contains(t, out.Execution.CompletedTasks, "task-99")
}

// answerHash must be deterministic and normalize whitespace so retries match.
func TestAnswerHash_Deterministic(t *testing.T) {
	a := answerHash(`{"add":["x"]}`)
	b := answerHash(`  {"add":["x"]}  `)
	c := answerHash(`{"add":["y"]}`)

	assert.Equal(t, a, b, "surrounding whitespace must not change the hash")
	assert.NotEqual(t, a, c, "different payloads must produce different hashes")
	assert.Len(t, a, 16, "hash must be 16 hex chars")
}
