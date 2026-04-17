package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	specp "github.com/pragmataW/tddmaster/internal/spec"
	"github.com/pragmataW/tddmaster/internal/state"
)

func newCancelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel the current spec",
		RunE:  runCancel,
	}
	cmd.Flags().String("spec", "", "Spec name")
	return cmd
}

func runCancel(cmd *cobra.Command, args []string) error {
	specFlag, _ := cmd.Flags().GetString("spec")
	specArgs := args
	if specFlag != "" {
		specArgs = append([]string{"--spec=" + specFlag}, args...)
	}
	return runCancelWithArgs(specArgs)
}

func runCancelWithArgs(args []string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}

	specResult := state.RequireSpecFlag(args)
	if !specResult.OK {
		return fmt.Errorf("%s", specResult.Error)
	}
	if err := rejectPositionalArgs(
		"cancel",
		args,
		"`cancel` terminates the entire spec and does not accept a task ID. Use `wontfix` if you need a reasoned terminal resolution.",
	); err != nil {
		return err
	}

	st, err := state.ResolveState(root, &specResult.Spec)
	if err != nil {
		return err
	}

	if st.Phase == state.PhaseIdle || st.Phase == state.PhaseUninitialized || st.Phase == state.PhaseCompleted {
		return fmt.Errorf("cannot cancel in phase: %s", st.Phase)
	}

	if st.Spec != nil && !specDirExists(root, *st.Spec) {
		return fmt.Errorf("active spec '%s' directory not found", *st.Spec)
	}

	user, _ := state.ResolveUser(root)
	userInfo := &state.UserInfo{Name: user.Name, Email: user.Email}

	fromPhase := st.Phase
	reason := state.CompletionReasonCancelled
	completedState, err := state.CompleteSpec(st, reason, nil)
	if err != nil {
		return err
	}
	reasonStr := "cancelled"
	completedState = state.RecordTransition(completedState, fromPhase, state.PhaseCompleted, userInfo, &reasonStr)

	if completedState.Spec != nil {
		_ = state.WriteSpecState(root, *completedState.Spec, completedState)
	}

	idleState, err := state.ResetToIdle(completedState)
	if err != nil {
		idleState = state.CreateInitialState()
	}
	if err := state.WriteState(root, idleState); err != nil {
		return err
	}

	if completedState.Spec != nil {
		if err := specp.UpdateSpecStatus(root, *completedState.Spec, "cancelled"); err != nil {
			printErr(fmt.Sprintf("warning: spec.md status update failed: %v", err))
		}
		if err := specp.UpdateProgressStatus(root, *completedState.Spec, "cancelled"); err != nil {
			printErr(fmt.Sprintf("warning: progress.json status update failed: %v", err))
		}
	}

	printErr("Spec cancelled.")
	return nil
}
