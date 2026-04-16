
package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newInvokeHookCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "invoke-hook",
		Short: "Invoke a lifecycle hook",
		RunE:  runInvokeHook,
	}
}

func runInvokeHook(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		printErr("Usage: tddmaster invoke-hook <hook-name> [args...]")
		printErr("  Available hooks: pre-tool-use, post-tool-use, pre-task, post-task, stop")
		return nil
	}

	hookName := args[0]
	hookArgs := args[1:]

	printErr(fmt.Sprintf("Invoking hook: %s", hookName))
	if len(hookArgs) > 0 {
		printErr(fmt.Sprintf("  Args: %s", strings.Join(hookArgs, " ")))
	}

	// Hook invocation is a no-op in the Go port for now.
	// In the full implementation, this would call the bridge package.
	return nil
}
