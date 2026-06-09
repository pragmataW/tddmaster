package phases

import (
	"testing"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func seedSettingsSpec(t *testing.T, root, slug string) {
	t.Helper()
	writeDiscoveryManifest(t, root)
	state := spec.State{
		Version: 1,
		Slug:    slug,
		Phase:   "spec-settings",
		Answers: map[string][]spec.Answer{},
	}
	if err := spec.SaveState(root, slug, state); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	if err := spec.SaveSettings(root, slug, spec.DefaultSettings()); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	if err := spec.SaveProgress(root, slug, spec.Progress{Spec: slug, Status: spec.StatusDraft}); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
}

func buildSettingsCtx(t *testing.T, root, slug string) *engine.Context {
	t.Helper()
	defs := []engine.PhaseDef{{ID: "spec-settings", Driver: SettingsDriver()}}
	ctx, err := engine.Build(root, slug, defs)
	if err != nil {
		t.Fatalf("engine.Build: %v", err)
	}
	return ctx
}

func TestSettingsDriver_Next_AsksWithMultiSelect(t *testing.T) {
	root := t.TempDir()
	seedSettingsSpec(t, root, "s")
	ctx := buildSettingsCtx(t, root, "s")

	action, done := (&settingsDriver{}).Next(ctx, nil)
	if done {
		t.Fatal("Next reported phase done before any answer")
	}
	if action.Action != engine.ActionAsk {
		t.Fatalf("action = %q, want %q", action.Action, engine.ActionAsk)
	}
	if !action.MultiSelect {
		t.Fatal("expected MultiSelect=true")
	}
	if len(action.InteractiveOptions) != 3 {
		t.Fatalf("expected 3 interactive options, got %d", len(action.InteractiveOptions))
	}
	if action.ExpectedInput.Format != engine.FormatJSON {
		t.Fatalf("expected JSON format, got %q", action.ExpectedInput.Format)
	}
}

func TestSettingsDriver_Submit_PersistsSettingsAndCompletesPhase(t *testing.T) {
	root := t.TempDir()
	seedSettingsSpec(t, root, "s")
	ctx := buildSettingsCtx(t, root, "s")

	payload := []byte(`{"tddEnabled":false,"skipVerifierEnabled":true,"importantTaskGateEnabled":true}`)
	_, done, err := (&settingsDriver{}).Submit(ctx, nil, payload)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if !done {
		t.Fatal("Submit did not mark phase done")
	}

	got, err := spec.LoadSettings(root, "s")
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	want := spec.Settings{TDDEnabled: false, SkipVerifierEnabled: true, ImportantTaskGateEnabled: true, MinTestCoverage: 80}
	if got != want {
		t.Fatalf("persisted settings = %+v, want %+v", got, want)
	}
}

func TestSettingsDriver_Submit_PartialPayloadKeepsDefaults(t *testing.T) {
	root := t.TempDir()
	seedSettingsSpec(t, root, "s")
	ctx := buildSettingsCtx(t, root, "s")

	_, _, err := (&settingsDriver{}).Submit(ctx, nil, []byte(`{"skipVerifierEnabled":true}`))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	got, _ := spec.LoadSettings(root, "s")
	if !got.TDDEnabled {
		t.Fatal("omitted tddEnabled should keep default true")
	}
	if !got.SkipVerifierEnabled {
		t.Fatal("skipVerifierEnabled should be true")
	}
	if got.ImportantTaskGateEnabled {
		t.Fatal("omitted importantTaskGateEnabled should keep default false")
	}
}

func TestSettingsDriver_Submit_InvalidJSON_ReturnsError(t *testing.T) {
	root := t.TempDir()
	seedSettingsSpec(t, root, "s")
	ctx := buildSettingsCtx(t, root, "s")

	_, done, err := (&settingsDriver{}).Submit(ctx, nil, []byte("not json"))
	if err == nil {
		t.Fatal("expected error on invalid JSON, got nil")
	}
	if done {
		t.Fatal("phase should not complete on invalid JSON")
	}
}

func TestSettingsDriver_Next_DoneAfterAnswerRecorded(t *testing.T) {
	root := t.TempDir()
	seedSettingsSpec(t, root, "s")
	ctx := buildSettingsCtx(t, root, "s")

	if _, _, err := (&settingsDriver{}).Submit(ctx, nil, []byte(`{"tddEnabled":true}`)); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	reloaded := buildSettingsCtx(t, root, "s")
	_, done := (&settingsDriver{}).Next(reloaded, nil)
	if !done {
		t.Fatal("Next should report done once spec_settings answer exists")
	}
}
