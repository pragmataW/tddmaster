package lifecycle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/errs"
	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/phasecatalog"
	"github.com/pragmataW/tddmaster/internal/spec"
)

type SpecInfo struct {
	Slug     string
	Phase    string
	Status   string
	Archived bool
}

func Rollback(root, slug, targetPhase string) ([]string, error) {
	if !spec.ValidSlug(slug) {
		return nil, errs.Newf(errs.KeyInvalidSlug, slug)
	}
	if !spec.Exists(root, slug) {
		return nil, errs.Newf(errs.KeySpecDoesNotExist, slug)
	}

	state, err := spec.LoadState(root, slug)
	if err != nil {
		return nil, err
	}
	prog, err := spec.LoadProgress(root, slug)
	if err != nil {
		return nil, err
	}
	settings, err := spec.LoadSettings(root, slug)
	if err != nil {
		return nil, err
	}

	targetIndex := -1
	currentIndex := -1
	validTargets := make([]string, 0, len(resetDescriptors))
	for i, d := range resetDescriptors {
		validTargets = append(validTargets, string(d.ID))
		if string(d.ID) == targetPhase {
			targetIndex = i
		}
		if string(d.ID) == state.Phase {
			currentIndex = i
		}
	}
	if targetIndex == -1 {
		return nil, errs.Newf(errs.KeyUnknownTargetPhase, targetPhase, validTargets)
	}
	if resetDescriptors[targetIndex].ID == phasecatalog.PhaseRuleLearning && !settings.RuleLearningEnabled {
		return nil, errs.Newf(errs.KeyRollbackRuleLearningDisabled, targetPhase)
	}
	if currentIndex == -1 {
		if state.Phase != string(engine.PhaseComplete) {
			return nil, errs.Newf(errs.KeyRollbackUnrecognizedPhase, state.Phase, validTargets)
		}
		currentIndex = len(resetDescriptors)
	}
	if targetIndex >= currentIndex {
		return nil, errs.Newf(errs.KeyRollbackNotEarlier, targetPhase, state.Phase)
	}

	warnings, err := ResetMemory(targetPhase, &state, &prog)
	if err != nil {
		return warnings, err
	}
	if !settings.RuleLearningEnabled {
		warnings = slices.DeleteFunc(warnings, func(w string) bool { return w == ruleLearningWarning })
	}

	state.Phase = targetPhase
	if err := spec.SaveState(root, slug, state); err != nil {
		return warnings, err
	}
	if err := spec.SaveProgress(root, slug, prog); err != nil {
		return warnings, err
	}
	if err := ResetArtifacts(targetPhase, root, slug); err != nil {
		return warnings, err
	}

	return warnings, nil
}

func Archive(root, slug string) error {
	if !spec.Exists(root, slug) {
		return errs.Newf(errs.KeySpecNotActive, slug)
	}
	archiveDir := paths.ArchiveSpecDir(root, slug)
	if _, err := os.Stat(archiveDir); err == nil {
		return errs.Newf(errs.KeySpecAlreadyArchived, slug)
	}
	if err := os.MkdirAll(filepath.Dir(archiveDir), 0o755); err != nil {
		return err
	}
	return os.Rename(paths.SpecDir(root, slug), archiveDir)
}

func Restore(root, slug string) error {
	archiveDir := paths.ArchiveSpecDir(root, slug)
	if _, err := os.Stat(archiveDir); err != nil {
		return errs.Newf(errs.KeyArchivedSpecNotFound, slug)
	}
	if spec.Exists(root, slug) {
		return errs.Newf(errs.KeyActiveSpecExists, slug)
	}
	specDir := paths.SpecDir(root, slug)
	if err := os.MkdirAll(filepath.Dir(specDir), 0o755); err != nil {
		return err
	}
	return os.Rename(archiveDir, specDir)
}

func Cancel(root, slug string) error {
	if !spec.ValidSlug(slug) {
		return errs.Newf(errs.KeyInvalidSlug, slug)
	}
	if !spec.Exists(root, slug) {
		return errs.Newf(errs.KeySpecDoesNotExist, slug)
	}
	return os.RemoveAll(paths.SpecDir(root, slug))
}

func List(root string) ([]SpecInfo, error) {
	infos := []SpecInfo{}

	active, err := listSpecDir(paths.Specs(root), false)
	if err != nil {
		return nil, err
	}
	infos = append(infos, active...)

	archived, err := listSpecDir(paths.Archive(root), true)
	if err != nil {
		return nil, err
	}
	infos = append(infos, archived...)

	return infos, nil
}

func listSpecDir(dir string, archived bool) ([]SpecInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	infos := make([]SpecInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		slug := entry.Name()
		info := SpecInfo{Slug: slug, Status: spec.StatusUnknown, Archived: archived}

		statePath := filepath.Join(dir, slug, paths.FileState)
		if data, err := os.ReadFile(statePath); err == nil {
			var state spec.State
			if err := json.Unmarshal(data, &state); err == nil {
				info.Phase = state.Phase
			}
		}

		progressPath := filepath.Join(dir, slug, paths.FileProgress)
		if data, err := os.ReadFile(progressPath); err == nil {
			var prog spec.Progress
			if err := json.Unmarshal(data, &prog); err == nil {
				info.Status = prog.Status
			}
		}

		infos = append(infos, info)
	}
	return infos, nil
}
