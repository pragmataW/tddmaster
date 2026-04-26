package service

import (
	"fmt"

	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/context/service/behavioral"
	"github.com/pragmataW/tddmaster/internal/context/service/concerns"
	"github.com/pragmataW/tddmaster/internal/context/service/discovery"
	"github.com/pragmataW/tddmaster/internal/context/service/execution"
	"github.com/pragmataW/tddmaster/internal/context/service/meta"
	"github.com/pragmataW/tddmaster/internal/context/service/phases"
	"github.com/pragmataW/tddmaster/internal/context/service/tdd"
	"github.com/pragmataW/tddmaster/internal/defaults"
	"github.com/pragmataW/tddmaster/internal/state"
)

// Compile turns state + ancillary data into the fully-populated NextOutput.
func Compile(in model.CompileInput) model.NextOutput {
	hints := model.DefaultHints
	if in.InteractionHints != nil {
		hints = *in.InteractionHints
	}

	r := NewRenderer(in.CommandPrefix)
	st := in.State
	activeConcerns := in.ActiveConcerns

	metaBlock := meta.Build(r, st, activeConcerns, hints)

	maxIter := model.DefaultMaxIter
	allowGit := false
	if in.Config != nil {
		if in.Config.MaxIterationsBeforeRestart > 0 {
			maxIter = in.Config.MaxIterationsBeforeRestart
		}
		allowGit = in.Config.AllowGit
	}

	// Determine verifierRequired early so behavioral.Build can gate sentences.
	// We compute a preliminary value here; the canonical value for ExecutionData
	// is computed later in the PhaseExecuting case below.
	prelimVerifierRequired := tdd.VerifierRequired(in.Config, st.Execution.TDDCycle)

	behavioralBlock := behavioral.Build(r, st, maxIter, allowGit, activeConcerns, in.ParsedSpec, hints, prelimVerifierRequired)

	if st.Phase == state.PhaseExecuting {
		_, tier2Reminders := concerns.SplitRemindersByTier(activeConcerns)
		totalT2 := in.Tier2Count + len(tier2Reminders)
		if totalT2 > 0 {
			summary := fmt.Sprintf("%d file-specific rules delivered via PreToolUse hook when editing matching files.", totalT2)
			behavioralBlock.Tier2Summary = &summary
		}
	}

	protocolGuide := meta.BuildProtocolGuide(r, st)
	roadmap := meta.BuildRoadmap(st.Phase)
	gate := meta.BuildGate(st, in.ParsedSpec)

	var phase string
	var discoveryData *model.DiscoveryOutput
	var discoveryReviewData *model.DiscoveryReviewOutput
	var specDraftData *model.SpecDraftOutput
	var specApprovedData *model.SpecApprovedOutput
	var executionData *model.ExecutionOutput
	var blockedData *model.BlockedOutput
	var completedData *model.CompletedOutput
	var idleData *model.IdleOutput

	switch st.Phase {
	case state.PhaseIdle:
		phase = "IDLE"
		allConcerns := defaults.DefaultConcerns()
		idle := phases.CompileIdle(activeConcerns, allConcerns, len(in.Rules), in.IdleContext)
		idleData = &idle

	case state.PhaseDiscovery:
		phase = "DISCOVERY"
		disc := discovery.Compile(r, st, activeConcerns, in.Rules, in.CurrentUser)
		discoveryData = &disc

	case state.PhaseDiscoveryRefinement:
		phase = "DISCOVERY_REFINEMENT"
		dr := discovery.CompileReview(r, st, activeConcerns)
		discoveryReviewData = &dr

	case state.PhaseSpecProposal:
		phase = "SPEC_PROPOSAL"
		sd := phases.CompileSpecDraft(r, st)
		specDraftData = &sd

	case state.PhaseSpecApproved:
		phase = "SPEC_APPROVED"
		sa := phases.CompileSpecApproved(r, st, in.Config, in.ParsedSpec)
		specApprovedData = &sa

	case state.PhaseExecuting:
		phase = "EXECUTING"
		exec := execution.Compile(r, st, activeConcerns, in.Rules, maxIter, in.ParsedSpec)
		currentTaskUsesTDD := state.ShouldRunTDDForCurrentTask(st, in.Config)
		if currentTaskUsesTDD && st.Execution.TDDCycle != "" {
			cycle := st.Execution.TDDCycle
			// AC-8: TDDPhase is always set (not gated by VerifierRequired).
			exec.TDDPhase = &cycle
			// AC-6: VerifierRequired determined by the VerifierRequired function.
			exec.VerifierRequired = tdd.VerifierRequired(in.Config, cycle)
			// AC-7: TDDVerificationContext only populated when VerifierRequired=true.
			if exec.VerifierRequired {
				exec.TDDVerificationContext = tdd.BuildVerificationContext(cycle)
			}
			exec.RefactorInstructions = tdd.BuildRefactorInstructions(st, getMaxRefactorRounds(in.Config), exec.VerifierRequired)
		} else {
			// AC-6: When TDD is off or no cycle, compute VerifierRequired from manifest alone.
			exec.VerifierRequired = tdd.VerifierRequired(in.Config, "")
		}
		if currentTaskUsesTDD &&
			st.Execution.LastVerification != nil && !st.Execution.LastVerification.Passed {
			maxRetries := 0
			if in.Config != nil && in.Config.Tdd != nil {
				maxRetries = in.Config.Tdd.MaxVerificationRetries
			}
			failCount := st.Execution.LastVerification.VerificationFailCount
			exec.TDDFailureReport = &model.TDDFailureReport{
				Reason:             "verification-failed",
				FailedACs:          st.Execution.LastVerification.FailedACs,
				UncoveredEdgeCases: st.Execution.LastVerification.UncoveredEdgeCases,
				RetryCount:         failCount,
				MaxRetries:         maxRetries,
				WillBlock:          maxRetries > 0 && failCount >= maxRetries,
			}
		}
		executionData = &exec

	case state.PhaseBlocked:
		phase = "BLOCKED"
		bl := phases.CompileBlocked(r, st)
		blockedData = &bl

	case state.PhaseCompleted:
		phase = "COMPLETED"
		comp := phases.CompileCompleted(st)
		completedData = &comp

	default:
		phase = "IDLE"
		allConcerns := defaults.DefaultConcerns()
		idle := phases.CompileIdle(activeConcerns, allConcerns, len(in.Rules), in.IdleContext)
		idleData = &idle
	}

	result := model.NextOutput{
		Phase:               phase,
		Meta:                metaBlock,
		Behavioral:          behavioralBlock,
		Roadmap:             roadmap,
		Gate:                gate,
		ProtocolGuide:       protocolGuide,
		DiscoveryData:       discoveryData,
		DiscoveryReviewData: discoveryReviewData,
		SpecDraftData:       specDraftData,
		SpecApprovedData:    specApprovedData,
		ExecutionData:       executionData,
		BlockedData:         blockedData,
		CompletedData:       completedData,
		IdleData:            idleData,
	}

	options := meta.BuildInteractiveOptions(r, st, activeConcerns, in.IdleContext, in.Config)
	if len(options) > 0 {
		publicOpts := make([]model.InteractiveOption, len(options))
		cmdMap := make(map[string]string, len(options))
		for i, opt := range options {
			publicOpts[i] = model.InteractiveOption{Label: opt.Label, Description: opt.Description}
			cmdMap[opt.Label] = opt.Command
		}

		toolHint := "AskUserQuestion"
		toolHintInstruction := "Use AskUserQuestion tool to present these options. Do NOT use prose."
		if hints.OptionPresentation != "tool" {
			toolHint = "prose-numbered-list"
			toolHintInstruction = "Present options as a numbered list. Ask user to pick a number."
		}

		result.InteractiveOptions = publicOpts
		result.CommandMap = cmdMap
		result.ToolHint = &toolHint
		result.ToolHintInstruction = &toolHintInstruction
	}

	return result
}

// getMaxRefactorRounds safely reads the MaxRefactorRounds field from cfg,
// returning 0 when cfg or cfg.Tdd is nil.
func getMaxRefactorRounds(cfg *state.NosManifest) int {
	if cfg == nil || cfg.Tdd == nil {
		return 0
	}
	return cfg.Tdd.MaxRefactorRounds
}
