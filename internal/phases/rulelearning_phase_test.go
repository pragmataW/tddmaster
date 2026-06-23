package phases

import (
	"testing"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/phasecatalog"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func TestPhaseRuleLearning_ConstantValue(t *testing.T) {
	if phasecatalog.PhaseRuleLearning != engine.PhaseID("rule-learning") {
		t.Fatalf("PhaseRuleLearning = %q, want %q", phasecatalog.PhaseRuleLearning, "rule-learning")
	}
}

func TestEnabled_RuleLearningEnabled_LastPhaseIsRuleLearning(t *testing.T) {
	s := spec.Settings{
		TDDEnabled:          true,
		RuleLearningEnabled: true,
	}
	defs := Enabled(s)
	if len(defs) == 0 {
		t.Fatal("Enabled returned empty slice")
	}
	last := defs[len(defs)-1]
	if last.ID != phasecatalog.PhaseRuleLearning {
		t.Fatalf("last phase = %q, want %q", last.ID, phasecatalog.PhaseRuleLearning)
	}
}

func TestEnabled_RuleLearningEnabled_RuleLearningAfterExecution(t *testing.T) {
	s := spec.Settings{
		TDDEnabled:          true,
		RuleLearningEnabled: true,
	}
	defs := Enabled(s)
	execIdx := -1
	rlIdx := -1
	for i, d := range defs {
		if d.ID == phasecatalog.PhaseExecution {
			execIdx = i
		}
		if d.ID == phasecatalog.PhaseRuleLearning {
			rlIdx = i
		}
	}
	if execIdx == -1 {
		t.Fatal("PhaseExecution not found in defs")
	}
	if rlIdx == -1 {
		t.Fatal("PhaseRuleLearning not found in defs")
	}
	if rlIdx <= execIdx {
		t.Fatalf("PhaseRuleLearning (idx=%d) must come after PhaseExecution (idx=%d)", rlIdx, execIdx)
	}
}

func TestEnabled_RuleLearningDisabled_NoRuleLearningPhase(t *testing.T) {
	s := spec.Settings{
		TDDEnabled:          true,
		RuleLearningEnabled: false,
	}
	defs := Enabled(s)
	for _, d := range defs {
		if d.ID == phasecatalog.PhaseRuleLearning {
			t.Fatalf("PhaseRuleLearning must not appear when RuleLearningEnabled=false")
		}
	}
}

func TestEnabled_RuleLearningDisabled_ExecutionIsLast(t *testing.T) {
	s := spec.Settings{
		TDDEnabled:          false,
		RuleLearningEnabled: false,
	}
	defs := Enabled(s)
	if len(defs) == 0 {
		t.Fatal("Enabled returned empty slice")
	}
	last := defs[len(defs)-1]
	if last.ID != phasecatalog.PhaseExecution {
		t.Fatalf("last phase = %q, want %q when RuleLearningEnabled=false", last.ID, phasecatalog.PhaseExecution)
	}
}
