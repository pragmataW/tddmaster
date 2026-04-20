
package cmd

import (
	"github.com/spf13/cobra"
)

// rootCmd is the base command for the tddmaster CLI.
var rootCmd = &cobra.Command{
	Use:   "tddmaster",
	Short: "tddmaster — state-machine orchestrator for AI agents",
	Long:  "tddmaster — state-machine orchestrator for AI agents",
}

// Execute runs the root command and returns any error.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(
		newInitCmd(),
		newStatusCmd(),
		newSpecCmd(),
		newNextCmd(),
		newApproveCmd(),
		newBlockCmd(),
		newResetCmd(),
		newDoneCmd(),
		newFreeCmd(),
		newCancelCmd(),
		newWontfixCmd(),
		newReopenCmd(),
		newUndoCmd(),
		newConcernCmd(),
		newWatchCmd(),
		newSyncCmd(),
		newLearnCmd(),
		newDiagramsCmd(),
		newPurgeCmd(),
		newInvokeHookCmd(),
		newRuleCmd(),
		newConfigCmd(),
		newPackCmd(),
		newSessionCmd(),
		newFollowupCmd(),
		newDelegateCmd(),
		newReviewCmd(),
	)
}
