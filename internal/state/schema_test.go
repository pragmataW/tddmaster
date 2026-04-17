package state

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateInitialState(t *testing.T) {
	s := CreateInitialState()

	assert.Equal(t, "0.1.0", s.Version)
	assert.Equal(t, PhaseIdle, s.Phase)
	assert.Nil(t, s.Spec)
	assert.Nil(t, s.Branch)
	assert.Equal(t, 0, s.Discovery.CurrentQuestion)
	assert.False(t, s.Discovery.Completed)
	assert.False(t, s.Discovery.Approved)
	assert.Equal(t, "human", s.Discovery.Audience)
	assert.Nil(t, s.Discovery.PlanPath)
	assert.Equal(t, []DiscoveryAnswer{}, s.Discovery.Answers)
	assert.Equal(t, "none", s.SpecState.Status)
	assert.Nil(t, s.SpecState.Path)
	assert.Equal(t, 0, s.Execution.Iteration)
	assert.Equal(t, []string{}, s.Execution.ModifiedFiles)
	assert.Equal(t, []string{}, s.Execution.CompletedTasks)
	assert.Equal(t, []Decision{}, s.Decisions)
	assert.Equal(t, []RevisitEntry{}, s.RevisitHistory)
	assert.Nil(t, s.Classification)
	assert.Nil(t, s.CompletionReason)
}

func TestNormalizeAnswer(t *testing.T) {
	t.Run("normalizes basic answer", func(t *testing.T) {
		a := DiscoveryAnswer{QuestionID: "Q1", Answer: "some answer"}
		result := NormalizeAnswer(a)

		assert.Equal(t, "Q1", result.QuestionID)
		assert.Equal(t, "some answer", result.Answer)
		assert.Equal(t, "Unknown User", result.User)
		assert.Equal(t, "", result.Email)
		assert.Equal(t, "", result.Timestamp)
		assert.Equal(t, "original", result.Type)
	})
}

func TestGetAnswersForQuestion(t *testing.T) {
	answers := []DiscoveryAnswer{
		{QuestionID: "Q1", Answer: "answer for Q1"},
		{QuestionID: "Q2", Answer: "answer for Q2"},
		{QuestionID: "Q1", Answer: "another answer for Q1"},
	}

	t.Run("returns answers for a given question", func(t *testing.T) {
		result := GetAnswersForQuestion(answers, "Q1")
		assert.Len(t, result, 2)
		assert.Equal(t, "Q1", result[0].QuestionID)
		assert.Equal(t, "answer for Q1", result[0].Answer)
		assert.Equal(t, "Q1", result[1].QuestionID)
		assert.Equal(t, "another answer for Q1", result[1].Answer)
	})

	t.Run("returns empty slice for unknown question", func(t *testing.T) {
		result := GetAnswersForQuestion(answers, "Q99")
		assert.Equal(t, []AttributedDiscoveryAnswer{}, result)
	})
}

func TestGetCombinedAnswer(t *testing.T) {
	t.Run("returns empty string when no answers", func(t *testing.T) {
		result := GetCombinedAnswer([]DiscoveryAnswer{}, "Q1")
		assert.Equal(t, "", result)
	})

	t.Run("returns single answer text directly", func(t *testing.T) {
		answers := []DiscoveryAnswer{
			{QuestionID: "Q1", Answer: "the answer"},
		}
		result := GetCombinedAnswer(answers, "Q1")
		assert.Equal(t, "the answer", result)
	})

	t.Run("combines multiple answers with attribution", func(t *testing.T) {
		answers := []DiscoveryAnswer{
			{QuestionID: "Q1", Answer: "first answer"},
			{QuestionID: "Q1", Answer: "second answer"},
		}
		result := GetCombinedAnswer(answers, "Q1")
		assert.Contains(t, result, "first answer")
		assert.Contains(t, result, "second answer")
		assert.Contains(t, result, "Unknown User")
	})
}

// =============================================================================
// TDDCycle field
// =============================================================================

func TestExecutionState_TDDCycle_ZeroValue(t *testing.T) {
	s := CreateInitialState()
	assert.Equal(t, "", s.Execution.TDDCycle, "TDDCycle must be empty string by default")
}

func TestExecutionState_TDDCycle_JSONRoundTrip(t *testing.T) {
	s := CreateInitialState()
	s.Execution.TDDCycle = "red"

	data, err := json.Marshal(s.Execution)
	require.NoError(t, err)

	var restored ExecutionState
	require.NoError(t, json.Unmarshal(data, &restored))
	assert.Equal(t, "red", restored.TDDCycle)
}

func TestExecutionState_TDDCycle_OmittedWhenEmpty(t *testing.T) {
	s := CreateInitialState()
	// empty string should be omitted from JSON (omitempty)
	data, err := json.Marshal(s.Execution)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "tddCycle")
}

// =============================================================================
// VerificationResult new fields
// =============================================================================

func TestVerificationResult_ZeroValues(t *testing.T) {
	var v VerificationResult
	assert.Nil(t, v.UncoveredEdgeCases, "UncoveredEdgeCases must be nil by default")
	assert.Equal(t, 0, v.VerificationFailCount, "VerificationFailCount must be 0 by default")
}

func TestVerificationResult_JSONRoundTrip(t *testing.T) {
	v := VerificationResult{
		Passed:                false,
		Output:                "test output",
		Timestamp:             "2026-01-01T00:00:00Z",
		UncoveredEdgeCases:    []string{"EC-1", "EC-2"},
		VerificationFailCount: 3,
	}

	data, err := json.Marshal(v)
	require.NoError(t, err)

	var restored VerificationResult
	require.NoError(t, json.Unmarshal(data, &restored))

	assert.Equal(t, v.Passed, restored.Passed)
	assert.Equal(t, v.Output, restored.Output)
	assert.Equal(t, v.UncoveredEdgeCases, restored.UncoveredEdgeCases)
	assert.Equal(t, v.VerificationFailCount, restored.VerificationFailCount)
}

func TestVerificationResult_UncoveredEdgeCases_OmittedWhenNil(t *testing.T) {
	v := VerificationResult{Passed: true, Output: "ok", Timestamp: "ts"}
	data, err := json.Marshal(v)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "uncoveredEdgeCases")
}

func TestVerificationResult_VerificationFailCount_OmittedWhenZero(t *testing.T) {
	v := VerificationResult{Passed: true, Output: "ok", Timestamp: "ts"}
	data, err := json.Marshal(v)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "verificationFailCount")
}

// =============================================================================
// IsTDDEnabled
// =============================================================================

func TestIsTDDEnabled_NilTdd_ReturnsFalse(t *testing.T) {
	m := NosManifest{Tdd: nil}
	assert.False(t, m.IsTDDEnabled())
}

func TestIsTDDEnabled_TddModeFalse_ReturnsFalse(t *testing.T) {
	m := NosManifest{Tdd: &Manifest{TddMode: false}}
	assert.False(t, m.IsTDDEnabled())
}

func TestIsTDDEnabled_TddModeTrue_ReturnsTrue(t *testing.T) {
	m := NosManifest{Tdd: &Manifest{TddMode: true}}
	assert.True(t, m.IsTDDEnabled())
}

func TestCreateInitialManifest(t *testing.T) {
	concerns := []string{"security"}
	tools := []CodingToolId{CodingToolClaudeCode}
	project := ProjectTraits{
		Languages:  []string{"go"},
		Frameworks: []string{},
		CI:         []string{},
		TestRunner: nil,
	}

	m := CreateInitialManifest(concerns, tools, project)

	assert.Equal(t, concerns, m.Concerns)
	assert.Equal(t, tools, m.Tools)
	assert.Equal(t, project, m.Project)
	assert.Equal(t, 15, m.MaxIterationsBeforeRestart)
	assert.Nil(t, m.VerifyCommand)
	assert.False(t, m.AllowGit)
	assert.Equal(t, "tddmaster", m.Command)
}

func TestCreateInitialManifest_TDDEnabled(t *testing.T) {
	m := CreateInitialManifest(nil, nil, ProjectTraits{})
	require.NotNil(t, m.Tdd, "CreateInitialManifest must set Tdd field")
	assert.True(t, m.Tdd.TddMode, "TddMode must be true by default")
	assert.Equal(t, 3, m.Tdd.MaxVerificationRetries, "MaxVerificationRetries must be 3 by default")
}

// =============================================================================
// RecordTDDVerification
// =============================================================================

func makeExecutingState() StateFile {
	s := CreateInitialState()
	specName := "test-spec"
	s.Phase = PhaseExecuting
	s.Spec = &specName
	s.Execution.CompletedTasks = []string{"ac-1", "ac-2", "ac-3"}
	return s
}

func TestRecordTDDVerification_PassedSetsVerificationResult(t *testing.T) {
	st := makeExecutingState()
	result, err := RecordTDDVerification(st, 3, true, "all pass", nil, nil)
	require.NoError(t, err)
	require.NotNil(t, result.Execution.LastVerification)
	assert.True(t, result.Execution.LastVerification.Passed)
	assert.Equal(t, "all pass", result.Execution.LastVerification.Output)
	assert.Equal(t, 0, result.Execution.LastVerification.VerificationFailCount)
}

func TestRecordTDDVerification_FailIncreasesFailCount(t *testing.T) {
	st := makeExecutingState()
	result, err := RecordTDDVerification(st, 5, false, "test failed", nil, nil)
	require.NoError(t, err)
	require.NotNil(t, result.Execution.LastVerification)
	assert.False(t, result.Execution.LastVerification.Passed)
	assert.Equal(t, 1, result.Execution.LastVerification.VerificationFailCount)
}

func TestRecordTDDVerification_FailRequeuesFailedACs(t *testing.T) {
	st := makeExecutingState()
	result, err := RecordTDDVerification(st, 5, false, "failed", []string{"ac-1", "ac-3"}, nil)
	require.NoError(t, err)
	// ac-1 and ac-3 removed from CompletedTasks, ac-2 remains
	assert.Equal(t, []string{"ac-2"}, result.Execution.CompletedTasks)
}

func TestRecordTDDVerification_AutoBlocksWhenMaxRetriesReached(t *testing.T) {
	st := makeExecutingState()
	// Seed a previous fail
	st.Execution.LastVerification = &VerificationResult{VerificationFailCount: 2}

	result, err := RecordTDDVerification(st, 3, false, "still failing", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, PhaseBlocked, result.Phase)
}

func TestRecordTDDVerification_ErrorWhenNotExecuting(t *testing.T) {
	st := CreateInitialState()
	// Phase is IDLE
	_, err := RecordTDDVerification(st, 3, true, "pass", nil, nil)
	require.Error(t, err)
}

func TestRecordTDDVerification_SetsUncoveredEdgeCases(t *testing.T) {
	st := makeExecutingState()
	result, err := RecordTDDVerification(st, 5, false, "failed", nil, []string{"EC-1", "EC-2"})
	require.NoError(t, err)
	require.NotNil(t, result.Execution.LastVerification)
	assert.Equal(t, []string{"EC-1", "EC-2"}, result.Execution.LastVerification.UncoveredEdgeCases)
}

// =============================================================================
// VerificationResult new TDD fields — RefactorNotes + Phase
// =============================================================================

func TestVerificationResult_RefactorNotes_JSONRoundTrip(t *testing.T) {
	v := VerificationResult{
		Passed:    true,
		Output:    "clean",
		Timestamp: "ts",
		Phase:     TDDCycleRefactor,
		RefactorNotes: []RefactorNote{
			{File: "a.go", Suggestion: "rename X to Y", Rationale: "clarity"},
		},
	}
	data, err := json.Marshal(v)
	require.NoError(t, err)

	var restored VerificationResult
	require.NoError(t, json.Unmarshal(data, &restored))
	assert.Equal(t, v.RefactorNotes, restored.RefactorNotes)
	assert.Equal(t, TDDCycleRefactor, restored.Phase)
}

func TestVerificationResult_RefactorNotes_OmittedWhenNil(t *testing.T) {
	v := VerificationResult{Passed: true, Output: "ok", Timestamp: "ts"}
	data, err := json.Marshal(v)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "refactorNotes")
	assert.NotContains(t, string(data), `"phase"`)
}

// =============================================================================
// ExecutionState new TDD round-tracking fields
// =============================================================================

func TestExecutionState_RefactorRoundFields_JSONRoundTrip(t *testing.T) {
	s := CreateInitialState()
	s.Execution.RefactorRounds = 2
	s.Execution.RefactorApplied = true

	data, err := json.Marshal(s.Execution)
	require.NoError(t, err)

	var restored ExecutionState
	require.NoError(t, json.Unmarshal(data, &restored))
	assert.Equal(t, 2, restored.RefactorRounds)
	assert.True(t, restored.RefactorApplied)
}

func TestExecutionState_RefactorRoundFields_OmittedWhenZero(t *testing.T) {
	s := CreateInitialState()
	data, err := json.Marshal(s.Execution)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "refactorRounds")
	assert.NotContains(t, string(data), "refactorApplied")
}

// =============================================================================
// Manifest MaxRefactorRounds
// =============================================================================

func TestCreateInitialManifest_MaxRefactorRoundsDefault(t *testing.T) {
	m := CreateInitialManifest(nil, nil, ProjectTraits{})
	require.NotNil(t, m.Tdd)
	assert.Equal(t, 3, m.Tdd.MaxRefactorRounds, "MaxRefactorRounds must default to 3")
}
