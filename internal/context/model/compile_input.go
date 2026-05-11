package model

import (
	"github.com/pragmataW/tddmaster/internal/spec"
	"github.com/pragmataW/tddmaster/internal/state"
)

// CompileInput bundles every input the context compiler needs. Prefer
// constructing named fields over positional arguments — the positional
// signature had grown to 10 parameters before this struct was introduced.
type CompileInput struct {
	State            state.StateFile
	ActiveConcerns   []state.ConcernDefinition
	Rules            []string
	Config           *state.NosManifest
	ParsedSpec       *spec.ParsedSpec
	IdleContext      *IdleContext
	InteractionHints *InteractionHints
	CurrentUser      *CurrentUser
	Tier2Count       int
	CommandPrefix    string

	// Root is the project root path. Optional. When provided, the compiler can
	// load per-task artifacts from disk (e.g. ProgressTaskPlan from progress.json
	// for the Important Task Gate flow). Tests that don't touch the gate can
	// leave it empty.
	Root string
}
