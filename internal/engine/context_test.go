package engine

import (
	"os"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func makeAskStep(id StepID, instruction string) StepDef {
	return StepDef{
		ID: id,
		Prompt: func(c *Context) Action {
			return Action{Action: ActionAsk, Instruction: instruction}
		},
	}
}

func makeOneStepPhase(phaseID PhaseID, stepInstruction string) PhaseDef {
	step := makeAskStep("step-1", stepInstruction)
	driver := &StepTableDriver{
		Modules: []ModuleDef{{ID: "mod-1", Steps: []StepDef{step}}},
	}
	return PhaseDef{ID: phaseID, Driver: driver}
}

func seedSpec(t *testing.T, root, slug, phase string) {
	t.Helper()
	st := spec.State{
		Version: 1,
		Slug:    slug,
		Phase:   phase,
		Answers: map[string][]spec.Answer{},
	}
	if err := spec.SaveState(root, slug, st); err != nil {
		t.Fatalf("seedSpec SaveState: %v", err)
	}
	pr := spec.Progress{
		Spec:   slug,
		Status: spec.StatusDraft,
		Tasks:  []spec.Task{},
	}
	if err := spec.SaveProgress(root, slug, pr); err != nil {
		t.Fatalf("seedSpec SaveProgress: %v", err)
	}
	if err := spec.SaveSettings(root, slug, spec.DefaultSettings()); err != nil {
		t.Fatalf("seedSpec SaveSettings: %v", err)
	}
}

func TestBuild_ErrorWhenSpecMissing(t *testing.T) {
	root := t.TempDir()
	defs := []PhaseDef{makeOneStepPhase("phase-a", "hello")}

	_, err := Build(root, "nope", defs)
	if err == nil {
		t.Fatalf("expected non-nil error when spec does not exist, got nil")
	}
}

func TestBuild_Success(t *testing.T) {
	root := t.TempDir()
	slug := "my-spec"
	seedSpec(t, root, slug, spec.PhaseInitial)

	defs := []PhaseDef{makeOneStepPhase(PhaseID(spec.PhaseInitial), "first question")}

	ctx, err := Build(root, slug, defs)
	if err != nil {
		t.Fatalf("Build returned unexpected error: %v", err)
	}
	if ctx == nil {
		t.Fatalf("Build returned nil context")
	}
	if ctx.Phase() != PhaseID(spec.PhaseInitial) {
		t.Fatalf("expected phase %q, got %q", spec.PhaseInitial, ctx.Phase())
	}
}

func TestNext_DelegatesToActivePhaseDriver(t *testing.T) {
	root := t.TempDir()
	slug := "delegate-spec"
	wantInstruction := "what is the target scope?"
	phaseID := PhaseID("phase-alpha")

	seedSpec(t, root, slug, string(phaseID))

	defs := []PhaseDef{makeOneStepPhase(phaseID, wantInstruction)}
	ctx, err := Build(root, slug, defs)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next returned error: %v", err)
	}
	if action.Action != ActionAsk {
		t.Fatalf("expected ActionAsk, got %q", action.Action)
	}
	if action.Instruction != wantInstruction {
		t.Fatalf("expected instruction %q, got %q", wantInstruction, action.Instruction)
	}
}

func TestSubmit_AdvancesPhaseWhenDriverReportsDone(t *testing.T) {
	root := t.TempDir()
	slug := "advance-spec"
	phaseA := PhaseID("phase-a")
	phaseB := PhaseID("phase-b")

	seedSpec(t, root, slug, string(phaseA))

	defs := []PhaseDef{
		makeOneStepPhase(phaseA, "question a"),
		makeOneStepPhase(phaseB, "question b"),
	}
	ctx, err := Build(root, slug, defs)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	_, err = ctx.Submit([]byte(`"answer"`))
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}

	if ctx.Phase() != phaseB {
		t.Fatalf("expected phase %q after advance, got %q", phaseB, ctx.Phase())
	}

	persisted, err := spec.LoadState(root, slug)
	if err != nil {
		t.Fatalf("LoadState after Submit: %v", err)
	}
	if persisted.Phase != string(phaseB) {
		t.Fatalf("persisted phase: expected %q, got %q", phaseB, persisted.Phase)
	}
}

func TestSubmit_AtLastPhase_BecomesPhaseComplete(t *testing.T) {
	root := t.TempDir()
	slug := "terminal-spec"
	phaseOnly := PhaseID("phase-only")

	seedSpec(t, root, slug, string(phaseOnly))

	defs := []PhaseDef{makeOneStepPhase(phaseOnly, "last question")}
	ctx, err := Build(root, slug, defs)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	_, err = ctx.Submit([]byte(`"done"`))
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}

	if ctx.Phase() != PhaseComplete {
		t.Fatalf("expected PhaseComplete %q, got %q", PhaseComplete, ctx.Phase())
	}

	persisted, err := spec.LoadState(root, slug)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if persisted.Phase != string(PhaseComplete) {
		t.Fatalf("persisted phase: expected %q, got %q", PhaseComplete, persisted.Phase)
	}
}

func TestBuild_ToleratesProgressWithNilExecution(t *testing.T) {
	root := t.TempDir()
	slug := "nil-exec-spec"

	st := spec.State{
		Version: 1,
		Slug:    slug,
		Phase:   spec.PhaseInitial,
		Answers: map[string][]spec.Answer{},
	}
	if err := spec.SaveState(root, slug, st); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	oldJSON := []byte(`{
  "spec": "nil-exec-spec",
  "status": "draft",
  "tasks": [],
  "updatedAt": "2024-01-01T00:00:00Z"
}`)
	dir := paths.SpecDir(root, slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(paths.SpecProgress(root, slug), oldJSON, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := spec.SaveSettings(root, slug, spec.DefaultSettings()); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	defs := []PhaseDef{makeOneStepPhase(PhaseID(spec.PhaseInitial), "q")}
	ctx, err := Build(root, slug, defs)
	if err != nil {
		t.Fatalf("Build with nil Execution progress: unexpected error: %v", err)
	}
	if ctx == nil {
		t.Fatalf("Build with nil Execution progress: returned nil context")
	}
}

func TestNext_TerminalAction_CarriesACompletionMessage(t *testing.T) {
	root := t.TempDir()
	slug := "done-spec"
	seedSpec(t, root, slug, "phase-a")
	if err := spec.SaveProgress(root, slug, spec.Progress{
		Spec:   slug,
		Status: spec.StatusCompleted,
		Tasks:  []spec.Task{{ID: "task-1", Done: true}, {ID: "task-2", Done: true}},
	}); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}

	// No phase def matches the stored phase, so the context is terminal.
	ctx, err := Build(root, slug, []PhaseDef{makeOneStepPhase("other-phase", "hi")})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if action.Action != ActionTerminal {
		t.Fatalf("expected terminal action, got %q", action.Action)
	}
	if action.Instruction == "" {
		t.Fatal("a terminal action must tell the user the spec is finished")
	}
	if !strings.Contains(action.Instruction, slug) || !strings.Contains(action.Instruction, "2/2") {
		t.Fatalf("terminal message must name the spec and its task count, got %q", action.Instruction)
	}
}

func TestNext_FillsSubmitCmdOnAsk(t *testing.T) {
	root := t.TempDir()
	seedSpec(t, root, "sc-ask", "p1")
	defs := []PhaseDef{makeOneStepPhase("p1", "answer me")}

	c, err := Build(root, "sc-ask", defs)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	action, err := c.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	want := "tddmaster next sc-ask --answer='<text>'"
	if action.ExpectedInput.SubmitCmd != want {
		t.Fatalf("SubmitCmd = %q, want %q", action.ExpectedInput.SubmitCmd, want)
	}
}

func TestWithSubmitCmd_PerActionAndTaskShapes(t *testing.T) {
	c := &Context{slug: "sl", command: "tddmaster"}

	notify := c.withSubmitCmd(Action{Action: ActionNotify})
	if notify.ExpectedInput.SubmitCmd != "tddmaster next sl" {
		t.Fatalf("notify SubmitCmd = %q", notify.ExpectedInput.SubmitCmd)
	}

	instruct := c.withSubmitCmd(Action{
		Action:        ActionInstruct,
		ExpectedInput: ExpectedInput{Format: FormatJSON},
		Tasks: []TaskAction{
			{TaskID: "task-1", ExpectedInput: ExpectedInput{Format: FormatJSON}},
			{TaskID: "task-2", ExpectedInput: ExpectedInput{Format: FormatFlag, SubmitCmd: "keep-me"}},
		},
	})
	if instruct.ExpectedInput.SubmitCmd != "tddmaster next sl --answer='<json>'" {
		t.Fatalf("instruct SubmitCmd = %q", instruct.ExpectedInput.SubmitCmd)
	}
	if instruct.Tasks[0].ExpectedInput.SubmitCmd != "tddmaster next sl --answer='<json>'" {
		t.Fatalf("task-1 SubmitCmd = %q", instruct.Tasks[0].ExpectedInput.SubmitCmd)
	}
	if instruct.Tasks[1].ExpectedInput.SubmitCmd != "keep-me" {
		t.Fatalf("an explicit task SubmitCmd must be preserved, got %q", instruct.Tasks[1].ExpectedInput.SubmitCmd)
	}

	terminal := c.withSubmitCmd(Action{Action: ActionTerminal})
	if terminal.ExpectedInput.SubmitCmd != "" {
		t.Fatalf("terminal must not carry a SubmitCmd, got %q", terminal.ExpectedInput.SubmitCmd)
	}
}

func TestRewindPhase_MovesSpecBackwards(t *testing.T) {
	root := t.TempDir()
	seedSpec(t, root, "sc-rewind", "p2")
	defs := []PhaseDef{makeOneStepPhase("p1", "one"), makeOneStepPhase("p2", "two")}

	c, err := Build(root, "sc-rewind", defs)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if err := c.RewindPhase("p1"); err != nil {
		t.Fatalf("RewindPhase: %v", err)
	}
	if c.Phase() != "p1" {
		t.Fatalf("in-memory phase = %q, want p1", c.Phase())
	}
	persisted, err := spec.LoadState(root, "sc-rewind")
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if persisted.Phase != "p1" {
		t.Fatalf("persisted phase = %q, want p1", persisted.Phase)
	}
}
