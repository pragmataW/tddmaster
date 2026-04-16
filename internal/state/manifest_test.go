package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// ParseManifest tests
// =============================================================================

func TestParseManifest_TddModeTrue(t *testing.T) {
	yaml := []byte(`tddMode: true`)
	m, err := ParseManifest(yaml)
	require.NoError(t, err)
	assert.True(t, m.TddMode, "tddMode should be true when set to true in YAML")
}

func TestParseManifest_TddModeFalse(t *testing.T) {
	yaml := []byte(`tddMode: false`)
	m, err := ParseManifest(yaml)
	require.NoError(t, err)
	assert.False(t, m.TddMode, "tddMode should be false when set to false in YAML")
}

func TestParseManifest_TddModeDefault(t *testing.T) {
	// When tddMode is not present in YAML, it must default to false.
	yaml := []byte(`testRunner: "go test ./..."`)
	m, err := ParseManifest(yaml)
	require.NoError(t, err)
	assert.False(t, m.TddMode, "tddMode should default to false when not specified")
}

func TestParseManifest_TestRunnerPresent(t *testing.T) {
	yaml := []byte(`testRunner: "go test ./..."`)
	m, err := ParseManifest(yaml)
	require.NoError(t, err)
	require.NotNil(t, m.TestRunner, "testRunner should not be nil when specified")
	assert.Equal(t, "go test ./...", *m.TestRunner)
}

func TestParseManifest_TestRunnerAbsent(t *testing.T) {
	// When testRunner is not present in YAML, it must be nil.
	yaml := []byte(`tddMode: true`)
	m, err := ParseManifest(yaml)
	require.NoError(t, err)
	assert.Nil(t, m.TestRunner, "testRunner should be nil when not specified")
}

func TestParseManifest_TestRunnerNull(t *testing.T) {
	// Explicit null in YAML should also result in nil pointer.
	yaml := []byte(`testRunner: null`)
	m, err := ParseManifest(yaml)
	require.NoError(t, err)
	assert.Nil(t, m.TestRunner, "testRunner should be nil when explicitly set to null")
}

func TestParseManifest_BothFields(t *testing.T) {
	yaml := []byte(`
tddMode: true
testRunner: "npm test"
`)
	m, err := ParseManifest(yaml)
	require.NoError(t, err)
	assert.True(t, m.TddMode)
	require.NotNil(t, m.TestRunner)
	assert.Equal(t, "npm test", *m.TestRunner)
}

func TestParseManifest_EmptyDocument(t *testing.T) {
	// An empty YAML document should produce zero values.
	m, err := ParseManifest([]byte(`{}`))
	require.NoError(t, err)
	assert.False(t, m.TddMode, "tddMode should default to false for empty document")
	assert.Nil(t, m.TestRunner, "testRunner should be nil for empty document")
}

func TestParseManifest_InvalidYAML(t *testing.T) {
	// Use a string that gopkg.in/yaml.v3 actually rejects.
	_, err := ParseManifest([]byte(`tddMode: @@@`))
	assert.Error(t, err, "invalid YAML should return an error")
}

// =============================================================================
// MarshalManifest tests
// =============================================================================

func TestMarshalManifest_TddModeTrue(t *testing.T) {
	m := Manifest{TddMode: true}
	data, err := MarshalManifest(m)
	require.NoError(t, err)

	// Round-trip: parse back and verify
	parsed, err := ParseManifest(data)
	require.NoError(t, err)
	assert.True(t, parsed.TddMode)
	assert.Nil(t, parsed.TestRunner)
}

func TestMarshalManifest_TestRunnerSet(t *testing.T) {
	runner := "go test -race ./..."
	m := Manifest{TddMode: false, TestRunner: &runner}
	data, err := MarshalManifest(m)
	require.NoError(t, err)

	parsed, err := ParseManifest(data)
	require.NoError(t, err)
	assert.False(t, parsed.TddMode)
	require.NotNil(t, parsed.TestRunner)
	assert.Equal(t, runner, *parsed.TestRunner)
}

func TestMarshalManifest_TestRunnerNil(t *testing.T) {
	m := Manifest{TddMode: false, TestRunner: nil}
	data, err := MarshalManifest(m)
	require.NoError(t, err)

	parsed, err := ParseManifest(data)
	require.NoError(t, err)
	assert.Nil(t, parsed.TestRunner)
}

func TestMarshalManifest_RoundTrip(t *testing.T) {
	runner := "make test"
	original := Manifest{TddMode: true, TestRunner: &runner}

	data, err := MarshalManifest(original)
	require.NoError(t, err)

	restored, err := ParseManifest(data)
	require.NoError(t, err)

	assert.Equal(t, original.TddMode, restored.TddMode)
	require.NotNil(t, restored.TestRunner)
	assert.Equal(t, *original.TestRunner, *restored.TestRunner)
}

// =============================================================================
// LoadManifest tests
// =============================================================================

func TestLoadManifest_ReadsTddInsideTddmasterSection(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, TddmasterDir), 0o755))

	manifestPath := filepath.Join(dir, TddmasterDir, "manifest.yml")
	err := os.WriteFile(manifestPath, []byte(`
tddmaster:
  command: custom-prefix
  allowGit: false
  tdd:
    tddMode: true
    testRunner: "go test ./..."
`), 0o644)
	require.NoError(t, err)

	loaded, err := LoadManifest(dir)
	require.NoError(t, err)
	assert.True(t, loaded.TddMode)
	require.NotNil(t, loaded.TestRunner)
	assert.Equal(t, "go test ./...", *loaded.TestRunner)
}

func TestLoadManifest_FallsBackToTopLevelTddKey(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, TddmasterDir), 0o755))

	manifestPath := filepath.Join(dir, TddmasterDir, "manifest.yml")
	err := os.WriteFile(manifestPath, []byte(`
tddmaster:
  command: custom-prefix
tdd:
  tddMode: true
  testRunner: "go test ./..."
`), 0o644)
	require.NoError(t, err)

	loaded, err := LoadManifest(dir)
	require.NoError(t, err)
	assert.True(t, loaded.TddMode)
	require.NotNil(t, loaded.TestRunner)
	assert.Equal(t, "go test ./...", *loaded.TestRunner)
}

func TestLoadManifest_WithoutNestedTddSectionReturnsZeroValue(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, TddmasterDir), 0o755))

	manifestPath := filepath.Join(dir, TddmasterDir, "manifest.yml")
	err := os.WriteFile(manifestPath, []byte(`
tddmaster:
  command: tddmaster
  allowGit: false
`), 0o644)
	require.NoError(t, err)

	loaded, err := LoadManifest(dir)
	require.NoError(t, err)
	assert.False(t, loaded.TddMode)
	assert.Nil(t, loaded.TestRunner)
}
