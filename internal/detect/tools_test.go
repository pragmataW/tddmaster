
package detect

import (
	"path/filepath"
	"testing"

	"github.com/pragmataW/tddmaster/internal/state"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// DetectCodingTools
// =============================================================================

func TestDetectCodingTools_ClaudeCode_CLAUDE_md(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"CLAUDE.md": "# Claude\n"})
	tools := DetectCodingTools(dir)
	assert.Contains(t, tools, state.CodingToolClaudeCode)
}

func TestDetectCodingTools_ClaudeCode_DotClaude(t *testing.T) {
	dir := makeTempDir(t, map[string]string{".claude/config.json": "{}"})
	tools := DetectCodingTools(dir)
	assert.Contains(t, tools, state.CodingToolClaudeCode)
}

func TestDetectCodingTools_Codex(t *testing.T) {
	dir := makeTempDir(t, map[string]string{".codex/config.toml": "[codex]\n"})
	tools := DetectCodingTools(dir)
	assert.Contains(t, tools, state.CodingToolCodex)
}

func TestDetectCodingTools_Empty(t *testing.T) {
	dir := t.TempDir()
	tools := DetectCodingTools(dir)
	assert.Empty(t, tools)
}

func TestDetectCodingTools_NoDuplicates(t *testing.T) {
	// Both CLAUDE.md and .claude present — should appear only once.
	dir := makeTempDir(t, map[string]string{
		"CLAUDE.md":            "# Claude\n",
		".claude/config.json":  "{}",
	})
	tools := DetectCodingTools(dir)

	count := 0
	for _, tool := range tools {
		if tool == state.CodingToolClaudeCode {
			count++
		}
	}
	assert.Equal(t, 1, count)
}

func TestDetectCodingTools_Multiple(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"CLAUDE.md":            "# Claude\n",
		".codex/config.toml":   "[codex]\n",
	})
	tools := DetectCodingTools(dir)

	assert.Contains(t, tools, state.CodingToolClaudeCode)
	assert.Contains(t, tools, state.CodingToolCodex)
}

func TestDetectCodingTools_PathJoin(t *testing.T) {
	// Verify that root path is properly joined (not concatenated with a slash).
	dir := makeTempDir(t, map[string]string{"CLAUDE.md": "# Claude\n"})
	tools := DetectCodingTools(filepath.Clean(dir))
	assert.Contains(t, tools, state.CodingToolClaudeCode)
}
