package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/charmbracelet/lipgloss"
	"github.com/pragmataW/tddmaster/internal/lifecycle"
	"github.com/pragmataW/tddmaster/internal/ui/theme"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List specs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot(cmd)
			if err != nil {
				return fmt.Errorf("resolve root: %w", err)
			}
			archived, err := cmd.Flags().GetBool("archived")
			if err != nil {
				return err
			}
			infos, err := lifecycle.List(root)
			if err != nil {
				return fmt.Errorf("list specs: %w", err)
			}

			headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.ColorIndigo)
			out := cmd.OutOrStdout()
			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "%s\t%s\t%s\n", headerStyle.Render("SLUG"), headerStyle.Render("PHASE"), headerStyle.Render("STATUS"))
			for _, info := range infos {
				if info.Archived != archived {
					continue
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", info.Slug, info.Phase, info.Status)
			}
			return w.Flush()
		},
	}
	addRootFlag(cmd)
	cmd.Flags().Bool("archived", false, "Show only archived specs")
	return cmd
}
