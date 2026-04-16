package spec

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pragmataW/tddmaster/internal/state"
)

// When OverrideTasks[i].Completed is true, GenerateSpec must write the task
// as status "done" in progress.json — overriding any stale "pending" value
// in the existing progress.json. This keeps progress.json in sync with the
// state-of-truth across refinement regenerations (fix for the bug where
// tasks reappeared as pending after a refinement).
func TestGenerateSpec_ProgressJsonUsesOverrideCompletedAsSource(t *testing.T) {
	root := t.TempDir()
	specName := "sync-test"

	// Minimal state: a spec with two override tasks, first one completed.
	userContext := "sync context"
	st := &state.StateFile{
		Spec: &specName,
		Discovery: state.DiscoveryState{
			UserContext: &userContext,
		},
		OverrideTasks: []state.SpecTask{
			{ID: "task-1", Title: "first", Completed: true},
			{ID: "task-2", Title: "second", Completed: false},
		},
	}

	// Pre-seed progress.json with STALE "pending" for task-1 — this is what
	// would exist if earlier writes had not updated it. The fix must override
	// it based on st.OverrideTasks[].Completed.
	p := state.Paths{}
	specDir := filepath.Join(root, p.SpecDir(specName))
	require.NoError(t, os.MkdirAll(specDir, 0o755))
	progressPath := filepath.Join(specDir, "progress.json")
	stalePayload := map[string]interface{}{
		"spec":   specName,
		"status": "draft",
		"tasks": []map[string]string{
			{"id": "task-1", "title": "first", "status": "pending"},
			{"id": "task-2", "title": "second", "status": "pending"},
		},
		"decisions": []interface{}{},
		"debt":      []interface{}{},
		"updatedAt": "2020-01-01T00:00:00Z",
	}
	staleBytes, _ := json.MarshalIndent(stalePayload, "", "  ")
	require.NoError(t, os.WriteFile(progressPath, staleBytes, 0o644))

	_, err := GenerateSpec(root, st, nil)
	require.NoError(t, err)

	// Re-read the regenerated progress.json.
	raw, err := os.ReadFile(progressPath)
	require.NoError(t, err)
	var got struct {
		Tasks []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"tasks"`
	}
	require.NoError(t, json.Unmarshal(raw, &got))

	statusByID := map[string]string{}
	for _, task := range got.Tasks {
		statusByID[task.ID] = task.Status
	}
	assert.Equal(t, "done", statusByID["task-1"],
		"OverrideTasks[].Completed=true must drive progress.json status=done")
	assert.Equal(t, "pending", statusByID["task-2"],
		"non-completed tasks remain pending")
}
