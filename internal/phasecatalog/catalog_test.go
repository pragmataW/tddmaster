package phasecatalog

import (
	"testing"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func TestPhaseID_Constants_HaveExpectedStringValues(t *testing.T) {
	if string(PhaseListenFirst) != "listen-first" {
		t.Fatalf("PhaseListenFirst = %q, want %q", PhaseListenFirst, "listen-first")
	}
	if string(PhaseDiscovery) != "discovery" {
		t.Fatalf("PhaseDiscovery = %q, want %q", PhaseDiscovery, "discovery")
	}
	if string(PhaseSpecProposal) != "spec-proposal" {
		t.Fatalf("PhaseSpecProposal = %q, want %q", PhaseSpecProposal, "spec-proposal")
	}
	if string(PhaseRefinement) != "refinement" {
		t.Fatalf("PhaseRefinement = %q, want %q", PhaseRefinement, "refinement")
	}
	if string(PhaseExecution) != "execution" {
		t.Fatalf("PhaseExecution = %q, want %q", PhaseExecution, "execution")
	}
}

func TestPhaseID_Constants_UseEnginePhaseIDType(t *testing.T) {
	var _ engine.PhaseID = PhaseListenFirst
	var _ engine.PhaseID = PhaseDiscovery
	var _ engine.PhaseID = PhaseSpecProposal
	var _ engine.PhaseID = PhaseRefinement
	var _ engine.PhaseID = PhaseExecution
}

func TestModuleID_Constants_Exist(t *testing.T) {
	ids := []engine.ModuleID{
		ModListenFirst,
		ModDiscovery,
		ModSpecProposal,
		ModRefinement,
		ModExecution,
	}
	for _, id := range ids {
		if string(id) == "" {
			t.Fatalf("ModuleID constant is empty")
		}
	}
}

func TestStepID_Constants_Exist(t *testing.T) {
	ids := []engine.StepID{
		StepListenFirstAsk,
		StepDiscoveryAsk,
		StepSpecProposalAsk,
		StepRefinementAsk,
		StepExecutionAsk,
	}
	for _, id := range ids {
		if string(id) == "" {
			t.Fatalf("StepID constant is empty")
		}
	}
}

func TestCatalog_HasExactlyFivePhaseDefs(t *testing.T) {
	cat := Catalog()
	if len(cat) != 5 {
		t.Fatalf("Catalog length = %d, want 5", len(cat))
	}
}

func TestCatalog_PhasesInCorrectLinearOrder(t *testing.T) {
	cat := Catalog()
	want := []engine.PhaseID{
		PhaseSettings,
		PhaseDiscovery,
		PhaseSpecProposal,
		PhaseRefinement,
		PhaseExecution,
	}
	for i, def := range cat {
		if def.ID != want[i] {
			t.Fatalf("Catalog[%d].ID = %q, want %q", i, def.ID, want[i])
		}
	}
}

func TestCatalog_AllPhaseDefsHaveNonNilDriver(t *testing.T) {
	cat := Catalog()
	for _, def := range cat {
		if def.Driver == nil {
			t.Fatalf("phase %q has nil Driver", def.ID)
		}
	}
}

func TestCatalog_Phases1To3DriversAreStepTableDriver(t *testing.T) {
	cat := Catalog()
	linearPhases := cat[:4]
	for _, def := range linearPhases {
		if _, ok := def.Driver.(*engine.StepTableDriver); !ok {
			t.Fatalf("phase %q driver is %T, want *engine.StepTableDriver", def.ID, def.Driver)
		}
	}
}

func TestCatalog_ExecutionPhaseDriverIsStepTableDriver(t *testing.T) {
	cat := Catalog()
	execDef := cat[4]
	if execDef.ID != PhaseExecution {
		t.Fatalf("last phase ID = %q, want %q", execDef.ID, PhaseExecution)
	}
	if _, ok := execDef.Driver.(*engine.StepTableDriver); !ok {
		t.Fatalf("execution phase driver is %T, want *engine.StepTableDriver (placeholder)", execDef.Driver)
	}
}

func TestCatalog_Phases1To3DriversHaveNonEmptyModules(t *testing.T) {
	cat := Catalog()
	for _, def := range cat[:4] {
		d, ok := def.Driver.(*engine.StepTableDriver)
		if !ok {
			t.Fatalf("phase %q driver is not *engine.StepTableDriver", def.ID)
		}
		if len(d.Modules) == 0 {
			t.Fatalf("phase %q StepTableDriver has no modules", def.ID)
		}
	}
}

func TestCatalog_EachLinearPhaseModuleHasAtLeastOneStep(t *testing.T) {
	cat := Catalog()
	for _, def := range cat[:4] {
		d := def.Driver.(*engine.StepTableDriver)
		for _, mod := range d.Modules {
			if len(mod.Steps) == 0 {
				t.Fatalf("phase %q module %q has no steps", def.ID, mod.ID)
			}
		}
	}
}

func TestNextPhase_SettingsToDiscovery(t *testing.T) {
	cat := Catalog()
	got := engine.NextPhase(cat, PhaseSettings)
	if got != PhaseDiscovery {
		t.Fatalf("NextPhase(spec-settings) = %q, want %q", got, PhaseDiscovery)
	}
}

func TestNextPhase_DiscoveryToSpecProposal(t *testing.T) {
	cat := Catalog()
	got := engine.NextPhase(cat, PhaseDiscovery)
	if got != PhaseSpecProposal {
		t.Fatalf("NextPhase(discovery) = %q, want %q", got, PhaseSpecProposal)
	}
}

func TestNextPhase_SpecProposalToRefinement(t *testing.T) {
	cat := Catalog()
	got := engine.NextPhase(cat, PhaseSpecProposal)
	if got != PhaseRefinement {
		t.Fatalf("NextPhase(spec-proposal) = %q, want %q", got, PhaseRefinement)
	}
}

func TestNextPhase_RefinementToExecution(t *testing.T) {
	cat := Catalog()
	got := engine.NextPhase(cat, PhaseRefinement)
	if got != PhaseExecution {
		t.Fatalf("NextPhase(refinement) = %q, want %q", got, PhaseExecution)
	}
}

func TestNextPhase_ExecutionReturnsPhaseComplete(t *testing.T) {
	cat := Catalog()
	got := engine.NextPhase(cat, PhaseExecution)
	if got != engine.PhaseComplete {
		t.Fatalf("NextPhase(execution) = %q, want %q", got, engine.PhaseComplete)
	}
}

func TestCatalog_FirstPhaseAlignsWithSpecPhaseInitial(t *testing.T) {
	cat := Catalog()
	if string(cat[0].ID) != spec.PhaseInitial {
		t.Fatalf("Catalog[0].ID = %q, want spec.PhaseInitial = %q", cat[0].ID, spec.PhaseInitial)
	}
}
