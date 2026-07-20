package lifecycle

import (
	"fmt"
	"os"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/phasecatalog"
	"github.com/pragmataW/tddmaster/internal/spec"
)

const ruleLearningWarning = "global rule files under .tddmaster/rules are preserved; review shared rule files manually"

type resetDescriptor struct {
	ID         engine.PhaseID
	AnswerKeys []string
	MemFn      func(state *spec.State, prog *spec.Progress)
	ArtifactFn func(root, slug string) error
}

var resetDescriptors = []resetDescriptor{
	{
		ID:         phasecatalog.PhaseSettings,
		AnswerKeys: []string{"spec_settings"},
		ArtifactFn: func(root, slug string) error {
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
		MemFn: func(state *spec.State, prog *spec.Progress) {
			prog.Tasks = []spec.Task{}
			prog.TaskSeq = 0
			prog.Status = spec.StatusDraft
		},
		ArtifactFn: func(root, slug string) error {
			if err := os.Remove(paths.SpecMd(root, slug)); err != nil && !os.IsNotExist(err) {
				return err
			}
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
		ArtifactFn: func(root, slug string) error {
			return spec.SaveAnalysis(root, slug, spec.Analysis{Verdict: "", Findings: []spec.Finding{}})
		},
	},
	{
		ID:         phasecatalog.PhaseExecution,
		AnswerKeys: []string{},
		MemFn: func(state *spec.State, prog *spec.Progress) {
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
		},
		ArtifactFn: func(root, slug string) error {
			return spec.SaveTraceability(root, slug, spec.Traceability{})
		},
	},
	{
		ID:         phasecatalog.PhaseRuleLearning,
		AnswerKeys: []string{"rule_proposal", "rule_approved", "rule_applied", "rule_feedback", "rule_attempt"},
	},
}

func resetDescriptorIndex(target string) (int, error) {
	for i, d := range resetDescriptors {
		if string(d.ID) == target {
			return i, nil
		}
	}
	return -1, fmt.Errorf("unknown reset target phase %q", target)
}

func ResetMemory(target string, state *spec.State, prog *spec.Progress) ([]string, error) {
	targetIndex, err := resetDescriptorIndex(target)
	if err != nil {
		return nil, err
	}

	var warnings []string
	for j := targetIndex; j < len(resetDescriptors); j++ {
		desc := resetDescriptors[j]
		for _, key := range desc.AnswerKeys {
			delete(state.Answers, key)
		}
		if desc.ID == phasecatalog.PhaseRuleLearning {
			warnings = append(warnings, ruleLearningWarning)
		}
		if desc.MemFn != nil {
			desc.MemFn(state, prog)
		}
	}
	return warnings, nil
}

func ResetArtifacts(target, root, slug string) error {
	targetIndex, err := resetDescriptorIndex(target)
	if err != nil {
		return err
	}

	for j := targetIndex; j < len(resetDescriptors); j++ {
		if fn := resetDescriptors[j].ArtifactFn; fn != nil {
			if err := fn(root, slug); err != nil {
				return err
			}
		}
	}
	return nil
}

func ResetFrom(target string, state *spec.State, prog *spec.Progress, root, slug string) ([]string, error) {
	warnings, err := ResetMemory(target, state, prog)
	if err != nil {
		return warnings, err
	}
	return warnings, ResetArtifacts(target, root, slug)
}
