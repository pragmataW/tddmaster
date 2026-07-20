package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pragmataW/tddmaster/internal/errs"
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
				return errs.Newf(errs.KeyInvalidSlug, slug)
			}
			root, err := resolveRoot(cmd)
			if err != nil {
				return errs.Wrap(errs.KeyResolveRoot, err)
			}
			if !spec.Exists(root, slug) {
				return errs.Newf(errs.KeySpecNotFoundRunStart, slug, slug)
			}
			state, err := spec.LoadState(root, slug)
			if err != nil {
				return errs.Wrap(errs.KeyLoadState, err)
			}
			if state.Phase != string(phasecatalog.PhaseRefinement) {
				return errs.Newf(errs.KeyRefineWrongPhase, state.Phase)
			}
			answer, _ := cmd.Flags().GetString("answer")
			if strings.TrimSpace(answer) == "" {
				return errs.New(errs.KeyAnswerRequired)
			}
			if !json.Valid([]byte(answer)) {
				return errs.New(errs.KeyInvalidJSONInAnswer)
			}
			var payload spec.RefinePayload
			if err := json.Unmarshal([]byte(answer), &payload); err != nil {
				return errs.Wrap(errs.KeyUnmarshalAnswer, err)
			}
			progress, err := spec.LoadProgress(root, slug)
			if err != nil {
				return errs.Wrap(errs.KeyLoadProgress, err)
			}
			settings, err := spec.LoadSettings(root, slug)
			if err != nil {
				return errs.Wrap(errs.KeyLoadSettings, err)
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
				return errs.Wrap(errs.KeySaveProgress, err)
			}
			content := spec.RenderSpecMd(slug, state, progress)
			if err := spec.SaveSpecMd(root, slug, content); err != nil {
				progress.Tasks = oldTasks
				progress.TaskSeq = oldSeq
				if rbErr := spec.SaveProgress(root, slug, progress); rbErr != nil {
					return errs.Newf(errs.KeySaveSpecMDRollback, err, rbErr)
				}
				return errs.Wrap(errs.KeySaveSpecMDRolledBack, err)
			}
			if len(payload.Remove) > 0 {
				if err := pruneTraceability(root, slug, payload.Remove); err != nil {
					return errs.Wrap(errs.KeyPruneTraceability, err)
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
				return errs.Wrap(errs.KeyMarshalOutput, err)
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
