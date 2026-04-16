package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pragmataW/tddmaster/internal/state"
)

// =============================================================================
// Generate
// =============================================================================

// GenerateSpec creates a spec directory, writes spec.md, and creates progress.json.
// Returns the path to the spec.md file.
func GenerateSpec(root string, st *state.StateFile, concerns []state.ConcernDefinition) (string, error) {
	if st.Spec == nil {
		return "", fmt.Errorf("no active spec")
	}

	p := state.Paths{}
	specDir := filepath.Join(root, p.SpecDir(*st.Spec))
	specFile := filepath.Join(root, p.SpecFile(*st.Spec))

	if err := os.MkdirAll(specDir, 0755); err != nil {
		return "", fmt.Errorf("creating spec dir: %w", err)
	}

	content := RenderSpec(
		*st.Spec,
		st.Discovery.Answers,
		st.Discovery.Premises,
		concerns,
		st.Decisions,
		st.Classification,
		st.CustomACs,
		st.SpecNotes,
		st.TransitionHistory,
		false,
		WithTaskOverride(st.OverrideTasks),
		WithOutOfScopeOverride(st.OverrideOutOfScope),
	)

	if err := state.WriteFileAtomic(specFile, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("writing spec.md: %w", err)
	}

	// Parse the spec we just wrote to extract tasks
	parsed := ParseSpecContent(*st.Spec, content)

	// Generate initial progress.json
	type progressTask struct {
		ID     string `json:"id"`
		Title  string `json:"title"`
		Status string `json:"status"`
	}
	type progressDecision struct {
		Question string `json:"question"`
		Choice   string `json:"choice"`
		Promoted bool   `json:"promoted"`
	}
	type progressFile struct {
		Spec      string             `json:"spec"`
		Status    string             `json:"status"`
		Tasks     []progressTask     `json:"tasks"`
		Decisions []progressDecision `json:"decisions"`
		Debt      []interface{}      `json:"debt"`
		UpdatedAt string             `json:"updatedAt"`
	}

	// Read existing progress.json to preserve task statuses (e.g. "done") across
	// spec regenerations such as refinements.
	progressPath := filepath.Join(specDir, "progress.json")
	existingStatuses := map[string]string{}
	if data, err := os.ReadFile(progressPath); err == nil {
		var existing struct {
			Tasks []struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"tasks"`
		}
		if json.Unmarshal(data, &existing) == nil {
			for _, t := range existing.Tasks {
				if t.Status != "" {
					existingStatuses[t.ID] = t.Status
				}
			}
		}
	}

	// Build authoritative completed set from state (OverrideTasks[].Completed).
	// This is updated by applyExecutorReport and is the single source of truth
	// for task status — it keeps progress.json in sync with spec.md across
	// refinements and regenerations.
	overrideCompleted := map[string]bool{}
	for _, ot := range st.OverrideTasks {
		if ot.Completed {
			overrideCompleted[ot.ID] = true
		}
	}

	tasks := make([]progressTask, len(parsed.Tasks))
	for i, t := range parsed.Tasks {
		status := "pending"
		if s, ok := existingStatuses[t.ID]; ok {
			status = s
		}
		if overrideCompleted[t.ID] {
			status = "done"
		}
		tasks[i] = progressTask{
			ID:     t.ID,
			Title:  t.Title,
			Status: status,
		}
	}

	decisionsCopy := make([]progressDecision, len(st.Decisions))
	for i, d := range st.Decisions {
		decisionsCopy[i] = progressDecision{
			Question: d.Question,
			Choice:   d.Choice,
			Promoted: d.Promoted,
		}
	}

	progress := progressFile{
		Spec:      *st.Spec,
		Status:    "draft",
		Tasks:     tasks,
		Decisions: decisionsCopy,
		Debt:      []interface{}{},
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	progressBytes, err := json.MarshalIndent(progress, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling progress.json: %w", err)
	}
	progressBytes = append(progressBytes, '\n')

	if err := state.WriteFileAtomic(progressPath, progressBytes, 0644); err != nil {
		return "", fmt.Errorf("writing progress.json: %w", err)
	}

	return specFile, nil
}
