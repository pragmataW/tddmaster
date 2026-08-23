package initform

import (
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/manifest"
	"github.com/pragmataW/tddmaster/internal/scaffold"
)

func TestRenderSummary_ListsEveryWrittenFile(t *testing.T) {
	res := scaffold.Result{
		FilesWritten: []string{
			"/proj/.tddmaster/manifest.json",
			"CLAUDE.md",
			".claude/agents/tddmaster-executor.md",
			".claude/agents/tddmaster-planner.md",
			".claude/agents/tddmaster-verifier.md",
		},
		Adapters: []manifest.ToolID{manifest.ToolClaudeCode},
	}

	got := RenderSummary(res, "tddmaster")
	for _, want := range []string{".tddmaster/manifest.json", "CLAUDE.md", "tddmaster-*"} {
		if !strings.Contains(got, want) {
			t.Fatalf("summary must mention %q, got:\n%s", want, got)
		}
	}
}

func TestRenderSummary_DoesNotHardcodeClaudeAgentsForOtherTools(t *testing.T) {
	res := scaffold.Result{
		FilesWritten: []string{
			"/proj/.tddmaster/manifest.json",
			"AGENTS.md",
			".opencode/agents/tddmaster-executor.md",
			".opencode/agents/tddmaster-planner.md",
			".opencode/agents/tddmaster-verifier.md",
		},
		Adapters: []manifest.ToolID{manifest.ToolOpenCode},
	}

	got := RenderSummary(res, "tddmaster")
	if strings.Contains(got, ".claude/agents") {
		t.Fatalf("opencode summary must not claim .claude/agents, got:\n%s", got)
	}
	if !strings.Contains(got, ".opencode/agents") || !strings.Contains(got, "AGENTS.md") {
		t.Fatalf("summary must list the opencode paths, got:\n%s", got)
	}
}

func TestSummaryFileLines_CollapsesOnlyAgentDirectories(t *testing.T) {
	got := summaryFileLines([]string{
		"CLAUDE.md",
		".claude/agents/a.md",
		".claude/agents/b.md",
		".claude/agents/c.md",
	})
	joined := strings.Join(got, "\n")
	if !strings.Contains(joined, "CLAUDE.md") {
		t.Fatalf("root-level file must survive, got %v", got)
	}
	if strings.Contains(joined, "a.md") {
		t.Fatalf("agent directory with 3 files must collapse, got %v", got)
	}
}
