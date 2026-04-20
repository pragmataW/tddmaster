// Package persistence reads and writes the main state.json and per-spec
// state JSON files under .tddmaster/.state/.
package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pragmataW/tddmaster/internal/state/model"
	"github.com/pragmataW/tddmaster/internal/state/service/atomic"
	"github.com/pragmataW/tddmaster/internal/state/service/paths"
)

var pathHelpers = paths.Paths{}

// CreateInitialState returns a new default StateFile.
func CreateInitialState() model.StateFile {
	return model.StateFile{
		Version:         "0.1.0",
		Phase:           model.PhaseIdle,
		Spec:            nil,
		SpecDescription: nil,
		Branch:          nil,
		Discovery: model.DiscoveryState{
			Answers:         []model.DiscoveryAnswer{},
			Prefills:        []model.DiscoveryPrefillQuestion{},
			Completed:       false,
			CurrentQuestion: 0,
			Audience:        "human",
			Approved:        false,
			PlanPath:        nil,
		},
		SpecState: model.SpecState{Path: nil, Status: "none"},
		Execution: model.ExecutionState{
			Iteration:            0,
			LastProgress:         nil,
			ModifiedFiles:        []string{},
			LastVerification:     nil,
			AwaitingStatusReport: false,
			Debt:                 nil,
			CompletedTasks:       []string{},
			DebtCounter:          0,
			NaItems:              []string{},
		},
		Decisions:        []model.Decision{},
		LastCalledAt:     nil,
		Classification:   nil,
		CompletionReason: nil,
		CompletedAt:      nil,
		CompletionNote:   nil,
		ReopenedFrom:     nil,
		RevisitHistory:   []model.RevisitEntry{},
	}
}

// ReadState reads the main state file, returning initial state on any error.
func ReadState(root string) (model.StateFile, error) {
	filePath := filepath.Join(root, paths.StateFilePath)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return CreateInitialState(), nil
	}

	var s model.StateFile
	if err := json.Unmarshal(data, &s); err != nil {
		return CreateInitialState(), nil
	}
	return s, nil
}

// ResolveState resolves state for a specific spec, or the active state if no spec given.
func ResolveState(root string, specName *string) (model.StateFile, error) {
	if specName == nil {
		return ReadState(root)
	}

	specDirPath := filepath.Join(root, pathHelpers.SpecDir(*specName))
	if _, err := os.Stat(specDirPath); err != nil {
		return model.StateFile{}, fmt.Errorf("spec '%s' not found. Run `tddmaster spec list` to see available specs", *specName)
	}

	specState, err := ReadSpecState(root, *specName)
	if err != nil {
		return model.StateFile{}, err
	}
	if specState.Spec != nil && *specState.Spec == *specName {
		return specState, nil
	}

	activeState, err := ReadState(root)
	if err != nil {
		return model.StateFile{}, err
	}
	if activeState.Spec != nil && *activeState.Spec == *specName {
		return activeState, nil
	}

	specState.Spec = specName
	return specState, nil
}

// WriteState writes the state file, creating directories as needed.
func WriteState(root string, s model.StateFile) error {
	filePath := filepath.Join(root, paths.StateFilePath)

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return atomic.WriteFileAtomic(filePath, data, 0o644)
}

// ReadActiveSpec returns the active spec name from state.json.
func ReadActiveSpec(root string) (*string, error) {
	s, err := ReadState(root)
	if err != nil {
		return nil, err
	}
	return s.Spec, nil
}

// ReadSpecState reads the per-spec state file.
func ReadSpecState(root, specName string) (model.StateFile, error) {
	filePath := filepath.Join(root, pathHelpers.SpecStateFile(specName))
	data, err := os.ReadFile(filePath)
	if err != nil {
		return CreateInitialState(), nil
	}

	var s model.StateFile
	if err := json.Unmarshal(data, &s); err != nil {
		return CreateInitialState(), nil
	}
	return s, nil
}

// WriteSpecState writes the per-spec state file.
func WriteSpecState(root, specName string, s model.StateFile) error {
	filePath := filepath.Join(root, pathHelpers.SpecStateFile(specName))

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return atomic.WriteFileAtomic(filePath, data, 0o644)
}

// ListSpecStates lists all spec names that have state files.
func ListSpecStates(root string) ([]model.SpecStateEntry, error) {
	dirPath := filepath.Join(root, paths.SpecStatesDir)
	var results []model.SpecStateEntry

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return results, nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		specName := strings.TrimSuffix(name, ".json")
		data, err := os.ReadFile(filepath.Join(dirPath, name))
		if err != nil {
			continue
		}
		var s model.StateFile
		if err := json.Unmarshal(data, &s); err != nil {
			continue
		}
		results = append(results, model.SpecStateEntry{Name: specName, State: s})
	}
	return results, nil
}

// WriteStateAndSpec writes main state AND the per-spec state file for the
// active spec. state.json is the primary source of truth; per-spec state is a
// derivative view. Each file is written atomically via WriteFileAtomic.
//
// If the per-spec write fails after the primary state was committed, the
// returned error explicitly signals that re-running the command will reconcile
// (the primary state is correct; the per-spec file will be rewritten from it).
func WriteStateAndSpec(root string, s model.StateFile) error {
	if err := WriteState(root, s); err != nil {
		return err
	}
	if s.Spec != nil {
		if err := WriteSpecState(root, *s.Spec, s); err != nil {
			return fmt.Errorf("primary state committed but per-spec state write failed; re-run command to reconcile: %w", err)
		}
	}
	return nil
}
