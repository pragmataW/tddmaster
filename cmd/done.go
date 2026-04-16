
package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/pragmataW/tddmaster/internal/output"
	specp "github.com/pragmataW/tddmaster/internal/spec"
	"github.com/pragmataW/tddmaster/internal/state"
)

func newDoneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "done",
		Short: "Mark the current task as done",
		RunE:  runDone,
	}
	cmd.Flags().String("spec", "", "Spec name")
	return cmd
}

func runDone(cmd *cobra.Command, args []string) error {
	specFlag, _ := cmd.Flags().GetString("spec")
	specArgs := args
	if specFlag != "" {
		specArgs = append([]string{"--spec=" + specFlag}, args...)
	}
	return runDoneWithArgs(specArgs)
}

func runDoneWithArgs(args []string) error {
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

	if st.Phase != state.PhaseExecuting {
		return fmt.Errorf("cannot complete in phase: %s. Only EXECUTING phase can transition to COMPLETED", st.Phase)
	}

	if st.Execution.AwaitingStatusReport {
		return fmt.Errorf("cannot complete: status report is pending. Submit a status report first: %s",
			output.Cmd(`next --answer='{"completed":[...],"remaining":[]}'`))
	}

	// Warn about unresolved debt
	if st.Execution.Debt != nil && len(st.Execution.Debt.Items) > 0 {
		printErr(fmt.Sprintf("Warning: %d unresolved debt item(s).", len(st.Execution.Debt.Items)))
		for _, item := range st.Execution.Debt.Items {
			printErr(fmt.Sprintf("  - %s: %s", item.ID, item.Text))
		}
	}

	if st.Spec != nil && !specDirExists(root, *st.Spec) {
		return fmt.Errorf("active spec '%s' directory not found", *st.Spec)
	}

	user, _ := state.ResolveUser(root)
	userInfo := &state.UserInfo{Name: user.Name, Email: user.Email}

	reason := state.CompletionReasonDone
	completedState, err := state.CompleteSpec(st, reason, nil)
	if err != nil {
		return err
	}
	fromPhase := state.PhaseExecuting
	reasonStr := "done"
	completedState = state.RecordTransition(completedState, fromPhase, state.PhaseCompleted, userInfo, &reasonStr)

	if completedState.Spec != nil {
		if err := state.WriteSpecState(root, *completedState.Spec, completedState); err != nil {
			return err
		}
	}

	idleState, err := state.ResetToIdle(completedState)
	if err != nil {
		// Force idle if reset fails
		idleState = state.CreateInitialState()
	}
	if err := state.WriteState(root, idleState); err != nil {
		return err
	}

	if completedState.Spec != nil {
		if err := specp.UpdateSpecStatus(root, *completedState.Spec, "completed"); err != nil {
			printErr(fmt.Sprintf("warning: spec.md status update failed: %v", err))
		}
		if err := specp.UpdateProgressStatus(root, *completedState.Spec, "completed"); err != nil {
			printErr(fmt.Sprintf("warning: progress.json status update failed: %v", err))
		}
	}

	specName := "unknown"
	if st.Spec != nil {
		specName = *st.Spec
	}

	printErr("Spec completed!")
	printErr("")
	printErr(fmt.Sprintf("  Spec:       %s", specName))
	printErr(fmt.Sprintf("  Iterations: %d", st.Execution.Iteration))
	printErr(fmt.Sprintf("  Decisions:  %d", len(st.Decisions)))
	printErr("")
	printErr(fmt.Sprintf("Start a new spec with: %s", output.Cmd(`spec new "..."`)))

	_ = time.Now // suppress import
	return nil
}
