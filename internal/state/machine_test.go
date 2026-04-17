package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanTransition(t *testing.T) {
	tests := []struct {
		from     Phase
		to       Phase
		expected bool
	}{
		{PhaseUninitialized, PhaseIdle, true},
		{PhaseIdle, PhaseDiscovery, true},
		{PhaseIdle, PhaseCompleted, true},
		{PhaseIdle, PhaseExecuting, false},
		{PhaseDiscovery, PhaseDiscoveryRefinement, true},
		{PhaseDiscovery, PhaseCompleted, true},
		{PhaseDiscovery, PhaseExecuting, false},
		{PhaseDiscoveryRefinement, PhaseSpecProposal, true},
		{PhaseDiscoveryRefinement, PhaseDiscoveryRefinement, true},
		{PhaseDiscoveryRefinement, PhaseCompleted, true},
		{PhaseSpecProposal, PhaseSpecApproved, true},
		{PhaseSpecProposal, PhaseCompleted, true},
		{PhaseSpecApproved, PhaseExecuting, true},
		{PhaseSpecApproved, PhaseCompleted, true},
		{PhaseExecuting, PhaseCompleted, true},
		{PhaseExecuting, PhaseBlocked, true},
		{PhaseBlocked, PhaseExecuting, true},
		{PhaseBlocked, PhaseCompleted, true},
		{PhaseCompleted, PhaseIdle, true},
		{PhaseCompleted, PhaseDiscovery, true},
		{PhaseCompleted, PhaseExecuting, true},
		{PhaseCompleted, PhaseBlocked, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			result := CanTransition(tt.from, tt.to)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAssertTransition(t *testing.T) {
	t.Run("valid transition returns nil", func(t *testing.T) {
		err := AssertTransition(PhaseIdle, PhaseDiscovery)
		assert.NoError(t, err)
	})

	t.Run("invalid transition returns error", func(t *testing.T) {
		err := AssertTransition(PhaseIdle, PhaseExecuting)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "IDLE")
		assert.Contains(t, err.Error(), "EXECUTING")
	})
}

func TestTransition(t *testing.T) {
	s := CreateInitialState()

	t.Run("valid transition updates phase", func(t *testing.T) {
		result, err := Transition(s, PhaseDiscovery)
		require.NoError(t, err)
		assert.Equal(t, PhaseDiscovery, result.Phase)
	})

	t.Run("invalid transition returns error", func(t *testing.T) {
		_, err := Transition(s, PhaseExecuting)
		assert.Error(t, err)
	})
}

func TestStartSpec(t *testing.T) {
	s := CreateInitialState()

	t.Run("starts spec and resets discovery state", func(t *testing.T) {
		desc := "my spec description"
		result, err := StartSpec(s, "my-spec", "main", &desc)
		require.NoError(t, err)

		assert.Equal(t, PhaseDiscovery, result.Phase)
		assert.Equal(t, "my-spec", *result.Spec)
		assert.Equal(t, "my spec description", *result.SpecDescription)
		assert.Equal(t, "main", *result.Branch)
		assert.False(t, result.Discovery.Completed)
		assert.Equal(t, 0, result.Discovery.CurrentQuestion)
		assert.Equal(t, []DiscoveryAnswer{}, result.Discovery.Answers)
		assert.Equal(t, "none", result.SpecState.Status)
		assert.Equal(t, 0, result.Execution.Iteration)
		assert.Equal(t, []Decision{}, result.Decisions)
	})

	t.Run("fails from wrong phase", func(t *testing.T) {
		executing := s
		executing.Phase = PhaseExecuting
		_, err := StartSpec(executing, "spec", "branch", nil)
		assert.Error(t, err)
	})
}

func TestSetDiscoveryMode(t *testing.T) {
	s := CreateInitialState()
	s.Phase = PhaseDiscovery

	t.Run("sets mode in DISCOVERY phase", func(t *testing.T) {
		result, err := SetDiscoveryMode(s, DiscoveryModeFull)
		require.NoError(t, err)
		assert.Equal(t, DiscoveryModeFull, *result.Discovery.Mode)
	})

	t.Run("fails in non-DISCOVERY phase", func(t *testing.T) {
		idle := CreateInitialState()
		_, err := SetDiscoveryMode(idle, DiscoveryModeFull)
		assert.Error(t, err)
	})
}

func TestAddDiscoveryAnswer(t *testing.T) {
	s := CreateInitialState()
	s.Phase = PhaseDiscovery

	t.Run("adds answer to discovery state", func(t *testing.T) {
		result, err := AddDiscoveryAnswer(s, "Q1", "this is a long enough answer for the validation", nil)
		require.NoError(t, err)
		assert.Len(t, result.Discovery.Answers, 1)
		assert.Equal(t, "Q1", result.Discovery.Answers[0].QuestionID)
	})

	t.Run("replaces existing answer for same question", func(t *testing.T) {
		s2, _ := AddDiscoveryAnswer(s, "Q1", "first answer that is long enough for validation", nil)
		s3, err := AddDiscoveryAnswer(s2, "Q1", "second answer that replaces the first one completely", nil)
		require.NoError(t, err)
		assert.Len(t, s3.Discovery.Answers, 1)
		assert.Equal(t, "second answer that replaces the first one completely", s3.Discovery.Answers[0].Answer)
	})

	t.Run("rejects answer shorter than 20 chars", func(t *testing.T) {
		_, err := AddDiscoveryAnswer(s, "Q1", "short", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "minimum 20 characters")
	})

	t.Run("rejects whitespace-only short answer", func(t *testing.T) {
		_, err := AddDiscoveryAnswer(s, "Q1", "   short   ", nil)
		assert.Error(t, err)
	})

	t.Run("fails in non-discovery phase", func(t *testing.T) {
		idle := CreateInitialState()
		_, err := AddDiscoveryAnswer(idle, "Q1", "this is a long enough answer for the validation", nil)
		assert.Error(t, err)
	})
}

func TestCompleteDiscovery(t *testing.T) {
	s := CreateInitialState()
	s.Phase = PhaseDiscovery
	specName := "my-spec"
	s.Spec = &specName

	t.Run("transitions to DISCOVERY_REFINEMENT", func(t *testing.T) {
		result, err := CompleteDiscovery(s)
		require.NoError(t, err)
		assert.Equal(t, PhaseDiscoveryRefinement, result.Phase)
		assert.True(t, result.Discovery.Completed)
		assert.Equal(t, "draft", result.SpecState.Status)
		assert.NotNil(t, result.SpecState.Path)
	})

	t.Run("blocks if pending follow-ups exist", func(t *testing.T) {
		s2 := AddFollowUp(s, "Q1", "follow up question here?", "agent")
		_, err := CompleteDiscovery(s2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "pending follow-up")
	})

	t.Run("fails in non-DISCOVERY phase", func(t *testing.T) {
		idle := CreateInitialState()
		_, err := CompleteDiscovery(idle)
		assert.Error(t, err)
	})
}

func TestApproveSpec(t *testing.T) {
	s := CreateInitialState()
	s.Phase = PhaseSpecProposal

	t.Run("transitions to SPEC_APPROVED", func(t *testing.T) {
		result, err := ApproveSpec(s)
		require.NoError(t, err)
		assert.Equal(t, PhaseSpecApproved, result.Phase)
		assert.Equal(t, "approved", result.SpecState.Status)
	})
}

func TestStartExecution(t *testing.T) {
	t.Run("transitions to EXECUTING and fully resets execution state", func(t *testing.T) {
		s := CreateInitialState()
		s.Phase = PhaseSpecApproved
		s.Discovery.Completed = true
		s.Discovery.Approved = true
		s.Discovery.Answers = []DiscoveryAnswer{
			{QuestionID: "verification", Answer: "- Cover failure path"},
		}
		progress := "working on task-3"
		s.Execution = ExecutionState{
			Iteration:            4,
			LastProgress:         &progress,
			ModifiedFiles:        []string{"internal/state/machine.go"},
			LastVerification:     &VerificationResult{Passed: false, Output: "failing tests", Timestamp: "2026-04-13T00:00:00Z"},
			AwaitingStatusReport: true,
			Debt: &DebtState{
				Items: []DebtItem{
					{ID: "debt-1", Text: "missing verifier evidence", Since: 2},
				},
				FromIteration:         1,
				UnaddressedIterations: 3,
			},
			CompletedTasks: []string{"task-1"},
			DebtCounter:    2,
			NaItems:        []string{"ac-2"},
			ConfidenceFindings: []ConfidenceFinding{
				{Finding: "verified by test", Confidence: 9, Basis: "read code"},
			},
		}

		result, err := StartExecution(s)
		require.NoError(t, err)
		assert.Equal(t, PhaseExecuting, result.Phase)
		assert.Equal(t, 0, result.Execution.Iteration)
		assert.True(t, result.Discovery.Completed)
		assert.False(t, result.Discovery.Approved)
		assert.Equal(t, s.Discovery.Answers, result.Discovery.Answers)
		assert.Nil(t, result.Execution.LastProgress)
		assert.Empty(t, result.Execution.ModifiedFiles)
		assert.Nil(t, result.Execution.LastVerification)
		assert.False(t, result.Execution.AwaitingStatusReport)
		assert.Nil(t, result.Execution.Debt)
		assert.Empty(t, result.Execution.CompletedTasks)
		assert.Zero(t, result.Execution.DebtCounter)
		assert.Empty(t, result.Execution.NaItems)
		assert.Empty(t, result.Execution.ConfidenceFindings)
	})

	t.Run("fails outside SPEC_APPROVED", func(t *testing.T) {
		s := CreateInitialState()
		_, err := StartExecution(s)
		assert.Error(t, err)
	})
}

func TestAdvanceExecution(t *testing.T) {
	s := CreateInitialState()
	s.Phase = PhaseExecuting

	t.Run("increments iteration and sets progress", func(t *testing.T) {
		result, err := AdvanceExecution(s, "progress update")
		require.NoError(t, err)
		assert.Equal(t, 1, result.Execution.Iteration)
		assert.Equal(t, "progress update", *result.Execution.LastProgress)
	})

	t.Run("fails in non-EXECUTING phase", func(t *testing.T) {
		idle := CreateInitialState()
		_, err := AdvanceExecution(idle, "progress")
		assert.Error(t, err)
	})
}

func TestBlockExecution(t *testing.T) {
	s := CreateInitialState()
	s.Phase = PhaseExecuting

	t.Run("transitions to BLOCKED", func(t *testing.T) {
		result, err := BlockExecution(s, "some blocker reason")
		require.NoError(t, err)
		assert.Equal(t, PhaseBlocked, result.Phase)
		assert.Contains(t, *result.Execution.LastProgress, "BLOCKED:")
		assert.Contains(t, *result.Execution.LastProgress, "some blocker reason")
	})
}

func TestCompleteSpec(t *testing.T) {
	s := CreateInitialState()
	s.Phase = PhaseExecuting

	t.Run("transitions to COMPLETED with reason", func(t *testing.T) {
		note := "all done"
		result, err := CompleteSpec(s, CompletionReasonDone, &note)
		require.NoError(t, err)
		assert.Equal(t, PhaseCompleted, result.Phase)
		assert.Equal(t, CompletionReasonDone, *result.CompletionReason)
		assert.NotNil(t, result.CompletedAt)
		assert.Equal(t, "all done", *result.CompletionNote)
	})
}

func TestReopenSpec(t *testing.T) {
	s := CreateInitialState()
	s.Phase = PhaseCompleted
	reason := CompletionReasonDone
	s.CompletionReason = &reason

	t.Run("transitions back to DISCOVERY", func(t *testing.T) {
		result, err := ReopenSpec(s)
		require.NoError(t, err)
		assert.Equal(t, PhaseDiscovery, result.Phase)
		assert.Nil(t, result.CompletionReason)
		assert.NotNil(t, result.ReopenedFrom)
		assert.Equal(t, "done", *result.ReopenedFrom)
	})

	t.Run("fails in non-COMPLETED phase", func(t *testing.T) {
		idle := CreateInitialState()
		_, err := ReopenSpec(idle)
		assert.Error(t, err)
	})
}

func TestResumeCompletedSpec(t *testing.T) {
	t.Run("restores executing phase and preserves execution state", func(t *testing.T) {
		s := CreateInitialState()
		s.Phase = PhaseCompleted
		reason := CompletionReasonDone
		s.CompletionReason = &reason
		s.Execution.Iteration = 4
		s.Execution.CompletedTasks = []string{"task-1"}
		s.TransitionHistory = []PhaseTransition{{From: PhaseExecuting, To: PhaseCompleted}}

		result, err := ResumeCompletedSpec(s)
		require.NoError(t, err)
		assert.Equal(t, PhaseExecuting, result.Phase)
		assert.Nil(t, result.CompletionReason)
		assert.Equal(t, 4, result.Execution.Iteration)
		assert.Equal(t, []string{"task-1"}, result.Execution.CompletedTasks)
	})

	t.Run("restores blocked phase when that was the terminal source", func(t *testing.T) {
		s := CreateInitialState()
		s.Phase = PhaseCompleted
		reason := CompletionReasonCancelled
		s.CompletionReason = &reason
		progress := "BLOCKED: waiting on DB"
		s.Execution.LastProgress = &progress
		s.TransitionHistory = []PhaseTransition{{From: PhaseBlocked, To: PhaseCompleted}}

		result, err := ResumeCompletedSpec(s)
		require.NoError(t, err)
		assert.Equal(t, PhaseBlocked, result.Phase)
		require.NotNil(t, result.Execution.LastProgress)
		assert.Equal(t, progress, *result.Execution.LastProgress)
	})

	t.Run("falls back to executing when no execution transition is available", func(t *testing.T) {
		s := CreateInitialState()
		s.Phase = PhaseCompleted
		reason := CompletionReasonDone
		s.CompletionReason = &reason
		s.TransitionHistory = []PhaseTransition{{From: PhaseSpecApproved, To: PhaseCompleted}}

		result, err := ResumeCompletedSpec(s)
		require.NoError(t, err)
		assert.Equal(t, PhaseExecuting, result.Phase)
	})
}

func TestRevisitSpec(t *testing.T) {
	s := CreateInitialState()
	s.Phase = PhaseExecuting

	t.Run("transitions to DISCOVERY and adds revisit entry", func(t *testing.T) {
		result, err := RevisitSpec(s, "need to clarify requirements")
		require.NoError(t, err)
		assert.Equal(t, PhaseDiscovery, result.Phase)
		assert.Len(t, result.RevisitHistory, 1)
		assert.Equal(t, PhaseExecuting, result.RevisitHistory[0].From)
		assert.Equal(t, "need to clarify requirements", result.RevisitHistory[0].Reason)
	})

	t.Run("fails from non-EXECUTING/BLOCKED phase", func(t *testing.T) {
		idle := CreateInitialState()
		_, err := RevisitSpec(idle, "reason")
		assert.Error(t, err)
	})
}

func TestAddDecision(t *testing.T) {
	s := CreateInitialState()
	d := Decision{ID: "d1", Question: "Use postgres?", Choice: "yes", Promoted: false, Timestamp: "2024-01-01T00:00:00Z"}

	result := AddDecision(s, d)
	assert.Len(t, result.Decisions, 1)
	assert.Equal(t, d, result.Decisions[0])
}

func TestRecordTransition(t *testing.T) {
	s := CreateInitialState()

	t.Run("records transition with user", func(t *testing.T) {
		user := &UserInfo{Name: "Alice", Email: "alice@example.com"}
		result := RecordTransition(s, PhaseIdle, PhaseDiscovery, user, nil)
		assert.Len(t, result.TransitionHistory, 1)
		assert.Equal(t, PhaseIdle, result.TransitionHistory[0].From)
		assert.Equal(t, PhaseDiscovery, result.TransitionHistory[0].To)
		assert.Equal(t, "Alice", result.TransitionHistory[0].User)
	})

	t.Run("records transition without user (defaults to Unknown User)", func(t *testing.T) {
		result := RecordTransition(s, PhaseIdle, PhaseDiscovery, nil, nil)
		assert.Len(t, result.TransitionHistory, 1)
		assert.Equal(t, "Unknown User", result.TransitionHistory[0].User)
	})
}

func TestAddCustomAC(t *testing.T) {
	s := CreateInitialState()
	result := AddCustomAC(s, "the system must do X", nil)
	assert.Len(t, result.CustomACs, 1)
	assert.Equal(t, "custom-ac-1", result.CustomACs[0].ID)
	assert.Equal(t, "the system must do X", result.CustomACs[0].Text)
	assert.Equal(t, "Unknown User", result.CustomACs[0].User)
}

func TestAddSpecNote(t *testing.T) {
	s := CreateInitialState()
	result := AddSpecNote(s, "this is a note", nil)
	assert.Len(t, result.SpecNotes, 1)
	assert.Equal(t, "note-1", result.SpecNotes[0].ID)
	assert.Equal(t, "this is a note", result.SpecNotes[0].Text)
}

func TestSetUserContext(t *testing.T) {
	s := CreateInitialState()
	result := SetUserContext(s, "user wants feature X")
	assert.Equal(t, "user wants feature X", *result.Discovery.UserContext)
	assert.Equal(t, false, *result.Discovery.UserContextProcessed)
}

func TestMarkUserContextProcessed(t *testing.T) {
	s := CreateInitialState()
	s = SetUserContext(s, "some context")
	result := MarkUserContextProcessed(s)
	assert.Equal(t, true, *result.Discovery.UserContextProcessed)
}

func TestClampConfidence(t *testing.T) {
	assert.Equal(t, 1, ClampConfidence(0))
	assert.Equal(t, 1, ClampConfidence(-5))
	assert.Equal(t, 1, ClampConfidence(1))
	assert.Equal(t, 5, ClampConfidence(5))
	assert.Equal(t, 10, ClampConfidence(10))
	assert.Equal(t, 10, ClampConfidence(15))
	assert.Equal(t, 5, ClampConfidence(5.4))
	assert.Equal(t, 6, ClampConfidence(5.5))
}

func TestAddConfidenceFinding(t *testing.T) {
	s := CreateInitialState()

	t.Run("adds finding with valid confidence", func(t *testing.T) {
		result, err := AddConfidenceFinding(s, "good finding", 5, "some basis")
		require.NoError(t, err)
		assert.Len(t, result.Execution.ConfidenceFindings, 1)
		assert.Equal(t, 5, result.Execution.ConfidenceFindings[0].Confidence)
	})

	t.Run("rejects high confidence without adequate basis", func(t *testing.T) {
		_, err := AddConfidenceFinding(s, "finding", 8, "short")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "basis")
	})

	t.Run("allows high confidence with adequate basis", func(t *testing.T) {
		result, err := AddConfidenceFinding(s, "finding", 8, "this is an adequate basis explanation")
		require.NoError(t, err)
		assert.Len(t, result.Execution.ConfidenceFindings, 1)
		assert.Equal(t, 8, result.Execution.ConfidenceFindings[0].Confidence)
	})
}

func TestGetLowConfidenceFindings(t *testing.T) {
	s := CreateInitialState()
	s, _ = AddConfidenceFinding(s, "low finding", 3, "basis")
	s, _ = AddConfidenceFinding(s, "medium finding", 5, "basis")
	s, _ = AddConfidenceFinding(s, "high finding", 8, "this is an adequate basis explanation")

	result := GetLowConfidenceFindings(s, 5)
	assert.Len(t, result, 1)
	assert.Equal(t, "low finding", result[0].Finding)
}

func TestGetAverageConfidence(t *testing.T) {
	t.Run("returns nil when no findings", func(t *testing.T) {
		s := CreateInitialState()
		result := GetAverageConfidence(s)
		assert.Nil(t, result)
	})

	t.Run("calculates average", func(t *testing.T) {
		s := CreateInitialState()
		s, _ = AddConfidenceFinding(s, "f1", 4, "basis")
		s, _ = AddConfidenceFinding(s, "f2", 6, "basis")
		result := GetAverageConfidence(s)
		require.NotNil(t, result)
		assert.Equal(t, 5.0, *result)
	})
}

func TestFollowUps(t *testing.T) {
	s := CreateInitialState()
	s.Phase = PhaseDiscovery

	t.Run("adds follow-up", func(t *testing.T) {
		result := AddFollowUp(s, "Q1", "follow-up question?", "agent")
		assert.Len(t, result.Discovery.FollowUps, 1)
		assert.Equal(t, "Q1a", result.Discovery.FollowUps[0].ID)
		assert.Equal(t, "pending", result.Discovery.FollowUps[0].Status)
	})

	t.Run("caps at 3 follow-ups per question", func(t *testing.T) {
		s2 := AddFollowUp(s, "Q1", "first question?", "agent")
		s2 = AddFollowUp(s2, "Q1", "second question?", "agent")
		s2 = AddFollowUp(s2, "Q1", "third question?", "agent")
		s2 = AddFollowUp(s2, "Q1", "fourth question?", "agent") // should be silently ignored
		assert.Len(t, s2.Discovery.FollowUps, 3)
	})

	t.Run("answers follow-up", func(t *testing.T) {
		s2 := AddFollowUp(s, "Q1", "follow-up?", "agent")
		s3 := AnswerFollowUp(s2, "Q1a", "the answer")
		assert.Equal(t, "answered", s3.Discovery.FollowUps[0].Status)
		assert.Equal(t, "the answer", *s3.Discovery.FollowUps[0].Answer)
		assert.NotNil(t, s3.Discovery.FollowUps[0].AnsweredAt)
	})

	t.Run("skips follow-up", func(t *testing.T) {
		s2 := AddFollowUp(s, "Q1", "follow-up?", "agent")
		s3 := SkipFollowUp(s2, "Q1a")
		assert.Equal(t, "skipped", s3.Discovery.FollowUps[0].Status)
	})

	t.Run("GetPendingFollowUps returns only pending", func(t *testing.T) {
		s2 := AddFollowUp(s, "Q1", "first?", "agent")
		s2 = AddFollowUp(s2, "Q2", "second?", "agent")
		s2 = SkipFollowUp(s2, "Q1a")

		pending := GetPendingFollowUps(s2)
		assert.Len(t, pending, 1)
		assert.Equal(t, "Q2a", pending[0].ID)
	})

	t.Run("GetFollowUpsForQuestion filters by parent", func(t *testing.T) {
		s2 := AddFollowUp(s, "Q1", "q1 follow-up?", "agent")
		s2 = AddFollowUp(s2, "Q2", "q2 follow-up?", "agent")

		result := GetFollowUpsForQuestion(s2, "Q1")
		assert.Len(t, result, 1)
		assert.Equal(t, "Q1a", result[0].ID)
	})
}

func TestDelegations(t *testing.T) {
	s := CreateInitialState()

	t.Run("adds delegation", func(t *testing.T) {
		result := AddDelegation(s, "Q1", "alice@example.com", "agent")
		assert.Len(t, result.Discovery.Delegations, 1)
		assert.Equal(t, "Q1", result.Discovery.Delegations[0].QuestionID)
		assert.Equal(t, "pending", result.Discovery.Delegations[0].Status)
	})

	t.Run("answers delegation", func(t *testing.T) {
		s2 := AddDelegation(s, "Q1", "alice@example.com", "agent")
		s3 := AnswerDelegation(s2, "Q1", "the delegated answer", "alice")
		assert.Equal(t, "answered", s3.Discovery.Delegations[0].Status)
		assert.Equal(t, "the delegated answer", *s3.Discovery.Delegations[0].Answer)
		assert.Equal(t, "alice", *s3.Discovery.Delegations[0].AnsweredBy)
	})

	t.Run("GetPendingDelegations returns only pending", func(t *testing.T) {
		s2 := AddDelegation(s, "Q1", "alice@example.com", "agent")
		s2 = AddDelegation(s2, "Q2", "bob@example.com", "agent")
		s2 = AnswerDelegation(s2, "Q1", "answer", "alice")

		pending := GetPendingDelegations(s2)
		assert.Len(t, pending, 1)
		assert.Equal(t, "Q2", pending[0].QuestionID)
	})
}

func TestSetContributors(t *testing.T) {
	s := CreateInitialState()
	result := SetContributors(s, []string{"alice", "bob"})
	assert.Equal(t, []string{"alice", "bob"}, result.Discovery.Contributors)
}

func TestResetToIdle(t *testing.T) {
	t.Run("resets from COMPLETED", func(t *testing.T) {
		s := CreateInitialState()
		s.Phase = PhaseCompleted
		specName := "my-spec"
		s.Spec = &specName

		result, err := ResetToIdle(s)
		require.NoError(t, err)
		assert.Equal(t, PhaseIdle, result.Phase)
		assert.Nil(t, result.Spec)
		assert.Nil(t, result.Branch)
		assert.Equal(t, []Decision{}, result.Decisions)
	})

	t.Run("resets from EXECUTING", func(t *testing.T) {
		s := CreateInitialState()
		s.Phase = PhaseExecuting

		result, err := ResetToIdle(s)
		require.NoError(t, err)
		assert.Equal(t, PhaseIdle, result.Phase)
	})

	t.Run("fails from DISCOVERY", func(t *testing.T) {
		s := CreateInitialState()
		s.Phase = PhaseDiscovery
		_, err := ResetToIdle(s)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cancel")
	})
}

// =============================================================================
// UncompleteTask
// =============================================================================

func makeExecutingStateWithTasks(completedIDs []string, overrideTasks []SpecTask) StateFile {
	s := CreateInitialState()
	s.Phase = PhaseExecuting
	s.Execution.CompletedTasks = completedIDs
	s.OverrideTasks = overrideTasks
	return s
}

func TestUncompleteTask_HappyPath_RemovesFromCompletedTasks(t *testing.T) {
	s := makeExecutingStateWithTasks([]string{"task-1", "task-2", "task-3"}, nil)
	result, err := UncompleteTask(s, "task-2")
	require.NoError(t, err)
	assert.Equal(t, []string{"task-1", "task-3"}, result.Execution.CompletedTasks)
}

func TestUncompleteTask_HappyPath_FlipsOverrideTaskCompleted(t *testing.T) {
	tasks := []SpecTask{
		{ID: "task-1", Title: "First", Completed: true},
		{ID: "task-2", Title: "Second", Completed: true},
	}
	s := makeExecutingStateWithTasks([]string{"task-1", "task-2"}, tasks)
	result, err := UncompleteTask(s, "task-1")
	require.NoError(t, err)

	// task-1 must be uncompleted
	assert.False(t, result.OverrideTasks[0].Completed)
	// task-2 must remain completed
	assert.True(t, result.OverrideTasks[1].Completed)
}

func TestUncompleteTask_UnknownID_ReturnsError(t *testing.T) {
	s := makeExecutingStateWithTasks([]string{"task-1"}, nil)
	_, err := UncompleteTask(s, "task-99")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task-99")
}

func TestUncompleteTask_DoesNotTouchPhaseOrIteration(t *testing.T) {
	s := makeExecutingStateWithTasks([]string{"task-1"}, nil)
	s.Execution.Iteration = 5
	result, err := UncompleteTask(s, "task-1")
	require.NoError(t, err)
	assert.Equal(t, PhaseExecuting, result.Phase)
	assert.Equal(t, 5, result.Execution.Iteration)
}
