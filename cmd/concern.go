
package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	ctxpkg "github.com/pragmataW/tddmaster/internal/context"
	"github.com/pragmataW/tddmaster/internal/output"
	"github.com/pragmataW/tddmaster/internal/state"
)

func newConcernCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "concern",
		Short: "Raise a concern about the current state",
		Long:  "Manage active concerns: add, remove, list.",
		RunE:  runConcern,
	}
}

func runConcern(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		printErr(fmt.Sprintf("Usage: %s concern <add <id> | remove <id> | list>", output.CmdPrefix()))
		return nil
	}

	switch args[0] {
	case "add":
		return concernAdd(args[1:])
	case "remove":
		return concernRemove(args[1:])
	case "list":
		return concernList()
	default:
		printErr(fmt.Sprintf("Usage: %s concern <add <id> | remove <id> | list>", output.CmdPrefix()))
		return nil
	}
}

func concernAdd(args []string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}

	var concernIds []string
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			concernIds = append(concernIds, a)
		}
	}

	if len(concernIds) == 0 {
		return fmt.Errorf("please provide concern ID(s): %s", output.Cmd("concern add open-source beautiful-product"))
	}

	config, _ := state.ReadManifest(root)
	if config == nil {
		return fmt.Errorf("tddmaster not initialized")
	}

	defaults := ctxpkg.LoadDefaultConcerns()
	var added []string

	for _, concernID := range concernIds {
		// Check local concerns first
		concern, _ := state.ReadConcern(root, concernID)

		if concern == nil {
			// Check defaults
			for _, d := range defaults {
				if d.ID == concernID {
					concern = &d
					break
				}
			}
			if concern != nil {
				_ = state.WriteConcern(root, *concern)
			}
		}

		if concern == nil {
			defaultIDs := make([]string, len(defaults))
			for i, d := range defaults {
				defaultIDs[i] = d.ID
			}
			printErr(fmt.Sprintf("Unknown concern: %s\n  Available: %s", concernID, strings.Join(defaultIDs, ", ")))
			continue
		}

		alreadyActive := false
		for _, id := range config.Concerns {
			if id == concernID {
				alreadyActive = true
				break
			}
		}
		if alreadyActive {
			printErr(fmt.Sprintf("Concern %q is already active.", concernID))
			continue
		}

		added = append(added, concernID)
	}

	if len(added) > 0 {
		config.Concerns = append(config.Concerns, added...)
		if err := state.WriteManifest(root, *config); err != nil {
			return err
		}
		printErr(fmt.Sprintf("Activated concerns: %s", strings.Join(added, ", ")))
	}

	return nil
}

func concernRemove(args []string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}

	if len(args) == 0 || args[0] == "" {
		return fmt.Errorf("please provide a concern ID: %s", output.Cmd("concern remove move-fast"))
	}
	concernID := args[0]

	config, _ := state.ReadManifest(root)
	if config == nil {
		return fmt.Errorf("tddmaster not initialized")
	}

	found := false
	for _, id := range config.Concerns {
		if id == concernID {
			found = true
			break
		}
	}
	if !found {
		printErr(fmt.Sprintf("Concern %q is not active.", concernID))
		return nil
	}

	var newConcerns []string
	for _, id := range config.Concerns {
		if id != concernID {
			newConcerns = append(newConcerns, id)
		}
	}
	config.Concerns = newConcerns

	if err := state.WriteManifest(root, *config); err != nil {
		return err
	}
	printErr(fmt.Sprintf("Deactivated concern: %s", concernID))
	return nil
}

func concernList() error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}

	config, _ := state.ReadManifest(root)
	activeConcernIds := map[string]bool{}
	if config != nil {
		for _, id := range config.Concerns {
			activeConcernIds[id] = true
		}
	}

	allConcerns := ctxpkg.LoadDefaultConcerns()

	printErr("Concerns\n")
	if len(allConcerns) == 0 {
		printErr("  No concerns defined.")
	} else {
		for _, c := range allConcerns {
			isActive := activeConcernIds[c.ID]
			marker := "○"
			if isActive {
				marker = "●"
			}
			desc := c.Description
			if len(desc) > 60 {
				desc = desc[:60] + "..."
			}
			printErr(fmt.Sprintf("  %s %s  %s", marker, c.ID, desc))
		}
	}
	return nil
}
