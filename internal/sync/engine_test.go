package sync_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pragmataW/tddmaster/internal/state"
	statesync "github.com/pragmataW/tddmaster/internal/sync"
	_ "github.com/pragmataW/tddmaster/internal/sync/adapters"
)

func writeSpecFile(t *testing.T, root, specName string) {
	t.Helper()

	specDir := filepath.Join(root, state.TddmasterDir, "specs", specName)
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatalf("MkdirAll spec dir: %v", err)
	}

	content := "# Spec: " + specName + "\n\n## Discovery Answers\n\n### status_quo\n\nplaceholder\n"
	if err := os.WriteFile(filepath.Join(specDir, "spec.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile spec.md: %v", err)
	}
}

func baseConfig(tools ...state.CodingToolId) state.NosManifest {
	config := state.CreateInitialManifest(nil, tools, state.ProjectTraits{})
	config.Command = "tddmaster"
	return config
}

func TestSyncAll_OpenCodePreservesOutputsWithoutCreatingConfig(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := state.ScaffoldDir(root); err != nil {
		t.Fatalf("ScaffoldDir: %v", err)
	}
	writeSpecFile(t, root, "demo-spec")

	config := baseConfig(state.CodingToolOpencode)

	if _, err := statesync.SyncAll(root, []state.CodingToolId{state.CodingToolOpencode}, &config); err != nil {
		t.Fatalf("SyncAll: %v", err)
	}

	for _, rel := range []string{
		filepath.Join(".opencode", "plugins", "tddmaster.ts"),
		filepath.Join(".opencode", "agents", "tddmaster-executor.md"),
		filepath.Join(".opencode", "agents", "tddmaster-verifier.md"),
		filepath.Join(".opencode", "skills", "demo-spec.md"),
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected %s to exist: %v", rel, err)
		}
	}

	if _, err := os.Stat(filepath.Join(root, "opencode.json")); err == nil {
		t.Fatalf("expected opencode.json to stay absent after SyncAll")
	} else if !os.IsNotExist(err) {
		t.Fatalf("Stat opencode.json: %v", err)
	}
}

func TestSyncAll_CodexPreservesOutputsWithoutCreatingConfig(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := state.ScaffoldDir(root); err != nil {
		t.Fatalf("ScaffoldDir: %v", err)
	}

	config := baseConfig(state.CodingToolCodex)

	if _, err := statesync.SyncAll(root, []state.CodingToolId{state.CodingToolCodex}, &config); err != nil {
		t.Fatalf("SyncAll: %v", err)
	}

	for _, rel := range []string{
		filepath.Join(".codex", "hooks.json"),
		filepath.Join(".codex", "agents", "tddmaster-executor.toml"),
		filepath.Join(".codex", "agents", "tddmaster-verifier.toml"),
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected %s to exist: %v", rel, err)
		}
	}

	if _, err := os.Stat(filepath.Join(root, ".codex", "config.toml")); err == nil {
		t.Fatalf("expected .codex/config.toml to stay absent after SyncAll")
	} else if !os.IsNotExist(err) {
		t.Fatalf("Stat .codex/config.toml: %v", err)
	}
}
