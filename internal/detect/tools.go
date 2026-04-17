package detect

import (
	"path/filepath"

	"github.com/pragmataW/tddmaster/internal/state"
)

// =============================================================================
// Coding Tool Detection (checks for config files in repo)
// =============================================================================

type toolSignal struct {
	id    state.CodingToolId
	paths []string
}

var toolSignals = []toolSignal{
	{id: state.CodingToolClaudeCode, paths: []string{"CLAUDE.md", ".claude"}},
	{id: state.CodingToolCodex, paths: []string{".codex", ".codex/config.toml"}},
	{id: state.CodingToolOpencode, paths: []string{".opencode", "opencode.json"}},
}

// DetectCodingTools detects available coding tools by checking for their
// config files in the given repository root.
func DetectCodingTools(root string) []state.CodingToolId {
	var detected []state.CodingToolId

	for _, signal := range toolSignals {
		for _, p := range signal.paths {
			if pathExists(filepath.Join(root, p)) {
				detected = append(detected, signal.id)
				break
			}
		}
	}

	return detected
}
