package cmd

import (
	"fmt"
	"os"

	"github.com/pragmataW/tddmaster/internal/ui/ruleform"
	"github.com/spf13/cobra"
)

func newRuleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rule",
		Short: "Manage project rule files",
	}
	cmd.AddCommand(newRuleAddCmd())
	return cmd
}

func newRuleAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new rule file interactively",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, _ := cmd.Flags().GetString("root")
			if root == "" {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("get cwd: %w", err)
				}
				root = cwd
			}
			return ruleform.Run(root)
		},
	}
	cmd.Flags().String("root", "", "Root directory (default: cwd)")
	return cmd
}
