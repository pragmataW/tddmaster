package adapter

import (
	"os"
	"path/filepath"

	"github.com/pragmataW/tddmaster/internal/errs"
	"github.com/pragmataW/tddmaster/internal/manifest"
	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/prompts"
)

type CodexCLIAdapter struct{}

func (CodexCLIAdapter) ID() manifest.ToolID { return manifest.ToolCodexCLI }

func (CodexCLIAdapter) Sync(ctx SyncContext) error {
	if err := os.MkdirAll(paths.CodexAgents(ctx.Root), 0o755); err != nil {
		return errs.Wrap(errs.KeyAdapterCreateDir, err, "codex")
	}

	rendered, err := prompts.Render("claude_md", prompts.RenderData{Command: ctx.CommandPrefix, ParallelSubagents: false})
	if err != nil {
		return errs.Wrap(errs.KeyAdapterRenderDoc, err)
	}
	if err := injectMarkedDoc(paths.AgentsMd(ctx.Root), rendered); err != nil {
		return err
	}

	for _, spec := range AgentSpecs {
		body, err := renderBody(spec, ctx.CommandPrefix)
		if err != nil {
			return err
		}
		content := "name = \"" + spec.Name + "\"\n" +
			"description = \"" + spec.Description + "\"\n" +
			"developer_instructions = \"\"\"" + body + "\"\"\"\n"
		filePath := filepath.Join(paths.CodexAgents(ctx.Root), spec.File+".toml")
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			return errs.Wrap(errs.KeyAdapterWriteAgent, err, "codex", spec.File)
		}
	}

	return nil
}
