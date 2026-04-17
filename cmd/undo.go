package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	specp "github.com/pragmataW/tddmaster/internal/spec"
	"github.com/pragmataW/tddmaster/internal/state"
)

func newUndoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "undo [task-id]",
		Short: "Undo the most recently completed task (or a specific task by ID)",
		Long: `undo reverses the completion of a task. By default it resets the most
recently completed task. Provide a task ID to undo a specific task.

If the spec is in COMPLETED phase you must use 'reopen' first.`,
		RunE: runUndo,
	}
	cmd.Flags().String("spec", "", "Spec name")
	return cmd
}

func runUndo(cmd *cobra.Command, args []string) error {
	specFlag, _ := cmd.Flags().GetString("spec")
	specArgs := args
	if specFlag != "" {
		specArgs = append([]string{"--spec=" + specFlag}, specArgs...)
	}
	return runUndoWithArgs(specArgs)
}

func runUndoWithArgs(args []string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}

	// Resolve optional --spec flag (same pattern as other commands).
	specResult := state.RequireSpecFlag(args)

	var st state.StateFile
	if specResult.OK {
		st, err = state.ResolveState(root, &specResult.Spec)
	} else {
		// No --spec flag: use the globally active state.
		st, err = state.ReadState(root)
	}
	if err != nil {
		return err
	}

	// Guard: completed spec requires reopen first.
	if st.Phase == state.PhaseCompleted {
		specName := "this spec"
		if st.Spec != nil {
			specName = *st.Spec
		}
		return fmt.Errorf("spec '%s' is completed — use `tddmaster spec %s reopen --resume-execution` first to reopen it before undoing tasks", specName, specName)
	}

	if len(st.Execution.CompletedTasks) == 0 {
		return fmt.Errorf("no completed tasks to undo")
	}

	// Determine which task ID to undo.
	var taskID string

	// positional args after flag stripping
	var positional []string
	for _, a := range args {
		if len(a) > 0 && a[0] != '-' {
			positional = append(positional, a)
		}
	}

	if len(positional) > 0 {
		taskID = positional[0]
	} else {
		// Default: last completed task.
		taskID = st.Execution.CompletedTasks[len(st.Execution.CompletedTasks)-1]
	}

	newState, err := state.UncompleteTask(st, taskID)
	if err != nil {
		return err
	}

	// Regenerate spec.md and update state files (pattern from approve.go:86-87).
	config, _ := state.ReadManifest(root)
	allConcerns, _ := state.ListConcerns(root)
	activeConcerns := filterConcerns(allConcerns, config.Concerns)

	if newState.Spec != nil {
		if _, genErr := specp.GenerateSpec(root, &newState, activeConcerns); genErr != nil {
			printErr(fmt.Sprintf("warning: undo succeeded but spec.md regeneration failed: %v", genErr))
		}
		if err := state.WriteSpecState(root, *newState.Spec, newState); err != nil {
			return err
		}
	}

	if err := state.WriteState(root, newState); err != nil {
		return err
	}

	printErr(fmt.Sprintf("Task '%s' marked as incomplete.", taskID))
	return nil
}
