package adapter

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pragmataW/tddmaster/internal/manifest"
	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/prompts"
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
	if err := injectMarkedDoc(paths.ClaudeMd(ctx.Root), rendered); err != nil {
		return err
	}

	for _, spec := range AgentSpecs {
		content, err := claudeAgentFile(spec, ctx.CommandPrefix)
		if err != nil {
			return err
		}
		filePath := filepath.Join(paths.ClaudeAgents(ctx.Root), spec.File+".md")
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write agent file %s: %w", spec.File, err)
		}
	}

	return nil
}

func claudeAgentFile(spec AgentSpec, cmd string) (string, error) {
	body, err := renderBody(spec, cmd)
	if err != nil {
		return "", err
	}
	frontmatter := "---\n" +
		"name: " + spec.Name + "\n" +
		"description: \"" + spec.Description + "\"\n" +
		"tools: " + spec.Tools + "\n" +
		"model: " + spec.Model + "\n" +
		"---\n"
	return frontmatter + body, nil
}
