
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pragmataW/tddmaster/internal/state"
)

func newReopenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reopen",
		Short: "Reopen a closed task",
		RunE:  runReopen,
	}
	cmd.Flags().String("spec", "", "Spec name")
	return cmd
}

func runReopen(cmd *cobra.Command, args []string) error {
	specFlag, _ := cmd.Flags().GetString("spec")
	specArgs := args
	if specFlag != "" {
		specArgs = append([]string{"--spec=" + specFlag}, args...)
	}
	return runReopenWithArgs(specArgs)
}

func runReopenWithArgs(args []string) error {
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

	if st.Phase != state.PhaseCompleted {
		return fmt.Errorf("cannot reopen in phase: %s. Only COMPLETED specs can be reopened", st.Phase)
	}

	user, _ := state.ResolveUser(root)
	userInfo := &state.UserInfo{Name: user.Name, Email: user.Email}

	newState, err := state.ReopenSpec(st)
	if err != nil {
		return err
	}
	reasonStr := "reopened"
	newState = state.RecordTransition(newState, state.PhaseCompleted, state.PhaseDiscovery, userInfo, &reasonStr)

	if err := state.WriteState(root, newState); err != nil {
		return err
	}
	if newState.Spec != nil {
		_ = state.WriteSpecState(root, *newState.Spec, newState)
	}

	printErr("Spec reopened. Discovery answers preserved — run `tddmaster next` to revise.")
	return nil
}
