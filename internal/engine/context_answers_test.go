package engine

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func writeManifest(t *testing.T, root string) {
	t.Helper()
	dir := paths.Tddmaster(root)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll manifest dir: %v", err)
	}
	payload := `{"selectedTools":["claude-code"],"maxIterationBeforeStart":15,"command":"tddmaster"}`
	if err := os.WriteFile(paths.Manifest(root), []byte(payload), 0o644); err != nil {
		t.Fatalf("WriteFile manifest: %v", err)
	}
}

func seedTempSpec(t *testing.T, root, slug, phase string) {
	t.Helper()
	writeManifest(t, root)
	now := time.Now().UTC()
	state := spec.State{
		Version:   1,
		Slug:      slug,
		Phase:     phase,
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

func buildCtxNoPhases(t *testing.T, root, slug string) *Context {
	t.Helper()
	ctx, err := Build(root, slug, nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return ctx
}

func TestAnswerValue_UnsetKeyReturnsEmpty(t *testing.T) {
	root := t.TempDir()
	slug := "test-slug"
	seedTempSpec(t, root, slug, "discovery")

	ctx := buildCtxNoPhases(t, root, slug)

	got := ctx.AnswerValue("listen_context")
	if got != "" {
		t.Errorf("AnswerValue for unset key = %q, want empty string", got)
	}
}

func TestHasAnswer_UnsetKeyReturnsFalse(t *testing.T) {
	root := t.TempDir()
	slug := "test-slug"
	seedTempSpec(t, root, slug, "discovery")

	ctx := buildCtxNoPhases(t, root, slug)

	if ctx.HasAnswer("listen_context") {
		t.Error("HasAnswer for unset key = true, want false")
	}
}

func TestSetAnswer_PersistsAndNewBuildReflects(t *testing.T) {
	root := t.TempDir()
	slug := "test-slug"
	seedTempSpec(t, root, slug, "discovery")

	ctx := buildCtxNoPhases(t, root, slug)

	if err := ctx.SetAnswer("listen_context", "some user context"); err != nil {
		t.Fatalf("SetAnswer returned error: %v", err)
	}

	ctx2 := buildCtxNoPhases(t, root, slug)

	got := ctx2.AnswerValue("listen_context")
	if got != "some user context" {
		t.Errorf("AnswerValue after SetAnswer = %q, want %q", got, "some user context")
	}
}

func TestHasAnswer_TrueAfterSetAnswer(t *testing.T) {
	root := t.TempDir()
	slug := "test-slug"
	seedTempSpec(t, root, slug, "discovery")

	ctx := buildCtxNoPhases(t, root, slug)

	if err := ctx.SetAnswer("mode", "validate"); err != nil {
		t.Fatalf("SetAnswer returned error: %v", err)
	}

	ctx2 := buildCtxNoPhases(t, root, slug)

	if !ctx2.HasAnswer("mode") {
		t.Error("HasAnswer after SetAnswer = false, want true")
	}
}

func TestAnswerValue_ReturnsFirstValueOnly(t *testing.T) {
	root := t.TempDir()
	slug := "test-slug"
	seedTempSpec(t, root, slug, "discovery")

	state, err := spec.LoadState(root, slug)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	state.Answers["multi_key"] = []spec.Answer{
		{Key: "multi_key", Value: "first"},
		{Key: "multi_key", Value: "second"},
	}
	if err := spec.SaveState(root, slug, state); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	ctx := buildCtxNoPhases(t, root, slug)

	got := ctx.AnswerValue("multi_key")
	if got != "first" {
		t.Errorf("AnswerValue with multiple answers = %q, want %q", got, "first")
	}
}

func TestSetAnswer_OverwritesPreviousValue(t *testing.T) {
	root := t.TempDir()
	slug := "test-slug"
	seedTempSpec(t, root, slug, "discovery")

	ctx := buildCtxNoPhases(t, root, slug)

	if err := ctx.SetAnswer("mode", "full"); err != nil {
		t.Fatalf("first SetAnswer: %v", err)
	}
	if err := ctx.SetAnswer("mode", "validate"); err != nil {
		t.Fatalf("second SetAnswer: %v", err)
	}

	ctx2 := buildCtxNoPhases(t, root, slug)

	got := ctx2.AnswerValue("mode")
	if got != "validate" {
		t.Errorf("AnswerValue after overwrite = %q, want %q", got, "validate")
	}
}

func TestSetAnswer_MultipleDistinctKeys(t *testing.T) {
	root := t.TempDir()
	slug := "test-slug"
	seedTempSpec(t, root, slug, "discovery")

	ctx := buildCtxNoPhases(t, root, slug)

	if err := ctx.SetAnswer("listen_context", "ctx value"); err != nil {
		t.Fatalf("SetAnswer listen_context: %v", err)
	}
	if err := ctx.SetAnswer("mode", "ship-fast"); err != nil {
		t.Fatalf("SetAnswer mode: %v", err)
	}

	ctx2 := buildCtxNoPhases(t, root, slug)

	if ctx2.AnswerValue("listen_context") != "ctx value" {
		t.Errorf("listen_context = %q, want %q", ctx2.AnswerValue("listen_context"), "ctx value")
	}
	if ctx2.AnswerValue("mode") != "ship-fast" {
		t.Errorf("mode = %q, want %q", ctx2.AnswerValue("mode"), "ship-fast")
	}
}

func TestSetAnswer_WritesExactlyOneAnswerEntry(t *testing.T) {
	root := t.TempDir()
	slug := "test-slug"
	seedTempSpec(t, root, slug, "discovery")

	ctx := buildCtxNoPhases(t, root, slug)

	if err := ctx.SetAnswer("foo", "bar"); err != nil {
		t.Fatalf("SetAnswer: %v", err)
	}

	data, err := os.ReadFile(paths.SpecState(root, slug))
	if err != nil {
		t.Fatalf("ReadFile state: %v", err)
	}
	var raw struct {
		Answers map[string][]json.RawMessage `json:"answers"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	entries := raw.Answers["foo"]
	if len(entries) != 1 {
		t.Errorf("expected exactly 1 answer entry for key %q, got %d", "foo", len(entries))
	}
}
