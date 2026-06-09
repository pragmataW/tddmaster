package spec

import (
	"fmt"
	"strconv"
	"strings"
)

type RefineOp struct {
	Title      *string  `json:"title,omitempty"`
	AC         []string `json:"ac,omitempty"`
	TDDEnabled *bool    `json:"tddEnabled,omitempty"`
	Important  *bool    `json:"important,omitempty"`
	EdgeCases  []string `json:"edgeCases,omitempty"`
}

type RefinePayload struct {
	Add    []RefineOp         `json:"add,omitempty"`
	Remove []string           `json:"remove,omitempty"`
	Update map[string]RefineOp `json:"update,omitempty"`
}

func ApplyRefinement(tasks []Task, p RefinePayload, tddDefault bool, seq int) ([]Task, int, error) {
	result := make([]Task, len(tasks))
	copy(result, tasks)

	maxN := seq
	for _, t := range tasks {
		suffix := strings.TrimPrefix(t.ID, "task-")
		if suffix == t.ID {
			continue
		}
		n, err := strconv.Atoi(suffix)
		if err != nil {
			continue
		}
		if n > maxN {
			maxN = n
		}
	}

	for _, id := range p.Remove {
		idx := -1
		for i, t := range result {
			if t.ID == id {
				idx = i
				break
			}
		}
		if idx == -1 {
			return nil, seq, fmt.Errorf("unknown task id: %s", id)
		}
		result = append(result[:idx], result[idx+1:]...)
	}

	for id, op := range p.Update {
		idx := -1
		for i, t := range result {
			if t.ID == id {
				idx = i
				break
			}
		}
		if idx == -1 {
			return nil, seq, fmt.Errorf("unknown task id: %s", id)
		}
		if op.Title != nil {
			result[idx].Title = *op.Title
		}
		if op.AC != nil {
			result[idx].AC = op.AC
		}
		if op.TDDEnabled != nil {
			result[idx].TDDEnabled = *op.TDDEnabled
		}
		if op.Important != nil {
			result[idx].Important = *op.Important
		}
		if op.EdgeCases != nil {
			result[idx].EdgeCases = op.EdgeCases
		}
	}

	for _, op := range p.Add {
		if op.Title == nil || *op.Title == "" {
			return nil, seq, fmt.Errorf("add op requires a non-empty title")
		}
		maxN++
		newTask := Task{
			ID:         fmt.Sprintf("task-%d", maxN),
			Title:      *op.Title,
			AC:         op.AC,
			Done:       false,
			TDDEnabled: tddDefault,
		}
		if op.TDDEnabled != nil {
			newTask.TDDEnabled = *op.TDDEnabled
		}
		if op.Important != nil {
			newTask.Important = *op.Important
		}
		newTask.EdgeCases = op.EdgeCases
		if newTask.AC == nil {
			newTask.AC = []string{}
		}
		if newTask.EdgeCases == nil {
			newTask.EdgeCases = []string{}
		}
		result = append(result, newTask)
	}

	return result, maxN, nil
}
