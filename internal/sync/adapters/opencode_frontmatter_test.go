package adapters

import (
	"strings"
	"testing"
)

func expectToolsBlock(t *testing.T, content string, tools ...string) {
	t.Helper()

	if !strings.Contains(content, "tools:\n") {
		t.Fatalf("expected YAML tools block, got:\n%s", content)
	}
	if strings.Contains(content, "tools: read,") {
		t.Fatalf("expected legacy single-line tools format to be absent, got:\n%s", content)
	}

	for _, tool := range tools {
		line := "  " + tool + ": true"
		if !strings.Contains(content, line) {
			t.Fatalf("expected tools block to contain %q, got:\n%s", line, content)
		}
	}
}

func TestOpenCodeExecutorAgentMd_UsesYamlToolsMap(t *testing.T) {
	t.Parallel()

	got := buildOpenCodeExecutorAgentMd("tddmaster")

	expectToolsBlock(t, got, "read", "write", "glob", "grep", "shell", "delegate")
}

func TestOpenCodeVerifierAgentMd_UsesYamlToolsMap(t *testing.T) {
	t.Parallel()

	got := buildOpenCodeVerifierAgentMd(nil)

	expectToolsBlock(t, got, "read", "glob", "grep", "shell")
}

func TestOpenCodeTestWriterAgentMd_UsesYamlToolsMap(t *testing.T) {
	t.Parallel()

	got := buildOpenCodeTestWriterAgentMd()

	expectToolsBlock(t, got, "read", "write", "glob", "grep", "shell")
}
