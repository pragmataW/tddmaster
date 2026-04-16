
package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// =============================================================================
// Settings.json generation (Claude Code hooks)
// =============================================================================

type claudeHookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
}

type claudeHookMatcher struct {
	Matcher string            `json:"matcher,omitempty"`
	Hooks   []claudeHookEntry `json:"hooks"`
}

type claudeSettings struct {
	Hooks map[string][]claudeHookMatcher `json:"hooks"`
}

// BuildClaudeSettings generates the Claude Code settings.json hooks structure.
func BuildClaudeSettings(commandPrefix string) map[string]interface{} {
	return map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Write|Edit|MultiEdit|Bash",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": fmt.Sprintf("%s invoke-hook pre-tool-use", commandPrefix),
							"timeout": 5,
						},
					},
				},
			},
			"PostToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Write|Edit|MultiEdit",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": fmt.Sprintf("%s invoke-hook post-file-write", commandPrefix),
							"timeout": 3,
						},
					},
				},
				map[string]interface{}{
					"matcher": "Bash",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": fmt.Sprintf("%s invoke-hook post-bash", commandPrefix),
							"timeout": 3,
						},
					},
				},
			},
			"Stop": []interface{}{
				map[string]interface{}{
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": fmt.Sprintf("%s invoke-hook stop", commandPrefix),
							"timeout": 10,
						},
					},
				},
			},
			"SessionStart": []interface{}{
				map[string]interface{}{
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": fmt.Sprintf("%s invoke-hook session-start", commandPrefix),
							"timeout": 5,
						},
					},
				},
			},
		},
	}
}

// SyncHooks generates or updates .claude/settings.json with tddmaster hook commands.
func SyncHooks(root, commandPrefix string) error {
	if commandPrefix == "" {
		commandPrefix = "tddmaster"
	}

	settingsPath := filepath.Join(root, ".claude", "settings.json")

	// Read existing settings (preserve non-hook keys)
	existingSettings := make(map[string]interface{})
	if data, err := os.ReadFile(settingsPath); err == nil {
		_ = json.Unmarshal(data, &existingSettings)
	}

	// Merge: keep existing keys, overwrite hooks
	newHooks := BuildClaudeSettings(commandPrefix)
	for k, v := range newHooks {
		existingSettings[k] = v
	}

	if err := os.MkdirAll(filepath.Join(root, ".claude"), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(existingSettings, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	return os.WriteFile(settingsPath, data, 0o644)
}
