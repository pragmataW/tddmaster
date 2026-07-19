package adapter

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pragmataW/tddmaster/internal/manifest"
	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/prompts"
)

type CursorAdapter struct{}

func (CursorAdapter) ID() manifest.ToolID { return manifest.ToolCursor }

func (CursorAdapter) Sync(ctx SyncContext) error {
	if err := os.MkdirAll(paths.CursorAgents(ctx.Root), 0o755); err != nil {
		return fmt.Errorf("create cursor agents dir: %w", err)
	}

	rendered, err := prompts.Render("claude_md", prompts.RenderData{Command: ctx.CommandPrefix, ParallelSubagents: false})
	if err != nil {
		return fmt.Errorf("render agents doc: %w", err)
	}
	if err := injectMarkedDoc(paths.AgentsMd(ctx.Root), rendered); err != nil {
		return err
	}

	for _, spec := range AgentSpecs {
		body, err := renderBody(spec, ctx.CommandPrefix)
		if err != nil {
			return err
		}
		content := "---\n" +
			"name: " + spec.Name + "\n" +
			"description: \"" + spec.Description + "\"\n" +
			"---\n" + body
		filePath := filepath.Join(paths.CursorAgents(ctx.Root), spec.File+".md")
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write cursor agent %s: %w", spec.File, err)
		}
	}

	return nil
}
