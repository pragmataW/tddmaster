package phases

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func seedRuleLearningSpec(t *testing.T, root, slug string, tasks []spec.Task) {
	t.Helper()
	writeDiscoveryManifest(t, root)
	state := spec.State{
		Version: 1,
		Slug:    slug,
		Phase:   "rule-learning",
		Answers: map[string][]spec.Answer{},
	}
	if err := spec.SaveState(root, slug, state); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	if err := spec.SaveSettings(root, slug, spec.DefaultSettings()); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	pr := spec.Progress{Spec: slug, Status: spec.StatusDraft, Tasks: tasks}
	if err := spec.SaveProgress(root, slug, pr); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
}

func buildRuleLearningCtx(t *testing.T, root, slug string) *engine.Context {
	t.Helper()
	defs := []engine.PhaseDef{{ID: "rule-learning", Driver: RuleLearningDriver()}}
	ctx, err := engine.Build(root, slug, defs)
	if err != nil {
		t.Fatalf("engine.Build: %v", err)
	}
	return ctx
}

func tasksWithLearnings() []spec.Task {
	return []spec.Task{
		{
			ID:       "task-1",
			Title:    "Alpha",
			Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac1"}},
			RefactorNotes: []spec.RefactorNote{
				{File: "foo.go", Suggestion: "extract helper", Rationale: "reduces duplication"},
			},
			FailedACReasons: []string{"missing validation"},
		},
		{
			ID:       "task-2",
			Title:    "Beta",
			Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac2"}},
			RefactorNotes: []spec.RefactorNote{
				{File: "bar.go", Suggestion: "inline constant", Rationale: "clarity"},
			},
			FailedACReasons: []string{"wrong response code"},
		},
	}
}

func tasksWithNoLearnings() []spec.Task {
	return []spec.Task{
		{ID: "task-1", Title: "Alpha", Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac1"}}},
		{ID: "task-2", Title: "Beta", Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac2"}}},
	}
}

const validProposalJSON = `{"rules":[{"scope":"executor","name":"r1","content":"always validate inputs","rationale":"prevents invalid state"}]}`

func TestRuleLearningDriver_NoLearnings_TerminalNoPhaseDone(t *testing.T) {
	root := t.TempDir()
	seedRuleLearningSpec(t, root, "s", tasksWithNoLearnings())
	ctx := buildRuleLearningCtx(t, root, "s")

	action, phaseDone := RuleLearningDriver().Next(ctx, &engine.PhaseDef{ID: "rule-learning"})
	if !phaseDone {
		t.Fatal("expected phaseDone=true when no learnings")
	}
	if action.DelegateAgent != "" {
		t.Fatalf("DelegateAgent must be empty (synthesizer not invoked), got %q", action.DelegateAgent)
	}
	if action.Action != "" && action.Action != engine.ActionTerminal {
		t.Fatalf("action = %q, want terminal or empty", action.Action)
	}
}

func TestRuleLearningDriver_WithLearnings_NoProposal_InstructsSynthesizer(t *testing.T) {
	root := t.TempDir()
	seedRuleLearningSpec(t, root, "s", tasksWithLearnings())
	ctx := buildRuleLearningCtx(t, root, "s")

	action, phaseDone := RuleLearningDriver().Next(ctx, &engine.PhaseDef{ID: "rule-learning"})
	if phaseDone {
		t.Fatal("phaseDone must be false when learnings exist and no proposal yet")
	}
	if action.Action != engine.ActionInstruct {
		t.Fatalf("action = %q, want %q", action.Action, engine.ActionInstruct)
	}
	if action.DelegateAgent != "tddmaster-rule-synthesizer" {
		t.Fatalf("DelegateAgent = %q, want %q", action.DelegateAgent, "tddmaster-rule-synthesizer")
	}
	if !strings.Contains(action.Instruction, "extract helper") {
		t.Error("instruction must contain refactor note suggestion 'extract helper'")
	}
	if !strings.Contains(action.Instruction, "missing validation") {
		t.Error("instruction must contain failed AC reason 'missing validation'")
	}
	if !strings.Contains(action.Instruction, "wrong response code") {
		t.Error("instruction must contain failed AC reason 'wrong response code'")
	}
	if !strings.Contains(action.Instruction, "inline constant") {
		t.Error("instruction must contain refactor note suggestion 'inline constant'")
	}
}

func TestRuleLearningDriver_ProposalPresentNotApproved_AsksAcceptReviseReject(t *testing.T) {
	root := t.TempDir()
	seedRuleLearningSpec(t, root, "s", tasksWithLearnings())
	ctx := buildRuleLearningCtx(t, root, "s")
	if err := ctx.SetAnswer("rule_proposal", validProposalJSON); err != nil {
		t.Fatalf("SetAnswer: %v", err)
	}

	action, phaseDone := RuleLearningDriver().Next(ctx, &engine.PhaseDef{ID: "rule-learning"})
	if phaseDone {
		t.Fatal("phaseDone must be false when proposal awaits approval")
	}
	if action.Action != engine.ActionAsk {
		t.Fatalf("action = %q, want %q", action.Action, engine.ActionAsk)
	}
	if len(action.InteractiveOptions) != 3 {
		t.Fatalf("expected 3 interactive options, got %d", len(action.InteractiveOptions))
	}

	labels := make([]string, len(action.InteractiveOptions))
	for i, o := range action.InteractiveOptions {
		labels[i] = strings.ToLower(o.Label)
	}
	hasAccept, hasRevise, hasReject := false, false, false
	for _, l := range labels {
		if strings.Contains(l, "accept") {
			hasAccept = true
		}
		if strings.Contains(l, "revise") {
			hasRevise = true
		}
		if strings.Contains(l, "reject") {
			hasReject = true
		}
	}
	if !hasAccept {
		t.Error("interactive options must include an 'accept' label")
	}
	if !hasRevise {
		t.Error("interactive options must include a 'revise' label")
	}
	if !hasReject {
		t.Error("interactive options must include a 'reject' label")
	}
	if len(action.CommandMap) == 0 && action.ExpectedInput.SubmitCmd == "" {
		t.Error("either CommandMap or ExpectedInput.SubmitCmd must be non-empty")
	}
}

func TestRuleLearningDriver_Approved_NotApplied_InstructsApply(t *testing.T) {
	root := t.TempDir()
	seedRuleLearningSpec(t, root, "s", tasksWithLearnings())
	ctx := buildRuleLearningCtx(t, root, "s")
	if err := ctx.SetAnswer("rule_proposal", validProposalJSON); err != nil {
		t.Fatalf("SetAnswer rule_proposal: %v", err)
	}
	if err := ctx.SetAnswer("rule_approved", "true"); err != nil {
		t.Fatalf("SetAnswer rule_approved: %v", err)
	}

	action, phaseDone := RuleLearningDriver().Next(ctx, &engine.PhaseDef{ID: "rule-learning"})
	if phaseDone {
		t.Fatal("phaseDone must be false when approved but not yet applied")
	}
	if action.Action != engine.ActionInstruct {
		t.Fatalf("action = %q, want %q", action.Action, engine.ActionInstruct)
	}
	if action.DelegateAgent != "tddmaster-rule-synthesizer" {
		t.Fatalf("DelegateAgent = %q, want %q", action.DelegateAgent, "tddmaster-rule-synthesizer")
	}
	if !strings.Contains(action.Instruction, "tddmaster rule add") {
		t.Error("instruction must contain 'tddmaster rule add'")
	}
	if !strings.Contains(action.Instruction, "r1") {
		t.Error("instruction must contain rule name 'r1'")
	}
	if !strings.Contains(action.Instruction, "executor") {
		t.Error("instruction must contain rule scope 'executor'")
	}
	if !strings.Contains(action.Instruction, "always validate inputs") {
		t.Error("instruction must contain the rule content verbatim so the fresh synthesizer can write it")
	}
}

func TestRuleLearningDriver_Applied_Terminal(t *testing.T) {
	root := t.TempDir()
	seedRuleLearningSpec(t, root, "s", tasksWithLearnings())
	ctx := buildRuleLearningCtx(t, root, "s")
	if err := ctx.SetAnswer("rule_proposal", validProposalJSON); err != nil {
		t.Fatalf("SetAnswer rule_proposal: %v", err)
	}
	if err := ctx.SetAnswer("rule_approved", "true"); err != nil {
		t.Fatalf("SetAnswer rule_approved: %v", err)
	}
	if err := ctx.SetAnswer("rule_applied", "true"); err != nil {
		t.Fatalf("SetAnswer rule_applied: %v", err)
	}

	action, phaseDone := RuleLearningDriver().Next(ctx, &engine.PhaseDef{ID: "rule-learning"})
	if !phaseDone {
		t.Fatal("phaseDone must be true after rule_applied=true")
	}
	if action.DelegateAgent != "" {
		t.Fatalf("DelegateAgent must be empty after apply, got %q", action.DelegateAgent)
	}
	if action.Action != "" && action.Action != engine.ActionTerminal {
		t.Fatalf("action = %q, want terminal or empty", action.Action)
	}
}

func TestRuleLearningDriver_Submit_ValidProposal_StoresRuleProposal(t *testing.T) {
	root := t.TempDir()
	seedRuleLearningSpec(t, root, "s", tasksWithLearnings())
	ctx := buildRuleLearningCtx(t, root, "s")

	_, _, err := RuleLearningDriver().Submit(ctx, &engine.PhaseDef{ID: "rule-learning"}, []byte(validProposalJSON))
	if err != nil {
		t.Fatalf("Submit with valid proposal returned error: %v", err)
	}
	if !ctx.HasAnswer("rule_proposal") {
		t.Fatal("rule_proposal must be stored after valid Submit")
	}
}

func TestRuleLearningDriver_Submit_AllScope_NormalizesToGlobal(t *testing.T) {
	root := t.TempDir()
	slug := "all-scope"
	seedRuleLearningSpec(t, root, slug, tasksWithLearnings())
	ctx := buildRuleLearningCtx(t, root, slug)

	proposal := `{"rules":[{"scope":"all","name":"shared-rule","content":"shared content","rationale":"applies everywhere"}]}`
	if _, _, err := RuleLearningDriver().Submit(ctx, nil, []byte(proposal)); err != nil {
		t.Fatalf("Submit all-scope proposal: %v", err)
	}

	var stored ruleProposal
	if err := json.Unmarshal([]byte(ctx.AnswerValue("rule_proposal")), &stored); err != nil {
		t.Fatalf("unmarshal stored proposal: %v", err)
	}
	if got := stored.Rules[0].Scope; got != "global" {
		t.Fatalf("stored scope = %q, want canonical global", got)
	}

	action, done := RuleLearningDriver().Next(ctx, nil)
	if done {
		t.Fatal("proposal must still await user approval")
	}
	if strings.Contains(action.Instruction, "scope: all") {
		t.Fatalf("review must not expose the unsupported all scope:\n%s", action.Instruction)
	}
	if !strings.Contains(action.Instruction, "scope: global") {
		t.Fatalf("review must show the canonical global scope:\n%s", action.Instruction)
	}
}

func TestRuleLearningDriver_Next_LegacyApprovedAllScope_EmitsGlobalApplyCommand(t *testing.T) {
	root := t.TempDir()
	slug := "legacy-all-scope"
	seedRuleLearningSpec(t, root, slug, tasksWithLearnings())
	ctx := buildRuleLearningCtx(t, root, slug)
	legacyProposal := `{"rules":[{"scope":"all","name":"shared-rule","content":"shared content","rationale":"applies everywhere"}]}`
	if err := ctx.SetAnswer("rule_proposal", legacyProposal); err != nil {
		t.Fatalf("SetAnswer rule_proposal: %v", err)
	}
	if err := ctx.SetAnswer("rule_approved", "true"); err != nil {
		t.Fatalf("SetAnswer rule_approved: %v", err)
	}

	action, done := RuleLearningDriver().Next(ctx, nil)
	if done {
		t.Fatal("approved proposal must await application")
	}
	if !strings.Contains(action.Instruction, "--scope global") {
		t.Fatalf("legacy all scope must emit a global apply command:\n%s", action.Instruction)
	}
	if strings.Contains(action.Instruction, "--scope all") {
		t.Fatalf("apply command must not retain unsupported all scope:\n%s", action.Instruction)
	}
}

func TestRuleLearningDriver_Submit_ApplyFailure_DoesNotCompletePhase(t *testing.T) {
	root := t.TempDir()
	slug := "apply-failure"
	seedRuleLearningSpec(t, root, slug, tasksWithLearnings())
	ctx := buildRuleLearningCtx(t, root, slug)
	if err := ctx.SetAnswer("rule_proposal", validProposalJSON); err != nil {
		t.Fatalf("SetAnswer rule_proposal: %v", err)
	}
	if err := ctx.SetAnswer("rule_approved", "true"); err != nil {
		t.Fatalf("SetAnswer rule_approved: %v", err)
	}

	action, done, err := RuleLearningDriver().Submit(ctx, nil, []byte(`unknown target "all"`))
	if err != nil {
		t.Fatalf("apply failure should be returned as an action, got error: %v", err)
	}
	if done {
		t.Fatal("apply failure must not complete rule learning")
	}
	if action.Action != engine.ActionError {
		t.Fatalf("action = %q, want error", action.Action)
	}
	if !strings.Contains(action.Instruction, `unknown target "all"`) {
		t.Fatalf("error action must preserve the apply failure, got %q", action.Instruction)
	}
	if ctx.HasAnswer("rule_applied") {
		t.Fatal("apply failure must not set rule_applied")
	}

	retry, retryDone := RuleLearningDriver().Next(ctx, nil)
	if retryDone || retry.Action != engine.ActionInstruct {
		t.Fatalf("next action must re-emit the apply instruction, done=%v action=%q", retryDone, retry.Action)
	}
}

func TestRuleLearningDriver_Submit_AppliedLiteral_CompletesPhase(t *testing.T) {
	root := t.TempDir()
	slug := "apply-success"
	seedRuleLearningSpec(t, root, slug, tasksWithLearnings())
	ctx := buildRuleLearningCtx(t, root, slug)
	if err := ctx.SetAnswer("rule_proposal", validProposalJSON); err != nil {
		t.Fatalf("SetAnswer rule_proposal: %v", err)
	}
	if err := ctx.SetAnswer("rule_approved", "true"); err != nil {
		t.Fatalf("SetAnswer rule_approved: %v", err)
	}

	action, done, err := RuleLearningDriver().Submit(ctx, nil, []byte("applied"))
	if err != nil {
		t.Fatalf("Submit applied: %v", err)
	}
	if !done {
		t.Fatal("literal applied must complete rule learning")
	}
	if action.Action != "" {
		t.Fatalf("successful apply action = %q, want empty", action.Action)
	}
	if !ctx.HasAnswer("rule_applied") {
		t.Fatal("successful apply must set rule_applied")
	}
}

func TestRuleLearningDriver_Submit_MalformedJSON_ReturnsError(t *testing.T) {
	root := t.TempDir()
	seedRuleLearningSpec(t, root, "s", tasksWithLearnings())
	ctx := buildRuleLearningCtx(t, root, "s")

	_, _, err := RuleLearningDriver().Submit(ctx, &engine.PhaseDef{ID: "rule-learning"}, []byte(`{not valid json`))
	if err == nil {
		t.Fatal("Submit with malformed JSON must return a non-nil error")
	}
	if ctx.HasAnswer("rule_proposal") {
		t.Fatal("rule_proposal must NOT be stored after malformed JSON")
	}
}

func TestRuleLearningDriver_Submit_EmptyRulesArray_ReturnsError(t *testing.T) {
	root := t.TempDir()
	seedRuleLearningSpec(t, root, "s", tasksWithLearnings())
	ctx := buildRuleLearningCtx(t, root, "s")

	_, _, err := RuleLearningDriver().Submit(ctx, &engine.PhaseDef{ID: "rule-learning"}, []byte(`{"rules":[]}`))
	if err == nil {
		t.Fatal("Submit with empty rules array must return a non-nil error")
	}
	if ctx.HasAnswer("rule_proposal") {
		t.Fatal("rule_proposal must NOT be stored after empty rules")
	}
}

func TestRuleLearningDriver_Submit_Accept_SetsRuleApproved(t *testing.T) {
	root := t.TempDir()
	seedRuleLearningSpec(t, root, "s", tasksWithLearnings())
	ctx := buildRuleLearningCtx(t, root, "s")
	if err := ctx.SetAnswer("rule_proposal", validProposalJSON); err != nil {
		t.Fatalf("SetAnswer: %v", err)
	}

	_, _, err := RuleLearningDriver().Submit(ctx, &engine.PhaseDef{ID: "rule-learning"}, []byte(`{"accepted":true}`))
	if err != nil {
		t.Fatalf("Submit accept returned error: %v", err)
	}
	if !ctx.HasAnswer("rule_approved") {
		t.Fatal("rule_approved must be set after accept")
	}
	if ctx.AnswerValue("rule_approved") != "true" {
		t.Fatalf("rule_approved = %q, want %q", ctx.AnswerValue("rule_approved"), "true")
	}
}

func TestRuleLearningDriver_Submit_Reject_Terminal(t *testing.T) {
	root := t.TempDir()
	seedRuleLearningSpec(t, root, "s", tasksWithLearnings())
	ctx := buildRuleLearningCtx(t, root, "s")
	if err := ctx.SetAnswer("rule_proposal", validProposalJSON); err != nil {
		t.Fatalf("SetAnswer: %v", err)
	}

	_, _, err := RuleLearningDriver().Submit(ctx, &engine.PhaseDef{ID: "rule-learning"}, []byte(`{"accepted":false}`))
	if err != nil {
		t.Fatalf("Submit reject returned error: %v", err)
	}

	action, phaseDone := RuleLearningDriver().Next(ctx, &engine.PhaseDef{ID: "rule-learning"})
	if !phaseDone {
		t.Fatal("phaseDone must be true after reject")
	}
	if action.DelegateAgent != "" {
		t.Fatalf("DelegateAgent must be empty after reject, got %q", action.DelegateAgent)
	}
}

func TestRuleLearningDriver_Submit_Revise_ClearsProposalStoresFeedback(t *testing.T) {
	root := t.TempDir()
	seedRuleLearningSpec(t, root, "s", tasksWithLearnings())
	ctx := buildRuleLearningCtx(t, root, "s")
	if err := ctx.SetAnswer("rule_proposal", validProposalJSON); err != nil {
		t.Fatalf("SetAnswer: %v", err)
	}

	_, _, err := RuleLearningDriver().Submit(ctx, &engine.PhaseDef{ID: "rule-learning"}, []byte(`{"planFeedback":"use stricter naming"}`))
	if err != nil {
		t.Fatalf("Submit revise returned error: %v", err)
	}

	if ctx.HasAnswer("rule_proposal") {
		t.Fatal("rule_proposal must be cleared after revise")
	}
	if !ctx.HasAnswer("rule_feedback") {
		t.Fatal("rule_feedback must be stored after revise")
	}
}

func TestRuleLearningDriver_Submit_Revise_NextReInstructsWithPriorFeedback(t *testing.T) {
	root := t.TempDir()
	seedRuleLearningSpec(t, root, "s", tasksWithLearnings())
	ctx := buildRuleLearningCtx(t, root, "s")
	if err := ctx.SetAnswer("rule_proposal", validProposalJSON); err != nil {
		t.Fatalf("SetAnswer: %v", err)
	}

	_, _, err := RuleLearningDriver().Submit(ctx, &engine.PhaseDef{ID: "rule-learning"}, []byte(`{"planFeedback":"use stricter naming"}`))
	if err != nil {
		t.Fatalf("Submit revise returned error: %v", err)
	}

	action, phaseDone := RuleLearningDriver().Next(ctx, &engine.PhaseDef{ID: "rule-learning"})
	if phaseDone {
		t.Fatal("phaseDone must be false after revise (should re-propose)")
	}
	if action.Action != engine.ActionInstruct {
		t.Fatalf("action = %q, want %q after revise", action.Action, engine.ActionInstruct)
	}
	if action.DelegateAgent != "tddmaster-rule-synthesizer" {
		t.Fatalf("DelegateAgent = %q after revise", action.DelegateAgent)
	}
	if !strings.Contains(action.Instruction, "use stricter naming") {
		t.Error("instruction must contain priorFeedback 'use stricter naming' on re-propose")
	}
}

func TestRuleLearningDriver_Submit_UnrecognizedJSONApprovalAnswer_ReturnsError(t *testing.T) {
	root := t.TempDir()
	seedRuleLearningSpec(t, root, "s", tasksWithLearnings())
	ctx := buildRuleLearningCtx(t, root, "s")
	if err := ctx.SetAnswer("rule_proposal", validProposalJSON); err != nil {
		t.Fatalf("SetAnswer: %v", err)
	}

	_, _, err := RuleLearningDriver().Submit(ctx, &engine.PhaseDef{ID: "rule-learning"}, []byte(`{"foo":"bar"}`))
	if err == nil {
		t.Fatal("Submit with unrecognized JSON approval answer must return a non-nil error")
	}

	if ctx.HasAnswer("rule_approved") {
		t.Fatal("rule_approved must NOT be set after unrecognized approval answer")
	}
	if ctx.HasAnswer("rule_applied") {
		t.Fatal("rule_applied must NOT be set after unrecognized approval answer")
	}
	if !ctx.HasAnswer("rule_proposal") {
		t.Fatal("rule_proposal must still be present after unrecognized approval answer (no state transition)")
	}
}

func TestRuleLearningDriver_Submit_GarbageStringApprovalAnswer_ReturnsError(t *testing.T) {
	root := t.TempDir()
	seedRuleLearningSpec(t, root, "s", tasksWithLearnings())
	ctx := buildRuleLearningCtx(t, root, "s")
	if err := ctx.SetAnswer("rule_proposal", validProposalJSON); err != nil {
		t.Fatalf("SetAnswer: %v", err)
	}

	_, _, err := RuleLearningDriver().Submit(ctx, &engine.PhaseDef{ID: "rule-learning"}, []byte(`maybe`))
	if err == nil {
		t.Fatal("Submit with garbage string approval answer must return a non-nil error")
	}

	if ctx.HasAnswer("rule_approved") {
		t.Fatal("rule_approved must NOT be set after garbage string approval answer")
	}
	if ctx.HasAnswer("rule_applied") {
		t.Fatal("rule_applied must NOT be set after garbage string approval answer")
	}
	if !ctx.HasAnswer("rule_proposal") {
		t.Fatal("rule_proposal must still be present after garbage string approval answer (no state transition)")
	}
}

func TestRuleLearningDriver_Submit_UnrecognizedAnswer_NextStillAsksApproval(t *testing.T) {
	root := t.TempDir()
	seedRuleLearningSpec(t, root, "s", tasksWithLearnings())
	ctx := buildRuleLearningCtx(t, root, "s")
	if err := ctx.SetAnswer("rule_proposal", validProposalJSON); err != nil {
		t.Fatalf("SetAnswer: %v", err)
	}

	_, _, _ = RuleLearningDriver().Submit(ctx, &engine.PhaseDef{ID: "rule-learning"}, []byte(`{"foo":"bar"}`))

	action, phaseDone := RuleLearningDriver().Next(ctx, &engine.PhaseDef{ID: "rule-learning"})
	if phaseDone {
		t.Fatal("phaseDone must be false: proposal still pending after unrecognized answer")
	}
	if action.Action != engine.ActionAsk {
		t.Fatalf("action = %q, want %q: should still prompt for approval", action.Action, engine.ActionAsk)
	}
}

func TestRuleLearningDriver_Submit_Revise_IncreasesAttemptCount(t *testing.T) {
	root := t.TempDir()
	seedRuleLearningSpec(t, root, "s", tasksWithLearnings())
	ctx := buildRuleLearningCtx(t, root, "s")

	if err := ctx.SetAnswer("rule_proposal", validProposalJSON); err != nil {
		t.Fatalf("SetAnswer: %v", err)
	}
	_, _, err := RuleLearningDriver().Submit(ctx, &engine.PhaseDef{ID: "rule-learning"}, []byte(`{"planFeedback":"first revision"}`))
	if err != nil {
		t.Fatalf("Submit revise #1: %v", err)
	}

	_, _, err = RuleLearningDriver().Submit(ctx, &engine.PhaseDef{ID: "rule-learning"}, []byte(validProposalJSON))
	if err != nil {
		t.Fatalf("Submit re-propose #1: %v", err)
	}
	if err := ctx.SetAnswer("rule_proposal", validProposalJSON); err != nil {
		t.Fatalf("SetAnswer proposal #2: %v", err)
	}

	_, _, err = RuleLearningDriver().Submit(ctx, &engine.PhaseDef{ID: "rule-learning"}, []byte(`{"planFeedback":"second revision"}`))
	if err != nil {
		t.Fatalf("Submit revise #2: %v", err)
	}

	action, phaseDone := RuleLearningDriver().Next(ctx, &engine.PhaseDef{ID: "rule-learning"})
	if phaseDone {
		t.Fatal("phaseDone must be false after second revise")
	}
	if !strings.Contains(action.Instruction, "second revision") {
		t.Error("instruction must contain latest priorFeedback after multiple revisions")
	}
	if strings.Contains(action.Instruction, "first revision") && !strings.Contains(action.Instruction, "second revision") {
		t.Error("stale feedback from first revision must not replace latest feedback")
	}
}

func TestRuleLearning_ApplyStep_CarriesSubmitContract(t *testing.T) {
	root := t.TempDir()
	slug := "rule-apply-contract"
	seedRuleLearningSpec(t, root, slug, tasksWithLearnings())
	ctx := buildRuleLearningCtx(t, root, slug)

	proposal := `{"rules":[{"scope":"executor","name":"single-prefix","content":"one prefix","rationale":"why"}]}`
	if err := ctx.SetAnswer("rule_proposal", proposal); err != nil {
		t.Fatalf("SetAnswer rule_proposal: %v", err)
	}
	if err := ctx.SetAnswer("rule_approved", "true"); err != nil {
		t.Fatalf("SetAnswer rule_approved: %v", err)
	}
	ctx = buildRuleLearningCtx(t, root, slug)

	action, done := RuleLearningDriver().Next(ctx, nil)
	if done {
		t.Fatal("apply step must still be pending")
	}
	if action.ExpectedInput.SubmitCmd == "" {
		t.Fatalf("apply step must tell the caller how to submit, got %+v", action.ExpectedInput)
	}
	if !strings.Contains(action.ExpectedInput.SubmitCmd, slug) {
		t.Fatalf("submitCmd must target this spec, got %q", action.ExpectedInput.SubmitCmd)
	}
	if action.ExpectedInput.Example == "" {
		t.Fatalf("apply step must carry an answer example, got %+v", action.ExpectedInput)
	}
}

func TestRuleLearning_ProposalPrompt_OmitsEmptyLearningHeaders(t *testing.T) {
	root := t.TempDir()
	slug := "rule-empty-headers"
	tasks := []spec.Task{{
		ID:            "task-1",
		Title:         "one",
		RefactorNotes: []spec.RefactorNote{{File: "a.go", Suggestion: "extract constant"}},
	}}
	seedRuleLearningSpec(t, root, slug, tasks)
	ctx := buildRuleLearningCtx(t, root, slug)

	action, done := RuleLearningDriver().Next(ctx, nil)
	if done {
		t.Fatal("rule learning must be pending with learnings present")
	}
	if !strings.Contains(action.Instruction, "extract constant") {
		t.Fatalf("prompt must carry the refactor note, got:\n%s", action.Instruction)
	}
	if strings.Contains(action.Instruction, "Failed AC reasons:") {
		t.Fatalf("empty learning header must be omitted, got:\n%s", action.Instruction)
	}
}

func TestRuleLearning_ProposeAction_CarriesExpectedInput(t *testing.T) {
	root := t.TempDir()
	slug := "propose-input"
	seedRuleLearningSpec(t, root, slug, tasksWithLearnings())

	ctx := buildRuleLearningCtx(t, root, slug)
	action, done := (&ruleLearningDriver{}).Next(ctx, nil)
	if done {
		t.Fatal("expected the propose step, phase reported done")
	}
	if action.ExpectedInput.Format != engine.FormatJSON {
		t.Fatalf("propose ExpectedInput.Format = %q, want json", action.ExpectedInput.Format)
	}
	if !strings.Contains(action.ExpectedInput.Example, `"rules"`) {
		t.Fatalf("propose example must show the rules array, got %q", action.ExpectedInput.Example)
	}
	if !strings.Contains(action.Instruction, `"rules"`) {
		t.Fatalf("propose instruction must state the return shape, got %q", action.Instruction)
	}
}

func TestRuleLearning_ReviewAction_RendersRulesReadably(t *testing.T) {
	root := t.TempDir()
	slug := "review-render"
	seedRuleLearningSpec(t, root, slug, tasksWithLearnings())

	ctx := buildRuleLearningCtx(t, root, slug)
	d := &ruleLearningDriver{}
	proposal := `{"rules":[{"scope":"executor","name":"no-globals","content":"Never use global variables.","rationale":"state leaks"}]}`
	if _, _, err := d.Submit(ctx, nil, []byte(proposal)); err != nil {
		t.Fatalf("propose Submit: %v", err)
	}

	action, _ := d.Next(ctx, nil)
	for _, want := range []string{"no-globals", "scope: executor", "Never use global variables.", "state leaks"} {
		if !strings.Contains(action.Instruction, want) {
			t.Fatalf("review instruction missing %q:\n%s", want, action.Instruction)
		}
	}
	if strings.Contains(action.Instruction, `{"rules":`) {
		t.Fatalf("review instruction should not dump raw JSON:\n%s", action.Instruction)
	}
	if action.ExpectedInput.Format == "" {
		t.Fatal("review action must declare an input format")
	}
}
