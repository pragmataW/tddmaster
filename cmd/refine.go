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
			if !spec.ValidSlug(slug) {
				return fmt.Errorf("invalid slug %q", slug)
			}
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
			settings, err := spec.LoadSettings(root, slug)
			if err != nil {
				return fmt.Errorf("load settings: %w", err)
			}
			newTasks, newSeq, err := spec.ApplyRefinement(progress.Tasks, payload, settings.TDDEnabled, progress.TaskSeq)
			if err != nil {
				return err
			}
			oldTasks := progress.Tasks
			oldSeq := progress.TaskSeq
			progress.Tasks = newTasks
			progress.TaskSeq = newSeq
			if err := spec.SaveProgress(root, slug, progress); err != nil {
				return fmt.Errorf("save progress: %w", err)
			}
			content := spec.RenderSpecMd(slug, state, progress)
			if err := spec.SaveSpecMd(root, slug, content); err != nil {
				progress.Tasks = oldTasks
				progress.TaskSeq = oldSeq
				if rbErr := spec.SaveProgress(root, slug, progress); rbErr != nil {
					return fmt.Errorf("save spec md: %v (progress rollback also failed: %w)", err, rbErr)
				}
				return fmt.Errorf("save spec md (progress rolled back): %w", err)
			}
			if len(payload.Remove) > 0 {
				if err := pruneTraceability(root, slug, payload.Remove); err != nil {
					return fmt.Errorf("prune traceability: %w", err)
				}
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

func pruneTraceability(root, slug string, removed []string) error {
	removedSet := make(map[string]bool, len(removed))
	for _, id := range removed {
		removedSet[id] = true
	}
	tr, err := spec.LoadTraceability(root, slug)
	if err != nil {
		return err
	}
	changed := false
	for file, entries := range tr.Entries {
		kept := entries[:0:0]
		for _, e := range entries {
			if !removedSet[e.TaskID] {
				kept = append(kept, e)
			}
		}
		if len(kept) == len(entries) {
			continue
		}
		changed = true
		if len(kept) == 0 {
			delete(tr.Entries, file)
		} else {
			tr.Entries[file] = kept
		}
	}
	if !changed {
		return nil
	}
	return spec.SaveTraceability(root, slug, tr)
}
