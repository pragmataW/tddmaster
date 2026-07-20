package spec

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/pragmataW/tddmaster/internal/errs"
)

type RefineOp struct {
	Title      *string     `json:"title,omitempty"`
	Criteria   []Criterion `json:"criteria,omitempty"`
	TDDEnabled *bool       `json:"tddEnabled,omitempty"`
	Important  *bool       `json:"important,omitempty"`
	EdgeCases  []string    `json:"edgeCases,omitempty"`
	DependsOn  *[]string   `json:"dependsOn,omitempty"`
}

type RefinePayload struct {
	Add    []RefineOp          `json:"add,omitempty"`
	Remove []string            `json:"remove,omitempty"`
	Update map[string]RefineOp `json:"update,omitempty"`
}

func ApplyRefinement(tasks []Task, p RefinePayload, tddDefault bool, seq int) ([]Task, int, error) {
	result := make([]Task, len(tasks))
	copy(result, tasks)

	maxN := seq
	for _, t := range tasks {
		suffix := strings.TrimPrefix(t.ID, TaskIDPrefix)
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

	removed := make(map[string]bool, len(p.Remove))
	for _, id := range p.Remove {
		idx := -1
		for i, t := range result {
			if t.ID == id {
				idx = i
				break
			}
		}
		if idx == -1 {
			if removed[id] {
				return nil, seq, errs.Newf(errs.KeyDupTaskIDRemove, id)
			}
			return nil, seq, errs.Newf(errs.KeyUnknownTaskID, id)
		}
		result = append(result[:idx], result[idx+1:]...)
		removed[id] = true
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
			return nil, seq, errs.Newf(errs.KeyUnknownTaskID, id)
		}
		if op.Title != nil {
			result[idx].Title = *op.Title
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
		if op.Criteria != nil {
			result[idx].Criteria = op.Criteria
			AssignCriterionIDs(&result[idx])
		}
		if op.DependsOn != nil {
			result[idx].DependsOn = *op.DependsOn
		}
	}

	for _, op := range p.Add {
		if op.Title == nil || *op.Title == "" {
			return nil, seq, errs.New(errs.KeyAddRequiresTitle)
		}
		maxN++
		newTask := Task{
			ID:         fmt.Sprintf("%s%d", TaskIDPrefix, maxN),
			Title:      *op.Title,
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
		if op.Criteria != nil {
			newTask.Criteria = op.Criteria
			AssignCriterionIDs(&newTask)
		}
		if newTask.EdgeCases == nil {
			newTask.EdgeCases = []string{}
		}
		if op.DependsOn != nil {
			newTask.DependsOn = *op.DependsOn
		} else {
			newTask.DependsOn = []string{}
		}
		result = append(result, newTask)
	}

	if issues := collectDAGIssues(result); len(issues) > 0 {
		seen := make(map[string]bool, len(issues))
		var msgs []string
		for _, issue := range issues {
			var msg string
			if issue.Kind == DAGErrorUnknown && removed[issue.DepID] {
				dependents := DependentsOf(result, issue.DepID)
				msg = errs.Msgf(errs.KeyCannotRemoveDeps, issue.DepID, strings.Join(dependents, ", "))
			} else {
				msg = issue.Error()
			}
			if seen[msg] {
				continue
			}
			seen[msg] = true
			msgs = append(msgs, msg)
		}
		return nil, seq, errors.New(strings.Join(msgs, "; "))
	}

	return result, maxN, nil
}
