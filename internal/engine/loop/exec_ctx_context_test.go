package loop

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func buildContextWithAnswers(t *testing.T, root, slug string, answers map[string][]spec.Answer) (*engine.Context, spec.Task) {
	t.Helper()
	st := spec.State{
		Version: 1,
		Slug:    slug,
		Phase:   "executing",
		Answers: answers,
	}
	if err := spec.SaveState(root, slug, st); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	tasks := []spec.Task{{ID: "t1", Title: "task one", Exec: &spec.ExecState{TDDCycle: cycleEmpty}}}
	if err := spec.SaveProgress(root, slug, spec.Progress{Spec: slug, Status: spec.StatusDraft, Tasks: tasks}); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
	if err := spec.SaveSettings(root, slug, spec.DefaultSettings()); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	ctx, err := engine.Build(root, slug, []engine.PhaseDef{{ID: "executing", Driver: NewLoopDriver()}})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return ctx, tasks[0]
}

func TestBuildExecCtx_CarriesDiscoveryModeAndRejectedPremises(t *testing.T) {
	root := t.TempDir()
	ctx, task := buildContextWithAnswers(t, root, "execctx-intent", map[string][]spec.Answer{
		"mode": {{Key: "mode", Value: "ship-fast"}},
		"premises": {{Key: "premises", Value: `{"premises":[
			{"text":"the cache can be cleared on deploy","agreed":true},
			{"text":"tokens can be rotated offline","agreed":false,"revision":"rotation must happen live"}
		]}`}},
	})

	execCtx := NewLoopDriver().buildExecCtx(ctx, task, 0, nil)

	if execCtx.Mode != "ship-fast" {
		t.Fatalf("ExecCtx.Mode = %q, want %q", execCtx.Mode, "ship-fast")
	}
	want := []string{"tokens can be rotated offline -> REVISED: rotation must happen live"}
	got := execCtx.Intent.ChallengedPremises
	if len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("ExecCtx.Intent.ChallengedPremises = %v, want %v", got, want)
	}
}

func TestBuildExecCtx_DetectsProjectCommandsFromRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/x\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	ctx, task := buildContextWithAnswers(t, root, "execctx-cmds", map[string][]spec.Answer{})

	execCtx := NewLoopDriver().buildExecCtx(ctx, task, 0, nil)

	var sawTest bool
	for _, c := range execCtx.ProjectCommands {
		if c.Label == "test" && c.Value == "go test ./..." {
			sawTest = true
		}
	}
	if !sawTest {
		t.Fatalf("ExecCtx.ProjectCommands did not detect the Go test command: %v", execCtx.ProjectCommands)
	}
}

func TestBuildExecCtx_ProjectCommandsEmptyForUnknownProject(t *testing.T) {
	root := t.TempDir()
	ctx, task := buildContextWithAnswers(t, root, "execctx-no-cmds", map[string][]spec.Answer{})

	execCtx := NewLoopDriver().buildExecCtx(ctx, task, 0, nil)

	if len(execCtx.ProjectCommands) != 0 {
		t.Fatalf("unknown project type must yield no commands, got %v", execCtx.ProjectCommands)
	}
}
