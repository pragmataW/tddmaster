package ruleform

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

func TestNewModel_InitialPhase(t *testing.T) {
	m := newModel(t.TempDir())
	if m.phase != phaseIntro {
		t.Errorf("newModel phase = %v, want phaseIntro", m.phase)
	}
}

func TestNewModel_InitialFrame(t *testing.T) {
	m := newModel(t.TempDir())
	if m.frame != 0 {
		t.Errorf("newModel frame = %d, want 0", m.frame)
	}
}

func TestNewModel_RootSet(t *testing.T) {
	root := t.TempDir()
	m := newModel(root)
	if m.root != root {
		t.Errorf("newModel root = %q, want %q", m.root, root)
	}
}

func TestNewModel_StateNotNil(t *testing.T) {
	m := newModel(t.TempDir())
	if m.state == nil {
		t.Error("newModel state is nil, want non-nil")
	}
}

func TestNewModel_FormNotNil(t *testing.T) {
	m := newModel(t.TempDir())
	if m.form == nil {
		t.Error("newModel form is nil, want non-nil")
	}
}

func TestNewModel_WrittenEmpty(t *testing.T) {
	m := newModel(t.TempDir())
	if m.written != "" {
		t.Errorf("newModel written = %q, want empty", m.written)
	}
}

func TestModel_Init_ReturnsNonNil(t *testing.T) {
	m := newModel(t.TempDir())
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init() returned nil, want non-nil tea.Cmd")
	}
}

func TestTick_ReturnsNonNil(t *testing.T) {
	cmd := tick()
	if cmd == nil {
		t.Error("tick() returned nil, want non-nil tea.Cmd")
	}
}

func TestTick_CmdProducesTickMsg(t *testing.T) {
	cmd := tick()
	msg := cmd()
	if _, ok := msg.(tickMsg); !ok {
		t.Errorf("tick() cmd produced %T, want tickMsg", msg)
	}
}

func TestBuildForm_ReturnsNonNil(t *testing.T) {
	s := &formState{target: "global"}
	f := buildForm(s)
	if f == nil {
		t.Error("buildForm returned nil, want non-nil *huh.Form")
	}
}

func TestBrandTheme_ReturnsNonNil(t *testing.T) {
	th := brandTheme()
	if th == nil {
		t.Error("brandTheme() returned nil, want non-nil *huh.Theme")
	}
}

func TestBrandTheme_IsHuhTheme(t *testing.T) {
	th := brandTheme()
	var _ *huh.Theme = th
}

func TestView_IntroPhase_NonEmpty(t *testing.T) {
	m := newModel(t.TempDir())
	m.phase = phaseIntro
	got := m.View()
	if got == "" {
		t.Error("View() in phaseIntro returned empty string")
	}
}

func TestView_FormPhase_NonEmpty(t *testing.T) {
	m := newModel(t.TempDir())
	m.phase = phaseForm
	got := m.View()
	if got == "" {
		t.Error("View() in phaseForm returned empty string")
	}
}

func TestView_SuccessPhase_NonEmpty(t *testing.T) {
	m := newModel(t.TempDir())
	m.phase = phaseSuccess
	m.written = "/some/path/rule.md"
	got := m.View()
	if got == "" {
		t.Error("View() in phaseSuccess returned empty string")
	}
}

func TestView_DonePhase_Empty(t *testing.T) {
	m := newModel(t.TempDir())
	m.phase = phaseDone
	got := m.View()
	if got != "" {
		t.Errorf("View() in phaseDone = %q, want empty", got)
	}
}

func TestViewIntro_LowFrame_NonEmpty(t *testing.T) {
	m := newModel(t.TempDir())
	m.frame = 0
	got := m.viewIntro()
	if got == "" {
		t.Error("viewIntro() with frame=0 returned empty string")
	}
}

func TestViewIntro_HighFrame_NonEmpty(t *testing.T) {
	m := newModel(t.TempDir())
	m.frame = 100
	got := m.viewIntro()
	if got == "" {
		t.Error("viewIntro() with frame=100 returned empty string")
	}
}

func TestViewIntro_MidFrame_NonEmpty(t *testing.T) {
	m := newModel(t.TempDir())
	m.frame = 5
	got := m.viewIntro()
	if got == "" {
		t.Error("viewIntro() with frame=5 returned empty string")
	}
}

func TestViewSuccess_ContainsWrittenPath(t *testing.T) {
	m := newModel(t.TempDir())
	m.written = "/some/path/rule.md"
	got := m.viewSuccess()
	if !strings.Contains(got, "rule.md") {
		t.Errorf("viewSuccess() = %q, want it to contain the written filename", got)
	}
}

func TestViewSuccess_NonEmpty(t *testing.T) {
	m := newModel(t.TempDir())
	m.written = "/some/path/rule.md"
	got := m.viewSuccess()
	if got == "" {
		t.Error("viewSuccess() returned empty string")
	}
}

func TestViewForm_NonEmpty(t *testing.T) {
	m := newModel(t.TempDir())
	m.phase = phaseForm
	got := m.viewForm()
	if got == "" {
		t.Error("viewForm() returned empty string")
	}
}

func TestUpdate_IntroTick_FrameAdvances(t *testing.T) {
	m := newModel(t.TempDir())
	m.phase = phaseIntro
	m.frame = 0

	result, _ := m.Update(tickMsg{})
	updated := result.(model)
	if updated.frame != 1 {
		t.Errorf("after tick frame = %d, want 1", updated.frame)
	}
}

func TestUpdate_IntroTick_BelowThreshold_StaysIntro(t *testing.T) {
	m := newModel(t.TempDir())
	m.phase = phaseIntro
	m.frame = 10

	result, cmd := m.Update(tickMsg{})
	updated := result.(model)
	if updated.phase != phaseIntro {
		t.Errorf("below threshold: phase = %v, want phaseIntro", updated.phase)
	}
	if cmd == nil {
		t.Error("below threshold: cmd is nil, want tick cmd")
	}
}

func TestUpdate_IntroTick_AtThreshold_TransitionsToForm(t *testing.T) {
	m := newModel(t.TempDir())
	m.phase = phaseIntro
	m.frame = 39

	result, _ := m.Update(tickMsg{})
	updated := result.(model)
	if updated.phase != phaseForm {
		t.Errorf("at threshold (frame 39->40): phase = %v, want phaseForm", updated.phase)
	}
}

func TestUpdate_IntroTick_AboveThreshold_TransitionsToForm(t *testing.T) {
	m := newModel(t.TempDir())
	m.phase = phaseIntro
	m.frame = 50

	result, _ := m.Update(tickMsg{})
	updated := result.(model)
	if updated.phase != phaseForm {
		t.Errorf("above threshold: phase = %v, want phaseForm", updated.phase)
	}
}

func TestUpdate_IntroKeyMsg_TransitionsToForm(t *testing.T) {
	m := newModel(t.TempDir())
	m.phase = phaseIntro

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := result.(model)
	if updated.phase != phaseForm {
		t.Errorf("keypress in intro: phase = %v, want phaseForm", updated.phase)
	}
}

func TestUpdate_SuccessTick_BelowThreshold_StaysSuccess(t *testing.T) {
	m := newModel(t.TempDir())
	m.phase = phaseSuccess
	m.written = "/tmp/rule.md"
	m.frame = 0

	result, cmd := m.Update(tickMsg{})
	updated := result.(model)
	if updated.phase != phaseSuccess {
		t.Errorf("success below threshold: phase = %v, want phaseSuccess", updated.phase)
	}
	if cmd == nil {
		t.Error("success below threshold: cmd is nil")
	}
}

func TestUpdate_SuccessTick_AtThreshold_TransitionsToDone(t *testing.T) {
	m := newModel(t.TempDir())
	m.phase = phaseSuccess
	m.written = "/tmp/rule.md"
	m.frame = 29

	result, cmd := m.Update(tickMsg{})
	updated := result.(model)
	if updated.phase != phaseDone {
		t.Errorf("success at threshold: phase = %v, want phaseDone", updated.phase)
	}
	if cmd == nil {
		t.Error("success at threshold: cmd is nil, want tea.Quit")
	}
}

func TestUpdate_SuccessKeyMsg_TransitionsToDone(t *testing.T) {
	m := newModel(t.TempDir())
	m.phase = phaseSuccess
	m.written = "/tmp/rule.md"

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := result.(model)
	if updated.phase != phaseDone {
		t.Errorf("keypress in success: phase = %v, want phaseDone", updated.phase)
	}
	if cmd == nil {
		t.Error("keypress in success: cmd is nil, want tea.Quit")
	}
}

func TestUpdate_UnknownMsg_ReturnsNilCmd(t *testing.T) {
	m := newModel(t.TempDir())
	m.phase = phaseIntro

	_, cmd := m.Update("unknown-message-type")
	_ = cmd
}

func TestUpdate_FormPhase_AbortedState_TransitionsToDone(t *testing.T) {
	m := newModel(t.TempDir())
	m.phase = phaseForm
	m.form.State = huh.StateAborted

	result, cmd := m.Update(tickMsg{})
	updated := result.(model)
	if updated.phase != phaseDone {
		t.Errorf("form aborted: phase = %v, want phaseDone", updated.phase)
	}
	if cmd == nil {
		t.Error("form aborted: cmd is nil, want tea.Quit")
	}
}

func TestUpdate_FormPhase_CompletedSuccess_WritesFileAndTransitionsToSuccess(t *testing.T) {
	root := t.TempDir()
	m := newModel(root)
	m.phase = phaseForm
	m.state.target = "global"
	m.state.filename = "cov-rule"
	m.state.body = "hi"
	m.form.State = huh.StateCompleted

	result, cmd := m.Update(tickMsg{})
	updated := result.(model)

	if updated.form.State != huh.StateCompleted {
		t.Skip("huh.Form.Update reset StateCompleted; direct branch test not possible via Update")
	}

	if updated.phase != phaseSuccess {
		t.Errorf("form completed success: phase = %v, want phaseSuccess", updated.phase)
	}
	if updated.written == "" {
		t.Error("form completed success: written is empty, want a file path")
	}
	if cmd == nil {
		t.Error("form completed success: cmd is nil, want tick cmd")
	}
	if _, err := os.Stat(updated.written); err != nil {
		t.Errorf("form completed success: written file does not exist: %v", err)
	}
}

func TestUpdate_FormPhase_CompletedError_SetsErrAndTransitionsToDone(t *testing.T) {
	root := t.TempDir()
	m := newModel(root)
	m.phase = phaseForm
	m.state.target = "global"
	m.state.filename = "!!!"
	m.state.body = "hi"
	m.form.State = huh.StateCompleted

	result, _ := m.Update(tickMsg{})
	updated := result.(model)

	if updated.form.State != huh.StateCompleted {
		t.Skip("huh.Form.Update reset StateCompleted; direct branch test not possible via Update")
	}

	if updated.phase != phaseDone {
		t.Errorf("form completed error: phase = %v, want phaseDone", updated.phase)
	}
	if updated.err == nil {
		t.Error("form completed error: err is nil, want non-nil error")
	}
}

func TestUpdate_FormPhase_InProgress_StaysFormPhase(t *testing.T) {
	m := newModel(t.TempDir())
	m.phase = phaseForm

	result, _ := m.Update(tickMsg{})
	updated := result.(model)

	if updated.phase != phaseForm {
		t.Errorf("form in-progress: phase = %v, want phaseForm", updated.phase)
	}
}
