package cmd

import (
	"encoding/json"
	"fmt"
	"net"
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

func writeSpecFile(t *testing.T, root, slug, name, content string) {
	t.Helper()
	dir := paths.SpecDir(root, slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir spec dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestVisualizeCmd_Registered(t *testing.T) {
	var found bool
	for _, sub := range newRootCmd().Commands() {
		if sub.Name() == "visualize" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'visualize' command to be registered in root command")
	}
}

func TestVisualizeCmd_NonExistentSlug(t *testing.T) {
	root := t.TempDir()
	cmd := newVisualizeCmd()
	cmd.SetArgs([]string{"non-existent-slug-12345", "--root", root})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent slug, got nil")
	}
	if !strings.Contains(err.Error(), "spec directory not found") {
		t.Errorf("expected 'spec directory not found' in error, got: %v", err)
	}
}

func TestVisualizeEndpoints(t *testing.T) {
	if getVisualizeHandler == nil {
		t.Fatal("getVisualizeHandler is not wired")
	}

	root := t.TempDir()
	slug := "test-slug"

	progressData := `{"spec":"test-slug","status":"executing","tasks":[{"id":"task-1","title":"First","done":true}]}`
	writeSpecFile(t, root, slug, paths.FileProgress, progressData)

	specData := "# Test Spec\nThis is a test spec."
	writeSpecFile(t, root, slug, paths.FileSpec, specData)

	stateData := `{"version":1,"slug":"test-slug","phase":"execution"}`
	writeSpecFile(t, root, slug, paths.FileState, stateData)

	settingsData := `{"tddEnabled":true,"skipVerifierEnabled":false,"importantTaskGateEnabled":true}`
	writeSpecFile(t, root, slug, paths.FileSettings, settingsData)

	handler, err := getVisualizeHandler(root, slug)
	if err != nil {
		t.Fatalf("getVisualizeHandler error: %v", err)
	}
	if handler == nil {
		t.Fatal("nil handler")
	}

	request := func(path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	cases := []struct {
		path, contentType, body string
	}{
		{"/progress.json", "application/json", progressData},
		{"/spec.md", "text/markdown", specData},
		{"/state.json", "application/json", stateData},
		{"/settings.json", "application/json", settingsData},
	}
	for _, c := range cases {
		t.Run(c.path, func(t *testing.T) {
			rec := request(c.path)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, c.contentType) {
				t.Errorf("content-type = %q, want prefix %q", ct, c.contentType)
			}
			if got := rec.Body.String(); got != c.body {
				t.Errorf("body = %q, want %q", got, c.body)
			}
		})
	}

	t.Run("/", func(t *testing.T) {
		rec := request("/")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `id="stepper"`) {
			t.Error("expected dashboard html served at /")
		}
	})

	t.Run("/api/status", func(t *testing.T) {
		rec := request("/api/status")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("content-type = %q, want application/json", ct)
		}
		var resp struct {
			Hash string `json:"hash"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("parse status: %v", err)
		}
		if resp.Hash != visualize.CalculateHash(root, slug) {
			t.Errorf("hash mismatch: %q vs %q", resp.Hash, visualize.CalculateHash(root, slug))
		}
	})
}

func TestVisualizeEndpoints_EdgeCases(t *testing.T) {
	if getVisualizeHandler == nil {
		t.Fatal("getVisualizeHandler is not wired")
	}

	do := func(t *testing.T, root, slug, path string) *httptest.ResponseRecorder {
		t.Helper()
		handler, err := getVisualizeHandler(root, slug)
		if err != nil {
			t.Fatalf("handler error: %v", err)
		}
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	mkDir := func(t *testing.T) (string, string) {
		root := t.TempDir()
		slug := "test-slug"
		if err := os.MkdirAll(paths.SpecDir(root, slug), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		return root, slug
	}

	t.Run("progress.json missing", func(t *testing.T) {
		root, slug := mkDir(t)
		if rec := do(t, root, slug, "/progress.json"); rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
	t.Run("progress.json malformed", func(t *testing.T) {
		root, slug := mkDir(t)
		writeSpecFile(t, root, slug, paths.FileProgress, `{malformed`)
		if rec := do(t, root, slug, "/progress.json"); rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want 500", rec.Code)
		}
	})
	t.Run("state.json missing", func(t *testing.T) {
		root, slug := mkDir(t)
		if rec := do(t, root, slug, "/state.json"); rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
	t.Run("settings.json missing", func(t *testing.T) {
		root, slug := mkDir(t)
		if rec := do(t, root, slug, "/settings.json"); rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
	t.Run("spec.md missing", func(t *testing.T) {
		root, slug := mkDir(t)
		if rec := do(t, root, slug, "/spec.md"); rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
	t.Run("unknown path", func(t *testing.T) {
		root, slug := mkDir(t)
		if rec := do(t, root, slug, "/nope"); rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
}

func TestVisualizeDashboardHTML(t *testing.T) {
	if dashboardHTML == "" {
		t.Fatal("dashboardHTML is empty; embed failed")
	}

	requiredElements := []string{
		`id="stepper"`,
		`id="spec-markdown"`,
		`id="progress-percentage"`,
		`id="progress-fill"`,
		`id="spec-name"`,
		`id="phase-badge"`,
		`id="stat-iteration"`,
		`id="stat-completed"`,
		`id="stat-refactors"`,
		`id="stat-debt"`,
		`id="task-detail-card"`,
		`id="debt-card"`,
		`id="settings-card"`,
		`id="theme-toggle"`,
	}
	for _, e := range requiredElements {
		if !strings.Contains(dashboardHTML, e) {
			t.Errorf("dashboard missing element %q", e)
		}
	}

	for _, kw := range []string{`'/api/status'`, "checkUpdates", "setInterval"} {
		if !strings.Contains(dashboardHTML, kw) {
			t.Errorf("dashboard missing polling keyword %q", kw)
		}
	}
}

func TestTraceabilityRoute_Returns200AndValidJSON(t *testing.T) {
	if getVisualizeHandler == nil {
		t.Fatal("getVisualizeHandler is not wired")
	}

	root := t.TempDir()
	slug := "trace-slug"

	traceability := spec.Traceability{
		Entries: map[string][]spec.TraceEntry{
			"task-1": {
				{FunctionName: "TestFoo", TaskID: "task-1", AC: []string{"AC1"}, EC: []string{}},
			},
		},
	}
	if err := spec.SaveTraceability(root, slug, traceability); err != nil {
		t.Fatalf("SaveTraceability: %v", err)
	}

	handler, err := getVisualizeHandler(root, slug)
	if err != nil {
		t.Fatalf("getVisualizeHandler error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/traceability.json", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("content-type = %q, want application/json", ct)
	}
	var parsed spec.Traceability
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if len(parsed.Entries["task-1"]) != 1 {
		t.Errorf("expected 1 entry for task-1, got %d", len(parsed.Entries["task-1"]))
	}
}

func TestTraceabilityRoute_EmptyTraceability_Returns200AndValidJSON(t *testing.T) {
	if getVisualizeHandler == nil {
		t.Fatal("getVisualizeHandler is not wired")
	}

	root := t.TempDir()
	slug := "trace-empty-slug"

	if err := spec.SaveTraceability(root, slug, spec.Traceability{}); err != nil {
		t.Fatalf("SaveTraceability: %v", err)
	}

	handler, err := getVisualizeHandler(root, slug)
	if err != nil {
		t.Fatalf("getVisualizeHandler error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/traceability.json", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (empty traceability must not 500)", rec.Code)
	}
	var parsed json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
}

func TestCalculateHash_IncludesTraceabilityFile(t *testing.T) {
	root := t.TempDir()
	slug := "hash-slug"

	if err := os.MkdirAll(paths.SpecDir(root, slug), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	initial := spec.Traceability{
		Entries: map[string][]spec.TraceEntry{
			"task-1": {
				{FunctionName: "TestAlpha", TaskID: "task-1", AC: []string{"AC1"}, EC: []string{}},
			},
		},
	}
	if err := spec.SaveTraceability(root, slug, initial); err != nil {
		t.Fatalf("SaveTraceability initial: %v", err)
	}

	hashBefore := visualize.CalculateHash(root, slug)

	updated := spec.Traceability{
		Entries: map[string][]spec.TraceEntry{
			"task-1": {
				{FunctionName: "TestAlpha", TaskID: "task-1", AC: []string{"AC1"}, EC: []string{}},
				{FunctionName: "TestBeta", TaskID: "task-1", AC: []string{"AC2"}, EC: []string{}},
			},
		},
	}
	if err := spec.SaveTraceability(root, slug, updated); err != nil {
		t.Fatalf("SaveTraceability updated: %v", err)
	}

	hashAfter := visualize.CalculateHash(root, slug)

	if hashBefore == hashAfter {
		t.Errorf("hash did not change after mutating traceability.json: both = %q", hashBefore)
	}
}

func TestServerPortAssignment(t *testing.T) {
	if listenOnFreePort == nil {
		t.Fatal("listenOnFreePort is not wired")
	}

	listener, addr, err := listenOnFreePort()
	if err != nil {
		t.Fatalf("listenOnFreePort error: %v", err)
	}
	if listener == nil {
		t.Fatal("nil listener")
	}
	defer listener.Close()

	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("expected *net.TCPAddr, got %T", listener.Addr())
	}
	if tcpAddr.Port == 0 {
		t.Error("expected non-zero port")
	}
	if want := fmt.Sprintf("http://127.0.0.1:%d", tcpAddr.Port); addr != want {
		t.Errorf("addr = %q, want %q", addr, want)
	}
}
