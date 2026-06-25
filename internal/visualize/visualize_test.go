package visualize_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/spec"
	"github.com/pragmataW/tddmaster/internal/visualize"
)

func writeProgressFile(t *testing.T, root, slug string, data []byte) {
	t.Helper()
	dir := paths.SpecDir(root, slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir spec dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, paths.FileProgress), data, 0o644); err != nil {
		t.Fatalf("write progress.json: %v", err)
	}
}

func TestVisualize_RendersCriterionGWTTable(t *testing.T) {
	tasks := []spec.Task{
		{
			ID:    "task-1",
			Title: "First task",
			Criteria: []spec.Criterion{
				{ID: "ac-1", Given: "g", When: "w", Then: "t"},
			},
		},
	}
	got := visualize.RenderCriteriaGWT(tasks)

	for _, want := range []string{"ac-1", "GIVEN", "g", "WHEN", "w", "THEN", "t"} {
		if !strings.Contains(got, want) {
			t.Errorf("RenderCriteriaGWT output missing %q; got:\n%s", want, got)
		}
	}
}

func TestVisualize_MultipleCriteria_EachRendered(t *testing.T) {
	tasks := []spec.Task{
		{
			ID:    "task-1",
			Title: "First task",
			Criteria: []spec.Criterion{
				{ID: "ac-1", Given: "given1", When: "when1", Then: "then1"},
				{ID: "ac-2", Given: "given2", When: "when2", Then: "then2"},
			},
		},
	}
	got := visualize.RenderCriteriaGWT(tasks)

	for _, want := range []string{"ac-1", "given1", "when1", "then1", "ac-2", "given2", "when2", "then2"} {
		if !strings.Contains(got, want) {
			t.Errorf("RenderCriteriaGWT output missing %q; got:\n%s", want, got)
		}
	}
}

func TestVisualize_CriteriaFromProgressJSON(t *testing.T) {
	root := t.TempDir()
	slug := "gwt-slug"

	progress := spec.Progress{
		Spec:   slug,
		Status: "executing",
		Tasks: []spec.Task{
			{
				ID:    "task-1",
				Title: "Task with criteria",
				Criteria: []spec.Criterion{
					{ID: "ac-1", Given: "a user exists", When: "login is called", Then: "session is created"},
				},
			},
		},
	}
	raw, err := json.Marshal(progress)
	if err != nil {
		t.Fatalf("marshal progress: %v", err)
	}
	writeProgressFile(t, root, slug, raw)

	handler, err := visualize.GetHandler(root, slug)
	if err != nil {
		t.Fatalf("GetHandler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/progress.json", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var loaded spec.Progress
	if err := json.Unmarshal(rec.Body.Bytes(), &loaded); err != nil {
		t.Fatalf("unmarshal progress body: %v", err)
	}

	if len(loaded.Tasks) == 0 {
		t.Fatal("no tasks in loaded progress")
	}
	if len(loaded.Tasks[0].Criteria) == 0 {
		t.Fatal("criteria did not survive the progress.json round-trip")
	}
	c := loaded.Tasks[0].Criteria[0]
	if c.ID != "ac-1" {
		t.Errorf("criterion ID = %q, want ac-1", c.ID)
	}
	if c.Given != "a user exists" {
		t.Errorf("Given = %q, want 'a user exists'", c.Given)
	}
	if c.When != "login is called" {
		t.Errorf("When = %q, want 'login is called'", c.When)
	}
	if c.Then != "session is created" {
		t.Errorf("Then = %q, want 'session is created'", c.Then)
	}
}

func TestVisualize_EmptyThenCriterion_NoBreak(t *testing.T) {
	tasks := []spec.Task{
		{
			ID:    "task-1",
			Title: "Task with empty Then",
			Criteria: []spec.Criterion{
				{ID: "ac-1", Given: "some given", When: "some when", Then: ""},
			},
		},
	}

	var got string
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("RenderCriteriaGWT panicked on empty Then: %v", r)
			}
		}()
		got = visualize.RenderCriteriaGWT(tasks)
	}()

	for _, want := range []string{"ac-1", "some given", "some when"} {
		if !strings.Contains(got, want) {
			t.Errorf("RenderCriteriaGWT with empty Then missing %q; got:\n%s", want, got)
		}
	}
}
