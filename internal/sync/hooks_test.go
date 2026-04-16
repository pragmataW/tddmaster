
package sync_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	statesync "github.com/pragmataW/tddmaster/internal/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncHooks_CreatesSettingsJSON(t *testing.T) {
	dir := t.TempDir()
	err := statesync.SyncHooks(dir, "tddmaster")
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
	require.NoError(t, err)

	var settings map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &settings))

	hooks, ok := settings["hooks"].(map[string]interface{})
	require.True(t, ok, "hooks should be an object")

	assert.Contains(t, hooks, "PreToolUse")
	assert.Contains(t, hooks, "PostToolUse")
	assert.Contains(t, hooks, "Stop")
	assert.Contains(t, hooks, "SessionStart")
}

func TestSyncHooks_PreservesExistingSettings(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".claude"), 0o755))

	existing := `{"allowedTools": ["*"], "theme": "dark"}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".claude", "settings.json"), []byte(existing), 0o644))

	err := statesync.SyncHooks(dir, "tddmaster")
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
	require.NoError(t, err)

	var settings map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &settings))

	assert.Contains(t, settings, "allowedTools")
	assert.Contains(t, settings, "theme")
	assert.Contains(t, settings, "hooks")
}

func TestBuildClaudeSettings_CommandPrefix(t *testing.T) {
	settings := statesync.BuildClaudeSettings("my-tool invoke")

	hooks, ok := settings["hooks"].(map[string]interface{})
	require.True(t, ok)

	preToolUse, ok := hooks["PreToolUse"].([]interface{})
	require.True(t, ok)
	require.NotEmpty(t, preToolUse)

	first, ok := preToolUse[0].(map[string]interface{})
	require.True(t, ok)

	hooksArr, ok := first["hooks"].([]interface{})
	require.True(t, ok)
	require.NotEmpty(t, hooksArr)

	hookEntry, ok := hooksArr[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "my-tool invoke invoke-hook pre-tool-use", hookEntry["command"])
}
