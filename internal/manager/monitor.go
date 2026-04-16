
// Right-top panel — shows spec state, roadmap, progress.

package manager

import (
	"fmt"
	"strings"
)

var roadmapPhases = []string{
	"IDLE",
	"DISCOVERY",
	"REVIEW",
	"DRAFT",
	"APPROVED",
	"EXECUTING",
	"DONE",
	"IDLE",
}

func buildRoadmap(phase *string) string {
	if phase == nil || *phase == "IDLE" {
		return "IDLE"
	}

	phaseMap := map[string]string{
		"DISCOVERY_REFINEMENT": "REVIEW",
		"SPEC_PROPOSAL":        "DRAFT",
		"SPEC_APPROVED":        "APPROVED",
		"COMPLETED":            "DONE",
	}

	mapped := *phase
	if m, ok := phaseMap[*phase]; ok {
		mapped = m
	}

	parts := make([]string, len(roadmapPhases))
	for i, p := range roadmapPhases {
		if p == mapped {
			parts[i] = styleBold.Render("\u2726" + p + "\u2726")
		} else {
			parts[i] = styleDim.Render(p)
		}
	}
	return strings.Join(parts, "\u2192")
}

func buildProgressBar(completed, total, width int) string {
	if total == 0 {
		return styleDim.Render("no tasks")
	}
	filled := (completed * width) / total
	empty := width - filled
	if filled < 0 {
		filled = 0
	}
	if empty < 0 {
		empty = 0
	}
	return styleGreen.Render(strings.Repeat("\u2588", filled)) +
		styleDim.Render(strings.Repeat("\u2591", empty)) +
		fmt.Sprintf(" %d/%d", completed, total)
}

// TaskInfo holds task completion information.
type TaskInfo struct {
	Completed int
	Total     int
}

// RenderMonitor renders the monitor panel as a string.
func RenderMonitor(
	tab *ManagerTab,
	panelX, panelY, panelW, panelH int,
	taskInfo *TaskInfo,
) string {
	var lines []string

	if tab == nil {
		lines = append(lines, styleBold.Render("Mode: ")+styleCyan.Render("IDLE"))
		lines = append(lines, styleDim.Render("No active spec"))
	} else if tab.Mode == TabModeFree {
		lines = append(lines, styleBold.Render("Mode: ")+styleCyan.Render("IDLE"))
		lines = append(lines, styleDim.Render("No active spec"))
		lines = append(lines, "")
		lines = append(lines, styleDim.Render(fmt.Sprintf("Session: %s", tab.SessionID)))
	} else {
		specName := "unknown"
		if tab.Spec != nil {
			specName = *tab.Spec
		}
		phaseName := "unknown"
		if tab.Phase != nil {
			phaseName = *tab.Phase
		}
		lines = append(lines, styleBold.Render("Spec: ")+specName)
		lines = append(lines, styleBold.Render("Phase: ")+phaseName)
		lines = append(lines, "")
		lines = append(lines, buildRoadmap(tab.Phase))
		lines = append(lines, "")
		if taskInfo != nil {
			lines = append(lines, styleBold.Render("Progress: ")+buildProgressBar(taskInfo.Completed, taskInfo.Total, 15))
		}
		lines = append(lines, "")
		lines = append(lines, styleDim.Render(fmt.Sprintf("Session: %s", tab.SessionID)))
	}

	return fillBox(panelX, panelY, panelW, panelH, "Monitor", lines)
}
