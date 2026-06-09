package cmd

import "github.com/spf13/cobra"

func addRootFlag(cmd *cobra.Command) {
	cmd.Flags().String("root", "", "Root directory (default: cwd)")
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "tddmaster",
		Short:         "TDD-driven spec orchestration tool",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.AddCommand(newInitCmd())
	root.AddCommand(newStartCmd())
	root.AddCommand(newNextCmd())
	root.AddCommand(newRefineCmd())
	root.AddCommand(newVisualizeCmd())
	return root
}

func Execute() error {
	return newRootCmd().Execute()
}
