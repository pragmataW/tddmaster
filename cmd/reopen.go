package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	specp "github.com/pragmataW/tddmaster/internal/spec"
	"github.com/pragmataW/tddmaster/internal/state"
)

func newReopenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reopen",
		Short: "Reopen a completed spec",
		RunE:  runReopen,
	}
	cmd.Flags().String("spec", "", "Spec name")
	cmd.Flags().Bool("resume-execution", false, "Resume the last EXECUTING/BLOCKED state instead of returning to DISCOVERY")
	return cmd
}

func runReopen(cmd *cobra.Command, args []string) error {
	specFlag, _ := cmd.Flags().GetString("spec")
	resumeExecution, _ := cmd.Flags().GetBool("resume-execution")
	specArgs := args
	if specFlag != "" {
		specArgs = append([]string{"--spec=" + specFlag}, args...)
	}
	if resumeExecution {
		specArgs = append(specArgs, "--resume-execution")
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
	filteredArgs, resumeExecution := stripExactArg(args, "--resume-execution")
	if err := rejectPositionalArgs(
		"reopen",
		filteredArgs,
		"`reopen` works at spec level. Use `--resume-execution` to recover the last execution state.",
	); err != nil {
		return err
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

	var newState state.StateFile
	if resumeExecution {
		newState, err = state.ResumeCompletedSpec(st)
		if err != nil {
			return err
		}
		reasonStr := "reopened-resume-execution"
		newState = state.RecordTransition(newState, state.PhaseCompleted, newState.Phase, userInfo, &reasonStr)
	} else {
		newState, err = state.ReopenSpec(st)
		if err != nil {
			return err
		}
		reasonStr := "reopened"
		newState = state.RecordTransition(newState, state.PhaseCompleted, state.PhaseDiscovery, userInfo, &reasonStr)
	}

	if err := state.WriteState(root, newState); err != nil {
		return err
	}
	if newState.Spec != nil {
		if err := state.WriteSpecState(root, *newState.Spec, newState); err != nil {
			return err
		}
		if resumeExecution {
			status := strings.ToLower(string(newState.Phase))
			if err := specp.UpdateSpecStatus(root, *newState.Spec, status); err != nil {
				printErr(fmt.Sprintf("warning: spec.md status update failed: %v", err))
			}
			if err := specp.UpdateProgressStatus(root, *newState.Spec, status); err != nil {
				printErr(fmt.Sprintf("warning: progress.json status update failed: %v", err))
			}
		}
	}

	if resumeExecution {
		printErr(fmt.Sprintf("Spec reopened and resumed in %s. Execution progress preserved.", newState.Phase))
		return nil
	}

	printErr("Spec reopened. Discovery answers preserved — run `tddmaster next` to revise.")
	return nil
}
