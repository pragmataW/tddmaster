
package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pragmataW/tddmaster/internal/state"
)

func newReviewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "review",
		Short: "Review spec or execution state",
		RunE:  runReview,
	}
	cmd.Flags().String("spec", "", "Spec name")
	return cmd
}

func runReview(cmd *cobra.Command, args []string) error {
	specFlag, _ := cmd.Flags().GetString("spec")
	specArgs := args
	if specFlag != "" {
		specArgs = append([]string{"--spec=" + specFlag}, args...)
	}
	return runReviewWithArgs(specArgs)
}

func runReviewWithArgs(args []string) error {
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

	config, _ := state.ReadManifest(root)
	concerns := []string{}
	if config != nil {
		concerns = config.Concerns
	}

	// Build review output
	review := map[string]interface{}{
		"spec":          specResult.Spec,
		"phase":         string(st.Phase),
		"iteration":     st.Execution.Iteration,
		"decisions":     len(st.Decisions),
		"concerns":      concerns,
		"customACs":     len(st.CustomACs),
		"specNotes":     len(st.SpecNotes),
		"modifiedFiles": st.Execution.ModifiedFiles,
	}

	if st.Execution.LastProgress != nil {
		review["lastProgress"] = *st.Execution.LastProgress
	}

	if st.Execution.Debt != nil {
		review["debtItems"] = len(st.Execution.Debt.Items)
	}

	_ = strings.TrimSpace // avoid unused import
	return writeJSON(review)
}
