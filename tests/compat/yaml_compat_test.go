// Package compat contains compatibility tests for YAML manifest round-trips.
package compat

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/pragmataW/tddmaster/internal/state"
)

// TestManifestRoundTrip verifies that a NosManifest can be written to YAML and read back.
func TestManifestRoundTrip(t *testing.T) {
	dir := t.TempDir()

	testRunner := "go test ./..."
	verifyCmd := "go build ./..."
	user := &state.UserConfig{Name: "Test User", Email: "test@example.com"}

	original := state.CreateInitialManifest(
		[]string{"security", "performance"},
		[]state.CodingToolId{state.CodingToolClaudeCode, state.CodingToolCodex},
		state.ProjectTraits{
			Languages:  []string{"go", "typescript"},
			Frameworks: []string{"cobra", "react"},
			CI:         []string{"github-actions"},
			TestRunner: &testRunner,
		},
	)
	original.VerifyCommand = &verifyCmd
	original.AllowGit = true
	original.MaxIterationsBeforeRestart = 20
	original.User = user

	// Write
	err := state.WriteManifest(dir, original)
	require.NoError(t, err)

	// Verify file exists
	manifestPath := filepath.Join(dir, ".tddmaster", "manifest.yml")
	_, err = os.Stat(manifestPath)
	require.NoError(t, err, "manifest.yml should exist")

	// Read back
	loaded, err := state.ReadManifest(dir)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, original.Concerns, loaded.Concerns)
	assert.Equal(t, original.Tools, loaded.Tools)
	assert.Equal(t, original.Project.Languages, loaded.Project.Languages)
	assert.Equal(t, original.Project.Frameworks, loaded.Project.Frameworks)
	assert.Equal(t, original.Project.CI, loaded.Project.CI)
	require.NotNil(t, loaded.Project.TestRunner)
	assert.Equal(t, testRunner, *loaded.Project.TestRunner)
	require.NotNil(t, loaded.VerifyCommand)
	assert.Equal(t, verifyCmd, *loaded.VerifyCommand)
	assert.Equal(t, true, loaded.AllowGit)
	assert.Equal(t, 20, loaded.MaxIterationsBeforeRestart)
	require.NotNil(t, loaded.User)
	assert.Equal(t, "Test User", loaded.User.Name)
	assert.Equal(t, "test@example.com", loaded.User.Email)
}

// TestManifestYAMLStructure verifies the YAML output has the expected structure
// under the "tddmaster" key (TS-compatible format).
func TestManifestYAMLStructure(t *testing.T) {
	dir := t.TempDir()

	original := state.CreateInitialManifest(
		[]string{"accessibility"},
		[]state.CodingToolId{state.CodingToolClaudeCode},
		state.ProjectTraits{
			Languages:  []string{"go"},
			Frameworks: []string{},
			CI:         []string{},
		},
	)

	err := state.WriteManifest(dir, original)
	require.NoError(t, err)

	manifestPath := filepath.Join(dir, ".tddmaster", "manifest.yml")
	data, err := os.ReadFile(manifestPath)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &raw))

	// Top-level must have "tddmaster" key
	require.Contains(t, raw, "tddmaster")

	nos, ok := raw["tddmaster"].(map[string]interface{})
	require.True(t, ok, "tddmaster value should be a map")

	// Required keys in tddmaster section
	assert.Contains(t, nos, "concerns")
	assert.Contains(t, nos, "tools")
	assert.Contains(t, nos, "project")
	assert.Contains(t, nos, "maxIterationsBeforeRestart")
	assert.Contains(t, nos, "allowGit")
	assert.Contains(t, nos, "command")

	// project sub-map
	proj, ok := nos["project"].(map[string]interface{})
	require.True(t, ok, "project should be a map")
	assert.Contains(t, proj, "languages")
	assert.Contains(t, proj, "frameworks")
	assert.Contains(t, proj, "ci")
}

// TestManifestPreservesOtherKeys verifies that other top-level YAML keys are preserved
// when writing the tddmaster section (important for TS compatibility).
func TestManifestPreservesOtherKeys(t *testing.T) {
	dir := t.TempDir()

	// Write a YAML file with extra keys first
	manifestDir := filepath.Join(dir, ".tddmaster")
	require.NoError(t, os.MkdirAll(manifestDir, 0o755))
	manifestPath := filepath.Join(manifestDir, "manifest.yml")

	existingYAML := `project:
  name: myproject
  version: "1.0"
custom_key: custom_value
`
	require.NoError(t, os.WriteFile(manifestPath, []byte(existingYAML), 0o644))

	// Now write tddmaster section
	manifest := state.CreateInitialManifest(
		[]string{},
		[]state.CodingToolId{},
		state.ProjectTraits{Languages: []string{"go"}, Frameworks: []string{}, CI: []string{}},
	)
	err := state.WriteManifest(dir, manifest)
	require.NoError(t, err)

	// Read back the file
	data, err := os.ReadFile(manifestPath)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &raw))

	// Both the tddmaster key and the original keys should be present
	assert.Contains(t, raw, "tddmaster", "tddmaster key should be added")
	assert.Contains(t, raw, "project", "original project key should be preserved")
	assert.Contains(t, raw, "custom_key", "custom_key should be preserved")
	assert.Equal(t, "custom_value", raw["custom_key"])
}

// TestManifestReadMissingFile verifies ReadManifest returns nil when file doesn't exist.
func TestManifestReadMissingFile(t *testing.T) {
	dir := t.TempDir()

	result, err := state.ReadManifest(dir)
	require.NoError(t, err)
	assert.Nil(t, result, "should return nil when manifest doesn't exist")
}

// TestManifestDefaultValues verifies that CreateInitialManifest produces expected defaults.
func TestManifestDefaultValues(t *testing.T) {
	manifest := state.CreateInitialManifest(
		[]string{},
		[]state.CodingToolId{},
		state.ProjectTraits{Languages: []string{}, Frameworks: []string{}, CI: []string{}},
	)

	assert.Equal(t, 15, manifest.MaxIterationsBeforeRestart)
	assert.Equal(t, false, manifest.AllowGit)
	assert.Nil(t, manifest.VerifyCommand)
	assert.Nil(t, manifest.User)
	assert.Equal(t, "tddmaster", manifest.Command)
}

// TestManifestTSCompatibleFieldNames verifies field names match TypeScript expectations.
// The TS side uses camelCase JSON tags; YAML tags must match the TS manifest schema.
func TestManifestTSCompatibleFieldNames(t *testing.T) {
	dir := t.TempDir()

	verifyCmd := "make test"
	manifest := state.CreateInitialManifest(
		[]string{"security"},
		[]state.CodingToolId{state.CodingToolCodex},
		state.ProjectTraits{
			Languages:  []string{"typescript"},
			Frameworks: []string{"next"},
			CI:         []string{"github-actions"},
		},
	)
	manifest.VerifyCommand = &verifyCmd
	manifest.AllowGit = true
	manifest.MaxIterationsBeforeRestart = 10

	err := state.WriteManifest(dir, manifest)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, ".tddmaster", "manifest.yml"))
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &raw))

	nos := raw["tddmaster"].(map[string]interface{})

	// camelCase field names as expected by TS
	assert.Contains(t, nos, "maxIterationsBeforeRestart")
	assert.Contains(t, nos, "verifyCommand")
	assert.Contains(t, nos, "allowGit")

	// Value checks
	assert.Equal(t, true, nos["allowGit"])
	assert.Equal(t, "make test", nos["verifyCommand"])
	assert.Equal(t, 10, nos["maxIterationsBeforeRestart"])

	proj := nos["project"].(map[string]interface{})
	assert.Equal(t, []interface{}{"typescript"}, proj["languages"])
}
