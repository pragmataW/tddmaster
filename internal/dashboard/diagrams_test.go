
package dashboard

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadWriteRegistry(t *testing.T) {
	dir := t.TempDir()

	// Create .tddmaster dir
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".tddmaster"), 0o755))

	entries := []DiagramEntry{
		{
			File:            "README.md",
			Line:            10,
			Type:            DiagramTypeMermaid,
			Hash:            "abc123",
			ReferencedFiles: []string{"main.go"},
			LastVerified:    "2024-01-01T10:00:00Z",
		},
	}

	require.NoError(t, WriteRegistry(dir, entries))

	loaded, err := ReadRegistry(dir)
	require.NoError(t, err)
	assert.Len(t, loaded, 1)
	assert.Equal(t, "README.md", loaded[0].File)
	assert.Equal(t, DiagramTypeMermaid, loaded[0].Type)
}

func TestReadRegistry_Empty(t *testing.T) {
	dir := t.TempDir()
	entries, err := ReadRegistry(dir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestVerifyDiagram(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".tddmaster"), 0o755))

	entries := []DiagramEntry{
		{
			File:            "docs/arch.md",
			Line:            5,
			Type:            DiagramTypeMermaid,
			Hash:            "xyz",
			ReferencedFiles: []string{},
			LastVerified:    "2020-01-01T00:00:00Z",
		},
	}
	require.NoError(t, WriteRegistry(dir, entries))

	found, err := VerifyDiagram(dir, "docs/arch.md", nil)
	require.NoError(t, err)
	assert.True(t, found)

	updated, err := ReadRegistry(dir)
	require.NoError(t, err)
	assert.NotEqual(t, "2020-01-01T00:00:00Z", updated[0].LastVerified)
}

func TestVerifyDiagram_NotFound(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".tddmaster"), 0o755))

	entries := []DiagramEntry{}
	require.NoError(t, WriteRegistry(dir, entries))

	found, err := VerifyDiagram(dir, "nonexistent.md", nil)
	require.NoError(t, err)
	assert.False(t, found)
}

func TestCheckStaleness(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".tddmaster"), 0o755))

	entries := []DiagramEntry{
		{
			File:            "README.md",
			Line:            1,
			Type:            DiagramTypeMermaid,
			Hash:            "abc",
			ReferencedFiles: []string{"internal/state/schema.go"},
			LastVerified:    "2024-01-01T10:00:00Z",
		},
	}
	require.NoError(t, WriteRegistry(dir, entries))

	stale, err := CheckStaleness(dir, []string{"internal/state/schema.go"})
	require.NoError(t, err)
	assert.Len(t, stale, 1)
	assert.Equal(t, "README.md", stale[0].File)
}

func TestCheckStaleness_NoMatch(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".tddmaster"), 0o755))

	entries := []DiagramEntry{
		{
			File:            "README.md",
			Line:            1,
			Type:            DiagramTypeMermaid,
			Hash:            "abc",
			ReferencedFiles: []string{"other/file.go"},
			LastVerified:    "2024-01-01T10:00:00Z",
		},
	}
	require.NoError(t, WriteRegistry(dir, entries))

	stale, err := CheckStaleness(dir, []string{"unrelated/path.go"})
	require.NoError(t, err)
	assert.Empty(t, stale)
}

func TestScanProject_Empty(t *testing.T) {
	dir := t.TempDir()
	entries, err := ScanProject(dir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestScanProject_MermaidBlock(t *testing.T) {
	dir := t.TempDir()

	mdContent := "# Architecture\n\n```mermaid\ngraph TD\n  A --> B\n```\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte(mdContent), 0o644))

	entries, err := ScanProject(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "README.md", entries[0].File)
	assert.Equal(t, DiagramTypeMermaid, entries[0].Type)
}

func TestHashContent(t *testing.T) {
	h1 := hashContent("hello world")
	h2 := hashContent("hello world")
	h3 := hashContent("different content")

	assert.Equal(t, h1, h2)
	assert.NotEqual(t, h1, h3)
}

func TestExtractReferences(t *testing.T) {
	content := "See internal/state/schema.go and main.ts for details"
	refs := extractReferences(content)
	assert.Contains(t, refs, "internal/state/schema.go")
	assert.Contains(t, refs, "main.ts")
}
