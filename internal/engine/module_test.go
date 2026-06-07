package engine

import (
	"encoding/json"
	"testing"
)

func TestModuleID_IsStringType(t *testing.T) {
	var id ModuleID = "recon"
	if string(id) != "recon" {
		t.Fatalf("ModuleID string conversion: got %q", string(id))
	}
}

func TestModuleDef_ZeroValue(t *testing.T) {
	var md ModuleDef
	if md.ID != "" {
		t.Fatalf("zero ModuleDef.ID should be empty")
	}
	if md.Steps != nil {
		t.Fatalf("zero ModuleDef.Steps should be nil")
	}
}

func TestModuleDef_Construct(t *testing.T) {
	md := ModuleDef{
		ID: ModuleID("discovery"),
		Steps: []StepDef{
			{ID: StepID("step-a")},
			{ID: StepID("step-b")},
		},
	}
	if md.ID != ModuleID("discovery") {
		t.Fatalf("ModuleDef.ID = %q, want %q", md.ID, "discovery")
	}
	if len(md.Steps) != 2 {
		t.Fatalf("ModuleDef.Steps len = %d, want 2", len(md.Steps))
	}
}

func TestModuleDef_EmptySteps(t *testing.T) {
	md := ModuleDef{
		ID:    ModuleID("empty-module"),
		Steps: []StepDef{},
	}
	if len(md.Steps) != 0 {
		t.Fatalf("ModuleDef.Steps len = %d, want 0", len(md.Steps))
	}
}

func TestModuleProgress_ZeroValue(t *testing.T) {
	var mp ModuleProgress
	if mp.Module != "" {
		t.Fatalf("zero ModuleProgress.Module should be empty")
	}
	if mp.Steps != nil {
		t.Fatalf("zero ModuleProgress.Steps should be nil")
	}
}

func TestModuleProgress_Construct(t *testing.T) {
	mp := ModuleProgress{
		Module: ModuleID("recon"),
		Steps: []StepProgress{
			{Step: StepID("step-1"), Answered: true},
		},
	}
	if mp.Module != ModuleID("recon") {
		t.Fatalf("ModuleProgress.Module = %q, want %q", mp.Module, "recon")
	}
	if len(mp.Steps) != 1 {
		t.Fatalf("ModuleProgress.Steps len = %d, want 1", len(mp.Steps))
	}
}

func TestModuleProgress_JSONRoundTrip(t *testing.T) {
	mp := ModuleProgress{
		Module: ModuleID("analysis"),
		Steps: []StepProgress{
			{Step: StepID("s1"), Answered: false},
			{Step: StepID("s2"), Answered: true, Answer: json.RawMessage(`"done"`)},
		},
	}
	b, err := json.Marshal(mp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got ModuleProgress
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Module != mp.Module {
		t.Fatalf("Module round-trip: got %q, want %q", got.Module, mp.Module)
	}
	if len(got.Steps) != 2 {
		t.Fatalf("Steps round-trip len: got %d, want 2", len(got.Steps))
	}
	if got.Steps[1].Answered != true {
		t.Fatalf("Steps[1].Answered round-trip: got %v, want true", got.Steps[1].Answered)
	}
}

func TestModuleProgress_EmptySteps(t *testing.T) {
	mp := ModuleProgress{
		Module: ModuleID("empty"),
		Steps:  []StepProgress{},
	}
	b, err := json.Marshal(mp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got ModuleProgress
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got.Steps) != 0 {
		t.Fatalf("Steps round-trip: got %d, want 0", len(got.Steps))
	}
}
