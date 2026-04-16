package state

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteFileAtomic_WritesContent(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "out.json")

	err := WriteFileAtomic(target, []byte("hello"), 0o644)
	require.NoError(t, err)

	got, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(got))
}

func TestWriteFileAtomic_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "out.json")

	require.NoError(t, os.WriteFile(target, []byte("old"), 0o644))
	require.NoError(t, WriteFileAtomic(target, []byte("new"), 0o644))

	got, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, "new", string(got))
}

func TestWriteFileAtomic_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "a", "b", "c", "out.json")

	require.NoError(t, WriteFileAtomic(target, []byte("x"), 0o644))

	got, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, "x", string(got))
}

func TestWriteFileAtomic_NoTempLeftOnSuccess(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "out.json")

	require.NoError(t, WriteFileAtomic(target, []byte("x"), 0o644))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		assert.False(t, strings.Contains(e.Name(), ".tmp-"),
			"temp file leaked: %s", e.Name())
	}
}

func TestWriteFileAtomic_CleansUpTempOnError(t *testing.T) {
	// Read-only parent dir → OpenFile for tmp fails.
	parent := t.TempDir()
	roDir := filepath.Join(parent, "ro")
	require.NoError(t, os.MkdirAll(roDir, 0o755))
	require.NoError(t, os.Chmod(roDir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(roDir, 0o755) })

	target := filepath.Join(roDir, "out.json")
	err := WriteFileAtomic(target, []byte("x"), 0o644)
	require.Error(t, err)

	entries, _ := os.ReadDir(roDir)
	for _, e := range entries {
		assert.False(t, strings.Contains(e.Name(), ".tmp-"),
			"temp file leaked on error: %s", e.Name())
	}
}

func TestWriteFileAtomic_ConcurrentWrites_NeverCorrupt(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "out.json")

	// Each writer writes a single-byte content from a small alphabet.
	// After the storm ends, the file must contain one of those exact bytes —
	// never a partial/mixed payload.
	const goroutines = 10
	const iters = 50
	valid := map[string]bool{"A": true, "B": true, "C": true, "D": true, "E": true}
	payloads := []string{"A", "B", "C", "D", "E"}

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := range goroutines {
		go func() {
			defer wg.Done()
			for i := range iters {
				p := payloads[(g+i)%len(payloads)]
				if err := WriteFileAtomic(target, []byte(p), 0o644); err != nil {
					t.Errorf("write failed: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()

	got, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.True(t, valid[string(got)], "file corrupted; got %q", string(got))

	// No temp files should remain.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		assert.False(t, strings.Contains(e.Name(), ".tmp-"),
			"temp file leaked after concurrent writes: %s", e.Name())
	}
}
