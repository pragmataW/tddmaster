package spec

import (
	"fmt"
	"slices"
	"strings"
)

type DAGErrorKind string

const (
	DAGErrorSelf    DAGErrorKind = "self"
	DAGErrorUnknown DAGErrorKind = "unknown"
	DAGErrorCycle   DAGErrorKind = "cycle"
)

type dagColor int

const (
	colorWhite dagColor = iota
	colorGray
	colorBlack
)

type DAGError struct {
	Kind   DAGErrorKind
	TaskID string
	DepID  string
	Cycle  []string
}

func (e *DAGError) Error() string {
	switch e.Kind {
	case DAGErrorSelf:
		return fmt.Sprintf("task %s cannot depend on itself", e.TaskID)
	case DAGErrorUnknown:
		return fmt.Sprintf("task %s depends on unknown task id: %s", e.TaskID, e.DepID)
	case DAGErrorCycle:
		return fmt.Sprintf("dependency cycle detected: %s", strings.Join(e.Cycle, " -> "))
	default:
		return fmt.Sprintf("invalid dependency on task %s", e.TaskID)
	}
}

func ValidateDAG(tasks []Task) error {
	if issues := collectDAGIssues(tasks); len(issues) > 0 {
		first := issues[0]
		return &first
	}
	return nil
}

func collectDAGIssues(tasks []Task) []DAGError {
	byID := make(map[string]Task, len(tasks))
	for _, t := range tasks {
		byID[t.ID] = t
	}

	var issues []DAGError
	for _, t := range tasks {
		for _, dep := range t.DependsOn {
			if dep == t.ID {
				issues = append(issues, DAGError{Kind: DAGErrorSelf, TaskID: t.ID})
			} else if _, ok := byID[dep]; !ok {
				issues = append(issues, DAGError{Kind: DAGErrorUnknown, TaskID: t.ID, DepID: dep})
			}
		}
	}

	noSelf := make(map[string]Task, len(tasks))
	for _, t := range tasks {
		filtered := t
		filtered.DependsOn = nil
		for _, dep := range t.DependsOn {
			if dep != t.ID {
				filtered.DependsOn = append(filtered.DependsOn, dep)
			}
		}
		noSelf[t.ID] = filtered
	}

	colors := make(map[string]dagColor, len(tasks))
	for _, t := range tasks {
		if colors[t.ID] == colorWhite {
			for _, cycle := range detectCycles(t.ID, noSelf, colors, nil) {
				issues = append(issues, DAGError{Kind: DAGErrorCycle, Cycle: cycle})
			}
		}
	}
	return issues
}

func detectCycles(id string, byID map[string]Task, colors map[string]dagColor, path []string) [][]string {
	colors[id] = colorGray
	path = append(path, id)
	var cycles [][]string
	for _, dep := range byID[id].DependsOn {
		switch colors[dep] {
		case colorGray:
			cycles = append(cycles, append(cycleTail(path, dep), dep))
		case colorWhite:
			cycles = append(cycles, detectCycles(dep, byID, colors, path)...)
		}
	}
	colors[id] = colorBlack
	return cycles
}

func cycleTail(path []string, start string) []string {
	for i, id := range path {
		if id == start {
			return append([]string(nil), path[i:]...)
		}
	}
	return append([]string(nil), path...)
}

func BlockedSet(tasks []Task) map[string]bool {
	blocked := make(map[string]bool)
	for _, t := range tasks {
		if t.Blocked && !t.Done {
			blocked[t.ID] = true
		}
	}
	for changed := true; changed; {
		changed = false
		for _, t := range tasks {
			if t.Done || blocked[t.ID] {
				continue
			}
			for _, dep := range t.DependsOn {
				if blocked[dep] {
					blocked[t.ID] = true
					changed = true
					break
				}
			}
		}
	}
	return blocked
}

func ReadyTaskIndices(tasks []Task) []int {
	done := make(map[string]bool, len(tasks))
	for _, t := range tasks {
		if t.Done {
			done[t.ID] = true
		}
	}
	blocked := BlockedSet(tasks)

	var ready []int
	for i, t := range tasks {
		if t.Done || blocked[t.ID] {
			continue
		}
		ok := true
		for _, dep := range t.DependsOn {
			if !done[dep] {
				ok = false
				break
			}
		}
		if ok {
			ready = append(ready, i)
		}
	}
	return ready
}

func DependentsOf(tasks []Task, id string) []string {
	var deps []string
	for _, t := range tasks {
		if slices.Contains(t.DependsOn, id) {
			deps = append(deps, t.ID)
		}
	}
	return deps
}

func LintDependencies(tasks []Task) []Finding {
	findings := []Finding{}
	for _, issue := range collectDAGIssues(tasks) {
		findings = append(findings, dagFinding(issue))
	}
	return findings
}

func dagFinding(e DAGError) Finding {
	switch e.Kind {
	case DAGErrorSelf:
		return Finding{
			Severity: SeverityBlock,
			Category: "dep-self",
			TaskID:   e.TaskID,
			Detail:   "Task depends on itself",
			Source:   SourceLinter,
		}
	case DAGErrorUnknown:
		return Finding{
			Severity: SeverityBlock,
			Category: "dep-unknown",
			TaskID:   e.TaskID,
			Detail:   fmt.Sprintf("Task depends on unknown task id: %s", e.DepID),
			Source:   SourceLinter,
		}
	default:
		return Finding{
			Severity: SeverityBlock,
			Category: "dep-cycle",
			TaskID:   e.Cycle[0],
			Detail:   "Dependency cycle detected: " + strings.Join(e.Cycle, " -> "),
			Source:   SourceLinter,
		}
	}
}
