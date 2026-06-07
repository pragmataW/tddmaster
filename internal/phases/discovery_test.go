package phases

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/promptregistry"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func writeDiscoveryManifest(t *testing.T, root string) {
	t.Helper()
	dir := paths.Tddmaster(root)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	payload := `{"selectedTools":["claude-code"],"maxIterationBeforeStart":15,"command":"tddmaster"}`
	if err := os.WriteFile(paths.Manifest(root), []byte(payload), 0o644); err != nil {
		t.Fatalf("WriteFile manifest: %v", err)
	}
}

func seedDiscoverySpec(t *testing.T, root, slug string) {
	t.Helper()
	writeDiscoveryManifest(t, root)
	now := time.Now().UTC()
	state := spec.State{
		Version:   1,
		Slug:      slug,
		Phase:     "discovery",
		Answers:   map[string][]spec.Answer{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := spec.SaveState(root, slug, state); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	if err := spec.SaveSettings(root, slug, spec.DefaultSettings()); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	p := spec.Progress{Spec: slug, Status: "draft", Tasks: []spec.Task{}}
	if err := spec.SaveProgress(root, slug, p); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
}

func buildDiscoveryCtx(t *testing.T, root, slug string) *engine.Context {
	t.Helper()
	defs := []engine.PhaseDef{{ID: "discovery", Driver: DiscoveryDriver()}}
	ctx, err := engine.Build(root, slug, defs)
	if err != nil {
		t.Fatalf("engine.Build: %v", err)
	}
	return ctx
}

func TestDiscoverySteps_ExactlyEleven(t *testing.T) {
	steps := DiscoverySteps()
	if len(steps) != 11 {
		t.Fatalf("DiscoverySteps() returned %d steps, want 11", len(steps))
	}
}

func TestDiscoverySteps_IDsInOrder(t *testing.T) {
	steps := DiscoverySteps()
	wantIDs := []engine.StepID{
		"step-listen-first",
		"step-mode-selection",
		"step-premise-challenge",
		"step-q-status_quo",
		"step-q-ambition",
		"step-q-reversibility",
		"step-q-user_impact",
		"step-q-verification",
		"step-q-scope_boundary",
		"step-q-edge_cases",
		"step-synthesis",
	}
	for i, s := range steps {
		if s.ID != wantIDs[i] {
			t.Errorf("step[%d].ID = %q, want %q", i, s.ID, wantIDs[i])
		}
	}
}

func TestDiscoverySteps_KeysInOrder(t *testing.T) {
	steps := DiscoverySteps()
	wantKeys := []string{
		"listen_context",
		"mode",
		"premises",
		"status_quo",
		"ambition",
		"reversibility",
		"user_impact",
		"verification",
		"scope_boundary",
		"edge_cases",
		"synthesis",
	}
	for i, s := range steps {
		if s.Key != wantKeys[i] {
			t.Errorf("step[%d].Key = %q, want %q", i, s.Key, wantKeys[i])
		}
	}
}

func TestDiscoverySteps_ListenFirst_ActionIsAsk(t *testing.T) {
	steps := DiscoverySteps()
	action := steps[0].Prompt("")
	if action.Action != engine.ActionAsk {
		t.Errorf("listen-first action = %q, want %q", action.Action, engine.ActionAsk)
	}
}

func TestDiscoverySteps_ListenFirst_InstructionMatchesRegistry(t *testing.T) {
	steps := DiscoverySteps()
	action := steps[0].Prompt("")
	want, ok := promptregistry.Instruction(promptregistry.KeyListenFirst)
	if !ok {
		t.Fatal("promptregistry.Instruction(KeyListenFirst) not found")
	}
	if action.Instruction != want {
		t.Errorf("listen-first instruction = %q, want %q", action.Instruction, want)
	}
}

func TestDiscoverySteps_ListenFirst_FormatIsText(t *testing.T) {
	steps := DiscoverySteps()
	action := steps[0].Prompt("")
	if action.ExpectedInput.Format != engine.FormatText {
		t.Errorf("listen-first format = %q, want %q", action.ExpectedInput.Format, engine.FormatText)
	}
}

func TestDiscoverySteps_ModeSelection_InstructionMatchesRegistry(t *testing.T) {
	steps := DiscoverySteps()
	action := steps[1].Prompt("")
	want, ok := promptregistry.Instruction(promptregistry.KeyModeSelection)
	if !ok {
		t.Fatal("promptregistry.Instruction(KeyModeSelection) not found")
	}
	if action.Instruction != want {
		t.Errorf("mode-selection instruction = %q, want %q", action.Instruction, want)
	}
}

func TestDiscoverySteps_ModeSelection_FiveInteractiveOptions(t *testing.T) {
	steps := DiscoverySteps()
	action := steps[1].Prompt("")
	if len(action.InteractiveOptions) != 5 {
		t.Fatalf("mode-selection interactive options count = %d, want 5", len(action.InteractiveOptions))
	}
}

func TestDiscoverySteps_ModeSelection_InteractiveOptionsMatchModeOptions(t *testing.T) {
	steps := DiscoverySteps()
	action := steps[1].Prompt("")
	for i, opt := range action.InteractiveOptions {
		want := promptregistry.ModeOptions[i]
		if opt.Label != want.Label {
			t.Errorf("option[%d].Label = %q, want %q", i, opt.Label, want.Label)
		}
		if opt.Description != want.Description {
			t.Errorf("option[%d].Description = %q, want %q", i, opt.Description, want.Description)
		}
	}
}

func TestDiscoverySteps_ModeSelection_CommandMapNonNilAndMapsLabelsToIDs(t *testing.T) {
	steps := DiscoverySteps()
	action := steps[1].Prompt("")
	if action.CommandMap == nil {
		t.Fatal("mode-selection CommandMap is nil")
	}
	for _, opt := range promptregistry.ModeOptions {
		id, ok := action.CommandMap[opt.Label]
		if !ok {
			t.Errorf("CommandMap missing key %q", opt.Label)
			continue
		}
		if id != opt.ID {
			t.Errorf("CommandMap[%q] = %q, want %q", opt.Label, id, opt.ID)
		}
	}
}

func TestDiscoverySteps_ModeSelection_FormatIsText(t *testing.T) {
	steps := DiscoverySteps()
	action := steps[1].Prompt("")
	if action.ExpectedInput.Format != engine.FormatText {
		t.Errorf("mode-selection format = %q, want %q", action.ExpectedInput.Format, engine.FormatText)
	}
}

func TestDiscoverySteps_PremiseChallenge_InstructionMatchesRegistry(t *testing.T) {
	steps := DiscoverySteps()
	action := steps[2].Prompt("")
	want, ok := promptregistry.Instruction(promptregistry.KeyPremiseChallenge)
	if !ok {
		t.Fatal("promptregistry.Instruction(KeyPremiseChallenge) not found")
	}
	if action.Instruction != want {
		t.Errorf("premise-challenge instruction = %q, want %q", action.Instruction, want)
	}
}

func TestDiscoverySteps_PremiseChallenge_FormatIsJSON(t *testing.T) {
	steps := DiscoverySteps()
	action := steps[2].Prompt("")
	if action.ExpectedInput.Format != engine.FormatJSON {
		t.Errorf("premise-challenge format = %q, want %q", action.ExpectedInput.Format, engine.FormatJSON)
	}
}

func TestDiscoverySteps_PremiseChallenge_ExampleContainsPremises(t *testing.T) {
	steps := DiscoverySteps()
	action := steps[2].Prompt("")
	if !strings.Contains(action.ExpectedInput.Example, "premises") {
		t.Errorf("premise-challenge example = %q, want it to contain 'premises'", action.ExpectedInput.Example)
	}
}

func TestDiscoverySteps_QuestionSteps_InstructionContainsQuestionText(t *testing.T) {
	steps := DiscoverySteps()
	questionSteps := steps[3:10]
	questionKeys := []string{
		"status_quo", "ambition", "reversibility",
		"user_impact", "verification", "scope_boundary", "edge_cases",
	}
	for i, s := range questionSteps {
		key := questionKeys[i]
		action := s.Prompt("")
		want, ok := promptregistry.Instruction(promptregistry.KeyDiscoveryQuestion(key))
		if !ok {
			t.Fatalf("Instruction(%q) not found", key)
		}
		if !strings.Contains(action.Instruction, want) {
			t.Errorf("step %q instruction does not contain question text %q", s.ID, want)
		}
	}
}

func TestDiscoverySteps_QuestionSteps_InstructionContainsAskWithSuggestionsDirective(t *testing.T) {
	steps := DiscoverySteps()
	for _, s := range steps[3:10] {
		action := s.Prompt("")
		if !strings.Contains(action.Instruction, promptregistry.AskWithSuggestionsDirective) {
			t.Errorf("question step %q instruction does not contain AskWithSuggestionsDirective", s.ID)
		}
		if !strings.Contains(action.Instruction, "AskUserQuestion") {
			t.Errorf("question step %q instruction does not mention AskUserQuestion", s.ID)
		}
	}
}

func TestDiscoverySteps_QuestionSteps_FormatIsText(t *testing.T) {
	steps := DiscoverySteps()
	for _, s := range steps[3:10] {
		action := s.Prompt("")
		if action.ExpectedInput.Format != engine.FormatText {
			t.Errorf("question step %q format = %q, want text", s.ID, action.ExpectedInput.Format)
		}
	}
}

func TestDiscoverySteps_QuestionStep_ModeValidate_InstructionContainsModeRules(t *testing.T) {
	steps := DiscoverySteps()
	action := steps[3].Prompt("validate")
	rules := promptregistry.ModeRules("validate")
	for _, rule := range rules {
		if !strings.Contains(action.Instruction, rule) {
			t.Errorf("status_quo prompt with mode=validate does not contain rule: %q", rule)
		}
	}
}

func TestDiscoverySteps_QuestionStep_NoMode_InstructionDoesNotContainValidateRules(t *testing.T) {
	steps := DiscoverySteps()
	action := steps[3].Prompt("")
	rules := promptregistry.ModeRules("validate")
	if len(rules) == 0 {
		t.Skip("no validate rules to check")
	}
	if strings.Contains(action.Instruction, rules[0]) {
		t.Errorf("status_quo prompt with empty mode must not contain validate rule: %q", rules[0])
	}
}

func TestDiscoverySteps_VerificationStep_InstructionContainsBuiltInExtras(t *testing.T) {
	steps := DiscoverySteps()
	verificationIdx := 7
	action := steps[verificationIdx].Prompt("")
	for _, extra := range promptregistry.BuiltInExtras {
		if !strings.Contains(action.Instruction, extra) {
			t.Errorf("verification step instruction does not contain built-in extra: %q", extra)
		}
	}
}

func TestDiscoverySteps_Synthesis_InstructionNonEmpty(t *testing.T) {
	steps := DiscoverySteps()
	action := steps[10].Prompt("")
	if strings.TrimSpace(action.Instruction) == "" {
		t.Error("synthesis instruction is empty")
	}
}

func TestDiscoverySteps_Synthesis_FormatIsFlag(t *testing.T) {
	steps := DiscoverySteps()
	action := steps[10].Prompt("")
	if action.ExpectedInput.Format != engine.FormatFlag {
		t.Errorf("synthesis format = %q, want %q", action.ExpectedInput.Format, engine.FormatFlag)
	}
}

func TestDiscoverySteps_Synthesis_ValidateApprove(t *testing.T) {
	steps := DiscoverySteps()
	s := steps[10]
	if s.Validate == nil {
		t.Fatal("synthesis Validate is nil")
	}
	if err := s.Validate([]byte("approve")); err != nil {
		t.Errorf("Validate(approve) = %v, want nil", err)
	}
}

func TestDiscoverySteps_Synthesis_ValidateApproveWithNewline(t *testing.T) {
	steps := DiscoverySteps()
	s := steps[10]
	if s.Validate == nil {
		t.Fatal("synthesis Validate is nil")
	}
	if err := s.Validate([]byte("approve\n")); err != nil {
		t.Errorf("Validate(approve\\n) = %v, want nil", err)
	}
}

func TestDiscoverySteps_Synthesis_ValidateRejectsNope(t *testing.T) {
	steps := DiscoverySteps()
	s := steps[10]
	if s.Validate == nil {
		t.Fatal("synthesis Validate is nil")
	}
	if err := s.Validate([]byte("nope")); err == nil {
		t.Error("Validate(nope) = nil, want non-nil error")
	}
}

func TestDiscoverySteps_Synthesis_ValidateRejectsEmpty(t *testing.T) {
	steps := DiscoverySteps()
	s := steps[10]
	if s.Validate == nil {
		t.Fatal("synthesis Validate is nil")
	}
	if err := s.Validate([]byte("")); err == nil {
		t.Error("Validate('') = nil, want non-nil error")
	}
}

func TestDiscoveryDriver_FreshSpec_NextReturnsListenFirstPrompt(t *testing.T) {
	root := t.TempDir()
	slug := "disc-slug"
	seedDiscoverySpec(t, root, slug)
	ctx := buildDiscoveryCtx(t, root, slug)

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}
	if action.Action != engine.ActionAsk {
		t.Errorf("action.Action = %q, want %q", action.Action, engine.ActionAsk)
	}
	want, _ := promptregistry.Instruction(promptregistry.KeyListenFirst)
	if action.Instruction != want {
		t.Errorf("action.Instruction = %q, want listen-first text", action.Instruction)
	}
}

func TestDiscoveryDriver_FreshSpec_NextPhaseDoneIsFalse(t *testing.T) {
	root := t.TempDir()
	slug := "disc-slug"
	seedDiscoverySpec(t, root, slug)
	ctx := buildDiscoveryCtx(t, root, slug)

	_, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}
}

func TestDiscoveryDriver_SubmitListenFirst_RecordsAnswer(t *testing.T) {
	root := t.TempDir()
	slug := "disc-slug"
	seedDiscoverySpec(t, root, slug)
	ctx := buildDiscoveryCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next: %v", err)
	}
	if _, err := ctx.Submit([]byte("user context here")); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	ctx2 := buildDiscoveryCtx(t, root, slug)
	if got := ctx2.AnswerValue("listen_context"); got != "user context here" {
		t.Errorf("AnswerValue(listen_context) = %q, want %q", got, "user context here")
	}
}

func TestDiscoveryDriver_SubmitListenFirst_ReturnsModeSelectionAction(t *testing.T) {
	root := t.TempDir()
	slug := "disc-slug"
	seedDiscoverySpec(t, root, slug)
	ctx := buildDiscoveryCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next: %v", err)
	}
	action, err := ctx.Submit([]byte("some context"))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if action.Action == "" {
		t.Fatal("Submit returned empty action, want next step prompt")
	}
	want, _ := promptregistry.Instruction(promptregistry.KeyModeSelection)
	if action.Instruction != want {
		t.Errorf("Submit action instruction = %q, want mode-selection text", action.Instruction)
	}
}

func TestDiscoveryDriver_AfterListenFirst_NextReturnsModeSelection(t *testing.T) {
	root := t.TempDir()
	slug := "disc-slug"
	seedDiscoverySpec(t, root, slug)
	ctx := buildDiscoveryCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("first Next: %v", err)
	}
	if _, err := ctx.Submit([]byte("some context")); err != nil {
		t.Fatalf("Submit listen_context: %v", err)
	}

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("second Next: %v", err)
	}
	want, _ := promptregistry.Instruction(promptregistry.KeyModeSelection)
	if action.Instruction != want {
		t.Errorf("second Next instruction = %q, want mode-selection text", action.Instruction)
	}
}

func TestDiscoveryDriver_AllAnswered_NextReturnsPhaseDone(t *testing.T) {
	root := t.TempDir()
	slug := "disc-slug"
	seedDiscoverySpec(t, root, slug)
	ctx := buildDiscoveryCtx(t, root, slug)

	answers := [][]byte{
		[]byte("user context"),
		[]byte("full"),
		[]byte(`{"premises":[{"text":"premise1","agreed":true,"revision":""}]}`),
		[]byte("status quo answer"),
		[]byte("ambition answer"),
		[]byte("reversibility answer"),
		[]byte("user impact answer"),
		[]byte("verification answer"),
		[]byte("scope boundary answer"),
		[]byte("edge cases answer"),
		[]byte("approve"),
	}

	for i, ans := range answers {
		if _, err := ctx.Next(); err != nil {
			t.Fatalf("Next before step %d: %v", i, err)
		}
		action, err := ctx.Submit(ans)
		if err != nil {
			t.Fatalf("Submit step %d: %v", i, err)
		}
		if i < len(answers)-1 {
			_ = action
		}
	}

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next after all answered: %v", err)
	}
	if action.Action != "" {
		t.Errorf("after all answered, Next action = %q, want empty (phase done)", action.Action)
	}
}

func TestDiscoveryDriver_NonfinalSubmit_PhaseDoneIsFalse(t *testing.T) {
	root := t.TempDir()
	slug := "disc-slug"
	seedDiscoverySpec(t, root, slug)
	ctx := buildDiscoveryCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next: %v", err)
	}
	if _, err := ctx.Submit([]byte("user context")); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	state, err := spec.LoadState(root, slug)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if state.Phase != "discovery" {
		t.Errorf("phase after non-final submit = %q, want %q", state.Phase, "discovery")
	}
}

func TestDiscoveryDriver_ValidateModeAfterModeSelection_QuestionContainsRules(t *testing.T) {
	root := t.TempDir()
	slug := "disc-slug"
	seedDiscoverySpec(t, root, slug)
	ctx := buildDiscoveryCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next listen-first: %v", err)
	}
	if _, err := ctx.Submit([]byte("some context")); err != nil {
		t.Fatalf("Submit listen_context: %v", err)
	}

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next mode-selection: %v", err)
	}
	if _, err := ctx.Submit([]byte("validate")); err != nil {
		t.Fatalf("Submit mode: %v", err)
	}

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next premise-challenge: %v", err)
	}
	if _, err := ctx.Submit([]byte(`{"premises":[{"text":"p","agreed":true,"revision":""}]}`)); err != nil {
		t.Fatalf("Submit premises: %v", err)
	}

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next question step: %v", err)
	}

	rules := promptregistry.ModeRules("validate")
	if len(rules) == 0 {
		t.Fatal("ModeRules('validate') is empty")
	}
	for _, rule := range rules {
		if !strings.Contains(action.Instruction, rule) {
			t.Errorf("question prompt after validate mode does not contain rule: %q", rule)
		}
	}
}

func TestDiscoveryDriver_SynthesisSubmitNope_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "disc-slug"
	seedDiscoverySpec(t, root, slug)
	ctx := buildDiscoveryCtx(t, root, slug)

	answers := [][]byte{
		[]byte("user context"),
		[]byte("full"),
		[]byte(`{"premises":[{"text":"premise1","agreed":true,"revision":""}]}`),
		[]byte("status quo answer"),
		[]byte("ambition answer"),
		[]byte("reversibility answer"),
		[]byte("user impact answer"),
		[]byte("verification answer"),
		[]byte("scope boundary answer"),
		[]byte("edge cases answer"),
	}

	for i, ans := range answers {
		if _, err := ctx.Next(); err != nil {
			t.Fatalf("Next before step %d: %v", i, err)
		}
		if _, err := ctx.Submit(ans); err != nil {
			t.Fatalf("Submit step %d: %v", i, err)
		}
	}

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next synthesis: %v", err)
	}
	_, err := ctx.Submit([]byte("nope"))
	if err == nil {
		t.Fatal("Submit(nope) synthesis = nil, want non-nil error")
	}
}

func TestDiscoveryDriver_SynthesisSubmitNope_PhaseNotDone(t *testing.T) {
	root := t.TempDir()
	slug := "disc-slug"
	seedDiscoverySpec(t, root, slug)
	ctx := buildDiscoveryCtx(t, root, slug)

	answers := [][]byte{
		[]byte("user context"),
		[]byte("full"),
		[]byte(`{"premises":[{"text":"premise1","agreed":true,"revision":""}]}`),
		[]byte("status quo answer"),
		[]byte("ambition answer"),
		[]byte("reversibility answer"),
		[]byte("user impact answer"),
		[]byte("verification answer"),
		[]byte("scope boundary answer"),
		[]byte("edge cases answer"),
	}

	for i, ans := range answers {
		if _, err := ctx.Next(); err != nil {
			t.Fatalf("Next before step %d: %v", i, err)
		}
		if _, err := ctx.Submit(ans); err != nil {
			t.Fatalf("Submit step %d: %v", i, err)
		}
	}

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next synthesis: %v", err)
	}
	ctx.Submit([]byte("nope"))

	state, err := spec.LoadState(root, slug)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if state.Phase != "discovery" {
		t.Errorf("phase after nope submit = %q, want %q", state.Phase, "discovery")
	}
}

func TestDiscoveryDriver_SynthesisSubmitNope_AnswerNotRecorded(t *testing.T) {
	root := t.TempDir()
	slug := "disc-slug"
	seedDiscoverySpec(t, root, slug)
	ctx := buildDiscoveryCtx(t, root, slug)

	answers := [][]byte{
		[]byte("user context"),
		[]byte("full"),
		[]byte(`{"premises":[{"text":"premise1","agreed":true,"revision":""}]}`),
		[]byte("status quo answer"),
		[]byte("ambition answer"),
		[]byte("reversibility answer"),
		[]byte("user impact answer"),
		[]byte("verification answer"),
		[]byte("scope boundary answer"),
		[]byte("edge cases answer"),
	}

	for i, ans := range answers {
		if _, err := ctx.Next(); err != nil {
			t.Fatalf("Next before step %d: %v", i, err)
		}
		if _, err := ctx.Submit(ans); err != nil {
			t.Fatalf("Submit step %d: %v", i, err)
		}
	}

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next synthesis: %v", err)
	}
	ctx.Submit([]byte("nope"))

	ctx2 := buildDiscoveryCtx(t, root, slug)
	if ctx2.HasAnswer("synthesis") {
		t.Error("synthesis answer recorded after nope submit, want not recorded")
	}
}

func TestDiscoveryDriver_AllAnswered_AllKeysPresent(t *testing.T) {
	root := t.TempDir()
	slug := "disc-slug"
	seedDiscoverySpec(t, root, slug)
	ctx := buildDiscoveryCtx(t, root, slug)

	answers := [][]byte{
		[]byte("user context"),
		[]byte("validate"),
		[]byte(`{"premises":[{"text":"premise1","agreed":true,"revision":""}]}`),
		[]byte("status quo answer"),
		[]byte("ambition answer"),
		[]byte("reversibility answer"),
		[]byte("user impact answer"),
		[]byte("verification answer"),
		[]byte("scope boundary answer"),
		[]byte("edge cases answer"),
		[]byte("approve"),
	}

	for i, ans := range answers {
		if _, err := ctx.Next(); err != nil {
			t.Fatalf("Next before step %d: %v", i, err)
		}
		if _, err := ctx.Submit(ans); err != nil {
			t.Fatalf("Submit step %d: %v", i, err)
		}
	}

	ctx2 := buildDiscoveryCtx(t, root, slug)
	wantKeys := []string{
		"listen_context", "mode", "premises",
		"status_quo", "ambition", "reversibility",
		"user_impact", "verification", "scope_boundary",
		"edge_cases", "synthesis",
	}
	for _, key := range wantKeys {
		if !ctx2.HasAnswer(key) {
			t.Errorf("key %q not found in answers after full discovery flow", key)
		}
	}
}
