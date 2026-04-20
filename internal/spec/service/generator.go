package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pragmataW/tddmaster/internal/spec/model"
	"github.com/pragmataW/tddmaster/internal/state"
)

// Generate writes spec.md + progress.json for the active spec in st.
// Returns the spec.md path. Existing progress.json task statuses are preserved
// across regenerations, with OverrideTasks[].Completed as the authoritative
// source of truth for completion.
func Generate(root string, st *state.StateFile, concerns []state.ConcernDefinition) (string, error) {
	if st.Spec == nil {
		return "", fmt.Errorf("no active spec")
	}

	p := state.Paths{}
	specDir := filepath.Join(root, p.SpecDir(*st.Spec))
	specFile := filepath.Join(root, p.SpecFile(*st.Spec))

	if err := os.MkdirAll(specDir, 0755); err != nil {
		return "", fmt.Errorf("creating spec dir: %w", err)
	}

	content := Render(
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

	parsed := parseContent(*st.Spec, content)
	progressPath := filepath.Join(specDir, "progress.json")

	progress := buildProgress(*st.Spec, parsed.Tasks, st.OverrideTasks, st.Decisions, readExistingStatuses(progressPath))

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

func readExistingStatuses(path string) map[string]string {
	statuses := map[string]string{}
	data, err := os.ReadFile(path)
	if err != nil {
		return statuses
	}
	var existing struct {
		Tasks []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"tasks"`
	}
	if json.Unmarshal(data, &existing) != nil {
		return statuses
	}
	for _, t := range existing.Tasks {
		if t.Status != "" {
			statuses[t.ID] = t.Status
		}
	}
	return statuses
}

func buildProgress(specName string, parsedTasks []model.ParsedTask, overrides []state.SpecTask, decisions []state.Decision, existing map[string]string) model.ProgressFile {
	overrideCompleted := map[string]bool{}
	for _, ot := range overrides {
		if ot.Completed {
			overrideCompleted[ot.ID] = true
		}
	}

	tasks := make([]model.ProgressTask, len(parsedTasks))
	for i, t := range parsedTasks {
		status := "pending"
		if s, ok := existing[t.ID]; ok {
			status = s
		}
		if overrideCompleted[t.ID] {
			status = "done"
		}
		tasks[i] = model.ProgressTask{ID: t.ID, Title: t.Title, Status: status}
	}

	decisionsCopy := make([]model.ProgressDecision, len(decisions))
	for i, d := range decisions {
		decisionsCopy[i] = model.ProgressDecision{
			Question: d.Question,
			Choice:   d.Choice,
			Promoted: d.Promoted,
		}
	}

	return model.ProgressFile{
		Spec:      specName,
		Status:    "draft",
		Tasks:     tasks,
		Decisions: decisionsCopy,
		Debt:      []any{},
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}
