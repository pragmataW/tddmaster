package cmd

import (
	"fmt"

	"github.com/pragmataW/tddmaster/internal/errs"
	"github.com/pragmataW/tddmaster/internal/lifecycle"
	"github.com/pragmataW/tddmaster/internal/spec"
	"github.com/pragmataW/tddmaster/internal/ui/theme"
	"github.com/spf13/cobra"
)

func newArchiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive <slug>",
		Short: "Archive an active spec",
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
			if err := lifecycle.Archive(root, slug); err != nil {
				return errs.Wrap(errs.KeyArchiveSpec, err)
			}
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, theme.SuccessStyle.Render(fmt.Sprintf("archived spec %s", slug)))
			return nil
		},
	}
	cmd.Flags().String("root", "", "Root directory (default: cwd)")
	return cmd
}
