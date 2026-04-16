
package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pragmataW/tddmaster/internal/state"
)

func newLearnCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "learn",
		Short: "Learn from agent interactions",
		RunE:  runLearn,
	}
	cmd.Flags().String("spec", "", "Spec name")
	return cmd
}

func runLearn(cmd *cobra.Command, args []string) error {
	specFlag, _ := cmd.Flags().GetString("spec")
	specArgs := args
	if specFlag != "" {
		specArgs = append([]string{"--spec=" + specFlag}, args...)
	}
	return runLearnWithArgs(specArgs)
}

func runLearnWithArgs(args []string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}

	specResult := state.RequireSpecFlag(args)
	if !specResult.OK {
		return fmt.Errorf("%s", specResult.Error)
	}

	st, err := state.ResolveState(root, &specResult.Spec)
	if err != nil {
		return err
	}

	if st.Phase != state.PhaseCompleted {
		return fmt.Errorf("learn is only available for COMPLETED specs (current phase: %s)", st.Phase)
	}

	// Collect decisions for promotion
	var promotable []state.Decision
	for _, d := range st.Decisions {
		if !d.Promoted {
			promotable = append(promotable, d)
		}
	}

	if len(promotable) == 0 {
		printErr(fmt.Sprintf("No decisions to promote for spec %s.", specResult.Spec))
		return nil
	}

	printErr(fmt.Sprintf("Decisions for spec %s (%d promotable):", specResult.Spec, len(promotable)))
	for i, d := range promotable {
		printErr(fmt.Sprintf("  %d. %s", i+1, d.Question))
		printErr(fmt.Sprintf("     Choice: %s", d.Choice))
	}
	printErr("")
	printErr(fmt.Sprintf("To promote a decision to a rule, run: tddmaster rule add \"%s\"", strings.TrimSpace(promotable[0].Choice)))

	return nil
}
