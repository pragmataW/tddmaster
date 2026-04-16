
package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/pragmataW/tddmaster/internal/state"
)

// =============================================================================
// spec.md updates
// =============================================================================

// UpdateSpecStatus updates the "## Status:" line in spec.md.
func UpdateSpecStatus(root, specName, newStatus string) error {
	p := state.Paths{}
	specFile := filepath.Join(root, p.SpecFile(specName))

	data, err := os.ReadFile(specFile)
	if err != nil {
		// spec.md doesn't exist yet — silently ignore
		return nil
	}

	re := regexp.MustCompile(`(?m)^## Status: .+$`)
	updated := re.ReplaceAllString(string(data), fmt.Sprintf("## Status: %s", newStatus))

	return state.WriteFileAtomic(specFile, []byte(updated), 0644)
}

// MarkTaskCompleted marks a task as completed in spec.md: "- [ ] task-N:" → "- [x] task-N:".
func MarkTaskCompleted(root, specName, taskID string) error {
	p := state.Paths{}
	specFile := filepath.Join(root, p.SpecFile(specName))

	data, err := os.ReadFile(specFile)
	if err != nil {
		// spec.md doesn't exist — silently ignore
		return nil
	}

	// Replace "- [ ] task-N:" with "- [x] task-N:"
	pattern := regexp.MustCompile(fmt.Sprintf(`(?m)^(- )\[ \]( %s:.*)$`, regexp.QuoteMeta(taskID)))
	updated := pattern.ReplaceAllString(string(data), "${1}[x]${2}")

	return state.WriteFileAtomic(specFile, []byte(updated), 0644)
}

// =============================================================================
// progress.json updates
// =============================================================================

// UpdateProgressTask updates a task's status in progress.json.
func UpdateProgressTask(root, specName, taskID, status string) error {
	p := state.Paths{}
	progressFile := filepath.Join(root, p.SpecDir(specName), "progress.json")

	data, err := os.ReadFile(progressFile)
	if err != nil {
		// progress.json doesn't exist — silently ignore
		return nil
	}

	var progress map[string]interface{}
	if err := json.Unmarshal(data, &progress); err != nil {
		return fmt.Errorf("parsing progress.json: %w", err)
	}

	tasks, ok := progress["tasks"].([]interface{})
	if ok {
		for _, t := range tasks {
			task, ok := t.(map[string]interface{})
			if !ok {
				continue
			}
			if task["id"] == taskID {
				task["status"] = status
			}
		}
	}

	progress["updatedAt"] = time.Now().UTC().Format(time.RFC3339)

	out, err := json.MarshalIndent(progress, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling progress.json: %w", err)
	}
	out = append(out, '\n')

	return state.WriteFileAtomic(progressFile, out, 0644)
}

// UpdateProgressStatus updates the spec status field in progress.json.
func UpdateProgressStatus(root, specName, status string) error {
	p := state.Paths{}
	progressFile := filepath.Join(root, p.SpecDir(specName), "progress.json")

	data, err := os.ReadFile(progressFile)
	if err != nil {
		// progress.json doesn't exist — silently ignore
		return nil
	}

	var progress map[string]interface{}
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
