
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newManagerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "manager",
		Short: "Run the manager process",
		Long:  "Run the tddmaster manager process (dashboard backend, session monitor).",
		RunE:  runManager,
	}
}

func runManager(_ *cobra.Command, _ []string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}

	printErr("Starting tddmaster manager...")
	printErr(fmt.Sprintf("  Project root: %s", root))
	printErr("  Manager not fully implemented in Go port.")
	printErr("  Press Ctrl+C to stop.")

	// Block forever (simulating manager daemon)
	select {}
}
