// Package concerns reads and writes concern definition JSON files under
// .tddmaster/concerns/.
package concerns

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/pragmataW/tddmaster/internal/state/model"
	"github.com/pragmataW/tddmaster/internal/state/service/atomic"
	"github.com/pragmataW/tddmaster/internal/state/service/paths"
)

var pathHelpers = paths.Paths{}

// ReadConcern reads a concern definition by ID.
func ReadConcern(root, concernID string) (*model.ConcernDefinition, error) {
	filePath := filepath.Join(root, pathHelpers.ConcernFile(concernID))
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil
	}

	var c model.ConcernDefinition
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, nil
	}
	return &c, nil
}

// WriteConcern writes a concern definition to disk.
func WriteConcern(root string, concern model.ConcernDefinition) error {
	filePath := filepath.Join(root, pathHelpers.ConcernFile(concern.ID))

	data, err := json.MarshalIndent(concern, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return atomic.WriteFileAtomic(filePath, data, 0o644)
}

// ListConcerns lists all concern definitions.
func ListConcerns(root string) ([]model.ConcernDefinition, error) {
	dirPath := filepath.Join(root, paths.ConcernsDir)
	var concerns []model.ConcernDefinition

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return concerns, nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dirPath, name))
		if err != nil {
			continue
		}
		var c model.ConcernDefinition
		if err := json.Unmarshal(data, &c); err != nil {
			continue
		}
		concerns = append(concerns, c)
	}
	return concerns, nil
}
