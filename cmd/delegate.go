
package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pragmataW/tddmaster/internal/state"
)

func newDelegateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delegate",
		Short: "Delegate a question to another user",
		RunE:  runDelegate,
	}
	cmd.Flags().String("spec", "", "Spec name")
	return cmd
}

func runDelegate(cmd *cobra.Command, args []string) error {
	specFlag, _ := cmd.Flags().GetString("spec")
	specArgs := args
	if specFlag != "" {
		specArgs = append([]string{"--spec=" + specFlag}, args...)
	}
	return runDelegateWithArgs(specArgs)
}

func runDelegateWithArgs(args []string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}

	specResult := state.RequireSpecFlag(args)
	if !specResult.OK {
		return fmt.Errorf("%s", specResult.Error)
	}

	var subArgs []string
	for _, a := range args {
		if !strings.HasPrefix(a, "--spec=") {
			subArgs = append(subArgs, a)
		}
	}

	if len(subArgs) < 2 {
		return fmt.Errorf("usage: tddmaster delegate --spec=%s <question-id> <to-user>", specResult.Spec)
	}

	questionID := subArgs[0]
	delegateTo := subArgs[1]

	st, err := state.ResolveState(root, &specResult.Spec)
	if err != nil {
		return err
	}

	user, _ := state.ResolveUser(root)
	st = state.AddDelegation(st, questionID, delegateTo, user.Name)

	if err := state.WriteStateAndSpec(root, st); err != nil {
		return err
	}

	printErr(fmt.Sprintf("Question %s delegated to %s.", questionID, delegateTo))
	return nil
}
