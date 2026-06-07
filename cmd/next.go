package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/phases"
	"github.com/pragmataW/tddmaster/internal/spec"
	"github.com/spf13/cobra"
)

func newNextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "next <slug>",
		Short: "Get or advance the next action for a spec",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			root, err := resolveRoot(cmd)
			if err != nil {
				return fmt.Errorf("resolve root: %w", err)
			}
			if !spec.Exists(root, slug) {
				return fmt.Errorf("spec %q not found: run tddmaster start %s first", slug, slug)
			}
			settings, err := spec.LoadSettings(root, slug)
			if err != nil {
				return fmt.Errorf("load settings: %w", err)
			}
			defs := phases.Enabled(settings)
			ctx, err := engine.Build(root, slug, defs)
			if err != nil {
				return fmt.Errorf("build context: %w", err)
			}
			answer, _ := cmd.Flags().GetString("answer")
			var action engine.Action
			if answer == "" {
				action, err = ctx.Next()
			} else {
				trimmed := strings.TrimSpace(answer)
				if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
					if !json.Valid([]byte(trimmed)) {
						return fmt.Errorf("invalid JSON in --answer: %q", trimmed)
					}
				}
				action, err = ctx.Submit([]byte(trimmed))
			}
			if err != nil {
				return fmt.Errorf("engine: %w", err)
			}
			data, err := json.MarshalIndent(action, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal action: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		},
	}
	addRootFlag(cmd)
	cmd.Flags().String("answer", "", "Answer to submit (JSON)")
	return cmd
}
