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

	existing := readExistingProgress(progressPath)
	progress := buildProgress(*st.Spec, parsed.Tasks, st.OverrideTasks, st.Decisions, existing)

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

// existingProgressData captures fields preserved across progress.json regenerations:
// per-task status + important flag, and the full TaskPlans slice (user-approved
// plans must survive regen so the executor keeps receiving them).
type existingProgressData struct {
	Statuses   map[string]string
	Importants map[string]bool
	TaskPlans  []model.ProgressTaskPlan
}

func readExistingProgress(path string) existingProgressData {
	out := existingProgressData{
		Statuses:   map[string]string{},
		Importants: map[string]bool{},
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	var existing struct {
		Tasks []struct {
			ID        string `json:"id"`
			Status    string `json:"status"`
			Important bool   `json:"important"`
		} `json:"tasks"`
		TaskPlans []model.ProgressTaskPlan `json:"taskPlans"`
	}
	if json.Unmarshal(data, &existing) != nil {
		return out
	}
	for _, t := range existing.Tasks {
		if t.Status != "" {
			out.Statuses[t.ID] = t.Status
		}
		if t.Important {
			out.Importants[t.ID] = true
		}
	}
	out.TaskPlans = existing.TaskPlans
	return out
}

func buildProgress(specName string, parsedTasks []model.ParsedTask, overrides []state.SpecTask, decisions []state.Decision, existing existingProgressData) model.ProgressFile {
	overrideCompleted := map[string]bool{}
	overrideImportant := map[string]*bool{}
	for _, ot := range overrides {
		if ot.Completed {
			overrideCompleted[ot.ID] = true
		}
		if ot.Important != nil {
			overrideImportant[ot.ID] = ot.Important
		}
	}

	tasks := make([]model.ProgressTask, len(parsedTasks))
	for i, t := range parsedTasks {
		status := "pending"
		if s, ok := existing.Statuses[t.ID]; ok {
			status = s
		}
		if overrideCompleted[t.ID] {
			status = "done"
		}

		// Override (state) wins over previous progress.json snapshot.
		important := existing.Importants[t.ID]
		if v, ok := overrideImportant[t.ID]; ok && v != nil {
			important = *v
		}

		tasks[i] = model.ProgressTask{ID: t.ID, Title: t.Title, Status: status, Important: important}
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
		TaskPlans: existing.TaskPlans,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}
