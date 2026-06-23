package phases

import (
	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/engine/loop"
	"github.com/pragmataW/tddmaster/internal/phasecatalog"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func Enabled(s spec.Settings) []engine.PhaseDef {
	defs := []engine.PhaseDef{
		{ID: phasecatalog.PhaseSettings, Driver: SettingsDriver()},
		{ID: phasecatalog.PhaseDiscovery, Driver: DiscoveryDriver()},
		{ID: phasecatalog.PhaseSpecProposal, Driver: SpecProposalDriver()},
		{ID: phasecatalog.PhaseRefinement, Driver: RefinementDriver()},
		{ID: phasecatalog.PhaseExecution, Driver: loop.NewLoopDriver()},
	}
	if s.RuleLearningEnabled {
		defs = append(defs, engine.PhaseDef{ID: phasecatalog.PhaseRuleLearning, Driver: RuleLearningDriver()})
	}
	return defs
}
