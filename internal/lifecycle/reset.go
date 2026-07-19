package lifecycle

import (
	"fmt"
	"os"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/phasecatalog"
	"github.com/pragmataW/tddmaster/internal/spec"
)

type resetDescriptor struct {
	ID         engine.PhaseID
	AnswerKeys []string
	ApplyFn    func(state *spec.State, prog *spec.Progress, root, slug string) error
}

var resetDescriptors = []resetDescriptor{
	{
		ID:         phasecatalog.PhaseSettings,
		AnswerKeys: []string{"spec_settings"},
		ApplyFn: func(state *spec.State, prog *spec.Progress, root, slug string) error {
			return spec.SaveSettings(root, slug, spec.DefaultSettings())
		},
	},
	{
		ID: phasecatalog.PhaseDiscovery,
		AnswerKeys: []string{
			"listen_context", "mode", "premises", "status_quo", "ambition",
			"reversibility", "user_impact", "verification", "scope_boundary",
			"edge_cases", "synthesis",
		},
	},
	{
		ID:         phasecatalog.PhaseSpecProposal,
		AnswerKeys: []string{"tasks_generated", "self_review"},
		ApplyFn: func(state *spec.State, prog *spec.Progress, root, slug string) error {
			if err := os.Remove(paths.SpecMd(root, slug)); err != nil && !os.IsNotExist(err) {
				return err
			}
			prog.Tasks = []spec.Task{}
			prog.TaskSeq = 0
			prog.Status = spec.StatusDraft
			return nil
		},
	},
	{
		ID:         phasecatalog.PhaseRefinement,
		AnswerKeys: []string{"refinement_approved"},
	},
	{
		ID:         phasecatalog.PhaseAnalysis,
		AnswerKeys: []string{"analysis_complete", "analysis_audited", "analysis_findings", "analysis_attempts"},
		ApplyFn: func(state *spec.State, prog *spec.Progress, root, slug string) error {
			return spec.SaveAnalysis(root, slug, spec.Analysis{Verdict: "", Findings: []spec.Finding{}})
		},
	},
	{
		ID:         phasecatalog.PhaseExecution,
		AnswerKeys: []string{},
		ApplyFn: func(state *spec.State, prog *spec.Progress, root, slug string) error {
			if err := spec.SaveTraceability(root, slug, spec.Traceability{}); err != nil {
				return err
			}
			for i := range prog.Tasks {
				prog.Tasks[i].Done = false
				prog.Tasks[i].Exec = nil
				prog.Tasks[i].RefactorNotes = nil
				prog.Tasks[i].FailedACReasons = nil
				prog.Tasks[i].Blocked = false
				prog.Tasks[i].BlockedReason = ""
			}
			prog.Status = spec.StatusDraft
			prog.Iterations = 0
			return nil
		},
	},
	{
		ID:         phasecatalog.PhaseRuleLearning,
		AnswerKeys: []string{"rule_proposal", "rule_approved", "rule_applied", "rule_feedback", "rule_attempt"},
	},
}

func ResetFrom(target string, state *spec.State, prog *spec.Progress, root, slug string) ([]string, error) {
	targetIndex := -1
	for i, d := range resetDescriptors {
		if string(d.ID) == target {
			targetIndex = i
			break
		}
	}
	if targetIndex == -1 {
		return nil, fmt.Errorf("unknown reset target phase %q", target)
	}

	var warnings []string
	for j := targetIndex; j < len(resetDescriptors); j++ {
		desc := resetDescriptors[j]
		for _, key := range desc.AnswerKeys {
			delete(state.Answers, key)
		}
		if desc.ID == phasecatalog.PhaseRuleLearning {
			warnings = append(warnings, "reset does not touch global rule-learning files; review shared rule files under .tddmaster/rules manually")
			continue
		}
		if desc.ApplyFn == nil {
			continue
		}
		if err := desc.ApplyFn(state, prog, root, slug); err != nil {
			return warnings, err
		}
	}
	return warnings, nil
}
