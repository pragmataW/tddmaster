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
	names := []string{"claude_md", "executor", "verifier", "planner", "test-writer"}
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

func TestRender_AgentTemplates_StartsWithFrontmatterDelimiter(t *testing.T) {
	agentNames := []string{"executor", "verifier", "planner", "test-writer"}
	for _, name := range agentNames {
		t.Run(name, func(t *testing.T) {
			out, err := Render(name, RenderData{Command: "tddmaster"})
			if err != nil {
				t.Fatalf("expected no error for %q, got: %v", name, err)
			}
			if !strings.HasPrefix(out, "---") {
				t.Fatalf("expected output for %q to start with ---, got: %s", name, out[:min(len(out), 50)])
			}
		})
	}
}

func TestRender_AgentTemplates_ContainNameAndDescription(t *testing.T) {
	agentNames := []string{"executor", "verifier", "planner", "test-writer"}
	for _, name := range agentNames {
		t.Run(name, func(t *testing.T) {
			out, err := Render(name, RenderData{Command: "tddmaster"})
			if err != nil {
				t.Fatalf("expected no error for %q, got: %v", name, err)
			}
			if !strings.Contains(out, "name:") {
				t.Fatalf("expected output for %q to contain 'name:'", name)
			}
			if !strings.Contains(out, "description:") {
				t.Fatalf("expected output for %q to contain 'description:'", name)
			}
		})
	}
}

func TestTemplateNames_ReturnsExactSortedNames(t *testing.T) {
	expected := []string{"claude_md", "executor", "planner", "test-writer", "verifier"}
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
	agentNames := []string{"executor", "verifier", "planner", "test-writer"}
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
