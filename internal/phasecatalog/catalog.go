package phasecatalog

import "github.com/pragmataW/tddmaster/internal/engine"

func askStep(id engine.StepID, prompt string) engine.StepDef {
	return engine.StepDef{
		ID: id,
		Prompt: func(c *engine.Context) engine.Action {
			return engine.Action{
				Action:      engine.ActionAsk,
				Instruction: prompt,
				ExpectedInput: engine.ExpectedInput{
					Format: engine.FormatText,
				},
			}
		},
	}
}

func singleStepPhase(phaseID engine.PhaseID, moduleID engine.ModuleID, stepID engine.StepID, prompt string) engine.PhaseDef {
	return engine.PhaseDef{
		ID: phaseID,
		Driver: &engine.StepTableDriver{
			Modules: []engine.ModuleDef{
				{
					ID:    moduleID,
					Steps: []engine.StepDef{askStep(stepID, prompt)},
				},
			},
		},
	}
}

func Catalog() []engine.PhaseDef {
	return []engine.PhaseDef{
		singleStepPhase(PhaseSettings, ModSettings, StepSettingsAsk, "Configure spec settings."),
		singleStepPhase(PhaseDiscovery, ModDiscovery, StepDiscoveryAsk, "What are the key requirements?"),
		singleStepPhase(PhaseSpecProposal, ModSpecProposal, StepSpecProposalAsk, "Does this spec proposal look correct?"),
		singleStepPhase(PhaseRefinement, ModRefinement, StepRefinementAsk, "Any refinements needed?"),
		singleStepPhase(PhaseAnalysis, ModAnalysis, StepAnalysisAsk, "Cross-artifact analysis phase."),
		singleStepPhase(PhaseExecution, ModExecution, StepExecutionAsk, "Execution phase placeholder."),
	}
}
