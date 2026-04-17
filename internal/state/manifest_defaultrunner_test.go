package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// compile-time guard: NosManifest must have a DefaultRunner field.
// This line will fail to compile until the executor adds DefaultRunner to NosManifest.
var _ = NosManifest{DefaultRunner: "claude-code"}

// TestNosManifest_DefaultRunnerFieldExists verifies that NosManifest exposes
// a DefaultRunner string field with JSON tag "defaultRunner" and omitempty.
func TestNosManifest_DefaultRunnerFieldExists(t *testing.T) {
	// Compile-time assignment already guards the field exists.
	// This runtime portion verifies the JSON round-trip with the correct tag.

	m := NosManifest{
		DefaultRunner: "opencode",
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("json.Marshal: unexpected error: %v", err)
	}

	jsonStr := string(data)
	if !strings.Contains(jsonStr, `"defaultRunner"`) {
		t.Errorf("marshaled JSON missing key defaultRunner; got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"opencode"`) {
		t.Errorf("marshaled JSON missing value opencode; got: %s", jsonStr)
	}

	// Unmarshal back and verify round-trip.
	var got NosManifest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal: unexpected error: %v", err)
	}
	if got.DefaultRunner != "opencode" {
		t.Errorf("DefaultRunner round-trip: want %q, got %q", "opencode", got.DefaultRunner)
	}
}

// TestNosManifest_DefaultRunner_Omitempty verifies that when DefaultRunner is
// the empty string, the JSON output does NOT contain the "defaultRunner" key.
func TestNosManifest_DefaultRunner_Omitempty(t *testing.T) {
	m := NosManifest{
		DefaultRunner: "", // explicitly empty
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("json.Marshal: unexpected error: %v", err)
	}

	if strings.Contains(string(data), `"defaultRunner"`) {
		t.Errorf("JSON must not contain defaultRunner when field is empty; got: %s", string(data))
	}
}

// TestNosManifest_ProvidersFieldRemoved asserts that no field named "Providers"
// exists on the NosManifest struct. Enforcement is via reflection so it detects
// regressions without coupling to a specific field index.
func TestNosManifest_ProvidersFieldRemoved(t *testing.T) {
	typ := reflect.TypeOf(NosManifest{})

	for i := 0; i < typ.NumField(); i++ {
		if typ.Field(i).Name == "Providers" {
			t.Errorf("NosManifest still has field Providers; it must be removed per task-11")
		}
	}
}

// TestCreateInitialManifest_NewSignature verifies that CreateInitialManifest
// accepts exactly 3 arguments (concerns, tools, project) — the providers
// parameter must have been removed. The returned manifest's DefaultRunner
// must be "" (empty) by default; callers set it explicitly when desired.
func TestCreateInitialManifest_NewSignature(t *testing.T) {
	// This call will fail to compile if the old 4-arg signature is still in place.
	m := CreateInitialManifest(nil, nil, ProjectTraits{})

	if m.DefaultRunner != "" {
		t.Errorf("DefaultRunner: want empty default, got %q", m.DefaultRunner)
	}
}

// TestReadManifest_LegacyProvidersFieldIgnored verifies that a manifest YAML
// containing the now-removed "providers" field can be read without error.
// Go's standard JSON/YAML unmarshal ignores unknown fields by default; this
// test is the explicit regression guard for that backward-compat contract.
func TestReadManifest_LegacyProvidersFieldIgnored(t *testing.T) {
	// Write a manifest that has a "providers" key at the tddmaster level.
	legacyYAML := `tddmaster:
  concerns: []
  tools: []
  providers:
    - anthropic
    - openai
  project:
    languages: []
    frameworks: []
    ci: []
`

	tmpDir := t.TempDir()
	tddDir := filepath.Join(tmpDir, TddmasterDir)
	if err := os.MkdirAll(tddDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	manifestPath := filepath.Join(tddDir, "manifest.yml")
	if err := os.WriteFile(manifestPath, []byte(legacyYAML), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	manifest, err := ReadManifest(tmpDir)
	if err != nil {
		t.Fatalf("ReadManifest returned error for legacy manifest: %v", err)
	}
	// ReadManifest returns nil when not found; here the file exists so we
	// expect a non-nil manifest.
	if manifest == nil {
		t.Fatal("ReadManifest: want non-nil manifest, got nil")
	}

	// The key point: no panic, no error — legacy field was silently ignored.
	// DefaultRunner must be zero value since it was absent from legacy YAML.
	if manifest.DefaultRunner != "" {
		t.Errorf("DefaultRunner: want empty string for legacy manifest, got %q", manifest.DefaultRunner)
	}
}
