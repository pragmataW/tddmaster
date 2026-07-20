package cmd

import (
	"fmt"

	"github.com/pragmataW/tddmaster/internal/errs"
	"github.com/pragmataW/tddmaster/internal/lifecycle"
	"github.com/pragmataW/tddmaster/internal/spec"
	"github.com/pragmataW/tddmaster/internal/ui/theme"
	"github.com/spf13/cobra"
)

func newRestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore <slug>",
		Short: "Restore an archived spec",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if !spec.ValidSlug(slug) {
				return errs.Newf(errs.KeyInvalidSlug, slug)
			}
			root, err := resolveRoot(cmd)
			if err != nil {
				return errs.Wrap(errs.KeyResolveRoot, err)
			}
			if spec.Exists(root, slug) {
				return errs.Newf(errs.KeyRestoreConflict, slug)
			}
			if err := lifecycle.Restore(root, slug); err != nil {
				return errs.Wrap(errs.KeyRestoreSpec, err)
			}
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, theme.SuccessStyle.Render(fmt.Sprintf("restored spec %s", slug)))
			return nil
		},
	}
	cmd.Flags().String("root", "", "Root directory (default: cwd)")
	return cmd
}
