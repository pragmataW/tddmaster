
package spec

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helpers

func writeSpecMd(t *testing.T, root, specName, content string) {
	t.Helper()
	specDir := filepath.Join(root, ".tddmaster", "specs", specName)
	require.NoError(t, os.MkdirAll(specDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(specDir, "spec.md"), []byte(content), 0644))
}

func readSpecMd(t *testing.T, root, specName string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".tddmaster", "specs", specName, "spec.md"))
	require.NoError(t, err)
	return string(data)
}

func writeProgressJSON(t *testing.T, root, specName string, data map[string]interface{}) {
	t.Helper()
	specDir := filepath.Join(root, ".tddmaster", "specs", specName)
	require.NoError(t, os.MkdirAll(specDir, 0755))
	b, err := json.MarshalIndent(data, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(specDir, "progress.json"), append(b, '\n'), 0644))
}

func readProgressJSON(t *testing.T, root, specName string) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".tddmaster", "specs", specName, "progress.json"))
	require.NoError(t, err)
	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &out))
	return out
}

// =============================================================================
// UpdateSpecStatus
// =============================================================================

func TestUpdateSpecStatus_UpdatesStatusLine(t *testing.T) {
	root := t.TempDir()
	writeSpecMd(t, root, "my-spec", "# Spec: my-spec\n\n## Status: draft\n\nSome content\n")

	err := UpdateSpecStatus(root, "my-spec", "executing")
	require.NoError(t, err)

	content := readSpecMd(t, root, "my-spec")
	assert.Contains(t, content, "## Status: executing")
	assert.NotContains(t, content, "## Status: draft")
}

func TestUpdateSpecStatus_NoErrorWhenFileMissing(t *testing.T) {
	root := t.TempDir()
	err := UpdateSpecStatus(root, "nonexistent", "approved")
	assert.NoError(t, err)
}

// =============================================================================
// MarkTaskCompleted
// =============================================================================

func TestMarkTaskCompleted_MarksTaskAsX(t *testing.T) {
	root := t.TempDir()
	writeSpecMd(t, root, "my-spec", "## Tasks\n\n- [ ] task-1: First task\n- [ ] task-2: Second task\n")

	err := MarkTaskCompleted(root, "my-spec", "task-1")
	require.NoError(t, err)

	content := readSpecMd(t, root, "my-spec")
	assert.Contains(t, content, "- [x] task-1: First task")
	assert.Contains(t, content, "- [ ] task-2: Second task")
}

func TestMarkTaskCompleted_NoErrorWhenFileMissing(t *testing.T) {
	root := t.TempDir()
	err := MarkTaskCompleted(root, "nonexistent", "task-1")
	assert.NoError(t, err)
}

func TestMarkTaskCompleted_DoesNotAffectAlreadyCompletedTask(t *testing.T) {
	root := t.TempDir()
	writeSpecMd(t, root, "my-spec", "## Tasks\n\n- [x] task-1: Already done\n- [ ] task-2: Pending\n")

	err := MarkTaskCompleted(root, "my-spec", "task-2")
	require.NoError(t, err)

	content := readSpecMd(t, root, "my-spec")
	assert.Contains(t, content, "- [x] task-1: Already done")
	assert.Contains(t, content, "- [x] task-2: Pending")
}

// =============================================================================
// UpdateProgressTask
// =============================================================================

func TestUpdateProgressTask_UpdatesTaskStatus(t *testing.T) {
	root := t.TempDir()
	writeProgressJSON(t, root, "my-spec", map[string]interface{}{
		"spec":   "my-spec",
		"status": "draft",
		"tasks": []interface{}{
			map[string]interface{}{"id": "task-1", "title": "First", "status": "pending"},
			map[string]interface{}{"id": "task-2", "title": "Second", "status": "pending"},
		},
	})

	err := UpdateProgressTask(root, "my-spec", "task-1", "done")
	require.NoError(t, err)

	data := readProgressJSON(t, root, "my-spec")
	tasks := data["tasks"].([]interface{})
	task1 := tasks[0].(map[string]interface{})
	task2 := tasks[1].(map[string]interface{})

	assert.Equal(t, "done", task1["status"])
	assert.Equal(t, "pending", task2["status"])
}

func TestUpdateProgressTask_NoErrorWhenFileMissing(t *testing.T) {
	root := t.TempDir()
	err := UpdateProgressTask(root, "nonexistent", "task-1", "done")
	assert.NoError(t, err)
}

// =============================================================================
// UpdateProgressStatus
// =============================================================================

func TestUpdateProgressStatus_UpdatesSpecStatus(t *testing.T) {
	root := t.TempDir()
	writeProgressJSON(t, root, "my-spec", map[string]interface{}{
		"spec":   "my-spec",
		"status": "draft",
		"tasks":  []interface{}{},
	})

	err := UpdateProgressStatus(root, "my-spec", "completed")
	require.NoError(t, err)

	data := readProgressJSON(t, root, "my-spec")
	assert.Equal(t, "completed", data["status"])
}

func TestUpdateProgressStatus_NoErrorWhenFileMissing(t *testing.T) {
	root := t.TempDir()
	err := UpdateProgressStatus(root, "nonexistent", "completed")
	assert.NoError(t, err)
}
