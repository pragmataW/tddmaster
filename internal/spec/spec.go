// Package spec is the entry point for reading, writing, and updating
// tddmaster spec artifacts (spec.md + progress.json). It splits into two
// sub-packages:
//
//   - internal/spec/model   — pure data shapes (ParsedSpec, ParsedTask,
//                             ProgressFile). No I/O, no logic.
//   - internal/spec/service — business logic: parsing spec.md, rendering
//                             new spec documents, deriving task and edge
//                             case lists from discovery state, updating
//                             status artifacts on disk.
//
// This root package only re-exports the wrappers that cmd/* and other
// internal packages call.
package spec

import (
	"github.com/pragmataW/tddmaster/internal/spec/model"
	"github.com/pragmataW/tddmaster/internal/spec/service"
	"github.com/pragmataW/tddmaster/internal/state"
)

// ParsedSpec re-exports model.ParsedSpec for callers of the root spec package.
type ParsedSpec = model.ParsedSpec

// ParsedTask re-exports model.ParsedTask for callers of the root spec package.
type ParsedTask = model.ParsedTask

// ParseSpec reads a spec.md file from disk and returns structured data.
func ParseSpec(root, specName string) (*ParsedSpec, error) {
	return service.ParseSpec(root, specName)
}

// GenerateSpec writes spec.md + progress.json for the active spec in st.
// Returns the spec.md path.
func GenerateSpec(root string, st *state.StateFile, concerns []state.ConcernDefinition) (string, error) {
	return service.Generate(root, st, concerns)
}

// DeriveEdgeCases extracts concrete edge cases from discovery answers and
// premises.
func DeriveEdgeCases(answers []state.DiscoveryAnswer, premises []state.Premise) []string {
	return service.DeriveEdgeCases(answers, premises)
}

// UpdateSpecStatus updates the "## Status:" line in spec.md.
func UpdateSpecStatus(root, specName, newStatus string) error {
	return service.UpdateSpecStatus(root, specName, newStatus)
}

// UpdateProgressStatus updates the spec-level status field in progress.json.
func UpdateProgressStatus(root, specName, status string) error {
	return service.UpdateProgressStatus(root, specName, status)
}

// MarkTaskImportant toggles the `important` flag on a task row in progress.json.
func MarkTaskImportant(root, specName, taskID string, important bool) error {
	return service.MarkTaskImportant(root, specName, taskID, important)
}

// AppendTaskPlan persists a user-approved plan to progress.json TaskPlans[].
// An existing plan for the same TaskID is overwritten.
func AppendTaskPlan(root, specName string, plan model.ProgressTaskPlan) error {
	return service.AppendTaskPlan(root, specName, plan)
}

// LoadTaskPlan returns the user-approved plan for the given taskID, or nil
// when none is recorded.
func LoadTaskPlan(root, specName, taskID string) (*model.ProgressTaskPlan, error) {
	return service.LoadTaskPlan(root, specName, taskID)
}

// ProgressTaskPlan re-exports the plan type for callers of the root spec package.
type ProgressTaskPlan = model.ProgressTaskPlan
