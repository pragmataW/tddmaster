
package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/pragmataW/tddmaster/internal/state"
)

func newWatchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch for state changes",
		RunE:  runWatch,
	}
	cmd.Flags().Int("interval", 2, "Poll interval in seconds")
	return cmd
}

func runWatch(cmd *cobra.Command, _ []string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}

	interval, _ := cmd.Flags().GetInt("interval")
	if interval < 1 {
		interval = 2
	}

	printErr("Watching for state changes (Ctrl+C to stop)...")

	var lastPhase state.Phase
	for {
		st, err := state.ReadState(root)
		if err != nil {
			time.Sleep(time.Duration(interval) * time.Second)
			continue
		}

		if st.Phase != lastPhase {
			lastPhase = st.Phase
			spec := "none"
			if st.Spec != nil {
				spec = *st.Spec
			}
			printErr(fmt.Sprintf("[%s] Phase: %s  Spec: %s",
				time.Now().Format("15:04:05"), st.Phase, spec))
		}

		time.Sleep(time.Duration(interval) * time.Second)
	}
}
