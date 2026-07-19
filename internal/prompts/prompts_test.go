package prompts

import (
	"strings"
	"testing"
)

func TestRender_ClaudeMd_NoErrorNonEmpty(t *testing.T) {
	out, err := Render("claude_md", RenderData{Command: "tddmaster"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if out == "" {
		t.Fatal("expected non-empty output")
	}
}

func TestRender_AllTemplates_NoErrorNonEmpty(t *testing.T) {
	names := []string{"claude_md", "executor", "verifier", "planner", "test-writer", "auditor"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			out, err := Render(name, RenderData{Command: "tddmaster"})
			if err != nil {
				t.Fatalf("expected no error for %q, got: %v", name, err)
			}
			if out == "" {
				t.Fatalf("expected non-empty output for %q", name)
			}
		})
	}
}

func TestRender_UnknownTemplate_ReturnsError(t *testing.T) {
	_, err := Render("nonexistent", RenderData{Command: "tddmaster"})
	if err == nil {
		t.Fatal("expected error for unknown template, got nil")
	}
}

func TestRender_CommandSubstitution_OutputContainsCommand(t *testing.T) {
	out, err := Render("claude_md", RenderData{Command: "mycli"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(out, "mycli") {
		t.Fatalf("expected output to contain %q, got: %s", "mycli", out)
	}
}

func TestRender_ClaudeMd_UsesStartSlugNotSpecNew(t *testing.T) {
	out, err := Render("claude_md", RenderData{Command: "tddmaster"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(out, "start <slug>") {
		t.Fatalf("expected output to contain %q", "start <slug>")
	}
	if strings.Contains(out, "spec new") {
		t.Fatalf("expected output to NOT contain %q", "spec new")
	}
}

func TestRender_AgentTemplates_AreBodyOnly_NoFrontmatter(t *testing.T) {
	agentNames := []string{"executor", "verifier", "planner", "test-writer", "auditor"}
	for _, name := range agentNames {
		t.Run(name, func(t *testing.T) {
			out, err := Render(name, RenderData{Command: "tddmaster"})
			if err != nil {
				t.Fatalf("expected no error for %q, got: %v", name, err)
			}
			if strings.HasPrefix(strings.TrimLeft(out, "\n"), "---") {
				t.Fatalf("expected body-only template for %q; frontmatter must live in the adapter, got leading ---", name)
			}
		})
	}
}

func TestRender_AgentTemplates_ContainBodyContent(t *testing.T) {
	cases := map[string]string{
		"executor":    "You are executing a single task",
		"verifier":    "You are verifying another agent's work",
		"planner":     "You are tddmaster-planner",
		"test-writer": "You are a TDD test-writer agent",
		"auditor":     "You are tddmaster-auditor",
	}
	for name, phrase := range cases {
		t.Run(name, func(t *testing.T) {
			out, err := Render(name, RenderData{Command: "tddmaster"})
			if err != nil {
				t.Fatalf("expected no error for %q, got: %v", name, err)
			}
			if !strings.Contains(out, phrase) {
				t.Fatalf("expected output for %q to contain %q", name, phrase)
			}
		})
	}
}

func TestTemplateNames_ReturnsExactSortedNames(t *testing.T) {
	expected := []string{"auditor", "claude_md", "executor", "planner", "rule-synthesizer", "test-writer", "verifier"}
	got := TemplateNames()
	if len(got) != len(expected) {
		t.Fatalf("expected %d names, got %d: %v", len(expected), len(got), got)
	}
	for i, name := range expected {
		if got[i] != name {
			t.Fatalf("expected name[%d] = %q, got %q", i, name, got[i])
		}
	}
}

func TestRender_AgentTemplates_NoUnresolvedTemplateSyntax(t *testing.T) {
	agentNames := []string{"executor", "verifier", "planner", "test-writer", "auditor"}
	for _, name := range agentNames {
		t.Run(name, func(t *testing.T) {
			out, err := Render(name, RenderData{Command: "tddmaster"})
			if err != nil {
				t.Fatalf("expected no error for %q, got: %v", name, err)
			}
			if strings.Contains(out, "{{") {
				t.Fatalf("expected no unresolved template syntax in %q output", name)
			}
		})
	}
}

func TestVerifierTmpl_GreenBlock_AuditsACsAndEdgeCases(t *testing.T) {
	out, err := Render("verifier", RenderData{})
	if err != nil {
		t.Fatalf("Render verifier: %v", err)
	}
	checks := []string{
		"each acceptance criterion",
		"uncoveredEdgeCases",
		"edge case",
	}
	for _, phrase := range checks {
		if !strings.Contains(strings.ToLower(out), strings.ToLower(phrase)) {
			t.Errorf("verifier GREEN block missing phrase %q", phrase)
		}
	}
	if !strings.Contains(out, "passed") || !strings.Contains(out, "true") {
		t.Error("verifier template must describe passed:true condition gated on ACs+ECs+tests")
	}
}

func TestVerifierTmpl_RefactorBlock_AuditsEdgeCases(t *testing.T) {
	out, err := Render("verifier", RenderData{})
	if err != nil {
		t.Fatalf("Render verifier: %v", err)
	}
	lower := strings.ToLower(out)
	if !strings.Contains(lower, "uncoveredEdgeCases") && !strings.Contains(out, "uncoveredEdgeCases") {
		t.Error("verifier REFACTOR block missing 'uncoveredEdgeCases' field")
	}
	if !strings.Contains(lower, "edge case") {
		t.Error("verifier REFACTOR block missing 'edge case' coverage check")
	}
}

func TestVerifierTmpl_GenericBlock_AuditsEdgeCases(t *testing.T) {
	out, err := Render("verifier", RenderData{})
	if err != nil {
		t.Fatalf("Render verifier: %v", err)
	}
	if !strings.Contains(out, "uncoveredEdgeCases") {
		t.Error("verifier generic block missing 'uncoveredEdgeCases' field for non-TDD flow")
	}
}

func TestRender_ClaudeMd_GateSubmitsIncludeTaskID(t *testing.T) {
	out, err := Render("claude_md", RenderData{Command: "tddmaster", ParallelSubagents: true})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(out, `--answer='{"taskId":"<taskId>","plan":{...},"accepted":true}'`) {
		t.Fatal("gate accept example must include taskId")
	}
	if !strings.Contains(out, `--answer='{"taskId":"<taskId>","planFeedback":"<reason>"}'`) {
		t.Fatal("gate revise/reject example must include taskId")
	}
}
