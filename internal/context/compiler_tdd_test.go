package context_test

import (
	"strings"
	"testing"

	ctx "github.com/pragmataW/tddmaster/internal/context"
	specpkg "github.com/pragmataW/tddmaster/internal/spec"
	"github.com/pragmataW/tddmaster/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// TDDRules
// =============================================================================

func TestTDDRules_ReturnsNonEmptySlice(t *testing.T) {
	rules := ctx.TDDRules()
	assert.NotEmpty(t, rules, "TDDRules must return at least one rule")
}

func TestTDDRules_ContainsRedGreenRefactor(t *testing.T) {
	rules := ctx.TDDRules()
	found := false
	for _, r := range rules {
		if strings.Contains(r, "red-green-refactor") {
			found = true
			break
		}
	}
	assert.True(t, found, "TDD rules must include red-green-refactor guidance")
}

func TestTDDRules_ContainsWriteTestsBeforeImplementation(t *testing.T) {
	rules := ctx.TDDRules()
	found := false
	for _, r := range rules {
		lower := strings.ToLower(r)
		if strings.Contains(lower, "write test") || strings.Contains(lower, "tests before") {
			found = true
			break
		}
	}
	assert.True(t, found, "TDD rules must include write-tests-before-implementation guidance")
}

func TestTDDRules_ContainsTestTaskComesFirst(t *testing.T) {
	rules := ctx.TDDRules()
	found := false
	for _, r := range rules {
		lower := strings.ToLower(r)
		if strings.Contains(lower, "test task") && (strings.Contains(lower, "first") || strings.Contains(lower, "before")) {
			found = true
			break
		}
	}
	assert.True(t, found, "TDD rules must include test-task-first ordering guidance")
}

func TestTDDRules_ContainsVerifyTestsFail(t *testing.T) {
	rules := ctx.TDDRules()
	found := false
	for _, r := range rules {
		lower := strings.ToLower(r)
		if strings.Contains(lower, "fail") && strings.Contains(lower, "implementation") {
			found = true
			break
		}
	}
	assert.True(t, found, "TDD rules must include verify-tests-fail-before-implementation guidance")
}

func TestTDDRules_ReturnsCopy(t *testing.T) {
	rules1 := ctx.TDDRules()
	rules2 := ctx.TDDRules()
	// Modifying one must not affect the other
	if len(rules1) > 0 {
		rules1[0] = "mutated"
		assert.NotEqual(t, rules1[0], rules2[0], "TDDRules must return independent copies")
	}
}

// =============================================================================
// InjectTDDRules
// =============================================================================

func TestInjectTDDRules_AppendsRulesToEmptySlice(t *testing.T) {
	result := ctx.InjectTDDRules(nil)
	tddRules := ctx.TDDRules()
	require.Equal(t, len(tddRules), len(result))
	for i, r := range tddRules {
		assert.Equal(t, r, result[i])
	}
}

func TestInjectTDDRules_AppendsRulesToExistingRules(t *testing.T) {
	existing := []string{"rule-a", "rule-b"}
	result := ctx.InjectTDDRules(existing)

	tddRules := ctx.TDDRules()
	expectedLen := len(existing) + len(tddRules)
	require.Len(t, result, expectedLen)

	// Original rules preserved at the start
	assert.Equal(t, "rule-a", result[0])
	assert.Equal(t, "rule-b", result[1])

	// TDD rules appended after
	for i, r := range tddRules {
		assert.Equal(t, r, result[len(existing)+i])
	}
}

func TestInjectTDDRules_DoesNotModifyOriginalSlice(t *testing.T) {
	original := []string{"rule-a", "rule-b"}
	originalCopy := make([]string, len(original))
	copy(originalCopy, original)

	_ = ctx.InjectTDDRules(original)

	assert.Equal(t, originalCopy, original, "InjectTDDRules must not modify the original slice")
}

func TestInjectTDDRules_TDDRulesAreTagged(t *testing.T) {
	result := ctx.InjectTDDRules(nil)
	for _, r := range result {
		assert.Contains(t, r, "TDD", "all injected TDD rules should be tagged with TDD")
	}
}

func TestCompile_SpecDraftIncludesDerivedEdgeCases(t *testing.T) {
	st := state.CreateInitialState()
	specName := "tdd-edge-cases"
	specPath := "/tmp/tdd-edge-cases/spec.md"
	revision := "If the verifier rejects a test, ask the user before changing it."

	st.Phase = state.PhaseSpecProposal
	st.Spec = &specName
	st.SpecState.Path = &specPath
	st.Classification = &state.SpecClassification{}
	st.Discovery.Answers = []state.DiscoveryAnswer{
		{QuestionID: "verification", Answer: "- Cover timeout recovery\n- Happy path smoke test"},
	}
	st.Discovery.Premises = []state.Premise{
		{Text: "Tests can be rewritten automatically", Agreed: false, Revision: &revision},
	}

	out := ctx.Compile(st, nil, nil, nil, nil, nil, nil, nil, nil, 0)
	require.NotNil(t, out.SpecDraftData)
	assert.Contains(t, out.SpecDraftData.EdgeCases, "Cover timeout recovery")
	assert.Contains(t, out.SpecDraftData.EdgeCases, "If the verifier rejects a test, ask the user before changing it.")
	require.NotNil(t, out.SpecDraftData.SelfReview)
	assert.Contains(t, strings.Join(out.SpecDraftData.SelfReview.Checks, "\n"), "Edge cases")
}

func TestCompile_ExecutionIncludesEdgeCasesAndTestWriterGuidance(t *testing.T) {
	st := state.CreateInitialState()
	specName := "tdd-edge-cases"
	revision := "If the verifier rejects a test, ask the user before changing it."

	st.Phase = state.PhaseExecuting
	st.Spec = &specName
	st.Discovery.Answers = []state.DiscoveryAnswer{
		{QuestionID: "verification", Answer: "- Cover timeout recovery\n- Happy path smoke test"},
	}
	st.Discovery.Premises = []state.Premise{
		{Text: "Tests can be rewritten automatically", Agreed: false, Revision: &revision},
	}

	parsedSpec := &specpkg.ParsedSpec{
		Name: "tdd-edge-cases",
		Tasks: []specpkg.ParsedTask{
			{ID: "task-1", Title: "Write tests before implementation"},
		},
	}

	out := ctx.Compile(st, nil, nil, nil, parsedSpec, nil, nil, nil, nil, 0)
	require.NotNil(t, out.ExecutionData)
	assert.Contains(t, out.ExecutionData.EdgeCases, "Cover timeout recovery")
	assert.Contains(t, out.ExecutionData.Instruction, "edge cases")
	assert.Contains(t, out.ExecutionData.Instruction, "test-writer")
	assert.Contains(t, strings.Join(out.Behavioral.Rules, "\n"), "pass them explicitly to the test-writer")
}

// =============================================================================
// TDDPhase in ExecutionOutput
// =============================================================================

func makeTDDManifest(tddMode bool, maxRetries int) *state.NosManifest {
	return &state.NosManifest{
		Tdd: &state.Manifest{TddMode: tddMode, MaxVerificationRetries: maxRetries},
	}
}

func TestCompile_TDDPhase_SetWhenTDDEnabledAndCycleSet(t *testing.T) {
	st := state.CreateInitialState()
	st.Phase = state.PhaseExecuting
	st.Execution.TDDCycle = "red"
	specName := "my-spec"
	st.Spec = &specName

	out := ctx.Compile(st, nil, nil, makeTDDManifest(true, 3), nil, nil, nil, nil, nil, 0)
	require.NotNil(t, out.ExecutionData)
	require.NotNil(t, out.ExecutionData.TDDPhase, "TDDPhase must be set when TDDEnabled=true and cycle is set")
	assert.Equal(t, "red", *out.ExecutionData.TDDPhase)
}

func TestCompile_TDDPhase_NilWhenTDDDisabled(t *testing.T) {
	st := state.CreateInitialState()
	st.Phase = state.PhaseExecuting
	st.Execution.TDDCycle = "red"
	specName := "my-spec"
	st.Spec = &specName

	out := ctx.Compile(st, nil, nil, makeTDDManifest(false, 3), nil, nil, nil, nil, nil, 0)
	require.NotNil(t, out.ExecutionData)
	assert.Nil(t, out.ExecutionData.TDDPhase, "TDDPhase must be nil when TDDEnabled=false")
}

func TestCompile_TDDPhase_NilWhenCycleEmpty(t *testing.T) {
	st := state.CreateInitialState()
	st.Phase = state.PhaseExecuting
	st.Execution.TDDCycle = ""
	specName := "my-spec"
	st.Spec = &specName

	out := ctx.Compile(st, nil, nil, makeTDDManifest(true, 3), nil, nil, nil, nil, nil, 0)
	require.NotNil(t, out.ExecutionData)
	assert.Nil(t, out.ExecutionData.TDDPhase, "TDDPhase must be nil when cycle is empty")
}

// =============================================================================
// TDDVerificationContext in ExecutionOutput
// =============================================================================

func TestCompile_TDDVerificationContext_Red(t *testing.T) {
	st := state.CreateInitialState()
	st.Phase = state.PhaseExecuting
	st.Execution.TDDCycle = "red"
	specName := "my-spec"
	st.Spec = &specName

	out := ctx.Compile(st, nil, nil, makeTDDManifest(true, 3), nil, nil, nil, nil, nil, 0)
	require.NotNil(t, out.ExecutionData)
	require.NotNil(t, out.ExecutionData.TDDVerificationContext)
	assert.Equal(t, "red", out.ExecutionData.TDDVerificationContext.Phase)
	assert.Contains(t, out.ExecutionData.TDDVerificationContext.Instruction, "READ-ONLY")
	assert.Contains(t, out.ExecutionData.TDDVerificationContext.Instruction, "DO NOT run tests")
	assert.Contains(t, out.ExecutionData.TDDVerificationContext.Instruction, "well-formed")
}

func TestCompile_TDDVerificationContext_Green(t *testing.T) {
	st := state.CreateInitialState()
	st.Phase = state.PhaseExecuting
	st.Execution.TDDCycle = "green"
	specName := "my-spec"
	st.Spec = &specName

	out := ctx.Compile(st, nil, nil, makeTDDManifest(true, 3), nil, nil, nil, nil, nil, 0)
	require.NotNil(t, out.ExecutionData)
	require.NotNil(t, out.ExecutionData.TDDVerificationContext)
	assert.Equal(t, "green", out.ExecutionData.TDDVerificationContext.Phase)
	assert.Contains(t, out.ExecutionData.TDDVerificationContext.Instruction, "expected-pass-but-failed")
}

func TestCompile_TDDVerificationContext_Refactor(t *testing.T) {
	st := state.CreateInitialState()
	st.Phase = state.PhaseExecuting
	st.Execution.TDDCycle = "refactor"
	specName := "my-spec"
	st.Spec = &specName

	out := ctx.Compile(st, nil, nil, makeTDDManifest(true, 3), nil, nil, nil, nil, nil, 0)
	require.NotNil(t, out.ExecutionData)
	require.NotNil(t, out.ExecutionData.TDDVerificationContext)
	assert.Equal(t, "refactor", out.ExecutionData.TDDVerificationContext.Phase)
	assert.Contains(t, out.ExecutionData.TDDVerificationContext.Instruction, "behavior-changed")
}

// =============================================================================
// TDDFailureReport in ExecutionOutput
// =============================================================================

func TestCompile_TDDFailureReport_PresentWhenVerificationFailed(t *testing.T) {
	st := state.CreateInitialState()
	st.Phase = state.PhaseExecuting
	specName := "my-spec"
	st.Spec = &specName
	st.Execution.LastVerification = &state.VerificationResult{
		Passed:               false,
		Output:               "2 tests failed",
		UncoveredEdgeCases:   []string{"EC-3"},
		VerificationFailCount: 1,
	}

	out := ctx.Compile(st, nil, nil, makeTDDManifest(true, 3), nil, nil, nil, nil, nil, 0)
	require.NotNil(t, out.ExecutionData)
	require.NotNil(t, out.ExecutionData.TDDFailureReport, "TDDFailureReport must be present when last verification failed")
	assert.Equal(t, "verification-failed", out.ExecutionData.TDDFailureReport.Reason)
	assert.Equal(t, []string{"EC-3"}, out.ExecutionData.TDDFailureReport.UncoveredEdgeCases)
	assert.Equal(t, 1, out.ExecutionData.TDDFailureReport.RetryCount)
	assert.Equal(t, 3, out.ExecutionData.TDDFailureReport.MaxRetries)
}

func TestCompile_TDDFailureReport_NilWhenVerificationPassed(t *testing.T) {
	st := state.CreateInitialState()
	st.Phase = state.PhaseExecuting
	specName := "my-spec"
	st.Spec = &specName
	st.Execution.LastVerification = &state.VerificationResult{
		Passed: true,
		Output: "all tests pass",
	}

	out := ctx.Compile(st, nil, nil, makeTDDManifest(true, 3), nil, nil, nil, nil, nil, 0)
	require.NotNil(t, out.ExecutionData)
	assert.Nil(t, out.ExecutionData.TDDFailureReport, "TDDFailureReport must be nil when last verification passed")
}

func TestCompile_TDDFailureReport_WillBlock_WhenFailCountAtMax(t *testing.T) {
	st := state.CreateInitialState()
	st.Phase = state.PhaseExecuting
	specName := "my-spec"
	st.Spec = &specName
	st.Execution.LastVerification = &state.VerificationResult{
		Passed:               false,
		Output:               "failed",
		VerificationFailCount: 3,
	}

	out := ctx.Compile(st, nil, nil, makeTDDManifest(true, 3), nil, nil, nil, nil, nil, 0)
	require.NotNil(t, out.ExecutionData)
	require.NotNil(t, out.ExecutionData.TDDFailureReport)
	assert.True(t, out.ExecutionData.TDDFailureReport.WillBlock, "WillBlock must be true when failCount >= maxRetries")
}

func TestCompile_TDDFailureReport_NilWhenTDDDisabled(t *testing.T) {
	st := state.CreateInitialState()
	st.Phase = state.PhaseExecuting
	specName := "my-spec"
	st.Spec = &specName
	st.Execution.LastVerification = &state.VerificationResult{
		Passed:               false,
		Output:               "failed",
		VerificationFailCount: 2,
	}

	out := ctx.Compile(st, nil, nil, makeTDDManifest(false, 3), nil, nil, nil, nil, nil, 0)
	require.NotNil(t, out.ExecutionData)
	assert.Nil(t, out.ExecutionData.TDDFailureReport, "TDDFailureReport must be nil when TDDEnabled=false")
}

// =============================================================================
// RefactorInstructions emission
// =============================================================================

func makeTDDRefactorManifest(maxRounds int) *state.NosManifest {
	return &state.NosManifest{
		Tdd: &state.Manifest{TddMode: true, MaxVerificationRetries: 3, MaxRefactorRounds: maxRounds},
	}
}

func TestCompile_RefactorInstructions_EmittedWhenNotesPresentAndNotApplied(t *testing.T) {
	st := state.CreateInitialState()
	specName := "spec-r"
	st.Phase = state.PhaseExecuting
	st.Spec = &specName
	st.Execution.TDDCycle = state.TDDCycleRefactor
	st.Execution.RefactorRounds = 1
	st.Execution.RefactorApplied = false
	st.Execution.LastVerification = &state.VerificationResult{
		Passed:    true,
		Phase:     state.TDDCycleRefactor,
		Timestamp: "ts",
		RefactorNotes: []state.RefactorNote{
			{File: "a.go", Suggestion: "rename X to Y", Rationale: "clarity"},
		},
	}

	out := ctx.Compile(st, nil, nil, makeTDDRefactorManifest(3), nil, nil, nil, nil, nil, 0)
	require.NotNil(t, out.ExecutionData)
	require.NotNil(t, out.ExecutionData.RefactorInstructions)
	assert.Len(t, out.ExecutionData.RefactorInstructions.Notes, 1)
	assert.Equal(t, 2, out.ExecutionData.RefactorInstructions.Round, "round must be RefactorRounds+1")
	assert.Equal(t, 3, out.ExecutionData.RefactorInstructions.MaxRounds)
	assert.Contains(t, out.ExecutionData.RefactorInstructions.Instruction, "refactorApplied")
}

func TestCompile_RefactorInstructions_NilWhenRefactorApplied(t *testing.T) {
	st := state.CreateInitialState()
	specName := "spec-r"
	st.Phase = state.PhaseExecuting
	st.Spec = &specName
	st.Execution.TDDCycle = state.TDDCycleRefactor
	st.Execution.RefactorApplied = true
	st.Execution.LastVerification = &state.VerificationResult{
		Passed:        true,
		Phase:         state.TDDCycleRefactor,
		RefactorNotes: []state.RefactorNote{{File: "a.go", Suggestion: "x", Rationale: "y"}},
	}

	out := ctx.Compile(st, nil, nil, makeTDDRefactorManifest(3), nil, nil, nil, nil, nil, 0)
	require.NotNil(t, out.ExecutionData)
	assert.Nil(t, out.ExecutionData.RefactorInstructions, "must be nil once executor has consumed the batch")
}

func TestCompile_RefactorInstructions_NilWhenNotesEmpty(t *testing.T) {
	st := state.CreateInitialState()
	specName := "spec-r"
	st.Phase = state.PhaseExecuting
	st.Spec = &specName
	st.Execution.TDDCycle = state.TDDCycleRefactor
	st.Execution.LastVerification = &state.VerificationResult{
		Passed:        true,
		Phase:         state.TDDCycleRefactor,
		RefactorNotes: nil,
	}

	out := ctx.Compile(st, nil, nil, makeTDDRefactorManifest(3), nil, nil, nil, nil, nil, 0)
	require.NotNil(t, out.ExecutionData)
	assert.Nil(t, out.ExecutionData.RefactorInstructions)
}

func TestCompile_RefactorInstructions_NilWhenCycleNotRefactor(t *testing.T) {
	st := state.CreateInitialState()
	specName := "spec-r"
	st.Phase = state.PhaseExecuting
	st.Spec = &specName
	st.Execution.TDDCycle = state.TDDCycleGreen
	st.Execution.LastVerification = &state.VerificationResult{
		Passed:        true,
		Phase:         state.TDDCycleRefactor,
		RefactorNotes: []state.RefactorNote{{File: "a.go", Suggestion: "x", Rationale: "y"}},
	}

	out := ctx.Compile(st, nil, nil, makeTDDRefactorManifest(3), nil, nil, nil, nil, nil, 0)
	require.NotNil(t, out.ExecutionData)
	assert.Nil(t, out.ExecutionData.RefactorInstructions)
}

// =============================================================================
// TDD verification-context instructions — updated contract keywords
// =============================================================================

func TestCompile_TDDVerificationContext_Red_SignalsExitCodeExpectation(t *testing.T) {
	st := state.CreateInitialState()
	specName := "my-spec"
	st.Phase = state.PhaseExecuting
	st.Spec = &specName
	st.Execution.TDDCycle = "red"

	out := ctx.Compile(st, nil, nil, makeTDDManifest(true, 3), nil, nil, nil, nil, nil, 0)
	require.NotNil(t, out.ExecutionData.TDDVerificationContext)
	assert.Contains(t, out.ExecutionData.TDDVerificationContext.Instruction, "READ-ONLY", "RED must be read-only — no test execution")
	assert.NotContains(t, out.ExecutionData.TDDVerificationContext.Instruction, "non-zero", "RED must not require non-zero exit code")
}

func TestCompile_TDDVerificationContext_Refactor_MentionsNotesContract(t *testing.T) {
	st := state.CreateInitialState()
	specName := "my-spec"
	st.Phase = state.PhaseExecuting
	st.Spec = &specName
	st.Execution.TDDCycle = "refactor"

	out := ctx.Compile(st, nil, nil, makeTDDManifest(true, 3), nil, nil, nil, nil, nil, 0)
	require.NotNil(t, out.ExecutionData.TDDVerificationContext)
	assert.Contains(t, out.ExecutionData.TDDVerificationContext.Instruction, "refactorNotes", "REFACTOR must describe refactorNotes contract")
}

// =============================================================================
// tddBehavioralRules — delegation table (rule #5 rewrite)
// =============================================================================

func TestTDDRules_RuleFiveDescribesDelegationTable(t *testing.T) {
	rules := ctx.TDDRules()
	combined := strings.Join(rules, "\n")
	assert.Contains(t, combined, "Delegation table", "rule #5 must describe the delegation table")
	assert.Contains(t, combined, "test-writer", "delegation table must reference test-writer")
	assert.Contains(t, combined, "tddmaster-executor", "delegation table must reference tddmaster-executor")
	assert.Contains(t, combined, "tddmaster-verifier", "delegation table must reference tddmaster-verifier")
	assert.Contains(t, combined, "refactorInstructions", "delegation table must reference refactorInstructions")
}

func TestCompile_ExecutionStatusReportCarriesEdgeCases(t *testing.T) {
	st := state.CreateInitialState()
	specName := "tdd-edge-cases"
	revision := "If the verifier rejects a test, ask the user before changing it."

	st.Phase = state.PhaseExecuting
	st.Spec = &specName
	st.Execution.AwaitingStatusReport = true
	st.Discovery.Answers = []state.DiscoveryAnswer{
		{QuestionID: "verification", Answer: "- Cover timeout recovery\n- Happy path smoke test"},
	}
	st.Discovery.Premises = []state.Premise{
		{Text: "Tests can be rewritten automatically", Agreed: false, Revision: &revision},
	}

	out := ctx.Compile(st, nil, nil, nil, nil, nil, nil, nil, nil, 0)
	require.NotNil(t, out.ExecutionData)
	assert.Contains(t, out.ExecutionData.EdgeCases, "Cover timeout recovery")
	assert.Contains(t, out.ExecutionData.EdgeCases, "If the verifier rejects a test, ask the user before changing it.")
}
