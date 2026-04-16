package state

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// WriteFileAtomic writes data to path via a temp file in the same directory,
// then rename. Ensures either the old content or the full new content is
// visible — no partial writes even on crash.
//
// Semantics:
//   - creates parent dirs (0o755) if missing
//   - writes to <path>.tmp-<random> in the same directory
//   - fsync temp file before rename so data is on disk
//   - rename overwrites target atomically on same filesystem
//   - fsync parent dir afterwards so the rename itself is durable
//   - on any error the temp file is cleaned up
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}

	var suffix [8]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return fmt.Errorf("random suffix: %w", err)
	}
	tmp := path + ".tmp-" + hex.EncodeToString(suffix[:])

	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("open tmp: %w", err)
	}

	cleanup := func() { _ = os.Remove(tmp) }

	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		cleanup()
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		cleanup()
		return fmt.Errorf("fsync tmp: %w", err)
	}
	if err := f.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		cleanup()
		return fmt.Errorf("rename: %w", err)
	}

	if dirf, derr := os.Open(dir); derr == nil {
		_ = dirf.Sync()
		_ = dirf.Close()
	}
	return nil
}
