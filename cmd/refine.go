package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pragmataW/tddmaster/internal/phasecatalog"
	"github.com/pragmataW/tddmaster/internal/spec"
	"github.com/spf13/cobra"
)

func newRefineCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refine <slug>",
		Short: "Apply refinement operations to a spec's task list",
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
			state, err := spec.LoadState(root, slug)
			if err != nil {
				return fmt.Errorf("load state: %w", err)
			}
			if state.Phase != string(phasecatalog.PhaseRefinement) {
				return fmt.Errorf("refine only valid in refinement phase, current phase: %s", state.Phase)
			}
			answer, _ := cmd.Flags().GetString("answer")
			if strings.TrimSpace(answer) == "" {
				return fmt.Errorf("--answer is required")
			}
			if !json.Valid([]byte(answer)) {
				return fmt.Errorf("invalid JSON in --answer")
			}
			var payload spec.RefinePayload
			if err := json.Unmarshal([]byte(answer), &payload); err != nil {
				return fmt.Errorf("unmarshal answer: %w", err)
			}
			progress, err := spec.LoadProgress(root, slug)
			if err != nil {
				return fmt.Errorf("load progress: %w", err)
			}
			newTasks, err := spec.ApplyRefinement(progress.Tasks, payload)
			if err != nil {
				return err
			}
			progress.Tasks = newTasks
			if err := spec.SaveProgress(root, slug, progress); err != nil {
				return fmt.Errorf("save progress: %w", err)
			}
			content := spec.RenderSpecMd(slug, state, progress)
			if err := spec.SaveSpecMd(root, slug, content); err != nil {
				return fmt.Errorf("save spec md: %w", err)
			}
			out := struct {
				Tasks []spec.Task `json:"tasks"`
				Hint  string      `json:"hint"`
			}{
				Tasks: newTasks,
				Hint:  fmt.Sprintf("run `tddmaster next %s` when satisfied", slug),
			}
			data, err := json.MarshalIndent(out, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal output: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		},
	}
	addRootFlag(cmd)
	cmd.Flags().String("answer", "", "Refinement payload (JSON)")
	return cmd
}
