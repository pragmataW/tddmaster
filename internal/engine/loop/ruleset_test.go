package loop

import (
	"testing"

	"github.com/pragmataW/tddmaster/internal/spec"
)

func TestNewRuleSet_ReturnsNonNilRuleSet(t *testing.T) {
	rs := newRuleSet()
	if rs == nil {
		t.Fatal("newRuleSet: got nil, want valid RuleSet")
	}
}

func TestRuleSet_Next_TDDOff_SkipVerifyOff_ReturnsExecutor(t *testing.T) {
	rs := newRuleSet()
	settings := makeSettings(false, false, false)
	task := makeTask("t-1", false, false)
	ctx := makeExecCtx(settings, task, makeExecState(""), 0, 3)

	stage, ok := rs.Next(ctx)

	if !ok {
		t.Fatal("Next: got ok=false, want ok=true (executor applies)")
	}
	if stage.ID() != StageIDExecutor {
		t.Errorf("Next stage ID: got %q, want %q", stage.ID(), StageIDExecutor)
	}
}

func TestRuleSet_Next_TDDOff_SkipVerifyOn_ReturnsExecutor(t *testing.T) {
	rs := newRuleSet()
	settings := makeSettings(false, true, false)
	task := makeTask("t-1", false, false)
	ctx := makeExecCtx(settings, task, makeExecState(""), 0, 3)

	stage, ok := rs.Next(ctx)

	if !ok {
		t.Fatal("Next: got ok=false, want ok=true")
	}
	if stage.ID() != StageIDExecutor {
		t.Errorf("Next stage ID: got %q, want %q", stage.ID(), StageIDExecutor)
	}
}

func TestRuleSet_Next_TDDOn_CycleRed_ReturnsRed(t *testing.T) {
	rs := newRuleSet()
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, makeExecState("red"), 0, 3)

	stage, ok := rs.Next(ctx)

	if !ok {
		t.Fatal("Next: got ok=false, want ok=true")
	}
	if stage.ID() != StageIDRed {
		t.Errorf("Next stage ID: got %q, want %q", stage.ID(), StageIDRed)
	}
}

func TestRuleSet_Next_TDDOn_CycleGreen_ReturnsGreen(t *testing.T) {
	rs := newRuleSet()
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, makeExecState("green"), 0, 3)

	stage, ok := rs.Next(ctx)

	if !ok {
		t.Fatal("Next: got ok=false, want ok=true")
	}
	if stage.ID() != StageIDGreen {
		t.Errorf("Next stage ID: got %q, want %q", stage.ID(), StageIDGreen)
	}
}

func TestRuleSet_Next_TDDOn_CycleRefactor_ReturnsRefactor(t *testing.T) {
	rs := newRuleSet()
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, makeExecState("refactor"), 0, 3)

	stage, ok := rs.Next(ctx)

	if !ok {
		t.Fatal("Next: got ok=false, want ok=true")
	}
	if stage.ID() != StageIDRefactor {
		t.Errorf("Next stage ID: got %q, want %q", stage.ID(), StageIDRefactor)
	}
}

func TestRuleSet_Next_GateEnabledImportantUnapproved_ReturnsGateFirst(t *testing.T) {
	rs := newRuleSet()
	settings := makeSettings(false, false, true)
	task := makeImportantTask("t-gate", false)
	ctx := makeExecCtx(settings, task, makeExecState(""), 0, 3)

	stage, ok := rs.Next(ctx)

	if !ok {
		t.Fatal("Next: got ok=false, want ok=true")
	}
	if stage.ID() != StageIDGate {
		t.Errorf("Next stage ID: got %q, want %q (gate must win over executor)", stage.ID(), StageIDGate)
	}
}

func TestRuleSet_Next_GateEnabled_ImportantApproved_GateFallsThrough(t *testing.T) {
	rs := newRuleSet()
	settings := makeSettings(false, false, true)
	task := makeImportantTask("t-gate", false)
	st := stateWithApprovedPlan()
	ctx := makeExecCtx(settings, task, st, 0, 3)

	stage, ok := rs.Next(ctx)

	if !ok {
		t.Fatal("Next: got ok=false, want ok=true (executor applies after gate falls through)")
	}
	if stage.ID() == StageIDGate {
		t.Error("Next stage ID: got gate, want executor (plan already approved)")
	}
	if stage.ID() != StageIDExecutor {
		t.Errorf("Next stage ID: got %q, want %q", stage.ID(), StageIDExecutor)
	}
}

func TestRuleSet_Next_GateFirst_TDDOnImportantUnapproved_GateBeforeRed(t *testing.T) {
	rs := newRuleSet()
	settings := makeSettings(true, false, true)
	task := makeImportantTask("t-gate", true)
	ctx := makeExecCtx(settings, task, makeExecState("red"), 0, 3)

	stage, ok := rs.Next(ctx)

	if !ok {
		t.Fatal("Next: got ok=false, want ok=true")
	}
	if stage.ID() != StageIDGate {
		t.Errorf("Next stage ID: got %q, want %q (gate before red in table order)", stage.ID(), StageIDGate)
	}
}

func TestRuleSet_Next_NoStageApplies_ReturnsFalse(t *testing.T) {
	rs := newRuleSet()
	settings := spec.Settings{
		TDDEnabled:               true,
		SkipVerifierEnabled:      true,
		ImportantTaskGateEnabled: false,
	}
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, makeExecState(""), 0, 3)

	_, ok := rs.Next(ctx)

	if ok {
		t.Error("Next: got ok=true, want ok=false (no stage applies: TDD on + empty cycle + skipVerify + no gate)")
	}
}

func TestRuleSet_Next_TDDOff_SkipVerifyOff_TableOrder_ExecutorBeforeVerifier(t *testing.T) {
	rs := newRuleSet()
	settings := makeSettings(false, false, false)
	task := makeTask("t-1", false, false)
	ctx := makeExecCtx(settings, task, makeExecState(""), 0, 3)

	stage, ok := rs.Next(ctx)

	if !ok {
		t.Fatal("Next: got ok=false, want ok=true")
	}
	if stage.ID() != StageIDExecutor {
		t.Errorf("table order: executor must precede verifier; got %q first", stage.ID())
	}
}

func TestRuleSet_Next_SkipVerifyOn_TDDOff_VerifierNeverSelected(t *testing.T) {
	rs := newRuleSet()
	settings := makeSettings(false, true, false)
	task := makeTask("t-1", false, false)
	ctx := makeExecCtx(settings, task, makeExecState(""), 0, 3)

	stage, ok := rs.Next(ctx)

	if !ok {
		t.Fatal("Next: got ok=false")
	}
	if stage.ID() == StageIDVerifier {
		t.Error("verifier must not be selected when skipVerify=on and TDD=off")
	}
}

func TestRuleSet_Next_SkipVerifyOn_TDDOn_CycleGreen_GreenBeforeVerifier(t *testing.T) {
	rs := newRuleSet()
	settings := makeSettings(true, true, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, makeExecState("green"), 0, 3)

	stage, ok := rs.Next(ctx)

	if !ok {
		t.Fatal("Next: got ok=false, want ok=true")
	}
	if stage.ID() != StageIDGreen {
		t.Errorf("table order: green must precede verifier; got %q", stage.ID())
	}
}

func TestRuleSet_Next_SkipVerifyOn_TDDOn_CycleRefactor_VerifierNotSelected(t *testing.T) {
	rs := newRuleSet()
	settings := makeSettings(true, true, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, makeExecState("refactor"), 0, 3)

	stage, ok := rs.Next(ctx)

	if !ok {
		t.Fatal("Next: got ok=false, want ok=true (refactor applies)")
	}
	if stage.ID() != StageIDRefactor {
		t.Errorf("Next stage ID: got %q, want %q", stage.ID(), StageIDRefactor)
	}
}

func TestRuleSet_Next_PerTaskTDD_TDDTask_RedPath(t *testing.T) {
	rs := newRuleSet()
	settings := makeSettings(true, false, false)
	tddTask := makeTask("t-tdd", true, false)
	ctx := makeExecCtx(settings, tddTask, makeExecState("red"), 0, 3)

	stage, ok := rs.Next(ctx)

	if !ok {
		t.Fatal("Next: got ok=false, want ok=true")
	}
	if stage.ID() != StageIDRed {
		t.Errorf("TDD task red cycle: got %q, want %q", stage.ID(), StageIDRed)
	}
}

func TestRuleSet_Next_PerTaskTDD_NonTDDTask_ExecutorPath(t *testing.T) {
	rs := newRuleSet()
	settings := makeSettings(true, false, false)
	nonTDDTask := makeTask("t-notdd", false, false)
	ctx := makeExecCtx(settings, nonTDDTask, makeExecState(""), 0, 3)

	stage, ok := rs.Next(ctx)

	if !ok {
		t.Fatal("Next: got ok=false, want ok=true")
	}
	if stage.ID() != StageIDExecutor {
		t.Errorf("non-TDD task: got %q, want %q", stage.ID(), StageIDExecutor)
	}
}

func TestRuleSet_Next_RedToGreen_Transition(t *testing.T) {
	rs := newRuleSet()
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)

	redCtx := makeExecCtx(settings, task, makeExecState("red"), 0, 3)
	redStageResult, ok := rs.Next(redCtx)
	if !ok || redStageResult.ID() != StageIDRed {
		t.Fatalf("expected red stage in red cycle, got ok=%v id=%q", ok, func() string {
			if ok {
				return redStageResult.ID()
			}
			return "<none>"
		}())
	}

	greenCtx := makeExecCtx(settings, task, makeExecState("green"), 0, 3)
	greenStageResult, ok := rs.Next(greenCtx)
	if !ok || greenStageResult.ID() != StageIDGreen {
		t.Errorf("expected green stage after advancing to green cycle, got ok=%v id=%q", ok, func() string {
			if ok {
				return greenStageResult.ID()
			}
			return "<none>"
		}())
	}
}

func TestRuleSet_Next_RefactorCap_RefactorStageReturned(t *testing.T) {
	rs := newRuleSet()
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	st := makeExecState("refactor")
	st.RefactorRounds = 2
	st.RefactorApplied = true
	ctx := makeExecCtx(settings, task, st, 0, 3)

	stage, ok := rs.Next(ctx)

	if !ok {
		t.Fatal("Next: got ok=false, want ok=true (refactor still applies at cap; OnReport handles cap logic)")
	}
	if stage.ID() != StageIDRefactor {
		t.Errorf("Next stage ID: got %q, want %q", stage.ID(), StageIDRefactor)
	}
}

func TestRuleSet_Next_GreenWithNotes_VerifierAlsoApplies_GreenSelectedFirst(t *testing.T) {
	rs := newRuleSet()
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, makeExecState("green"), 0, 3)

	stage, ok := rs.Next(ctx)

	if !ok {
		t.Fatal("Next: got ok=false")
	}
	if stage.ID() != StageIDGreen {
		t.Errorf("table order: green (index 2) before verifier (index 5); got %q first", stage.ID())
	}
}
