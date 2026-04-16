
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pragmataW/tddmaster/internal/output"
)

func newFreeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "free",
		Short: "Free a blocked state",
		Long:  "Deprecated. IDLE is the default permissive state.",
		RunE:  runFree,
	}
}

func runFree(_ *cobra.Command, _ []string) error {
	printErr("tddmaster starts in idle mode with no enforcement. To work on a spec, run:")
	fmt.Printf("  %s\n", output.Cmd(`spec new "description"`))
	return nil
}
