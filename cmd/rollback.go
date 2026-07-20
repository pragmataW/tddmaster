package cmd

import (
	"fmt"

	"github.com/pragmataW/tddmaster/internal/errs"
	"github.com/pragmataW/tddmaster/internal/lifecycle"
	"github.com/pragmataW/tddmaster/internal/spec"
	"github.com/pragmataW/tddmaster/internal/ui/theme"
	"github.com/spf13/cobra"
)

func newRollbackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollback <slug> <target-phase>",
		Short: "Roll back a spec to an earlier phase",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			targetPhase := args[1]
			if !spec.ValidSlug(slug) {
				return errs.Newf(errs.KeyInvalidSlug, slug)
			}
			root, err := resolveRoot(cmd)
			if err != nil {
				return errs.Wrap(errs.KeyResolveRoot, err)
			}
			warnings, err := lifecycle.Rollback(root, slug, targetPhase)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, theme.SuccessStyle.Render(fmt.Sprintf("✓ rolled back spec %s to phase %s", slug, targetPhase)))
			for _, w := range warnings {
				fmt.Fprintln(out, theme.MutedStyle.Render(w))
			}
			return nil
		},
	}
	addRootFlag(cmd)
	return cmd
}
