package adapter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pragmataW/tddmaster/internal/manifest"
	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/prompts"
)

const (
	markerStart = "<!-- tddmasterStart -->"
	markerEnd   = "<!-- tddmasterEnd -->"
)

type ClaudeCodeAdapter struct{}

func (ClaudeCodeAdapter) ID() manifest.ToolID { return manifest.ToolClaudeCode }

func (ClaudeCodeAdapter) Sync(ctx SyncContext) error {
	if err := os.MkdirAll(paths.ClaudeAgents(ctx.Root), 0o755); err != nil {
		return fmt.Errorf("create agents dir: %w", err)
	}

	rendered, err := prompts.Render("claude_md", prompts.RenderData{Command: ctx.CommandPrefix})
	if err != nil {
		return fmt.Errorf("render claude_md: %w", err)
	}

	block := markerStart + "\n" + rendered + "\n" + markerEnd

	claudeMdPath := paths.ClaudeMd(ctx.Root)
	existing, readErr := os.ReadFile(claudeMdPath)
	if readErr != nil && !os.IsNotExist(readErr) {
		return fmt.Errorf("read CLAUDE.md: %w", readErr)
	}

	var newContent string
	if readErr != nil {
		newContent = block
	} else {
		content := string(existing)
		startIdx := strings.Index(content, markerStart)
		endIdx := -1
		if startIdx != -1 {
			rel := strings.Index(content[startIdx+len(markerStart):], markerEnd)
			if rel != -1 {
				endIdx = startIdx + len(markerStart) + rel
			}
		}
		if startIdx != -1 && endIdx != -1 {
			newContent = content[:startIdx] + block + content[endIdx+len(markerEnd):]
		} else {
			content = strings.ReplaceAll(content, markerStart, "")
			content = strings.ReplaceAll(content, markerEnd, "")
			if len(content) > 0 && !strings.HasSuffix(content, "\n") {
				newContent = content + "\n" + block
			} else {
				newContent = content + block
			}
		}
	}

	if err := os.WriteFile(claudeMdPath, []byte(newContent), 0o644); err != nil {
		return fmt.Errorf("write CLAUDE.md: %w", err)
	}

	agentFiles := []struct {
		filename string
		template string
	}{
		{"tddmaster-executor.md", "executor"},
		{"tddmaster-verifier.md", "verifier"},
		{"tddmaster-planner.md", "planner"},
		{"tddmaster-test-writer.md", "test-writer"},
		{"tddmaster-rule-synthesizer.md", "rule-synthesizer"},
		{"tddmaster-auditor.md", "auditor"},
	}

	for _, af := range agentFiles {
		content, err := prompts.Render(af.template, prompts.RenderData{Command: ctx.CommandPrefix})
		if err != nil {
			return fmt.Errorf("render %s: %w", af.template, err)
		}
		filePath := filepath.Join(paths.ClaudeAgents(ctx.Root), af.filename)
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write agent file %s: %w", af.filename, err)
		}
	}

	return nil
}
