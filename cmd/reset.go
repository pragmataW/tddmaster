
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pragmataW/tddmaster/internal/state"
)

func newResetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset the current state",
		RunE:  runReset,
	}
	cmd.Flags().String("spec", "", "Spec name")
	return cmd
}

func runReset(cmd *cobra.Command, args []string) error {
	specFlag, _ := cmd.Flags().GetString("spec")
	specArgs := args
	if specFlag != "" {
		specArgs = append([]string{"--spec=" + specFlag}, args...)
	}
	return runResetWithArgs(specArgs)
}

func runResetWithArgs(args []string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}

	specResult := state.RequireSpecFlag(args)
	if !specResult.OK {
		return fmt.Errorf("%s", specResult.Error)
	}

	st, err := state.ResolveState(root, &specResult.Spec)
	if err != nil {
		return err
	}

	if st.Phase == state.PhaseIdle || st.Phase == state.PhaseUninitialized {
		printErr("Already idle. Nothing to reset.")
		return nil
	}

	specName := st.Spec

	newState, err := state.ResetToIdle(st)
	if err != nil {
		// Force idle
		newState = state.CreateInitialState()
	}

	if err := state.WriteState(root, newState); err != nil {
		return err
	}
	if specName != nil {
		_ = state.WriteSpecState(root, *specName, newState)
	}

	printErr("Reset to idle.")
	if specName != nil {
		printErr(fmt.Sprintf("Spec %q state cleared. Files in %s/specs/%s/ preserved.", *specName, state.TddmasterDir, *specName))
	}

	return nil
}
