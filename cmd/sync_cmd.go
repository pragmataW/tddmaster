
package cmd

import (
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync state with remote",
		Long:  "`sync` has been merged into `init`. Running `tddmaster init`...",
		RunE:  runSync,
	}
}

func runSync(cmd *cobra.Command, args []string) error {
	printErr("`sync` has been merged into `init`. Running `tddmaster init`...")
	printErr("")
	return runInit(cmd, args)
}
