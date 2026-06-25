package phasecatalog

import "github.com/pragmataW/tddmaster/internal/engine"

const (
	PhaseListenFirst  engine.PhaseID = "listen-first"
	PhaseSettings     engine.PhaseID = "spec-settings"
	PhaseDiscovery    engine.PhaseID = "discovery"
	PhaseSpecProposal engine.PhaseID = "spec-proposal"
	PhaseRefinement   engine.PhaseID = "refinement"
	PhaseAnalysis     engine.PhaseID = "cross-artifact-analysis"
	PhaseExecution    engine.PhaseID = "execution"
	PhaseRuleLearning engine.PhaseID = "rule-learning"
)

const (
	ModListenFirst  engine.ModuleID = "mod-listen-first"
	ModSettings     engine.ModuleID = "mod-settings"
	ModDiscovery    engine.ModuleID = "mod-discovery"
	ModSpecProposal engine.ModuleID = "mod-spec-proposal"
	ModRefinement   engine.ModuleID = "mod-refinement"
	ModAnalysis     engine.ModuleID = "mod-cross-artifact-analysis"
	ModExecution    engine.ModuleID = "mod-execution"
	ModRuleLearning engine.ModuleID = "mod-rule-learning"
)

const (
	StepListenFirstAsk  engine.StepID = "step-listen-first-ask"
	StepSettingsAsk     engine.StepID = "step-settings-ask"
	StepDiscoveryAsk    engine.StepID = "step-discovery-ask"
	StepSpecProposalAsk engine.StepID = "step-spec-proposal-ask"
	StepRefinementAsk   engine.StepID = "step-refinement-ask"
	StepAnalysisAsk     engine.StepID = "step-cross-artifact-analysis-ask"
	StepExecutionAsk    engine.StepID = "step-execution-ask"
	StepRuleLearningAsk engine.StepID = "step-rule-learning-ask"
)
