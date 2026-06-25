package phases

import (
	"testing"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/engine/loop"
	"github.com/pragmataW/tddmaster/internal/phasecatalog"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func TestEnabled_ReturnsExactlySixPhaseDefs(t *testing.T) {
	phases := Enabled(spec.DefaultSettings())
	if len(phases) != 6 {
		t.Fatalf("Enabled returned %d phases, want 6", len(phases))
	}
}

func TestEnabled_PhasesInCorrectLinearOrder(t *testing.T) {
	phases := Enabled(spec.DefaultSettings())
	want := []engine.PhaseID{
		phasecatalog.PhaseSettings,
		phasecatalog.PhaseDiscovery,
		phasecatalog.PhaseSpecProposal,
		phasecatalog.PhaseRefinement,
		phasecatalog.PhaseAnalysis,
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
	execDef := phases[5]
	if execDef.ID != phasecatalog.PhaseExecution {
		t.Fatalf("phases[5].ID = %q, want %q", execDef.ID, phasecatalog.PhaseExecution)
	}
	if _, ok := execDef.Driver.(*loop.LoopDriver); !ok {
		t.Fatalf("execution phase driver is %T, want *loop.LoopDriver", execDef.Driver)
	}
}

func TestEnabled_ExecutionPhaseDriverIsNotStepTableDriver(t *testing.T) {
	phases := Enabled(spec.DefaultSettings())
	execDef := phases[5]
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
	if got != phasecatalog.PhaseAnalysis {
		t.Fatalf("NextPhase(refinement) = %q, want %q", got, phasecatalog.PhaseAnalysis)
	}

	got = engine.NextPhase(phases, phasecatalog.PhaseAnalysis)
	if got != phasecatalog.PhaseExecution {
		t.Fatalf("NextPhase(cross-artifact-analysis) = %q, want %q", got, phasecatalog.PhaseExecution)
	}

	got = engine.NextPhase(phases, phasecatalog.PhaseExecution)
	if got != engine.PhaseComplete {
		t.Fatalf("NextPhase(execution) = %q, want %q", got, engine.PhaseComplete)
	}
}

func TestEnabled_SettingsTDDOnAndOff_SameSixPhasesWithLoopDriver(t *testing.T) {
	settingsTDDOn := spec.Settings{TDDEnabled: true, SkipVerifierEnabled: false, ImportantTaskGateEnabled: false}
	settingsTDDOff := spec.Settings{TDDEnabled: false, SkipVerifierEnabled: false, ImportantTaskGateEnabled: false}

	phasesOn := Enabled(settingsTDDOn)
	phasesOff := Enabled(settingsTDDOff)

	if len(phasesOn) != 6 {
		t.Fatalf("TDDEnabled=true: Enabled returned %d phases, want 6", len(phasesOn))
	}
	if len(phasesOff) != 6 {
		t.Fatalf("TDDEnabled=false: Enabled returned %d phases, want 6", len(phasesOff))
	}

	if _, ok := phasesOn[5].Driver.(*loop.LoopDriver); !ok {
		t.Fatalf("TDDEnabled=true: execution driver is %T, want *loop.LoopDriver", phasesOn[5].Driver)
	}
	if _, ok := phasesOff[5].Driver.(*loop.LoopDriver); !ok {
		t.Fatalf("TDDEnabled=false: execution driver is %T, want *loop.LoopDriver", phasesOff[5].Driver)
	}
}

func TestEnabled_SettingsSkipVerifierAndImportantGate_SameSixPhasesWithLoopDriver(t *testing.T) {
	settings := spec.Settings{TDDEnabled: true, SkipVerifierEnabled: true, ImportantTaskGateEnabled: true}

	phases := Enabled(settings)

	if len(phases) != 6 {
		t.Fatalf("Enabled returned %d phases, want 6", len(phases))
	}
	if _, ok := phases[5].Driver.(*loop.LoopDriver); !ok {
		t.Fatalf("execution driver is %T, want *loop.LoopDriver", phases[5].Driver)
	}
}

func TestEnabled_FirstPhaseIDMatchesPhaseInitial(t *testing.T) {
	phases := Enabled(spec.DefaultSettings())
	if string(phases[0].ID) != spec.PhaseInitial {
		t.Fatalf("Enabled[0].ID = %q, want spec.PhaseInitial = %q", phases[0].ID, spec.PhaseInitial)
	}
}

func TestEnabled_AnalysisPhase_Unconditional(t *testing.T) {
	allOff := spec.Settings{
		TDDEnabled:               false,
		SkipVerifierEnabled:      false,
		ImportantTaskGateEnabled: false,
		MinTestCoverage:          0,
		RuleLearningEnabled:      false,
	}
	phases := Enabled(allOff)

	analysisIdx := -1
	refinementIdx := -1
	executionIdx := -1
	for i, d := range phases {
		if d.ID == phasecatalog.PhaseAnalysis {
			analysisIdx = i
		}
		if d.ID == phasecatalog.PhaseRefinement {
			refinementIdx = i
		}
		if d.ID == phasecatalog.PhaseExecution {
			executionIdx = i
		}
	}

	if analysisIdx == -1 {
		t.Fatal("PhaseAnalysis not found even when all settings flags are false")
	}
	if analysisIdx <= refinementIdx {
		t.Fatalf("PhaseAnalysis (idx=%d) must come after PhaseRefinement (idx=%d)", analysisIdx, refinementIdx)
	}
	if analysisIdx >= executionIdx {
		t.Fatalf("PhaseAnalysis (idx=%d) must come before PhaseExecution (idx=%d)", analysisIdx, executionIdx)
	}
}

func TestEnabled_AnalysisDriver_IsAnalysisDriver(t *testing.T) {
	phases := Enabled(spec.DefaultSettings())
	var analysisDef *engine.PhaseDef
	for i := range phases {
		if phases[i].ID == phasecatalog.PhaseAnalysis {
			analysisDef = &phases[i]
			break
		}
	}
	if analysisDef == nil {
		t.Fatal("PhaseAnalysis not found in Enabled()")
	}
	if _, ok := analysisDef.Driver.(*analysisDriver); !ok {
		t.Fatalf("analysis phase driver is %T, want *analysisDriver", analysisDef.Driver)
	}
	if _, ok := analysisDef.Driver.(*loop.LoopDriver); ok {
		t.Fatal("analysis phase driver must NOT be *loop.LoopDriver")
	}
	if _, ok := analysisDef.Driver.(*engine.StepTableDriver); ok {
		t.Fatal("analysis phase driver must NOT be *engine.StepTableDriver")
	}
}
