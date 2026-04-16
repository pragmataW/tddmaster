package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeExecutingCycleState(cycle string) StateFile {
	s := CreateInitialState()
	specName := "spec-x"
	s.Phase = PhaseExecuting
	s.Spec = &specName
	s.Execution.CompletedTasks = []string{"ac-1"}
	s.Execution.TDDCycle = cycle
	return s
}

func TestRecordTDDVerificationFull_RedPassed_AdvancesToGreen(t *testing.T) {
	st := makeExecutingCycleState(TDDCycleRed)
	result, err := RecordTDDVerificationFull(st, 3, 3, true, "tests fail as expected", nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, TDDCycleGreen, result.Execution.TDDCycle)
	require.NotNil(t, result.Execution.LastVerification)
	assert.Equal(t, TDDCycleRed, result.Execution.LastVerification.Phase, "LastVerification.Phase must snapshot the phase at verify time")
	assert.Equal(t, 0, result.Execution.LastVerification.VerificationFailCount)
}

func TestRecordTDDVerificationFull_GreenPassed_WithNotes_AdvancesToRefactor(t *testing.T) {
	st := makeExecutingCycleState(TDDCycleGreen)
	notes := []RefactorNote{{File: "a.go", Suggestion: "extract fn", Rationale: "dedupe"}}
	result, err := RecordTDDVerificationFull(st, 3, 3, true, "all pass", nil, nil, notes)
	require.NoError(t, err)
	assert.Equal(t, TDDCycleRefactor, result.Execution.TDDCycle)
	assert.Equal(t, 0, result.Execution.RefactorRounds)
	assert.False(t, result.Execution.RefactorApplied)
	assert.Equal(t, TDDCycleGreen, result.Execution.LastVerification.Phase)
	assert.Equal(t, notes, result.Execution.LastVerification.RefactorNotes)
}

func TestRecordTDDVerificationFull_GreenPassed_NoNotes_SkipsRefactor(t *testing.T) {
	st := makeExecutingCycleState(TDDCycleGreen)
	result, err := RecordTDDVerificationFull(st, 3, 3, true, "all pass, code is clean", nil, nil, nil)
	require.NoError(t, err)
	// No refactor notes → green verifier found nothing to improve → clear the
	// cycle; cmd/next.go re-seeds RED for the next TDD task (or leaves empty
	// for non-TDD tasks).
	assert.Equal(t, "", result.Execution.TDDCycle)
	assert.Equal(t, 0, result.Execution.RefactorRounds)
	assert.False(t, result.Execution.RefactorApplied)
}

func TestRecordTDDVerificationFull_RefactorPassed_NoNotes_ResetsToRed(t *testing.T) {
	st := makeExecutingCycleState(TDDCycleRefactor)
	result, err := RecordTDDVerificationFull(st, 3, 3, true, "clean", nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "", result.Execution.TDDCycle, "empty refactor notes on first pass should clear the cycle so cmd/next.go can re-seed for the next task")
	assert.Equal(t, 0, result.Execution.RefactorRounds)
	assert.False(t, result.Execution.RefactorApplied)
}

func TestRecordTDDVerificationFull_RefactorPassed_WithNotes_StaysInRefactor(t *testing.T) {
	st := makeExecutingCycleState(TDDCycleRefactor)
	notes := []RefactorNote{
		{File: "a.go", Suggestion: "rename", Rationale: "clarity"},
	}
	result, err := RecordTDDVerificationFull(st, 3, 3, true, "clean but has notes", nil, nil, notes)
	require.NoError(t, err)
	assert.Equal(t, TDDCycleRefactor, result.Execution.TDDCycle)
	assert.False(t, result.Execution.RefactorApplied, "executor hasn't applied yet")
	require.NotNil(t, result.Execution.LastVerification)
	assert.Equal(t, notes, result.Execution.LastVerification.RefactorNotes)
}

func TestRecordTDDVerificationFull_RefactorAppliedThenVerify_EmptyNotes_ResetsToRed(t *testing.T) {
	st := makeExecutingCycleState(TDDCycleRefactor)
	st.Execution.RefactorApplied = true
	st.Execution.RefactorRounds = 0
	result, err := RecordTDDVerificationFull(st, 3, 3, true, "clean after apply", nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "", result.Execution.TDDCycle)
	assert.Equal(t, 0, result.Execution.RefactorRounds)
	assert.False(t, result.Execution.RefactorApplied)
}

func TestRecordTDDVerificationFull_RefactorAppliedThenVerify_WithNotes_IteratesRound(t *testing.T) {
	st := makeExecutingCycleState(TDDCycleRefactor)
	st.Execution.RefactorApplied = true
	st.Execution.RefactorRounds = 0
	notes := []RefactorNote{{File: "b.go", Suggestion: "extract fn", Rationale: "dedupe"}}
	result, err := RecordTDDVerificationFull(st, 3, 3, true, "more to improve", nil, nil, notes)
	require.NoError(t, err)
	assert.Equal(t, TDDCycleRefactor, result.Execution.TDDCycle)
	assert.Equal(t, 1, result.Execution.RefactorRounds)
	assert.False(t, result.Execution.RefactorApplied, "cleared so executor applies again")
}

func TestRecordTDDVerificationFull_RefactorRoundCap_AdvancesDespiteNotes(t *testing.T) {
	st := makeExecutingCycleState(TDDCycleRefactor)
	st.Execution.RefactorApplied = true
	st.Execution.RefactorRounds = 2 // next increment hits cap=3
	notes := []RefactorNote{{File: "c.go", Suggestion: "more", Rationale: "x"}}
	result, err := RecordTDDVerificationFull(st, 3, 3, true, "still notes", nil, nil, notes)
	require.NoError(t, err)
	assert.Equal(t, "", result.Execution.TDDCycle, "cap reached → clear cycle so cmd/next.go can re-seed for the next task")
	assert.Equal(t, 0, result.Execution.RefactorRounds)
}

func TestRecordTDDVerificationFull_RefactorFail_TreatedLikeGreenRetry(t *testing.T) {
	st := makeExecutingCycleState(TDDCycleRefactor)
	st.Execution.RefactorApplied = true
	result, err := RecordTDDVerificationFull(st, 5, 3, false, "behavior broke", []string{"ac-1"}, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, TDDCycleRefactor, result.Execution.TDDCycle, "cycle preserved on fail")
	assert.Equal(t, 1, result.Execution.LastVerification.VerificationFailCount)
	assert.NotContains(t, result.Execution.CompletedTasks, "ac-1", "failed AC requeued")
}

func TestRecordTDDVerificationFull_RefactorFail_MaxRetriesBlocks(t *testing.T) {
	st := makeExecutingCycleState(TDDCycleRefactor)
	st.Execution.RefactorApplied = true
	st.Execution.LastVerification = &VerificationResult{VerificationFailCount: 2}
	result, err := RecordTDDVerificationFull(st, 3, 3, false, "still broken", nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, PhaseBlocked, result.Phase)
}

func TestRecordTDDVerificationFull_EmptyCycle_NoTransition(t *testing.T) {
	// Non-TDD mode: TDDCycle empty → pass-through behavior preserved.
	st := CreateInitialState()
	specName := "spec"
	st.Phase = PhaseExecuting
	st.Spec = &specName
	result, err := RecordTDDVerificationFull(st, 3, 3, true, "ok", nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "", result.Execution.TDDCycle, "no cycle set means no transition")
}

func TestStartTDDCycleForTask_InitializesRed(t *testing.T) {
	s := CreateInitialState()
	s.Execution.TDDCycle = TDDCycleRefactor
	s.Execution.RefactorRounds = 2
	s.Execution.RefactorApplied = true

	StartTDDCycleForTask(&s)

	assert.Equal(t, TDDCycleRed, s.Execution.TDDCycle)
	assert.Equal(t, 0, s.Execution.RefactorRounds)
	assert.False(t, s.Execution.RefactorApplied)
}

func TestMarkRefactorApplied_FlipsFlag(t *testing.T) {
	s := CreateInitialState()
	s.Execution.RefactorApplied = false

	MarkRefactorApplied(&s)

	assert.True(t, s.Execution.RefactorApplied)
}
