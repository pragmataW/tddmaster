package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/pragmataW/tddmaster/internal/spec"
)

func scaffoldSpec(t *testing.T, root, slug string) {
	t.Helper()
	s := spec.State{
		Version:   1,
		Slug:      slug,
		Phase:     spec.PhaseInitial,
		Answers:   map[string][]spec.Answer{},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := spec.SaveState(root, slug, s); err != nil {
		t.Fatalf("scaffoldSpec SaveState: %v", err)
	}
	if err := spec.SaveSettings(root, slug, spec.DefaultSettings()); err != nil {
		t.Fatalf("scaffoldSpec SaveSettings: %v", err)
	}
	p := spec.Progress{
		Spec:      slug,
		Status:    spec.StatusDraft,
		Tasks:     []spec.Task{},
		UpdatedAt: time.Now().UTC(),
	}
	if err := spec.SaveProgress(root, slug, p); err != nil {
		t.Fatalf("scaffoldSpec SaveProgress: %v", err)
	}
}

func executeNext(t *testing.T, root string, extraArgs ...string) (string, error) {
	t.Helper()
	root_cmd := newRootCmd()
	var buf bytes.Buffer
	root_cmd.SetOut(&buf)
	root_cmd.SetErr(&buf)
	args := append([]string{"next"}, extraArgs...)
	if root != "" {
		args = append(args, "--root", root)
	}
	root_cmd.SetArgs(args)
	err := root_cmd.Execute()
	return buf.String(), err
}

func Test_NextCmd_NoSlugArg_ReturnsError(t *testing.T) {
	root := t.TempDir()
	_, err := executeNext(t, root)
	if err == nil {
		t.Fatal("expected error when no slug arg provided, got nil")
	}
}

func Test_NextCmd_SpecDirMissing_ReturnsStartFirstError(t *testing.T) {
	root := t.TempDir()
	_, err := executeNext(t, root, "nonexistent-slug")
	if err == nil {
		t.Fatal("expected error when spec dir does not exist, got nil")
	}
	msg := err.Error()
	if !strings.Contains(strings.ToLower(msg), "start") {
		t.Errorf("expected error mentioning 'start', got: %q", msg)
	}
}

func Test_NextCmd_HappyPath_EmitsValidJSON(t *testing.T) {
	root := t.TempDir()
	scaffoldSpec(t, root, "my-spec")

	out, err := executeNext(t, root, "my-spec")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var action map[string]any
	if jsonErr := json.Unmarshal([]byte(out), &action); jsonErr != nil {
		t.Fatalf("stdout is not valid JSON: %v\noutput: %q", jsonErr, out)
	}
}

func Test_NextCmd_HappyPath_JSONHasActionField(t *testing.T) {
	root := t.TempDir()
	scaffoldSpec(t, root, "my-spec")

	out, err := executeNext(t, root, "my-spec")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var action map[string]any
	if err := json.Unmarshal([]byte(out), &action); err != nil {
		t.Fatalf("stdout is not valid JSON: %v", err)
	}

	if _, ok := action["action"]; !ok {
		t.Errorf("expected JSON to have 'action' field, got keys: %v", keys(action))
	}
}

func Test_NextCmd_HappyPath_JSONIsIndented(t *testing.T) {
	root := t.TempDir()
	scaffoldSpec(t, root, "my-spec")

	out, err := executeNext(t, root, "my-spec")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "\n") {
		t.Errorf("expected indented (multi-line) JSON output, got single line: %q", out)
	}
}

func Test_NextCmd_AnswerFlag_EmitsValidJSON(t *testing.T) {
	root := t.TempDir()
	scaffoldSpec(t, root, "my-spec")

	_, _ = executeNext(t, root, "my-spec")

	out, err := executeNext(t, root, "my-spec", "--answer", `{"tddEnabled":true,"skipVerifierEnabled":false,"importantTaskGateEnabled":false}`)
	if err != nil {
		t.Fatalf("unexpected error on --answer submit: %v", err)
	}

	var action map[string]any
	if jsonErr := json.Unmarshal([]byte(out), &action); jsonErr != nil {
		t.Fatalf("stdout after --answer is not valid JSON: %v\noutput: %q", jsonErr, out)
	}
}

func Test_NextCmd_AnswerFlag_MalformedJSON_ReturnsError(t *testing.T) {
	root := t.TempDir()
	scaffoldSpec(t, root, "my-spec")

	_, _ = executeNext(t, root, "my-spec")

	_, err := executeNext(t, root, "my-spec", "--answer", "{bad json")
	if err == nil {
		t.Fatal("expected error for malformed JSON answer, got nil")
	}
}

func Test_NextCmd_RegisteredOnRoot(t *testing.T) {
	root := newRootCmd()
	var found bool
	for _, sub := range root.Commands() {
		if sub.Name() == "next" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'next' subcommand registered on root, but not found")
	}
}

func Test_NextCmd_MultipleCallsWithoutAnswer_AreIdempotent(t *testing.T) {
	root := t.TempDir()
	scaffoldSpec(t, root, "my-spec")

	out1, err1 := executeNext(t, root, "my-spec")
	out2, err2 := executeNext(t, root, "my-spec")

	if err1 != nil {
		t.Fatalf("first next: unexpected error: %v", err1)
	}
	if err2 != nil {
		t.Fatalf("second next: unexpected error: %v", err2)
	}

	var a1, a2 map[string]any
	if err := json.Unmarshal([]byte(out1), &a1); err != nil {
		t.Fatalf("first output not valid JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(out2), &a2); err != nil {
		t.Fatalf("second output not valid JSON: %v", err)
	}

	if a1["action"] != a2["action"] {
		t.Errorf("repeated next without answer should return same action type: %v vs %v", a1["action"], a2["action"])
	}
}

func keys(m map[string]any) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
