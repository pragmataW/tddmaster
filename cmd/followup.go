
package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pragmataW/tddmaster/internal/state"
)

func newFollowupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "followup",
		Short: "Manage follow-up questions",
		RunE:  runFollowup,
	}
	cmd.Flags().String("spec", "", "Spec name")
	return cmd
}

func runFollowup(cmd *cobra.Command, args []string) error {
	specFlag, _ := cmd.Flags().GetString("spec")
	specArgs := args
	if specFlag != "" {
		specArgs = append([]string{"--spec=" + specFlag}, args...)
	}
	return runFollowupWithArgs(specArgs)
}

func runFollowupWithArgs(args []string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}

	specResult := state.RequireSpecFlag(args)
	if !specResult.OK {
		return fmt.Errorf("%s", specResult.Error)
	}

	// Find subcommand after --spec=
	var subArgs []string
	for _, a := range args {
		if !strings.HasPrefix(a, "--spec=") {
			subArgs = append(subArgs, a)
		}
	}

	if len(subArgs) == 0 {
		return followupList(root, specResult.Spec)
	}

	switch subArgs[0] {
	case "add":
		return followupAdd(root, specResult.Spec, subArgs[1:])
	case "answer":
		return followupAnswer(root, specResult.Spec, subArgs[1:])
	case "list":
		return followupList(root, specResult.Spec)
	default:
		return fmt.Errorf("unknown followup subcommand: %s (use add, answer, list)", subArgs[0])
	}
}

func followupList(root, specName string) error {
	st, err := state.ResolveState(root, &specName)
	if err != nil {
		return err
	}

	followUps := state.GetPendingFollowUps(st)
	if len(followUps) == 0 {
		printErr(fmt.Sprintf("No pending follow-ups for spec %s.", specName))
		return nil
	}

	printErr(fmt.Sprintf("Pending follow-ups for spec %s:", specName))
	for _, fu := range followUps {
		printErr(fmt.Sprintf("  %s: %s", fu.ID, fu.Question))
	}
	return nil
}

func followupAdd(root, specName string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: tddmaster followup --spec=%s add <question-id> <question text>", specName)
	}

	questionID := args[0]
	questionText := strings.Join(args[1:], " ")
	if questionText == "" {
		return fmt.Errorf("please provide the follow-up question text")
	}

	st, err := state.ResolveState(root, &specName)
	if err != nil {
		return err
	}

	user, _ := state.ResolveUser(root)
	st = state.AddFollowUp(st, questionID, questionText, user.Name)

	if err := state.WriteStateAndSpec(root, st); err != nil {
		return err
	}

	printErr(fmt.Sprintf("Follow-up added: %s", questionID))
	return nil
}

func followupAnswer(root, specName string, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: tddmaster followup --spec=%s answer <follow-up-id> <answer>", specName)
	}

	followUpID := args[0]
	answer := strings.Join(args[1:], " ")

	st, err := state.ResolveState(root, &specName)
	if err != nil {
		return err
	}

	st = state.AnswerFollowUp(st, followUpID, answer)

	if err := state.WriteStateAndSpec(root, st); err != nil {
		return err
	}

	printErr(fmt.Sprintf("Follow-up %s answered.", followUpID))
	return nil
}
