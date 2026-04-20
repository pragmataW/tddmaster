// Package service decodes the embedded default concern payloads into
// state.ConcernDefinition values. Raw JSON lives in the sibling model
// package; this package owns the parsing.
package service

import (
	"encoding/json"

	"github.com/pragmataW/tddmaster/internal/defaults/model"
	"github.com/pragmataW/tddmaster/internal/state"
)

// DefaultConcerns decodes and returns the built-in concern definitions.
// Each call yields a freshly allocated slice so callers can safely mutate
// their own copy without affecting other callers. A malformed payload is
// a programmer error (the JSON is authored in this repo), so it panics
// rather than bubbling up.
func DefaultConcerns() []state.ConcernDefinition {
	concerns := make([]state.ConcernDefinition, 0, len(model.DefaultConcernJSONs))
	for _, raw := range model.DefaultConcernJSONs {
		var c state.ConcernDefinition
		if err := json.Unmarshal([]byte(raw), &c); err != nil {
			panic("defaults: failed to parse embedded concern JSON: " + err.Error())
		}
		concerns = append(concerns, c)
	}
	return concerns
}
