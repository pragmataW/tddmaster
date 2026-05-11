package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pragmataW/tddmaster/internal/spec/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFakeProgress(t *testing.T, root, specName string, body string) string {
	t.Helper()
	dir := filepath.Join(root, ".tddmaster", "specs", specName)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	path := filepath.Join(dir, "progress.json")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	return path
}

func TestMarkTaskImportant_SetsAndClearsFlag(t *testing.T) {
	root := t.TempDir()
	path := writeFakeProgress(t, root, "demo", `{
	  "spec": "demo",
	  "status": "draft",
	  "tasks": [
	    {"id": "task-1", "title": "Add reset", "status": "pending"},
	    {"id": "task-2", "title": "Refactor", "status": "pending"}
	  ],
	  "decisions": [],
	  "debt": [],
	  "updatedAt": "2026-01-01T00:00:00Z"
	}`)

	require.NoError(t, MarkTaskImportant(root, "demo", "task-1", true))

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var pf model.ProgressFile
	require.NoError(t, json.Unmarshal(raw, &pf))
	require.Len(t, pf.Tasks, 2)
	assert.True(t, pf.Tasks[0].Important, "task-1 should be marked important")
	assert.False(t, pf.Tasks[1].Important, "task-2 should remain unmarked")

	// Toggle back off.
	require.NoError(t, MarkTaskImportant(root, "demo", "task-1", false))
	raw, err = os.ReadFile(path)
	require.NoError(t, err)
	var pfClear model.ProgressFile
	require.NoError(t, json.Unmarshal(raw, &pfClear))
	assert.False(t, pfClear.Tasks[0].Important, "task-1 should be cleared")
}

func TestAppendTaskPlan_NewAndReplace(t *testing.T) {
	root := t.TempDir()
	writeFakeProgress(t, root, "demo", `{
	  "spec": "demo",
	  "status": "draft",
	  "tasks": [{"id":"task-1","title":"x","status":"pending"}],
	  "decisions": [],
	  "debt": [],
	  "updatedAt": "2026-01-01T00:00:00Z"
	}`)

	first := model.ProgressTaskPlan{
		TaskID:         "task-1",
		Assumptions:    []string{"existing schema is fine"},
		TouchedFiles:   []string{"api/auth.go"},
		DesignPatterns: []string{"Command"},
		BestPractices:  []string{"validate at boundary"},
		Approach:       "Add reset endpoint guarded by rate-limit middleware.",
		AttemptCount:   1,
		ApprovedAt:     "2026-01-02T00:00:00Z",
		ApprovedBy:     "Alice",
	}
	require.NoError(t, AppendTaskPlan(root, "demo", first))

	got, err := LoadTaskPlan(root, "demo", "task-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, first.Approach, got.Approach)
	assert.Equal(t, []string{"api/auth.go"}, got.TouchedFiles)

	// Replace: same TaskID, new content.
	second := first
	second.Approach = "Updated narrative after revise/reject cycle."
	second.AttemptCount = 3
	require.NoError(t, AppendTaskPlan(root, "demo", second))

	got, err = LoadTaskPlan(root, "demo", "task-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Updated narrative after revise/reject cycle.", got.Approach)
	assert.Equal(t, 3, got.AttemptCount, "AttemptCount must reflect the latest submission")
}

func TestLoadTaskPlan_AbsentReturnsNil(t *testing.T) {
	root := t.TempDir()
	writeFakeProgress(t, root, "demo", `{"spec":"demo","status":"draft","tasks":[],"decisions":[],"debt":[],"updatedAt":""}`)

	got, err := LoadTaskPlan(root, "demo", "task-999")
	require.NoError(t, err)
	assert.Nil(t, got, "missing plan must return nil, not error")
}
