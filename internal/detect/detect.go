// Package detect is the entry point for project-trait and coding-tool
// detection. It splits into two sub-packages:
//
//   - internal/detect/model   — signal tables and canonical identifiers
//                               (pure data, no I/O, no logic).
//   - internal/detect/service — business logic: filesystem probes,
//                               JSON parsing, signal-table iteration.
//
// This root package only re-exports the wrapper helpers that cmd/*
// calls.
package detect

import (
	"github.com/pragmataW/tddmaster/internal/detect/service"
	"github.com/pragmataW/tddmaster/internal/state"
)

// DetectProject detects project traits by scanning marker files in root.
func DetectProject(root string) state.ProjectTraits {
	return service.DetectProject(root)
}

// DetectCodingTools detects available coding tools by checking for
// their marker files in root.
func DetectCodingTools(root string) []state.CodingToolId {
	return service.DetectCodingTools(root)
}
