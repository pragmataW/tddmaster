
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pragmataW/tddmaster/internal/defaults"
	"github.com/pragmataW/tddmaster/internal/output"
)

func newPackCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pack",
		Short: "Pack project artifacts",
		Long:  "Manage rule packs (install, list, remove).",
		RunE:  runPack,
	}
}

func runPack(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		printErr(fmt.Sprintf("Usage: %s pack <list | install <name> | remove <name>>", output.CmdPrefix()))
		return nil
	}

	switch args[0] {
	case "list":
		return packList()
	case "install":
		if len(args) < 2 {
			return fmt.Errorf("usage: tddmaster pack install <name>")
		}
		return packInstall(args[1])
	case "remove":
		if len(args) < 2 {
			return fmt.Errorf("usage: tddmaster pack remove <name>")
		}
		return packRemove(args[1])
	default:
		printErr(fmt.Sprintf("Usage: %s pack <list | install <name> | remove <name>>", output.CmdPrefix()))
		return nil
	}
}

func packList() error {
	packs := defaults.DefaultPacks()
	printErr("Available packs:")
	printErr("")
	for name, pack := range packs {
		printErr(fmt.Sprintf("  %s", name))
		if pack.Manifest.Description != "" {
			printErr(fmt.Sprintf("    %s", pack.Manifest.Description))
		}
	}
	return nil
}

func packInstall(name string) error {
	packs := defaults.DefaultPacks()
	if _, ok := packs[name]; !ok {
		return fmt.Errorf("pack %q not found. Run `tddmaster pack list` to see available packs", name)
	}
	printErr(fmt.Sprintf("Pack %q installed (stub — not fully implemented).", name))
	return nil
}

func packRemove(name string) error {
	printErr(fmt.Sprintf("Pack %q removed (stub — not fully implemented).", name))
	return nil
}
