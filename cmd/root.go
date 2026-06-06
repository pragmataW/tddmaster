package cmd

import "github.com/spf13/cobra"

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "tddmaster",
		Short: "TDD-driven spec orchestration tool",
	}
	root.AddCommand(newInitCmd())
	return root
}

func Execute() error {
	return newRootCmd().Execute()
}
