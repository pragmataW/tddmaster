
package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pragmataW/tddmaster/internal/state"
)

func newDiagramsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diagrams",
		Short: "Generate state machine diagrams",
		RunE:  runDiagrams,
	}
}

func runDiagrams(_ *cobra.Command, _ []string) error {
	printErr("Generating state machine diagram...")
	fmt.Println(generateStateDiagram())
	return nil
}

// generateStateDiagram returns a Mermaid state diagram of the machine.
func generateStateDiagram() string {
	var sb strings.Builder
	sb.WriteString("stateDiagram-v2\n")

	// Show all transitions
	for from, tos := range state.ValidTransitions {
		for _, to := range tos {
			sb.WriteString(fmt.Sprintf("  %s --> %s\n", from, to))
		}
	}

	return sb.String()
}
