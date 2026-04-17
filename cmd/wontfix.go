
package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	specp "github.com/pragmataW/tddmaster/internal/spec"
	"github.com/pragmataW/tddmaster/internal/state"
)

func newWontfixCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wontfix",
		Short: "Mark the current spec as wontfix",
		RunE:  runWontfix,
	}
	cmd.Flags().String("spec", "", "Spec name")
	return cmd
}

func runWontfix(cmd *cobra.Command, args []string) error {
	specFlag, _ := cmd.Flags().GetString("spec")
	specArgs := args
	if specFlag != "" {
		specArgs = append([]string{"--spec=" + specFlag}, args...)
	}
	return runWontfixWithArgs(specArgs)
}

func runWontfixWithArgs(args []string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}

	specResult := state.RequireSpecFlag(args)
	if !specResult.OK {
		return fmt.Errorf("%s", specResult.Error)
	}

	var filteredArgs []string
	for _, a := range args {
		if !strings.HasPrefix(a, "--spec=") {
			filteredArgs = append(filteredArgs, a)
		}
	}
	reason := strings.Join(filteredArgs, " ")

	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf(`a reason is required: tddmaster wontfix "reason text"`)
	}

	st, err := state.ResolveState(root, &specResult.Spec)
	if err != nil {
		return err
	}

	if st.Phase == state.PhaseIdle || st.Phase == state.PhaseUninitialized || st.Phase == state.PhaseCompleted {
		return fmt.Errorf("cannot mark as won't fix in phase: %s", st.Phase)
	}

	if st.Spec != nil && !specDirExists(root, *st.Spec) {
		return fmt.Errorf("active spec '%s' directory not found", *st.Spec)
	}

	user, _ := state.ResolveUser(root)
	userInfo := &state.UserInfo{Name: user.Name, Email: user.Email}

	fromPhase := st.Phase
	wontfixReason := state.CompletionReasonWontfix
	note := reason
	completedState, err := state.CompleteSpec(st, wontfixReason, &note)
	if err != nil {
		return err
	}
	completedState = state.RecordTransition(completedState, fromPhase, state.PhaseCompleted, userInfo, &reason)

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
		if err := specp.UpdateSpecStatus(root, *completedState.Spec, "wontfix"); err != nil {
			printErr(fmt.Sprintf("warning: spec.md status update failed: %v", err))
		}
		if err := specp.UpdateProgressStatus(root, *completedState.Spec, "wontfix"); err != nil {
			printErr(fmt.Sprintf("warning: progress.json status update failed: %v", err))
		}
	}

	printErr(fmt.Sprintf("Spec marked as won't fix: %s", reason))
	return nil
}
