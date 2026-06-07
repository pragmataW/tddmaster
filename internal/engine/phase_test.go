package engine

import (
	"testing"
)

func TestPhaseComplete_Value(t *testing.T) {
	if PhaseComplete != PhaseID("completed") {
		t.Fatalf("PhaseComplete = %q, want %q", PhaseComplete, "completed")
	}
}

func TestPhaseID_IsStringType(t *testing.T) {
	var id PhaseID = "discovery"
	if string(id) != "discovery" {
		t.Fatalf("PhaseID string conversion: got %q, want %q", string(id), "discovery")
	}
}

func TestPhaseDef_ZeroValue(t *testing.T) {
	var pd PhaseDef
	if pd.ID != "" {
		t.Fatalf("zero PhaseDef.ID should be empty, got %q", pd.ID)
	}
	if pd.Driver != nil {
		t.Fatalf("zero PhaseDef.Driver should be nil")
	}
}

func TestPhaseDef_Construct(t *testing.T) {
	pd := PhaseDef{
		ID:     PhaseID("discovery"),
		Driver: nil,
	}
	if pd.ID != PhaseID("discovery") {
		t.Fatalf("PhaseDef.ID = %q, want %q", pd.ID, "discovery")
	}
}

func TestNextPhase_ReturnsNextID(t *testing.T) {
	defs := []PhaseDef{
		{ID: PhaseID("discovery")},
		{ID: PhaseID("execution")},
	}
	got := NextPhase(defs, PhaseID("discovery"))
	if got != PhaseID("execution") {
		t.Fatalf("NextPhase = %q, want %q", got, "execution")
	}
}

func TestNextPhase_LastPhaseReturnsComplete(t *testing.T) {
	defs := []PhaseDef{
		{ID: PhaseID("discovery")},
		{ID: PhaseID("execution")},
	}
	got := NextPhase(defs, PhaseID("execution"))
	if got != PhaseComplete {
		t.Fatalf("NextPhase last = %q, want %q", got, PhaseComplete)
	}
}

func TestNextPhase_UnknownIDReturnsComplete(t *testing.T) {
	defs := []PhaseDef{
		{ID: PhaseID("discovery")},
	}
	got := NextPhase(defs, PhaseID("nonexistent"))
	if got != PhaseComplete {
		t.Fatalf("NextPhase unknown = %q, want %q", got, PhaseComplete)
	}
}

func TestNextPhase_EmptyDefsReturnsComplete(t *testing.T) {
	got := NextPhase([]PhaseDef{}, PhaseID("anything"))
	if got != PhaseComplete {
		t.Fatalf("NextPhase empty = %q, want %q", got, PhaseComplete)
	}
}

func TestPhaseProgress_ZeroValue(t *testing.T) {
	var pp PhaseProgress
	if pp.Phase != "" {
		t.Fatalf("zero PhaseProgress.Phase should be empty, got %q", pp.Phase)
	}
	if pp.Modules != nil {
		t.Fatalf("zero PhaseProgress.Modules should be nil")
	}
}

func TestPhaseProgress_Construct(t *testing.T) {
	pp := PhaseProgress{
		Phase:   PhaseID("discovery"),
		Modules: []ModuleProgress{},
	}
	if pp.Phase != PhaseID("discovery") {
		t.Fatalf("PhaseProgress.Phase = %q", pp.Phase)
	}
}
