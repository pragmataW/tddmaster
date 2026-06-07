package phases

import (
	"testing"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/engine/loop"
	"github.com/pragmataW/tddmaster/internal/phasecatalog"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func TestEnabled_ReturnsExactlyFivePhaseDefs(t *testing.T) {
	phases := Enabled(spec.DefaultSettings())
	if len(phases) != 5 {
		t.Fatalf("Enabled returned %d phases, want 5", len(phases))
	}
}

func TestEnabled_PhasesInCorrectLinearOrder(t *testing.T) {
	phases := Enabled(spec.DefaultSettings())
	want := []engine.PhaseID{
		phasecatalog.PhaseSettings,
		phasecatalog.PhaseDiscovery,
		phasecatalog.PhaseSpecProposal,
		phasecatalog.PhaseRefinement,
		phasecatalog.PhaseExecution,
	}
	for i, def := range phases {
		if def.ID != want[i] {
			t.Fatalf("Enabled[%d].ID = %q, want %q", i, def.ID, want[i])
		}
	}
}

func TestEnabled_AllPhaseDefsHaveNonNilDriver(t *testing.T) {
	phases := Enabled(spec.DefaultSettings())
	for _, def := range phases {
		if def.Driver == nil {
			t.Fatalf("phase %q has nil Driver", def.ID)
		}
	}
}

func TestEnabled_SettingsDriverIsSettingsDriver(t *testing.T) {
	phases := Enabled(spec.DefaultSettings())
	if _, ok := phases[0].Driver.(*settingsDriver); !ok {
		t.Fatalf("settings phase driver is %T, want *settingsDriver", phases[0].Driver)
	}
}

func TestEnabled_DiscoveryDriverIsDiscoveryDriver(t *testing.T) {
	phases := Enabled(spec.DefaultSettings())
	if _, ok := phases[1].Driver.(*discoveryDriver); !ok {
		t.Fatalf("discovery phase driver is %T, want *discoveryDriver", phases[1].Driver)
	}
}

func TestEnabled_SpecProposalDriverIsSpecProposalDriver(t *testing.T) {
	phases := Enabled(spec.DefaultSettings())
	if _, ok := phases[2].Driver.(*specProposalDriver); !ok {
		t.Fatalf("spec-proposal phase driver is %T, want *specProposalDriver", phases[2].Driver)
	}
}

func TestEnabled_RefinementDriverIsRefinementDriver(t *testing.T) {
	phases := Enabled(spec.DefaultSettings())
	if _, ok := phases[3].Driver.(*refinementDriver); !ok {
		t.Fatalf("refinement phase driver is %T, want *refinementDriver", phases[3].Driver)
	}
}

func TestEnabled_ExecutionPhaseDriverIsLoopDriver(t *testing.T) {
	phases := Enabled(spec.DefaultSettings())
	execDef := phases[4]
	if execDef.ID != phasecatalog.PhaseExecution {
		t.Fatalf("last phase ID = %q, want %q", execDef.ID, phasecatalog.PhaseExecution)
	}
	if _, ok := execDef.Driver.(*loop.LoopDriver); !ok {
		t.Fatalf("execution phase driver is %T, want *loop.LoopDriver", execDef.Driver)
	}
}

func TestEnabled_ExecutionPhaseDriverIsNotStepTableDriver(t *testing.T) {
	phases := Enabled(spec.DefaultSettings())
	execDef := phases[4]
	if _, ok := execDef.Driver.(*engine.StepTableDriver); ok {
		t.Fatalf("execution phase driver is still *engine.StepTableDriver (placeholder not swapped)")
	}
}

func TestEnabled_NextPhaseChainIsCorrect(t *testing.T) {
	phases := Enabled(spec.DefaultSettings())

	got := engine.NextPhase(phases, phasecatalog.PhaseSettings)
	if got != phasecatalog.PhaseDiscovery {
		t.Fatalf("NextPhase(spec-settings) = %q, want %q", got, phasecatalog.PhaseDiscovery)
	}

	got = engine.NextPhase(phases, phasecatalog.PhaseDiscovery)
	if got != phasecatalog.PhaseSpecProposal {
		t.Fatalf("NextPhase(discovery) = %q, want %q", got, phasecatalog.PhaseSpecProposal)
	}

	got = engine.NextPhase(phases, phasecatalog.PhaseSpecProposal)
	if got != phasecatalog.PhaseRefinement {
		t.Fatalf("NextPhase(spec-proposal) = %q, want %q", got, phasecatalog.PhaseRefinement)
	}

	got = engine.NextPhase(phases, phasecatalog.PhaseRefinement)
	if got != phasecatalog.PhaseExecution {
		t.Fatalf("NextPhase(refinement) = %q, want %q", got, phasecatalog.PhaseExecution)
	}

	got = engine.NextPhase(phases, phasecatalog.PhaseExecution)
	if got != engine.PhaseComplete {
		t.Fatalf("NextPhase(execution) = %q, want %q", got, engine.PhaseComplete)
	}
}

func TestEnabled_SettingsTDDOnAndOff_SameFivePhasesWithLoopDriver(t *testing.T) {
	settingsTDDOn := spec.Settings{TDDEnabled: true, SkipVerifierEnabled: false, ImportantTaskGateEnabled: false}
	settingsTDDOff := spec.Settings{TDDEnabled: false, SkipVerifierEnabled: false, ImportantTaskGateEnabled: false}

	phasesOn := Enabled(settingsTDDOn)
	phasesOff := Enabled(settingsTDDOff)

	if len(phasesOn) != 5 {
		t.Fatalf("TDDEnabled=true: Enabled returned %d phases, want 5", len(phasesOn))
	}
	if len(phasesOff) != 5 {
		t.Fatalf("TDDEnabled=false: Enabled returned %d phases, want 5", len(phasesOff))
	}

	if _, ok := phasesOn[4].Driver.(*loop.LoopDriver); !ok {
		t.Fatalf("TDDEnabled=true: execution driver is %T, want *loop.LoopDriver", phasesOn[4].Driver)
	}
	if _, ok := phasesOff[4].Driver.(*loop.LoopDriver); !ok {
		t.Fatalf("TDDEnabled=false: execution driver is %T, want *loop.LoopDriver", phasesOff[4].Driver)
	}
}

func TestEnabled_SettingsSkipVerifierAndImportantGate_SameFivePhasesWithLoopDriver(t *testing.T) {
	settings := spec.Settings{TDDEnabled: true, SkipVerifierEnabled: true, ImportantTaskGateEnabled: true}

	phases := Enabled(settings)

	if len(phases) != 5 {
		t.Fatalf("Enabled returned %d phases, want 5", len(phases))
	}
	if _, ok := phases[4].Driver.(*loop.LoopDriver); !ok {
		t.Fatalf("execution driver is %T, want *loop.LoopDriver", phases[4].Driver)
	}
}

func TestEnabled_FirstPhaseIDMatchesPhaseInitial(t *testing.T) {
	phases := Enabled(spec.DefaultSettings())
	if string(phases[0].ID) != spec.PhaseInitial {
		t.Fatalf("Enabled[0].ID = %q, want spec.PhaseInitial = %q", phases[0].ID, spec.PhaseInitial)
	}
}
