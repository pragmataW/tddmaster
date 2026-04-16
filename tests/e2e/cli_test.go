// Package e2e contains end-to-end tests that build and run the tddmaster binary.
package e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// binaryPath holds the path of the compiled test binary (set by TestMain).
var binaryPath string

// moduleRoot is the absolute path of the Go module root (set by TestMain).
var moduleRoot string

// TestMain builds the binary once, runs all tests, then cleans up.
func TestMain(m *testing.M) {
	// Determine module root: this file is at tests/e2e/cli_test.go
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller failed")
	}
	moduleRoot = filepath.Join(filepath.Dir(filename), "..", "..")

	// Build binary to a temp dir that persists for the whole test binary run.
	tmpDir, err := os.MkdirTemp("", "tddmaster_e2e_*")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}

	binaryPath = filepath.Join(tmpDir, "tddmaster_e2e")
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = moduleRoot
	if out, err := buildCmd.CombinedOutput(); err != nil {
		panic("failed to build tddmaster binary: " + err.Error() + "\n" + string(out))
	}

	code := m.Run()
	os.RemoveAll(tmpDir)
	os.Exit(code)
}

// buildBinary returns the pre-built binary path (built in TestMain).
func buildBinary(t *testing.T) string {
	t.Helper()
	require.NotEmpty(t, binaryPath, "binaryPath should be set by TestMain")
	return binaryPath
}

// run executes the tddmaster binary with the given args in the given directory.
// Returns stdout, stderr, and exit code.
func run(t *testing.T, dir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	bin := buildBinary(t)
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	// Set TDDMASTER_PROJECT_ROOT so CLI doesn't walk up looking for .tddmaster/
	cmd.Env = append(os.Environ(), "TDDMASTER_PROJECT_ROOT="+dir)

	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}
	return
}

func seedSpecApprovedState(t *testing.T, dir, specName string, tddMode bool) {
	t.Helper()

	_, stderr, exitCode := run(t, dir, "init", "--non-interactive")
	require.Equalf(t, 0, exitCode, "init should succeed: %s", stderr)

	// Update tddMode in manifest via ReadManifest/WriteManifest to use canonical tddmaster.tdd location.
	manifest, err := state.ReadManifest(dir)
	require.NoError(t, err)
	require.NotNil(t, manifest)
	manifest.Tdd = &state.Manifest{TddMode: tddMode}
	require.NoError(t, state.WriteManifest(dir, *manifest))

	specDir := filepath.Join(dir, ".tddmaster", "specs", specName)
	require.NoError(t, os.MkdirAll(specDir, 0o755))

	specPath := filepath.Join(specDir, "spec.md")
	require.NoError(t, os.WriteFile(specPath, []byte("# Spec\n\n## Tasks\n- [ ] task-1: Example task\n"), 0o644))

	st := state.CreateInitialState()
	st.Phase = state.PhaseSpecApproved
	st.Spec = &specName
	st.Discovery.Completed = true
	st.Discovery.Approved = true
	st.SpecState.Path = &specPath
	st.SpecState.Status = "approved"

	require.NoError(t, state.WriteStateAndSpec(dir, st))
}

// TestBinaryBuildAndHelp verifies the binary builds successfully and --help works.
func TestBinaryBuildAndHelp(t *testing.T) {
	bin := buildBinary(t)
	_, err := os.Stat(bin)
	require.NoError(t, err, "binary should exist after build")

	dir := t.TempDir()
	stdout, _, exitCode := run(t, dir, "--help")
	assert.Equal(t, 0, exitCode, "help should exit 0")
	assert.Contains(t, stdout, "tddmaster")
}

// TestHelpContainsAllCommands verifies the help output lists all expected commands.
func TestHelpContainsAllCommands(t *testing.T) {
	dir := t.TempDir()
	stdout, _, exitCode := run(t, dir, "--help")
	assert.Equal(t, 0, exitCode)

	expectedCommands := []string{
		"init",
		"spec",
		"next",
		"approve",
		"done",
		"block",
		"reset",
		"cancel",
		"wontfix",
		"reopen",
		"status",
		"concern",
		"run",
		"watch",
		"sync",
		"learn",
		"purge",
		"rule",
		"config",
		"pack",
		"session",
		"review",
		"delegate",
		"followup",
	}

	for _, cmd := range expectedCommands {
		assert.Contains(t, stdout, cmd, "help should mention command %q", cmd)
	}
}

// TestInitCreatesDirectory verifies that `tddmaster init --non-interactive` creates .tddmaster/.
func TestInitCreatesDirectory(t *testing.T) {
	dir := t.TempDir()

	_, stderr, exitCode := run(t, dir, "init", "--non-interactive")
	t.Logf("stderr: %s", stderr)
	assert.Equal(t, 0, exitCode, "init should exit 0")

	// .tddmaster/ should exist
	tddmasterDir := filepath.Join(dir, ".tddmaster")
	info, err := os.Stat(tddmasterDir)
	require.NoError(t, err, ".tddmaster/ should be created")
	assert.True(t, info.IsDir())
}

// TestInitCreatesManifest verifies that init creates manifest.yml with tddmaster section.
func TestInitCreatesManifest(t *testing.T) {
	dir := t.TempDir()
	_, _, exitCode := run(t, dir, "init", "--non-interactive")
	assert.Equal(t, 0, exitCode)

	manifestPath := filepath.Join(dir, ".tddmaster", "manifest.yml")
	data, err := os.ReadFile(manifestPath)
	require.NoError(t, err, "manifest.yml should be created")

	content := string(data)
	assert.Contains(t, content, "tddmaster", "manifest should contain tddmaster section")
}

// TestInitCreatesStateFile verifies that init creates .tddmaster/.state/state.json.
func TestInitCreatesStateFile(t *testing.T) {
	dir := t.TempDir()
	_, _, exitCode := run(t, dir, "init", "--non-interactive")
	assert.Equal(t, 0, exitCode)

	stateFilePath := filepath.Join(dir, ".tddmaster", ".state", "state.json")
	data, err := os.ReadFile(stateFilePath)
	require.NoError(t, err, "state.json should be created")

	var stateMap map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &stateMap))
	assert.Equal(t, "IDLE", stateMap["phase"])
}

// TestInitIsIdempotent verifies that running init twice does not fail.
func TestInitIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	_, _, exitCode1 := run(t, dir, "init", "--non-interactive")
	assert.Equal(t, 0, exitCode1, "first init should succeed")

	_, _, exitCode2 := run(t, dir, "init", "--non-interactive")
	assert.Equal(t, 0, exitCode2, "second init should succeed (idempotent)")
}

// TestSpecLifecycle verifies: init → spec new → spec list.
func TestSpecLifecycle(t *testing.T) {
	dir := t.TempDir()

	// Step 1: init
	_, _, exitCode := run(t, dir, "init", "--non-interactive")
	require.Equal(t, 0, exitCode, "init should succeed")

	// Step 2: spec new
	stdout, stderr, exitCode := run(t, dir, "spec", "new", "Add user authentication feature")
	t.Logf("spec new stdout: %s", stdout)
	t.Logf("spec new stderr: %s", stderr)
	assert.Equal(t, 0, exitCode, "spec new should succeed")

	// stderr should mention the spec was started
	assert.Contains(t, stderr, "Spec started:")

	// Step 3: spec list
	stdout, stderr, exitCode = run(t, dir, "spec", "list")
	t.Logf("spec list stdout: %s", stdout)
	t.Logf("spec list stderr: %s", stderr)
	assert.Equal(t, 0, exitCode, "spec list should succeed")

	// Parse JSON output
	var specs []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(stdout)), &specs))
	require.NotEmpty(t, specs, "spec list should return at least one spec")

	// Verify the spec appears with DISCOVERY phase
	found := false
	for _, s := range specs {
		if phase, ok := s["phase"].(string); ok && phase == "DISCOVERY" {
			found = true
			break
		}
	}
	assert.True(t, found, "newly created spec should be in DISCOVERY phase")
}

// TestSpecNewRequiresInit verifies that spec new fails without initialization.
func TestSpecNewRequiresInit(t *testing.T) {
	dir := t.TempDir()

	_, _, exitCode := run(t, dir, "spec", "new", "My new feature")
	assert.NotEqual(t, 0, exitCode, "spec new without init should fail")
}

// TestSpecNewCreatesDirectory verifies that spec new creates the spec directory.
func TestSpecNewCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	_, _, _ = run(t, dir, "init", "--non-interactive")

	_, _, exitCode := run(t, dir, "spec", "new", "test feature for directory creation")
	require.Equal(t, 0, exitCode)

	// The spec dir should exist under .tddmaster/specs/
	specsDir := filepath.Join(dir, ".tddmaster", "specs")
	entries, err := os.ReadDir(specsDir)
	require.NoError(t, err)
	assert.NotEmpty(t, entries, "specs directory should contain at least one spec")
}

// TestSpecListEmpty verifies spec list returns valid JSON when no specs exist.
// The output may be null or [] — both are acceptable empty-list representations.
func TestSpecListEmpty(t *testing.T) {
	dir := t.TempDir()
	_, _, exitCode := run(t, dir, "init", "--non-interactive")
	require.Equal(t, 0, exitCode)

	stdout, _, exitCode := run(t, dir, "spec", "list")
	assert.Equal(t, 0, exitCode)

	// Should be valid JSON (null or [])
	trimmed := strings.TrimSpace(stdout)
	require.NotEmpty(t, trimmed, "spec list should produce output")

	// Accept both null and [] as valid empty responses
	assert.True(t, trimmed == "null" || (strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")),
		"spec list should return JSON array or null, got: %q", trimmed)
}

// TestStatusCommand verifies `tddmaster status` works after init.
func TestStatusCommand(t *testing.T) {
	dir := t.TempDir()
	_, _, exitCode := run(t, dir, "init", "--non-interactive")
	require.Equal(t, 0, exitCode)

	stdout, _, exitCode := run(t, dir, "status")
	t.Logf("status stdout: %s", stdout)
	assert.Equal(t, 0, exitCode, "status should succeed after init")
}

// TestInitWithConcerns verifies that --concerns flag is accepted.
func TestInitWithConcerns(t *testing.T) {
	dir := t.TempDir()

	_, stderr, exitCode := run(t, dir, "init", "--non-interactive", "--concerns=security")
	t.Logf("stderr: %s", stderr)
	// Should succeed (unknown concern IDs are ignored gracefully)
	assert.Equal(t, 0, exitCode)
}

// TestInitWithTools verifies that --tools flag is accepted.
func TestInitWithTools(t *testing.T) {
	dir := t.TempDir()

	_, stderr, exitCode := run(t, dir, "init", "--non-interactive", "--tools=claude-code")
	t.Logf("stderr: %s", stderr)
	assert.Equal(t, 0, exitCode)

	manifestPath := filepath.Join(dir, ".tddmaster", "manifest.yml")
	data, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "claude-code")
}

// TestMultipleSpecsLifecycle verifies creating and listing multiple specs.
func TestMultipleSpecsLifecycle(t *testing.T) {
	dir := t.TempDir()
	_, _, exitCode := run(t, dir, "init", "--non-interactive")
	require.Equal(t, 0, exitCode)

	// Create two specs
	_, _, c1 := run(t, dir, "spec", "new", "First feature implementation")
	require.Equal(t, 0, c1, "first spec new should succeed")

	_, _, c2 := run(t, dir, "spec", "new", "Second feature implementation")
	require.Equal(t, 0, c2, "second spec new should succeed")

	// List should show both
	stdout, _, exitCode := run(t, dir, "spec", "list")
	assert.Equal(t, 0, exitCode)

	var specs []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(stdout)), &specs))
	assert.GreaterOrEqual(t, len(specs), 2, "should have at least 2 specs")
}

func TestNextSpecApprovedWithTddModeShowsPreExecutionPrompt(t *testing.T) {
	dir := t.TempDir()
	specName := "tdd-pre-exec"
	seedSpecApprovedState(t, dir, specName, true)

	stdout, stderr, exitCode := run(t, dir, "next", "--spec="+specName)
	require.Equalf(t, 0, exitCode, "next should succeed: %s", stderr)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(stdout)), &payload))
	assert.Equal(t, "SPEC_APPROVED", payload["phase"])

	tddMode, ok := payload["tddMode"].(map[string]interface{})
	require.True(t, ok, "tddMode prompt should be present")
	assert.Equal(t, true, tddMode["active"])

	instruction, ok := tddMode["instruction"].(string)
	require.True(t, ok)
	assert.Contains(t, instruction, "TDD mode is active")
	assert.Contains(t, instruction, "Confirm to proceed")
}

func TestNextSpecApprovedWithoutTddModeSkipsPreExecutionPrompt(t *testing.T) {
	dir := t.TempDir()
	specName := "non-tdd-pre-exec"
	seedSpecApprovedState(t, dir, specName, false)

	stdout, stderr, exitCode := run(t, dir, "next", "--spec="+specName)
	require.Equalf(t, 0, exitCode, "next should succeed: %s", stderr)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(stdout)), &payload))
	assert.Equal(t, "SPEC_APPROVED", payload["phase"])

	_, exists := payload["tddMode"]
	assert.False(t, exists, "tddMode prompt should be omitted when tdd.tddMode=false")
}
