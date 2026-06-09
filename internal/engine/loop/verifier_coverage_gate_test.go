package loop

import (
	"testing"

	"github.com/pragmataW/tddmaster/internal/spec"
)

func makeSettingsWithCoverage(tdd bool, minCoverage int) spec.Settings {
	return spec.Settings{
		TDDEnabled:      tdd,
		MinTestCoverage: minCoverage,
	}
}

func makeGreenTDDCtx(minCoverage int) ExecCtx {
	settings := makeSettingsWithCoverage(true, minCoverage)
	task := makeTask("t-cov", true, false)
	st := makeExecState(cycleGreen)
	st.Implemented = true
	return makeExecCtx(settings, task, st, 0, 3)
}

func passedReport(files []FileCoverageEntry) StageReport {
	return StageReport{
		Passed:       true,
		FileCoverage: files,
	}
}

func TestVerifierOnReport_NotEffectivePassed_SetsImplementedFalse_NoCoverageConsulted(t *testing.T) {
	ctx := makeGreenTDDCtx(80)
	report := StageReport{
		Passed:    false,
		FailedACs: []string{"AC-1"},
		FileCoverage: []FileCoverageEntry{
			{File: "foo.go", Coverage: 100},
		},
	}

	result, err := verifierStageImpl{}.OnReport(ctx, report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.State.Implemented {
		t.Error("Implemented: got true, want false when report not effective-passed")
	}
	if result.State.TDDCycle != cycleGreen {
		t.Errorf("TDDCycle: got %q, want %q — coverage gate must not run when report failed", result.State.TDDCycle, cycleGreen)
	}
	if result.State.LastCoverage != nil {
		t.Error("LastCoverage: must not be written when report not effective-passed")
	}
}

func TestVerifierOnReport_NonTDD_NoCoverageWritten_NoGate(t *testing.T) {
	settings := makeSettingsWithCoverage(false, 80)
	task := makeTask("t-nontdd", false, false)
	st := makeExecState("")
	st.Implemented = true
	ctx := makeExecCtx(settings, task, st, 0, 3)

	report := passedReport([]FileCoverageEntry{
		{File: "bar.go", Coverage: 10},
	})

	result, err := verifierStageImpl{}.OnReport(ctx, report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.State.LastCoverage != nil {
		t.Error("LastCoverage: must not be written for non-TDD context")
	}
	if result.State.TDDCycle == cycleRed {
		t.Error("TDDCycle: must not become red for non-TDD context")
	}
}

func TestVerifierOnReport_GreenEnforced_LowCoverage_GateTriggersRedAndNotImplemented(t *testing.T) {
	ctx := makeGreenTDDCtx(80)

	report := passedReport([]FileCoverageEntry{
		{File: "main.go", Coverage: 50},
	})

	result, err := verifierStageImpl{}.OnReport(ctx, report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.State.TDDCycle != cycleRed {
		t.Errorf("TDDCycle: got %q, want %q — low coverage should trigger gate back to red", result.State.TDDCycle, cycleRed)
	}
	if result.State.Implemented {
		t.Error("Implemented: got true, want false when coverage gate triggers")
	}
	if result.State.LastCoverage == nil {
		t.Fatal("LastCoverage: must be set even when gate triggers")
	}
	if result.State.LastCoverage["main.go"] != 50 {
		t.Errorf("LastCoverage[main.go]: got %d, want 50", result.State.LastCoverage["main.go"])
	}
}

func TestVerifierOnReport_GreenEnforced_EmptyFileCoverage_StaysGreenUnreported_EC1(t *testing.T) {
	ctx := makeGreenTDDCtx(80)

	report := passedReport([]FileCoverageEntry{})

	result, err := verifierStageImpl{}.OnReport(ctx, report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.State.TDDCycle != cycleGreen {
		t.Errorf("TDDCycle: got %q, want %q — empty FileCoverage is a verifier-fail; cycle must stay green so the verifier re-measures, not red (EC-1)", result.State.TDDCycle, cycleGreen)
	}
	if result.State.Implemented {
		t.Error("Implemented: got true, want false so the verifier re-runs (EC-1)")
	}
	if !result.State.CoverageUnreported {
		t.Error("CoverageUnreported: got false, want true to flag the missing measurement (EC-1)")
	}
}

func TestVerifierOnReport_GreenEnforced_ReportedCoverage_ClearsUnreportedFlag(t *testing.T) {
	ctx := makeGreenTDDCtx(80)
	st := ctx.State
	st.CoverageUnreported = true
	ctx.State = st

	report := passedReport([]FileCoverageEntry{
		{File: "a.go", Coverage: 90},
	})

	result, err := verifierStageImpl{}.OnReport(ctx, report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.State.CoverageUnreported {
		t.Error("CoverageUnreported: got true, want false once a non-empty coverage report arrives")
	}
}

func TestVerifierOnReport_GreenEnforced_BoundaryExactThreshold_Advances_EC2(t *testing.T) {
	ctx := makeGreenTDDCtx(80)

	report := passedReport([]FileCoverageEntry{
		{File: "pkg.go", Coverage: 80},
	})

	result, err := verifierStageImpl{}.OnReport(ctx, report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.State.TDDCycle == cycleRed {
		t.Error("TDDCycle: got red, want advanced — coverage exactly at threshold must pass gate (EC-2)")
	}
	if result.State.LastCoverage == nil {
		t.Fatal("LastCoverage: must be set when coverage met")
	}
	if result.State.LastCoverage["pkg.go"] != 80 {
		t.Errorf("LastCoverage[pkg.go]: got %d, want 80", result.State.LastCoverage["pkg.go"])
	}
}

func TestVerifierOnReport_GreenEnforced_CoverageMet_AdvancesNormally(t *testing.T) {
	ctx := makeGreenTDDCtx(80)

	report := passedReport([]FileCoverageEntry{
		{File: "a.go", Coverage: 90},
		{File: "b.go", Coverage: 85},
	})

	result, err := verifierStageImpl{}.OnReport(ctx, report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.State.TDDCycle == cycleRed {
		t.Error("TDDCycle: got red, want advanced cycle — all files above threshold must not trigger gate")
	}
	if result.State.TDDCycle == cycleGreen {
		t.Error("TDDCycle: still green, want advanced to refactor after verifier pass")
	}
	if result.State.LastCoverage == nil {
		t.Fatal("LastCoverage: must be populated when coverage met and cycle advances")
	}
	if result.State.LastCoverage["a.go"] != 90 {
		t.Errorf("LastCoverage[a.go]: got %d, want 90", result.State.LastCoverage["a.go"])
	}
	if result.State.LastCoverage["b.go"] != 85 {
		t.Errorf("LastCoverage[b.go]: got %d, want 85", result.State.LastCoverage["b.go"])
	}
}

func TestVerifierOnReport_GateDisabled_LowCoverage_AdvancesNormally(t *testing.T) {
	ctx := makeGreenTDDCtx(0)
	st := ctx.State
	st.LastCoverage = nil
	ctx.State = st

	report := passedReport([]FileCoverageEntry{
		{File: "x.go", Coverage: 10},
	})

	result, err := verifierStageImpl{}.OnReport(ctx, report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.State.TDDCycle == cycleRed {
		t.Error("TDDCycle: got red, want advanced — gate disabled (MinTestCoverage 0) must not block progression")
	}
	if result.State.TDDCycle == cycleGreen {
		t.Error("TDDCycle: still green, want advanced when gate disabled and report passed")
	}
}

func TestVerifierOnReport_GateDisabled_EmptyFileCoverage_AdvancesNormally(t *testing.T) {
	ctx := makeGreenTDDCtx(0)

	report := passedReport([]FileCoverageEntry{})

	result, err := verifierStageImpl{}.OnReport(ctx, report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.State.TDDCycle == cycleRed {
		t.Error("TDDCycle: got red, want advanced — gate disabled with empty coverage must not trigger red")
	}
}

func TestVerifierOnReport_GreenEnforced_CoverageGate_DoesNotAdvanceCycle(t *testing.T) {
	ctx := makeGreenTDDCtx(80)

	report := passedReport([]FileCoverageEntry{
		{File: "impl.go", Coverage: 79},
	})

	result, err := verifierStageImpl{}.OnReport(ctx, report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.State.TDDCycle != cycleRed {
		t.Errorf("TDDCycle: got %q, want %q — gate must reset to red without advancing through refactor", result.State.TDDCycle, cycleRed)
	}
	if result.State.TDDCycle == cycleRefactor {
		t.Error("TDDCycle: must not reach refactor when gate blocks at green")
	}
}
