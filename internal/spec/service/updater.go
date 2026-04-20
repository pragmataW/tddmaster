package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/pragmataW/tddmaster/internal/state"
)

var statusLineRe = regexp.MustCompile(`(?m)^## Status: .+$`)

// UpdateSpecStatus updates the "## Status:" line in spec.md. Silently no-ops
// when the spec.md file does not exist.
func UpdateSpecStatus(root, specName, newStatus string) error {
	specFile := filepath.Join(root, state.Paths{}.SpecFile(specName))

	data, err := os.ReadFile(specFile)
	if err != nil {
		return nil
	}

	updated := statusLineRe.ReplaceAllString(string(data), fmt.Sprintf("## Status: %s", newStatus))
	return state.WriteFileAtomic(specFile, []byte(updated), 0644)
}

// UpdateProgressStatus updates the spec-level status field in progress.json.
// Silently no-ops when progress.json does not exist. Uses map[string]any
// round-trip to preserve unknown fields; see docs/bugs.md S7 for the known
// key-ordering limitation.
func UpdateProgressStatus(root, specName, status string) error {
	progressFile := filepath.Join(root, state.Paths{}.SpecDir(specName), "progress.json")

	data, err := os.ReadFile(progressFile)
	if err != nil {
		return nil
	}

	var progress map[string]any
	if err := json.Unmarshal(data, &progress); err != nil {
		return fmt.Errorf("parsing progress.json: %w", err)
	}

	progress["status"] = status
	progress["updatedAt"] = time.Now().UTC().Format(time.RFC3339)

	out, err := json.MarshalIndent(progress, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling progress.json: %w", err)
	}
	out = append(out, '\n')

	return state.WriteFileAtomic(progressFile, out, 0644)
}
