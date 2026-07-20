package cmd

import (
	"os"

	"github.com/pragmataW/tddmaster/internal/errs"
	"github.com/pragmataW/tddmaster/internal/ui/ruleform"
	"github.com/spf13/cobra"
)

func newRuleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rule",
		Short: "Manage project rule files",
	}
	cmd.AddCommand(newRuleAddCmd())
	return cmd
}

func newRuleAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new rule file interactively or non-interactively",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, _ := cmd.Flags().GetString("root")
			if root == "" {
				cwd, err := os.Getwd()
				if err != nil {
					return errs.Wrap(errs.KeyGetCwd, err)
				}
				root = cwd
			}
			scope, _ := cmd.Flags().GetString("scope")
			name, _ := cmd.Flags().GetString("name")
			if (scope != "") != (name != "") {
				return errs.New(errs.KeyRuleNonInteractive)
			}
			if scope != "" && name != "" {
				content, _ := cmd.Flags().GetString("content")
				contentFile, _ := cmd.Flags().GetString("content-file")
				if content != "" && contentFile != "" {
					return errs.New(errs.KeyContentExclusive)
				}
				_, err := runRuleAddNonInteractive(root, scope, name, content, contentFile)
				return err
			}
			return ruleform.Run(root)
		},
	}
	cmd.Flags().String("root", "", "Root directory (default: cwd)")
	cmd.Flags().String("scope", "", "Scope / target agent (e.g. executor, global)")
	cmd.Flags().String("name", "", "Rule name")
	cmd.Flags().String("content", "", "Rule content")
	cmd.Flags().String("content-file", "", "Path to file whose content becomes the rule body")
	return cmd
}

func runRuleAddNonInteractive(root, scope, name, content, contentFile string) (string, error) {
	body := content
	if contentFile != "" {
		raw, err := os.ReadFile(contentFile)
		if err != nil {
			return "", errs.Wrap(errs.KeyReadContentFile, err, contentFile)
		}
		body = string(raw)
	}

	return ruleform.WriteRuleNoOverwrite(root, scope, name, body)
}
