package manifest

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestDefaults_ReturnsExpectedValues(t *testing.T) {
	m := Defaults()
	if m.SelectedTools == nil {
		t.Fatal("SelectedTools must be non-nil")
	}
	if len(m.SelectedTools) != 0 {
		t.Fatalf("SelectedTools must be empty, got %v", m.SelectedTools)
	}
	if m.MaxIterationBeforeStart != 15 {
		t.Fatalf("MaxIterationBeforeStart want 15, got %d", m.MaxIterationBeforeStart)
	}
	if m.Command != "tddmaster" {
		t.Fatalf("Command want tddmaster, got %q", m.Command)
	}
}

func TestNormalize_DedupSelectedTools_OrderPreserved(t *testing.T) {
	m := &Manifest{
		SelectedTools:           []ToolID{ToolClaudeCode, ToolClaudeCode},
		MaxIterationBeforeStart: 15,
		Command:                 "tddmaster",
	}
	Normalize(m)
	if len(m.SelectedTools) != 1 {
		t.Fatalf("expected 1 tool after dedup, got %d: %v", len(m.SelectedTools), m.SelectedTools)
	}
	if m.SelectedTools[0] != ToolClaudeCode {
		t.Fatalf("expected %q, got %q", ToolClaudeCode, m.SelectedTools[0])
	}
}

func TestNormalize_MaxIterationZero_SetsDefault(t *testing.T) {
	m := &Manifest{
		SelectedTools:           []ToolID{},
		MaxIterationBeforeStart: 0,
		Command:                 "tddmaster",
	}
	Normalize(m)
	if m.MaxIterationBeforeStart != 15 {
		t.Fatalf("want 15, got %d", m.MaxIterationBeforeStart)
	}
}

func TestNormalize_MaxIterationNegative_SetsDefault(t *testing.T) {
	m := &Manifest{
		SelectedTools:           []ToolID{},
		MaxIterationBeforeStart: -5,
		Command:                 "tddmaster",
	}
	Normalize(m)
	if m.MaxIterationBeforeStart != 15 {
		t.Fatalf("want 15, got %d", m.MaxIterationBeforeStart)
	}
}

func TestNormalize_MaxIterationPositive_Unchanged(t *testing.T) {
	m := &Manifest{
		SelectedTools:           []ToolID{},
		MaxIterationBeforeStart: 20,
		Command:                 "tddmaster",
	}
	Normalize(m)
	if m.MaxIterationBeforeStart != 20 {
		t.Fatalf("want 20, got %d", m.MaxIterationBeforeStart)
	}
}

func TestNormalize_EmptyCommand_SetsDefault(t *testing.T) {
	m := &Manifest{
		SelectedTools:           []ToolID{},
		MaxIterationBeforeStart: 15,
		Command:                 "",
	}
	Normalize(m)
	if m.Command != "tddmaster" {
		t.Fatalf("want tddmaster, got %q", m.Command)
	}
}

func TestNormalize_NonEmptyCommand_Preserved(t *testing.T) {
	m := &Manifest{
		SelectedTools:           []ToolID{},
		MaxIterationBeforeStart: 15,
		Command:                 "custom-cmd",
	}
	Normalize(m)
	if m.Command != "custom-cmd" {
		t.Fatalf("want custom-cmd, got %q", m.Command)
	}
}

func TestManifest_JSONRoundTrip_FieldsPreserved(t *testing.T) {
	original := Manifest{
		SelectedTools:           []ToolID{ToolClaudeCode},
		MaxIterationBeforeStart: 10,
		Command:                 "tddmaster",
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded Manifest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(decoded.SelectedTools) != 1 || decoded.SelectedTools[0] != ToolClaudeCode {
		t.Fatalf("SelectedTools mismatch: %v", decoded.SelectedTools)
	}
	if decoded.MaxIterationBeforeStart != 10 {
		t.Fatalf("MaxIterationBeforeStart mismatch: %d", decoded.MaxIterationBeforeStart)
	}
	if decoded.Command != "tddmaster" {
		t.Fatalf("Command mismatch: %q", decoded.Command)
	}
	for _, key := range []string{`"selectedTools"`, `"maxIterationBeforeStart"`, `"command"`} {
		if !bytes.Contains(data, []byte(key)) {
			t.Fatalf("JSON output missing key %s in %s", key, data)
		}
	}
}

func TestToolClaudeCode_ConstantValue(t *testing.T) {
	if ToolClaudeCode != "claude-code" {
		t.Fatalf("ToolClaudeCode want claude-code, got %q", ToolClaudeCode)
	}
}

func TestCatalog_ContainsExactlyOneEntry_WithClaudeCodeID(t *testing.T) {
	if len(Catalog) != 1 {
		t.Fatalf("Catalog must have exactly 1 entry, got %d", len(Catalog))
	}
	if Catalog[0].ID != ToolClaudeCode {
		t.Fatalf("Catalog[0].ID want %q, got %q", ToolClaudeCode, Catalog[0].ID)
	}
}
