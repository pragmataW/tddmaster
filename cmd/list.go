package cmd

import (
	"bufio"
	"fmt"

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
			out := bufio.NewWriter(cmd.OutOrStdout())

			type row struct{ slug, phase, status string }
			var rows []row
			for _, info := range infos {
				if info.Archived != archived {
					continue
				}
				rows = append(rows, row{info.Slug, info.Phase, info.Status})
			}

			wSlug, wPhase := len("SLUG"), len("PHASE")
			for _, r := range rows {
				if len(r.slug) > wSlug {
					wSlug = len(r.slug)
				}
				if len(r.phase) > wPhase {
					wPhase = len(r.phase)
				}
			}

			fmt.Fprintf(out, "%s  %s  %s\n",
				headerStyle.Render(fmt.Sprintf("%-*s", wSlug, "SLUG")),
				headerStyle.Render(fmt.Sprintf("%-*s", wPhase, "PHASE")),
				headerStyle.Render("STATUS"))
			for _, r := range rows {
				fmt.Fprintf(out, "%-*s  %-*s  %s\n", wSlug, r.slug, wPhase, r.phase, r.status)
			}
			return out.Flush()
		},
	}
	addRootFlag(cmd)
	cmd.Flags().Bool("archived", false, "Show only archived specs")
	return cmd
}
