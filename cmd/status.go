
package cmd

import (
	"github.com/spf13/cobra"

	ctxpkg "github.com/pragmataW/tddmaster/internal/context"
	"github.com/pragmataW/tddmaster/internal/output"
	"github.com/pragmataW/tddmaster/internal/spec"
	"github.com/pragmataW/tddmaster/internal/state"
)

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current state machine status",
		RunE:  runStatus,
	}
	cmd.Flags().String("spec", "", "Spec name to show status for")
	return cmd
}

func runStatus(cmd *cobra.Command, args []string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}

	initialized, _ := state.IsInitialized(root)
	if !initialized {
		return writeJSON(map[string]string{"error": "tddmaster is not initialized. Run: " + output.Cmd("init")})
	}

	specFlag, _ := cmd.Flags().GetString("spec")
	var specPtr *string
	if specFlag != "" {
		specPtr = &specFlag
	}
	if specPtr == nil {
		specPtr = state.ParseSpecFlag(args)
	}

	st, err := state.ResolveState(root, specPtr)
	if err != nil {
		return writeJSON(map[string]string{"error": err.Error()})
	}

	config, _ := state.ReadManifest(root)

	concerns := []string{}
	tools := []state.CodingToolId{}
	if config != nil {
		concerns = config.Concerns
		tools = config.Tools
	}

	// Build status data
	statusData := map[string]interface{}{
		"phase":     string(st.Phase),
		"spec":      st.Spec,
		"branch":    st.Branch,
		"concerns":  concerns,
		"tools":     tools,
		"decisions": len(st.Decisions),
	}

	if st.Phase == state.PhaseDiscovery || st.Phase == state.PhaseDiscoveryRefinement {
		statusData["discovery"] = map[string]interface{}{
			"answered": len(st.Discovery.Answers),
		}
	}

	if st.Phase == state.PhaseExecuting || st.Phase == state.PhaseBlocked {
		debtLen := 0
		if st.Execution.Debt != nil {
			debtLen = len(st.Execution.Debt.Items)
		}
		var verPassed *bool
		if st.Execution.LastVerification != nil {
			v := st.Execution.LastVerification.Passed
			verPassed = &v
		}
		statusData["execution"] = map[string]interface{}{
			"iteration":          st.Execution.Iteration,
			"lastProgress":       st.Execution.LastProgress,
			"debt":               debtLen,
			"verificationPassed": verPassed,
		}
	}

	// Compile full output
	allConcerns, _ := state.ListConcerns(root)
	activeConcerns := filterConcerns(allConcerns, concerns)

	tier1, hints, tier2Count, _ := loadRulesAndHints(root, st, config)

	var parsedSpec *spec.ParsedSpec
	if st.Spec != nil {
		parsedSpec, _ = spec.ParseSpec(root, *st.Spec)
	}

	compiled := ctxpkg.Compile(st, activeConcerns, tier1, config, parsedSpec, nil, nil, hints, nil, tier2Count)
	merged := mergeMap(statusData, compiledToMap(compiled))

	return writeJSON(merged)
}

// filterConcerns returns concern definitions whose IDs are in the given list.
func filterConcerns(all []state.ConcernDefinition, ids []string) []state.ConcernDefinition {
	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}
	var result []state.ConcernDefinition
	for _, c := range all {
		if idSet[c.ID] {
			result = append(result, c)
		}
	}
	return result
}
