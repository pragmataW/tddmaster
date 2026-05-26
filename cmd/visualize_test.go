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

	"github.com/pragmataW/tddmaster/internal/state"
	"github.com/pragmataW/tddmaster/internal/visualize"
)

// getVisualizeHandler is a hook that the implementation should define (e.g. in cmd/visualize.go's init or
// inside a refactor) to return the http.Handler for the visualize server.
// If it is nil, the test will fail, indicating that the handler extraction has not been implemented yet.

// listenOnFreePort is defined in cmd/visualize.go.

func setupTestRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := state.ScaffoldDir(root); err != nil {
		t.Fatalf("failed to scaffold test root: %v", err)
	}
	t.Setenv("TDDMASTER_PROJECT_ROOT", root)
	return root
}

func TestVisualizeCmd_Registered(t *testing.T) {
	var found bool
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "visualize" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'visualize' command to be registered in rootCmd")
	}
}

func TestVisualizeCmd_NonExistentSlug(t *testing.T) {
	setupTestRoot(t)

	cmd := newVisualizeCmd()
	cmd.SetArgs([]string{"non-existent-slug-12345"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when running visualize command with a non-existent slug, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "spec directory not found") {
		t.Errorf("expected error message to contain 'spec directory not found', got: %v", err)
	}
}

func TestVisualizeEndpoints(t *testing.T) {
	if getVisualizeHandler == nil {
		t.Fatal("getVisualizeHandler is not implemented (failing TDD test)")
	}

	root := setupTestRoot(t)
	slug := "test-slug"

	// Create specs directory for the slug
	specDir := filepath.Join(root, ".tddmaster", "specs", slug)
	if err := os.MkdirAll(specDir, 0755); err != nil {
		t.Fatalf("failed to create spec directory: %v", err)
	}

	// 1. Write progress.json
	progressData := `{"completed": ["task-1"], "remaining": []}`
	progressPath := filepath.Join(specDir, "progress.json")
	if err := os.WriteFile(progressPath, []byte(progressData), 0644); err != nil {
		t.Fatalf("failed to write progress.json: %v", err)
	}

	// 2. Write spec.md
	specData := "# Test Spec\nThis is a test spec."
	specPath := filepath.Join(specDir, "spec.md")
	if err := os.WriteFile(specPath, []byte(specData), 0644); err != nil {
		t.Fatalf("failed to write spec.md: %v", err)
	}

	// 3. Write state.json using state.WriteSpecState
	st := state.CreateInitialState()
	st.Phase = state.PhaseExecuting
	if err := state.WriteSpecState(root, slug, st); err != nil {
		t.Fatalf("failed to write spec state: %v", err)
	}

	// Retrieve the visualize handler under test
	handler, err := getVisualizeHandler(root, slug)
	if err != nil {
		t.Fatalf("getVisualizeHandler returned error: %v", err)
	}
	if handler == nil {
		t.Fatal("getVisualizeHandler returned nil handler")
	}

	// Helper function to perform HTTP GET requests to the handler
	request := func(path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	// A. Test /progress.json endpoint
	t.Run("progress.json", func(t *testing.T) {
		rec := request("/progress.json")
		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", contentType)
		}
		body := rec.Body.String()
		if body != progressData {
			t.Errorf("expected body %q, got %q", progressData, body)
		}
	})

	// B. Test /spec.md endpoint
	t.Run("spec.md", func(t *testing.T) {
		rec := request("/spec.md")
		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if contentType := rec.Header().Get("Content-Type"); contentType != "text/markdown" {
			t.Errorf("expected Content-Type text/markdown, got %q", contentType)
		}
		body := rec.Body.String()
		if body != specData {
			t.Errorf("expected body %q, got %q", specData, body)
		}
	})

	// C. Test /state.json endpoint
	t.Run("state.json", func(t *testing.T) {
		rec := request("/state.json")
		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", contentType)
		}

		var returnedState state.StateFile
		if err := json.Unmarshal(rec.Body.Bytes(), &returnedState); err != nil {
			t.Fatalf("failed to parse returned state JSON: %v", err)
		}
		if returnedState.Phase != state.PhaseExecuting {
			t.Errorf("expected returned state phase %q, got %q", state.PhaseExecuting, returnedState.Phase)
		}
	})

	// D. Test /api/status endpoint
	t.Run("api/status", func(t *testing.T) {
		rec := request("/api/status")
		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", contentType)
		}

		type statusResponse struct {
			Hash string `json:"hash"`
		}
		var resp statusResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse status response JSON: %v", err)
		}

		expectedHash := visualize.CalculateHash(root, slug)
		if resp.Hash != expectedHash {
			t.Errorf("expected hash %q, got %q", expectedHash, resp.Hash)
		}
	})
}

func TestVisualizeDashboardHTML(t *testing.T) {
	// 1. Verify dashboardHTML is successfully embedded and not empty
	if dashboardHTML == "" {
		t.Error("expected dashboardHTML to be embedded and not empty, but it was empty")
	}

	// 2. Verify critical DOM elements required by the spec
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
		`id="decisions-card"`,
		`id="debt-card"`,
	}

	for _, elem := range requiredElements {
		if !strings.Contains(dashboardHTML, elem) {
			t.Errorf("expected dashboardHTML to contain element/attribute %q", elem)
		}
	}

	// 3. Verify JavaScript polling mechanism for /api/status
	pollingKeywords := []string{
		`"/api/status"`,
		`checkUpdates`,
		`setInterval`,
	}

	for _, keyword := range pollingKeywords {
		if !strings.Contains(dashboardHTML, keyword) {
			t.Errorf("expected dashboardHTML to contain JavaScript polling keyword %q", keyword)
		}
	}
}

func TestVisualizeEndpoints_EdgeCases(t *testing.T) {
	if getVisualizeHandler == nil {
		t.Fatal("getVisualizeHandler is not implemented (failing TDD test)")
	}

	t.Run("progress.json missing", func(t *testing.T) {
		root := setupTestRoot(t)
		slug := "test-slug"
		specDir := filepath.Join(root, ".tddmaster", "specs", slug)
		if err := os.MkdirAll(specDir, 0755); err != nil {
			t.Fatalf("failed to create spec directory: %v", err)
		}

		handler, err := getVisualizeHandler(root, slug)
		if err != nil {
			t.Fatalf("getVisualizeHandler returned error: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/progress.json", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404 for missing progress.json, got %d", rec.Code)
		}
	})

	t.Run("progress.json malformed", func(t *testing.T) {
		root := setupTestRoot(t)
		slug := "test-slug"
		specDir := filepath.Join(root, ".tddmaster", "specs", slug)
		if err := os.MkdirAll(specDir, 0755); err != nil {
			t.Fatalf("failed to create spec directory: %v", err)
		}

		// Write malformed JSON
		progressPath := filepath.Join(specDir, "progress.json")
		if err := os.WriteFile(progressPath, []byte(`{malformed json}`), 0644); err != nil {
			t.Fatalf("failed to write progress.json: %v", err)
		}

		handler, err := getVisualizeHandler(root, slug)
		if err != nil {
			t.Fatalf("getVisualizeHandler returned error: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/progress.json", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500 for malformed progress.json, got %d", rec.Code)
		}
	})

	t.Run("state.json missing", func(t *testing.T) {
		root := setupTestRoot(t)
		slug := "test-slug"
		specDir := filepath.Join(root, ".tddmaster", "specs", slug)
		if err := os.MkdirAll(specDir, 0755); err != nil {
			t.Fatalf("failed to create spec directory: %v", err)
		}
		// Do not write state.json

		handler, err := getVisualizeHandler(root, slug)
		if err != nil {
			t.Fatalf("getVisualizeHandler returned error: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/state.json", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404 for missing state.json, got %d", rec.Code)
		}
	})

	t.Run("state.json malformed", func(t *testing.T) {
		root := setupTestRoot(t)
		slug := "test-slug"
		specDir := filepath.Join(root, ".tddmaster", "specs", slug)
		if err := os.MkdirAll(specDir, 0755); err != nil {
			t.Fatalf("failed to create spec directory: %v", err)
		}

		// Write malformed JSON directly to state.json path
		statePath := filepath.Join(specDir, "state.json")
		if err := os.WriteFile(statePath, []byte(`{invalid json`), 0644); err != nil {
			t.Fatalf("failed to write state.json: %v", err)
		}

		handler, err := getVisualizeHandler(root, slug)
		if err != nil {
			t.Fatalf("getVisualizeHandler returned error: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/state.json", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500 for malformed state.json, got %d", rec.Code)
		}
	})

	t.Run("spec.md missing", func(t *testing.T) {
		root := setupTestRoot(t)
		slug := "test-slug"
		specDir := filepath.Join(root, ".tddmaster", "specs", slug)
		if err := os.MkdirAll(specDir, 0755); err != nil {
			t.Fatalf("failed to create spec directory: %v", err)
		}
		// Do not write spec.md

		handler, err := getVisualizeHandler(root, slug)
		if err != nil {
			t.Fatalf("getVisualizeHandler returned error: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/spec.md", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404 for missing spec.md, got %d", rec.Code)
		}
	})
}

func TestServerPortAssignment(t *testing.T) {
	if listenOnFreePort == nil {
		t.Fatal("listenOnFreePort is not implemented (failing TDD test)")
	}

	listener, addr, err := listenOnFreePort()
	if err != nil {
		t.Fatalf("listenOnFreePort returned error: %v", err)
	}
	if listener == nil {
		t.Fatal("expected non-nil listener")
	}
	defer listener.Close()

	// Verify that the listener is actually listening
	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("expected listener to be TCP listener, got %T", listener.Addr())
	}

	if tcpAddr.Port == 0 {
		t.Error("expected listener to bind to a free port, but port is 0")
	}

	// Verify that the returned address has the correct port
	expectedAddr := fmt.Sprintf("http://127.0.0.1:%d", tcpAddr.Port)
	if addr != expectedAddr {
		t.Errorf("expected listening address %q, got %q", expectedAddr, addr)
	}
}

func TestVisualizeDocumentation(t *testing.T) {
	// Find README.md by climbing up directories from the current working directory
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	var readmePath string
	for {
		candidate := filepath.Join(dir, "README.md")
		if _, err := os.Stat(candidate); err == nil {
			readmePath = candidate
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	if readmePath == "" {
		t.Fatal("README.md file not found in repository root or parent directories")
	}

	contentBytes, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	content := string(contentBytes)

	// Check that the content contains the section/header ## Visualize
	// or similar documentation section detailing the tddmaster visualize <slug> command
	// and its dynamic features.
	if !strings.Contains(content, "## Visualize") {
		t.Error("expected README.md to contain a '## Visualize' section")
	}

	if !strings.Contains(content, "tddmaster visualize") {
		t.Error("expected README.md to contain documentation detailing 'tddmaster visualize <slug>' command")
	}

	// Dynamic features details: e.g., real-time updates / progress / polling / dashboard
	if !strings.Contains(content, "polling") && !strings.Contains(content, "real-time") && !strings.Contains(content, "dynamic") && !strings.Contains(content, "dashboard") {
		t.Error("expected README.md to detail dynamic features of the visualize command (such as real-time updates, polling, or dynamic dashboard)")
	}
}


