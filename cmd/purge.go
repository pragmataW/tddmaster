
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/pragmataW/tddmaster/internal/state"
)

func newPurgeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "purge",
		Short: "Purge old state data",
		RunE:  runPurge,
	}
	cmd.Flags().Bool("force", false, "Purge all content without prompting")
	cmd.Flags().Bool("specs", false, "Purge specs directory")
	cmd.Flags().Bool("state", false, "Purge state directory")
	cmd.Flags().Bool("rules", false, "Purge rules directory")
	cmd.Flags().Bool("concerns", false, "Purge concerns directory")
	return cmd
}

func runPurge(cmd *cobra.Command, _ []string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}

	force, _ := cmd.Flags().GetBool("force")
	purgeSpecs, _ := cmd.Flags().GetBool("specs")
	purgeState, _ := cmd.Flags().GetBool("state")
	purgeRules, _ := cmd.Flags().GetBool("rules")
	purgeConcerns, _ := cmd.Flags().GetBool("concerns")

	if !force && !purgeSpecs && !purgeState && !purgeRules && !purgeConcerns {
		printErr("Usage: tddmaster purge [--force | --specs | --state | --rules | --concerns]")
		printErr("  --force     Purge all content")
		printErr("  --specs     Purge specs directory")
		printErr("  --state     Purge state directory")
		printErr("  --rules     Purge rules directory")
		printErr("  --concerns  Purge concerns directory")
		return nil
	}

	var purged []string

	if force || purgeState {
		stateDir := filepath.Join(root, state.TddmasterDir, ".state")
		if err := os.RemoveAll(stateDir); err != nil && !os.IsNotExist(err) {
			return err
		}
		purged = append(purged, ".state/")
	}

	if force || purgeSpecs {
		specsDir := filepath.Join(root, state.TddmasterDir, "specs")
		if err := os.RemoveAll(specsDir); err != nil && !os.IsNotExist(err) {
			return err
		}
		purged = append(purged, "specs/")
	}

	if force || purgeRules {
		rulesDir := filepath.Join(root, state.TddmasterDir, "rules")
		if err := os.RemoveAll(rulesDir); err != nil && !os.IsNotExist(err) {
			return err
		}
		purged = append(purged, "rules/")
	}

	if force || purgeConcerns {
		concernsDir := filepath.Join(root, state.TddmasterDir, "concerns")
		if err := os.RemoveAll(concernsDir); err != nil && !os.IsNotExist(err) {
			return err
		}
		purged = append(purged, "concerns/")
	}

	if len(purged) > 0 {
		printErr("Purge complete.")
		for _, p := range purged {
			printErr(fmt.Sprintf("  Removed: %s", p))
		}
	} else {
		printErr("Nothing to purge.")
	}

	return nil
}
