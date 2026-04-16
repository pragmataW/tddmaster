
package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pragmataW/tddmaster/internal/output"
	"github.com/pragmataW/tddmaster/internal/state"
)

func newBlockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block",
		Short: "Block the current state",
		RunE:  runBlock,
	}
	cmd.Flags().String("spec", "", "Spec name")
	return cmd
}

func runBlock(cmd *cobra.Command, args []string) error {
	specFlag, _ := cmd.Flags().GetString("spec")
	specArgs := args
	if specFlag != "" {
		specArgs = append([]string{"--spec=" + specFlag}, args...)
	}
	return runBlockWithArgs(specArgs)
}

func runBlockWithArgs(args []string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}

	specResult := state.RequireSpecFlag(args)
	if !specResult.OK {
		return fmt.Errorf("%s", specResult.Error)
	}

	// Extract reason from non-spec args
	var filteredArgs []string
	for _, a := range args {
		if !strings.HasPrefix(a, "--spec=") {
			filteredArgs = append(filteredArgs, a)
		}
	}
	reason := strings.Join(filteredArgs, " ")
	if strings.TrimSpace(reason) == "" {
		reason = "No reason given"
	}

	st, err := state.ResolveState(root, &specResult.Spec)
	if err != nil {
		return err
	}

	if st.Spec != nil && !specDirExists(root, *st.Spec) {
		return fmt.Errorf("active spec '%s' directory not found", *st.Spec)
	}

	if st.Phase != state.PhaseExecuting {
		return fmt.Errorf("cannot block in phase: %s", st.Phase)
	}

	user, _ := state.ResolveUser(root)
	userInfo := &state.UserInfo{Name: user.Name, Email: user.Email}

	newState, err := state.BlockExecution(st, reason)
	if err != nil {
		return err
	}
	reasonStr := reason
	newState = state.RecordTransition(newState, state.PhaseExecuting, state.PhaseBlocked, userInfo, &reasonStr)

	if err := state.WriteState(root, newState); err != nil {
		return err
	}
	if newState.Spec != nil {
		_ = state.WriteSpecState(root, *newState.Spec, newState)
	}

	printErr(fmt.Sprintf("Spec blocked: %s", reason))
	printErr(fmt.Sprintf("Resolve with: %s", output.Cmd(`next --answer="resolution"`)))

	return nil
}
