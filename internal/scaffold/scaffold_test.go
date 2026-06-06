package scaffold

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/pragmataW/tddmaster/internal/adapter"
	"github.com/pragmataW/tddmaster/internal/manifest"
	"github.com/pragmataW/tddmaster/internal/paths"
)

func claudeCodeManifest() *manifest.Manifest {
	return &manifest.Manifest{
		SelectedTools:           []manifest.ToolID{manifest.ToolClaudeCode},
		MaxIterationBeforeStart: 15,
		Command:                 "tddmaster",
	}
}

func TestScaffold_EmptySelectedTools_ReturnsError(t *testing.T) {
	tmp := t.TempDir()
	opts := Options{
		Root: tmp,
		Manifest: &manifest.Manifest{
			SelectedTools:           []manifest.ToolID{},
			MaxIterationBeforeStart: 15,
			Command:                 "tddmaster",
		},
	}

	_, err := Scaffold(opts)

	if err == nil {
		t.Fatal("expected error for empty SelectedTools, got nil")
	}
	if !strings.Contains(err.Error(), "tool") {
		t.Errorf("error message %q does not contain 'tool'", err.Error())
	}
}

func TestScaffold_HappyPath_NoError(t *testing.T) {
	tmp := t.TempDir()
	opts := Options{Root: tmp, Manifest: claudeCodeManifest()}

	result, err := Scaffold(opts)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	manifestPath := paths.Manifest(tmp)
	if _, statErr := os.Stat(manifestPath); os.IsNotExist(statErr) {
		t.Errorf("manifest.json not found at %s", manifestPath)
	}

	found := false
	for _, id := range result.Adapters {
		if id == manifest.ToolClaudeCode {
			found = true
		}
	}
	if !found {
		t.Errorf("Result.Adapters does not contain claude-code, got: %v", result.Adapters)
	}

	claudeMdPath := paths.ClaudeMd(tmp)
	if _, statErr := os.Stat(claudeMdPath); os.IsNotExist(statErr) {
		t.Errorf("CLAUDE.md not found at %s", claudeMdPath)
	}
}

func TestScaffold_ManifestContent_ValidJSONAndIndented(t *testing.T) {
	tmp := t.TempDir()
	opts := Options{Root: tmp, Manifest: claudeCodeManifest()}

	_, err := Scaffold(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, readErr := os.ReadFile(paths.Manifest(tmp))
	if readErr != nil {
		t.Fatalf("failed to read manifest.json: %v", readErr)
	}

	var m manifest.Manifest
	if unmarshalErr := json.Unmarshal(data, &m); unmarshalErr != nil {
		t.Fatalf("manifest.json is not valid JSON: %v", unmarshalErr)
	}

	if len(m.SelectedTools) != 1 || m.SelectedTools[0] != manifest.ToolClaudeCode {
		t.Errorf("SelectedTools = %v, want [claude-code]", m.SelectedTools)
	}
	if m.MaxIterationBeforeStart != 15 {
		t.Errorf("MaxIterationBeforeStart = %d, want 15", m.MaxIterationBeforeStart)
	}
	if m.Command != "tddmaster" {
		t.Errorf("Command = %q, want 'tddmaster'", m.Command)
	}

	if !strings.Contains(string(data), "\n") {
		t.Errorf("manifest.json appears compact (no newlines); expected indented JSON")
	}
}

func TestScaffold_UnknownTool_WarningNotError(t *testing.T) {
	tmp := t.TempDir()
	opts := Options{
		Root: tmp,
		Manifest: &manifest.Manifest{
			SelectedTools:           []manifest.ToolID{manifest.ToolClaudeCode, "bogus-tool"},
			MaxIterationBeforeStart: 15,
			Command:                 "tddmaster",
		},
	}

	result, err := Scaffold(opts)

	if err != nil {
		t.Fatalf("expected no error for unknown tool, got: %v", err)
	}

	if len(result.Warnings) == 0 {
		t.Fatal("expected at least one warning for unknown tool, got none")
	}

	foundWarning := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "bogus-tool") {
			foundWarning = true
		}
	}
	if !foundWarning {
		t.Errorf("warnings %v do not mention 'bogus-tool'", result.Warnings)
	}

	claudeMdPath := paths.ClaudeMd(tmp)
	if _, statErr := os.Stat(claudeMdPath); os.IsNotExist(statErr) {
		t.Errorf("CLAUDE.md not found; claude-code adapter should have run despite unknown tool")
	}
}

func TestScaffold_NormalizeApplied_DeduplicatesAndSetsDefaults(t *testing.T) {
	tmp := t.TempDir()
	opts := Options{
		Root: tmp,
		Manifest: &manifest.Manifest{
			SelectedTools:           []manifest.ToolID{manifest.ToolClaudeCode, manifest.ToolClaudeCode},
			MaxIterationBeforeStart: 0,
			Command:                 "tddmaster",
		},
	}

	_, err := Scaffold(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, readErr := os.ReadFile(paths.Manifest(tmp))
	if readErr != nil {
		t.Fatalf("failed to read manifest.json: %v", readErr)
	}

	var m manifest.Manifest
	if unmarshalErr := json.Unmarshal(data, &m); unmarshalErr != nil {
		t.Fatalf("manifest.json is not valid JSON: %v", unmarshalErr)
	}

	if m.MaxIterationBeforeStart != 15 {
		t.Errorf("MaxIterationBeforeStart = %d after normalize, want 15", m.MaxIterationBeforeStart)
	}

	count := 0
	for _, id := range m.SelectedTools {
		if id == manifest.ToolClaudeCode {
			count++
		}
	}
	if count != 1 {
		t.Errorf("SelectedTools has %d claude-code entries after normalize, want 1", count)
	}
}

func TestLoadManifestOrDefaults_MissingFile_ReturnsDefaults(t *testing.T) {
	tmp := t.TempDir()

	m := LoadManifestOrDefaults(tmp)

	defaults := manifest.Defaults()
	if m.Command != defaults.Command {
		t.Errorf("Command = %q, want %q", m.Command, defaults.Command)
	}
	if m.MaxIterationBeforeStart != defaults.MaxIterationBeforeStart {
		t.Errorf("MaxIterationBeforeStart = %d, want %d", m.MaxIterationBeforeStart, defaults.MaxIterationBeforeStart)
	}
	if len(m.SelectedTools) != 0 {
		t.Errorf("SelectedTools = %v, want empty", m.SelectedTools)
	}
}

func TestLoadManifestOrDefaults_ExistingManifest_ReturnsStoredValues(t *testing.T) {
	tmp := t.TempDir()
	stored := manifest.Manifest{
		SelectedTools:           []manifest.ToolID{manifest.ToolClaudeCode},
		MaxIterationBeforeStart: 7,
		Command:                 "tddmaster",
	}
	data, _ := json.MarshalIndent(stored, "", "  ")
	tddDir := paths.Tddmaster(tmp)
	if mkErr := os.MkdirAll(tddDir, 0o755); mkErr != nil {
		t.Fatalf("failed to create .tddmaster dir: %v", mkErr)
	}
	if writeErr := os.WriteFile(paths.Manifest(tmp), data, 0o644); writeErr != nil {
		t.Fatalf("failed to write manifest.json: %v", writeErr)
	}

	m := LoadManifestOrDefaults(tmp)

	if m.MaxIterationBeforeStart != 7 {
		t.Errorf("MaxIterationBeforeStart = %d, want 7", m.MaxIterationBeforeStart)
	}
	if len(m.SelectedTools) != 1 || m.SelectedTools[0] != manifest.ToolClaudeCode {
		t.Errorf("SelectedTools = %v, want [claude-code]", m.SelectedTools)
	}
}

func TestScaffold_Idempotent_SecondRunSucceedsAndSingleMarker(t *testing.T) {
	tmp := t.TempDir()
	opts := Options{Root: tmp, Manifest: claudeCodeManifest()}

	if _, err := Scaffold(opts); err != nil {
		t.Fatalf("first Scaffold call failed: %v", err)
	}

	if _, err := Scaffold(opts); err != nil {
		t.Fatalf("second Scaffold call failed: %v", err)
	}

	claudeMdPath := filepath.Join(tmp, "CLAUDE.md")
	data, readErr := os.ReadFile(claudeMdPath)
	if readErr != nil {
		t.Fatalf("failed to read CLAUDE.md: %v", readErr)
	}

	markerStart := "<!-- tddmasterStart -->"
	count := strings.Count(string(data), markerStart)
	if count != 1 {
		t.Errorf("tddmasterStart marker appears %d times in CLAUDE.md, want exactly 1", count)
	}
}
