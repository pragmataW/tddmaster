package cmd

import (
	"github.com/spf13/cobra"
)

// newPackCmd registers the `pack` subcommand. Pack management (install,
// remove, list) is not implemented — the command is kept as a visible
// stub so the CLI surface documents the intent.
func newPackCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pack",
		Short: "Manage rule packs (not yet implemented)",
		Long:  "Manage rule packs (install, remove, list). This feature is not yet implemented.",
		RunE: func(_ *cobra.Command, _ []string) error {
			printErr("Pack management is not implemented yet.")
			return nil
		},
	}
}
