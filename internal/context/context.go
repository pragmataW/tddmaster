// Package context is the entry point for turning a tddmaster StateFile into
// the `tddmaster next` JSON output (NextOutput). It splits into two sub-
// packages:
//
//   - internal/context/model   — pure data shapes, JSON contract types,
//                                hardcoded rule/instruction strings, named
//                                numeric constants. No I/O, no logic.
//   - internal/context/service — business logic: phase compilers, concern
//                                filtering, behavioural rules, discovery
//                                enrichment, acceptance-criteria building.
//
// This root package only re-exports the wrapper helpers that cmd/* calls.
// Callers needing specific JSON types should import `internal/context/model`
// directly.
package context

import (
	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/context/service"
	"github.com/pragmataW/tddmaster/internal/context/service/concerns"
	"github.com/pragmataW/tddmaster/internal/context/service/discovery"
	"github.com/pragmataW/tddmaster/internal/context/service/tdd"
	"github.com/pragmataW/tddmaster/internal/state"
)

// commandPrefixOverride captures the CLI binary name (set once via
// SetCommandPrefix) so that legacy callers do not need to thread a prefix
// through every Compile invocation.
var commandPrefixOverride string

// SetCommandPrefix overrides the CLI binary prefix used inside the generated
// instruction strings. cmd/next sets this once at startup.
func SetCommandPrefix(prefix string) {
	commandPrefixOverride = prefix
}

// Compile delegates to service.Compile. Callers fill a model.CompileInput and
// this wrapper fills in CommandPrefix from the module-level override when the
// caller has not set it explicitly.
func Compile(in model.CompileInput) model.NextOutput {
	if in.CommandPrefix == "" {
		in.CommandPrefix = commandPrefixOverride
	}
	return service.Compile(in)
}

// InjectTDDRules appends the canonical TDD behavioural rules to the supplied
// rule list without mutating the original slice.
func InjectTDDRules(rules []string) []string {
	return tdd.InjectRules(rules)
}

// LoadDefaultConcerns returns the embedded default concern definitions.
func LoadDefaultConcerns() []state.ConcernDefinition {
	return concerns.LoadDefault()
}

// GetQuestionsWithExtras returns all discovery questions enriched with
// built-in and concern-specific extras.
func GetQuestionsWithExtras(activeConcerns []state.ConcernDefinition) []model.QuestionWithExtras {
	return discovery.GetQuestionsWithExtras(activeConcerns)
}

// ExtractUserContextPrefills turns a rich listen-first message into reviewable
// discovery suggestions.
func ExtractUserContextPrefills(raw string) []state.DiscoveryPrefillQuestion {
	return discovery.ExtractUserContextPrefills(raw)
}
