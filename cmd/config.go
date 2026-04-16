
package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pragmataW/tddmaster/internal/output"
	"github.com/pragmataW/tddmaster/internal/state"
)

func newConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  "Manage tddmaster configuration (user identity, etc.).",
		RunE:  runConfig,
	}
}

func runConfig(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		printErr(fmt.Sprintf("Usage: %s config <set-user | get-user | clear-user>", output.CmdPrefix()))
		return nil
	}

	switch args[0] {
	case "set-user":
		return configSetUser(args[1:])
	case "get-user":
		return configGetUser()
	case "clear-user":
		return configClearUser()
	default:
		printErr(fmt.Sprintf("Usage: %s config <set-user | get-user | clear-user>", output.CmdPrefix()))
		return nil
	}
}

func configSetUser(args []string) error {
	var name, email string
	var fromGit bool

	for _, arg := range args {
		if strings.HasPrefix(arg, "--name=") {
			name = arg[len("--name="):]
		} else if strings.HasPrefix(arg, "--email=") {
			email = arg[len("--email="):]
		} else if arg == "--from-git" {
			fromGit = true
		}
	}

	if fromGit {
		gitUser, _ := state.DetectGitUser()
		if gitUser == nil {
			return fmt.Errorf("could not read git user config")
		}
		name = gitUser.Name
		email = gitUser.Email
	}

	if strings.TrimSpace(name) == "" {
		return fmt.Errorf(`please provide a name: %s config set-user --name="Your Name" --email="you@example.com"`, output.CmdPrefix())
	}

	user := state.User{Name: name, Email: email}
	if err := state.SetCurrentUser(user); err != nil {
		return err
	}

	printErr(fmt.Sprintf("User set: %s", state.FormatUser(user)))
	printErr(fmt.Sprintf("  Stored in: %s", state.GetUserFilePath()))
	return nil
}

func configGetUser() error {
	user, _ := state.GetCurrentUser()
	if user == nil {
		printErr("No user configured.")
		printErr(fmt.Sprintf("Set one with: %s config set-user --name=\"Your Name\"", output.CmdPrefix()))
	} else {
		printErr(fmt.Sprintf("User: %s", state.FormatUser(*user)))
		printErr(fmt.Sprintf("  File: %s", state.GetUserFilePath()))
	}
	return nil
}

func configClearUser() error {
	removed, _ := state.ClearCurrentUser()
	if removed {
		printErr("User identity cleared.")
	} else {
		printErr("No user configured.")
	}
	return nil
}
