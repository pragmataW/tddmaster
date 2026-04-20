package tdd_test

import (
	"testing"

	"github.com/pragmataW/tddmaster/internal/state/model"
	"github.com/pragmataW/tddmaster/internal/state/service/tdd"
)

// helper: build a minimal executing StateFile.
func execState(phase string) model.StateFile {
	return model.StateFile{
		Phase: model.PhaseExecuting,
		Execution: model.ExecutionState{
			TDDCycle: phase,
		},
	}
}

// helper: pointer-to-bool convenience.
func boolPtr(b bool) *bool { return &b }

// helper: build a NosManifest with TddMode and SkipVerify set.
func manifest(tddMode, skipVerify bool) *model.NosManifest {
	return &model.NosManifest{
		Tdd: &model.Manifest{
			TddMode:    tddMode,
			SkipVerify: skipVerify,
		},
	}
}

// helper: build a slice of RefactorNote values.
func notes(texts ...string) []model.RefactorNote {
	out := make([]model.RefactorNote, len(texts))
	for i, t := range texts {
		out[i] = model.RefactorNote{Suggestion: t}
	}
	return out
}

// ---------------------------------------------------------------------------
// AC-1: skip=true + TDD=on + GREEN verifier passed + refactorNotes present
//   → PendingRefactorNotes written, TDDCycle becomes "refactor"
// ---------------------------------------------------------------------------

func TestRecordTDDVerificationFull_SkipVerify_Green_WithRefactorNotes_TransitionsToRefactor(t *testing.T) {
	st := execState(model.TDDCycleGreen)
	cfg := manifest(true, true) // TDD on, skipVerify on
	rNotes := notes("note1", "note2")

	got, err := tdd.RecordTDDVerificationFull(st, 0, 0, true, "ok", nil, nil, rNotes, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Execution.TDDCycle != model.TDDCycleRefactor {
		t.Errorf("TDDCycle: got %q, want %q", got.Execution.TDDCycle, model.TDDCycleRefactor)
	}
	if len(got.Execution.PendingRefactorNotes) != 2 {
		t.Fatalf("PendingRefactorNotes length: got %d, want 2", len(got.Execution.PendingRefactorNotes))
	}
	if got.Execution.PendingRefactorNotes[0].Suggestion != "note1" {
		t.Errorf("PendingRefactorNotes[0]: got %q, want %q", got.Execution.PendingRefactorNotes[0].Suggestion, "note1")
	}
	if got.Execution.PendingRefactorNotes[1].Suggestion != "note2" {
		t.Errorf("PendingRefactorNotes[1]: got %q, want %q", got.Execution.PendingRefactorNotes[1].Suggestion, "note2")
	}
}

// ---------------------------------------------------------------------------
// AC-2: skip=true + TDD=on + GREEN verifier passed + refactorNotes=[]
//   → task completes (TDDCycle clears to ""), no REFACTOR entered
// ---------------------------------------------------------------------------

func TestRecordTDDVerificationFull_SkipVerify_Green_EmptyRefactorNotes_ClearsToNextTask(t *testing.T) {
	st := execState(model.TDDCycleGreen)
	cfg := manifest(true, true)

	got, err := tdd.RecordTDDVerificationFull(st, 0, 0, true, "ok", nil, nil, nil, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Execution.TDDCycle != "" {
		t.Errorf("TDDCycle: got %q, want empty (next-task reset)", got.Execution.TDDCycle)
	}
	if len(got.Execution.PendingRefactorNotes) != 0 {
		t.Errorf("PendingRefactorNotes should be empty, got %v", got.Execution.PendingRefactorNotes)
	}
}

// ---------------------------------------------------------------------------
// AC-3: REFACTOR bypass guard — when PendingRefactorNotes present, REFACTOR
//   phase must be preserved even under skip mode (refactorNotes non-empty).
//   This mirrors cmd/next.go:1395-1418 guard: if refactorNotes exist,
//   REFACTOR is mandatory regardless of skipVerify.
// ---------------------------------------------------------------------------

func TestRecordTDDVerificationFull_SkipVerify_RefactorPhase_NotesPresent_StaysRefactor(t *testing.T) {
	st := execState(model.TDDCycleRefactor)
	// Simulate executor having applied the notes (RefactorApplied=true)
	st.Execution.RefactorApplied = true
	st.Execution.RefactorRounds = 0
	cfg := manifest(true, true)
	rNotes := notes("still-more-work")

	got, err := tdd.RecordTDDVerificationFull(st, 0, 3, true, "ok", nil, nil, rNotes, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Notes present → should stay in REFACTOR, not advance to next task.
	if got.Execution.TDDCycle != model.TDDCycleRefactor {
		t.Errorf("TDDCycle: got %q, want %q (refactor guard)", got.Execution.TDDCycle, model.TDDCycleRefactor)
	}
}

func TestRecordTDDVerificationFull_SkipVerify_RefactorPhase_NotesPresent_PendingRefactorNotesUpdated(t *testing.T) {
	st := execState(model.TDDCycleRefactor)
	st.Execution.RefactorApplied = true
	st.Execution.RefactorRounds = 0
	cfg := manifest(true, true)
	rNotes := notes("refactor-suggestion")

	got, err := tdd.RecordTDDVerificationFull(st, 0, 3, true, "ok", nil, nil, rNotes, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Execution.PendingRefactorNotes) == 0 {
		t.Error("expected PendingRefactorNotes to be populated when refactorNotes provided in REFACTOR phase")
	}
}

// ---------------------------------------------------------------------------
// AC-4: skip=false regression — existing RecordTDDVerificationFull behavior
//   preserved when skipVerify is false.
// ---------------------------------------------------------------------------

// GREEN with notes, skipVerify=false → REFACTOR (same as before).
func TestRecordTDDVerificationFull_NoSkip_Green_WithNotes_TransitionsToRefactor(t *testing.T) {
	st := execState(model.TDDCycleGreen)
	cfg := manifest(true, false) // TDD on, skipVerify OFF

	got, err := tdd.RecordTDDVerificationFull(st, 0, 0, true, "ok", nil, nil, notes("refactor-me"), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Execution.TDDCycle != model.TDDCycleRefactor {
		t.Errorf("TDDCycle: got %q, want %q", got.Execution.TDDCycle, model.TDDCycleRefactor)
	}
}

// GREEN with no notes, skipVerify=false → task completes.
func TestRecordTDDVerificationFull_NoSkip_Green_EmptyNotes_ClearsToNextTask(t *testing.T) {
	st := execState(model.TDDCycleGreen)
	cfg := manifest(true, false)

	got, err := tdd.RecordTDDVerificationFull(st, 0, 0, true, "ok", nil, nil, nil, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Execution.TDDCycle != "" {
		t.Errorf("TDDCycle: got %q, want empty", got.Execution.TDDCycle)
	}
}

// RED → GREEN transition still works with skipVerify=false.
func TestRecordTDDVerificationFull_NoSkip_Red_Passed_TransitionsToGreen(t *testing.T) {
	st := execState(model.TDDCycleRed)
	cfg := manifest(true, false)

	got, err := tdd.RecordTDDVerificationFull(st, 0, 0, true, "ok", nil, nil, nil, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Execution.TDDCycle != model.TDDCycleGreen {
		t.Errorf("TDDCycle: got %q, want %q", got.Execution.TDDCycle, model.TDDCycleGreen)
	}
}

// Non-passing verification still re-queues failed ACs (regression, skipVerify=false).
func TestRecordTDDVerificationFull_NoSkip_Failed_IncrementsFailCount(t *testing.T) {
	st := execState(model.TDDCycleGreen)
	cfg := manifest(true, false)

	got, err := tdd.RecordTDDVerificationFull(st, 0, 0, false, "fail output", []string{"AC-1"}, nil, nil, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Execution.LastVerification == nil {
		t.Fatal("LastVerification should be set after failed verification")
	}
	if got.Execution.LastVerification.VerificationFailCount != 1 {
		t.Errorf("VerificationFailCount: got %d, want 1", got.Execution.LastVerification.VerificationFailCount)
	}
}

// nil cfg behaves the same as skipVerify=false (no regression for callers
// that pass nil).
func TestRecordTDDVerificationFull_NilCfg_Green_WithNotes_TransitionsToRefactor(t *testing.T) {
	st := execState(model.TDDCycleGreen)

	got, err := tdd.RecordTDDVerificationFull(st, 0, 0, true, "ok", nil, nil, notes("x"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Execution.TDDCycle != model.TDDCycleRefactor {
		t.Errorf("TDDCycle: got %q, want %q (nil cfg should default to no-skip)", got.Execution.TDDCycle, model.TDDCycleRefactor)
	}
}

// ---------------------------------------------------------------------------
// AC-5: MaxRefactorRounds limit still enforced in skip mode.
// ---------------------------------------------------------------------------

func TestRecordTDDVerificationFull_SkipVerify_MaxRefactorRoundsReached_CompletesTask(t *testing.T) {
	st := execState(model.TDDCycleRefactor)
	st.Execution.RefactorApplied = true
	st.Execution.RefactorRounds = 2 // already at 2; max is 3, so rounds++ → 3 ≥ 3 → complete
	cfg := manifest(true, true)
	rNotes := notes("one-more-note") // notes present but cap hit

	got, err := tdd.RecordTDDVerificationFull(st, 0, 3, true, "ok", nil, nil, rNotes, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Execution.TDDCycle != "" {
		t.Errorf("TDDCycle: got %q, want empty (max rounds cap)", got.Execution.TDDCycle)
	}
}

func TestRecordTDDVerificationFull_NoSkip_MaxRefactorRoundsReached_CompletesTask(t *testing.T) {
	st := execState(model.TDDCycleRefactor)
	st.Execution.RefactorApplied = true
	st.Execution.RefactorRounds = 2
	cfg := manifest(true, false)
	rNotes := notes("one-more-note")

	got, err := tdd.RecordTDDVerificationFull(st, 0, 3, true, "ok", nil, nil, rNotes, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Execution.TDDCycle != "" {
		t.Errorf("TDDCycle: got %q, want empty (max rounds cap, skipVerify=false)", got.Execution.TDDCycle)
	}
}

// MaxRefactorRounds=0 means unlimited; notes present → stays in REFACTOR.
func TestRecordTDDVerificationFull_SkipVerify_UnlimitedRefactorRounds_StaysRefactorWhenNotesPresent(t *testing.T) {
	st := execState(model.TDDCycleRefactor)
	st.Execution.RefactorApplied = true
	st.Execution.RefactorRounds = 99
	cfg := manifest(true, true)
	rNotes := notes("keep-going")

	got, err := tdd.RecordTDDVerificationFull(st, 0, 0, true, "ok", nil, nil, rNotes, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Execution.TDDCycle != model.TDDCycleRefactor {
		t.Errorf("TDDCycle: got %q, want %q (unlimited rounds)", got.Execution.TDDCycle, model.TDDCycleRefactor)
	}
}

// ---------------------------------------------------------------------------
// Additional: wrong phase guard preserved regardless of skipVerify.
// ---------------------------------------------------------------------------

func TestRecordTDDVerificationFull_NonExecutingPhase_ReturnsError(t *testing.T) {
	st := model.StateFile{
		Phase: model.PhaseDiscovery,
		Execution: model.ExecutionState{
			TDDCycle: model.TDDCycleGreen,
		},
	}
	cfg := manifest(true, true)

	_, err := tdd.RecordTDDVerificationFull(st, 0, 0, true, "ok", nil, nil, nil, cfg)
	if err == nil {
		t.Error("expected error for non-executing phase, got nil")
	}
}
