package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pragmataW/tddmaster/internal/errs"
	"github.com/pragmataW/tddmaster/internal/manifest"
	"github.com/pragmataW/tddmaster/internal/scaffold"
	"github.com/pragmataW/tddmaster/internal/ui/initform"
	"github.com/spf13/cobra"
)

func isTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func commandName() string {
	if len(os.Args) == 0 {
		return "tddmaster"
	}
	name := filepath.Base(os.Args[0])
	if name == "" {
		return "tddmaster"
	}
	return name
}

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize tddmaster in the current directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			nonInteractive, _ := cmd.Flags().GetBool("non-interactive")
			toolsFlag, _ := cmd.Flags().GetStringSlice("tools")
			maxIter, _ := cmd.Flags().GetInt("max-iteration")

			root, err := os.Getwd()
			if err != nil {
				return errs.Wrap(errs.KeyGetCwd, err)
			}

			cmdName := commandName()

			if !nonInteractive && !isTTY() {
				return errs.New(errs.KeyNoTTYNonInteractive)
			}

			if nonInteractive {
				var tools []manifest.ToolID
				for _, raw := range toolsFlag {
					for _, part := range strings.Split(raw, ",") {
						part = strings.TrimSpace(part)
						if part != "" {
							tools = append(tools, manifest.ToolID(part))
						}
					}
				}

				if len(tools) == 0 {
					return errs.New(errs.KeyToolRequiredInit)
				}

				if maxIter <= 0 {
					maxIter = 15
				}

				m := manifest.Manifest{
					SelectedTools:           tools,
					MaxIterationBeforeStart: maxIter,
					Command:                 cmdName,
				}

				res, err := scaffold.Scaffold(scaffold.Options{
					Root:           root,
					NonInteractive: true,
					Manifest:       &m,
				})
				if err != nil {
					return errs.Wrap(errs.KeyScaffold, err)
				}

				fmt.Fprintln(cmd.OutOrStdout(), initform.RenderSummary(res, cmdName))
				return nil
			}

			existing := scaffold.LoadManifestOrDefaults(root)
			initform.PlayIntro()
			formRes, err := initform.Run(existing)
			if err != nil {
				return errs.Wrap(errs.KeyForm, err)
			}

			if !formRes.Confirmed {
				fmt.Fprintln(cmd.OutOrStdout(), "init aborted.")
				return nil
			}

			m := manifest.Manifest{
				SelectedTools:           formRes.Tools,
				MaxIterationBeforeStart: formRes.MaxIteration,
				Command:                 cmdName,
			}

			res, err := scaffold.Scaffold(scaffold.Options{
				Root:     root,
				Manifest: &m,
			})
			if err != nil {
				return errs.Wrap(errs.KeyScaffold, err)
			}

			initform.PlayOutro(res, cmdName)
			return nil
		},
	}

	cmd.Flags().Bool("non-interactive", false, "Skip interactive prompts")
	cmd.Flags().StringSlice("tools", nil, "Comma-separated tool IDs (e.g. claude-code)")
	cmd.Flags().Int("max-iteration", 15, "Max verification iterations before start")

	return cmd
}
