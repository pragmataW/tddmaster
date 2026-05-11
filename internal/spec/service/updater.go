package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/pragmataW/tddmaster/internal/spec/model"
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

// AppendTaskPlan appends or replaces a user-approved ProgressTaskPlan in
// progress.json. A plan keyed to an existing TaskID is overwritten (re-approval
// after revise/reject loop wins). Errors when progress.json is missing — gate
// flow requires the plan be persisted so it can be injected into downstream
// executor spawns.
func AppendTaskPlan(root, specName string, plan model.ProgressTaskPlan) error {
	progressFile := filepath.Join(root, state.Paths{}.SpecDir(specName), "progress.json")

	data, err := os.ReadFile(progressFile)
	if err != nil {
		return fmt.Errorf("reading progress.json for spec %q: %w", specName, err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parsing progress.json: %w", err)
	}

	plans, _ := raw["taskPlans"].([]any)
	replaced := false
	for i, p := range plans {
		pm, ok := p.(map[string]any)
		if !ok {
			continue
		}
		if id, _ := pm["taskId"].(string); id == plan.TaskID {
			b, err := json.Marshal(plan)
			if err != nil {
				return fmt.Errorf("marshaling plan: %w", err)
			}
			var pmNew map[string]any
			if err := json.Unmarshal(b, &pmNew); err != nil {
				return fmt.Errorf("re-encoding plan: %w", err)
			}
			plans[i] = pmNew
			replaced = true
			break
		}
	}
	if !replaced {
		b, err := json.Marshal(plan)
		if err != nil {
			return fmt.Errorf("marshaling plan: %w", err)
		}
		var pmNew map[string]any
		if err := json.Unmarshal(b, &pmNew); err != nil {
			return fmt.Errorf("re-encoding plan: %w", err)
		}
		plans = append(plans, pmNew)
	}
	raw["taskPlans"] = plans
	raw["updatedAt"] = time.Now().UTC().Format(time.RFC3339)

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling progress.json: %w", err)
	}
	out = append(out, '\n')
	return state.WriteFileAtomic(progressFile, out, 0644)
}

// MarkTaskImportant toggles the `important` flag on a single progress.json task
// row. Silently no-ops when progress.json or the task ID is missing.
func MarkTaskImportant(root, specName, taskID string, important bool) error {
	progressFile := filepath.Join(root, state.Paths{}.SpecDir(specName), "progress.json")

	data, err := os.ReadFile(progressFile)
	if err != nil {
		return nil
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parsing progress.json: %w", err)
	}

	tasks, _ := raw["tasks"].([]any)
	for i, t := range tasks {
		tm, ok := t.(map[string]any)
		if !ok {
			continue
		}
		if id, _ := tm["id"].(string); id == taskID {
			if important {
				tm["important"] = true
			} else {
				delete(tm, "important")
			}
			tasks[i] = tm
			break
		}
	}
	raw["tasks"] = tasks
	raw["updatedAt"] = time.Now().UTC().Format(time.RFC3339)

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling progress.json: %w", err)
	}
	out = append(out, '\n')
	return state.WriteFileAtomic(progressFile, out, 0644)
}

// LoadTaskPlan returns the user-approved plan for the given taskID, or nil
// when no plan has been approved yet. Read-only — used by compile to embed
// the plan into the executor prompt.
func LoadTaskPlan(root, specName, taskID string) (*model.ProgressTaskPlan, error) {
	progressFile := filepath.Join(root, state.Paths{}.SpecDir(specName), "progress.json")

	data, err := os.ReadFile(progressFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var pf model.ProgressFile
	if err := json.Unmarshal(data, &pf); err != nil {
		return nil, fmt.Errorf("parsing progress.json: %w", err)
	}
	for i := range pf.TaskPlans {
		if pf.TaskPlans[i].TaskID == taskID {
			return &pf.TaskPlans[i], nil
		}
	}
	return nil, nil
}
