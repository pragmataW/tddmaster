package phasecatalog

import "github.com/pragmataW/tddmaster/internal/engine"

const (
	PhaseListenFirst  engine.PhaseID = "listen-first"
	PhaseSettings     engine.PhaseID = "spec-settings"
	PhaseDiscovery    engine.PhaseID = "discovery"
	PhaseSpecProposal engine.PhaseID = "spec-proposal"
	PhaseRefinement   engine.PhaseID = "refinement"
	PhaseExecution    engine.PhaseID = "execution"
)

const (
	ModListenFirst  engine.ModuleID = "mod-listen-first"
	ModSettings     engine.ModuleID = "mod-settings"
	ModDiscovery    engine.ModuleID = "mod-discovery"
	ModSpecProposal engine.ModuleID = "mod-spec-proposal"
	ModRefinement   engine.ModuleID = "mod-refinement"
	ModExecution    engine.ModuleID = "mod-execution"
)

const (
	StepListenFirstAsk  engine.StepID = "step-listen-first-ask"
	StepSettingsAsk     engine.StepID = "step-settings-ask"
	StepDiscoveryAsk    engine.StepID = "step-discovery-ask"
	StepSpecProposalAsk engine.StepID = "step-spec-proposal-ask"
	StepRefinementAsk   engine.StepID = "step-refinement-ask"
	StepExecutionAsk    engine.StepID = "step-execution-ask"
)
