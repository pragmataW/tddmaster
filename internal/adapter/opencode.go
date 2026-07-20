package adapter

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pragmataW/tddmaster/internal/errs"
	"github.com/pragmataW/tddmaster/internal/manifest"
	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/prompts"
)

type OpenCodeAdapter struct{}

func (OpenCodeAdapter) ID() manifest.ToolID { return manifest.ToolOpenCode }

func (OpenCodeAdapter) Sync(ctx SyncContext) error {
	if err := os.MkdirAll(paths.OpenCodeAgents(ctx.Root), 0o755); err != nil {
		return errs.Wrap(errs.KeyAdapterCreateDir, err, "opencode")
	}

	rendered, err := prompts.Render("claude_md", prompts.RenderData{Command: ctx.CommandPrefix, ParallelSubagents: true})
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
		// OpenCode agent name comes from the filename; no name field.
		// Model is omitted: spec.Model is a Claude Code shorthand, not
		// OpenCode's provider/model-id format — agents inherit the user's model.
		content := "---\n" +
			"description: \"" + spec.Description + "\"\n" +
			"mode: subagent\n" +
			openCodePermission(spec.Tools) +
			"---\n" + body
		filePath := filepath.Join(paths.OpenCodeAgents(ctx.Root), spec.File+".md")
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			return errs.Wrap(errs.KeyAdapterWriteAgent, err, "opencode", spec.File)
		}
	}

	return nil
}

// openCodePermission maps the spec's Claude Code tool list to OpenCode
// permission denies. Empty string when the agent has full access.
func openCodePermission(tools string) string {
	var denies []string
	if !strings.Contains(tools, "Edit") && !strings.Contains(tools, "Write") {
		denies = append(denies, "  edit: deny\n")
	}
	if !strings.Contains(tools, "Bash") {
		denies = append(denies, "  bash: deny\n")
	}
	if len(denies) == 0 {
		return ""
	}
	return "permission:\n" + strings.Join(denies, "")
}
