// Package defaults exposes the built-in concern definitions shipped with
// tddmaster. The public entry point DefaultConcerns delegates to the
// service layer; raw JSON payloads live in the model layer.
package defaults

import (
	"github.com/pragmataW/tddmaster/internal/defaults/service"
	"github.com/pragmataW/tddmaster/internal/state"
)

// DefaultConcerns returns the built-in concern definitions, freshly
// decoded on every call.
func DefaultConcerns() []state.ConcernDefinition {
	return service.DefaultConcerns()
}
