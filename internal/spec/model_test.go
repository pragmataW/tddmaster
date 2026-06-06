package spec

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestPhaseInitial(t *testing.T) {
	if PhaseInitial != "listen-first" {
		t.Fatalf("expected PhaseInitial to be %q, got %q", "listen-first", PhaseInitial)
	}
}

func TestDefaultSettings(t *testing.T) {
	got := DefaultSettings()
	want := Settings{
		TDDEnabled:               true,
		SkipVerifierEnabled:      false,
		ImportantTaskGateEnabled: false,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DefaultSettings() = %+v, want %+v", got, want)
	}
}

func TestState_JSONTags(t *testing.T) {
	now := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	original := State{
		Version: 1,
		Slug:    "x",
		Phase:   PhaseInitial,
		Answers: map[string][]Answer{
			"q1": {{Key: "k1", Value: "v1"}},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	raw := string(data)
	for _, tag := range []string{`"version"`, `"slug"`, `"phase"`, `"answers"`, `"createdAt"`, `"updatedAt"`, `"key"`, `"value"`} {
		if !strings.Contains(raw, tag) {
			t.Errorf("JSON missing tag %s in: %s", tag, raw)
		}
	}

	var roundTripped State
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if roundTripped.Version != original.Version {
		t.Errorf("Version mismatch: got %d, want %d", roundTripped.Version, original.Version)
	}
	if roundTripped.Slug != original.Slug {
		t.Errorf("Slug mismatch: got %q, want %q", roundTripped.Slug, original.Slug)
	}
	if roundTripped.Phase != original.Phase {
		t.Errorf("Phase mismatch: got %q, want %q", roundTripped.Phase, original.Phase)
	}
	if !reflect.DeepEqual(roundTripped.Answers, original.Answers) {
		t.Errorf("Answers mismatch: got %+v, want %+v", roundTripped.Answers, original.Answers)
	}
	if !roundTripped.CreatedAt.Equal(original.CreatedAt) {
		t.Errorf("CreatedAt mismatch: got %v, want %v", roundTripped.CreatedAt, original.CreatedAt)
	}
	if !roundTripped.UpdatedAt.Equal(original.UpdatedAt) {
		t.Errorf("UpdatedAt mismatch: got %v, want %v", roundTripped.UpdatedAt, original.UpdatedAt)
	}
}

func TestSettings_JSONTags(t *testing.T) {
	s := DefaultSettings()
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	raw := string(data)
	for _, tag := range []string{`"tddEnabled"`, `"skipVerifierEnabled"`, `"importantTaskGateEnabled"`} {
		if !strings.Contains(raw, tag) {
			t.Errorf("JSON missing tag %s in: %s", tag, raw)
		}
	}

	if !strings.Contains(raw, `"tddEnabled":true`) {
		t.Errorf("expected tddEnabled to be true in: %s", raw)
	}
	if !strings.Contains(raw, `"skipVerifierEnabled":false`) {
		t.Errorf("expected skipVerifierEnabled to be false in: %s", raw)
	}
	if !strings.Contains(raw, `"importantTaskGateEnabled":false`) {
		t.Errorf("expected importantTaskGateEnabled to be false in: %s", raw)
	}
}

func TestProgress_JSONTags(t *testing.T) {
	now := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	original := Progress{
		Spec:   "my-spec",
		Status: "draft",
		Tasks: []Task{
			{
				ID:         "t1",
				Title:      "first task",
				AC:         []string{"ac1", "ac2"},
				Done:       false,
				TDDEnabled: true,
				Important:  false,
			},
		},
		UpdatedAt: now,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	raw := string(data)
	for _, tag := range []string{`"spec"`, `"status"`, `"tasks"`, `"id"`, `"title"`, `"ac"`, `"done"`, `"tddEnabled"`, `"important"`} {
		if !strings.Contains(raw, tag) {
			t.Errorf("JSON missing tag %s in: %s", tag, raw)
		}
	}

	var roundTripped Progress
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if roundTripped.Spec != original.Spec {
		t.Errorf("Spec mismatch: got %q, want %q", roundTripped.Spec, original.Spec)
	}
	if roundTripped.Status != original.Status {
		t.Errorf("Status mismatch: got %q, want %q", roundTripped.Status, original.Status)
	}
	if !reflect.DeepEqual(roundTripped.Tasks, original.Tasks) {
		t.Errorf("Tasks mismatch: got %+v, want %+v", roundTripped.Tasks, original.Tasks)
	}
}

func TestProgress_EmptyTasksSerializesAsArray(t *testing.T) {
	p := Progress{
		Spec:   "s",
		Status: "draft",
		Tasks:  []Task{},
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	raw := string(data)
	if !strings.Contains(raw, `"tasks":[]`) {
		t.Errorf("expected tasks to serialize as [] but got: %s", raw)
	}
	if strings.Contains(raw, `"tasks":null`) {
		t.Errorf("tasks must not serialize as null, got: %s", raw)
	}
}

func TestState_EmptyAnswersMap(t *testing.T) {
	s := State{
		Version: 1,
		Slug:    "y",
		Phase:   PhaseInitial,
		Answers: map[string][]Answer{},
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	raw := string(data)
	if !strings.Contains(raw, `"answers":{}`) {
		t.Errorf("expected answers to serialize as {} but got: %s", raw)
	}
	if strings.Contains(raw, `"answers":null`) {
		t.Errorf("answers must not serialize as null, got: %s", raw)
	}
}
