
// Cross-session learnings — persistent JSONL log of mistakes, conventions,
// and successes discovered during spec execution.

package dashboard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pragmataW/tddmaster/internal/state"
)

// =============================================================================
// Types
// =============================================================================

// LearningType represents the type of a learning.
type LearningType string

const (
	LearningTypeMistake    LearningType = "mistake"
	LearningTypeConvention LearningType = "convention"
	LearningTypeSuccess    LearningType = "success"
	LearningTypeDependency LearningType = "dependency"
)

// Learning is a single learning entry.
type Learning struct {
	Ts       string       `json:"ts"`
	Spec     string       `json:"spec"`
	Type     LearningType `json:"type"`
	Text     string       `json:"text"`
	Severity string       `json:"severity"` // "high" | "medium" | "low"
}

// =============================================================================
// Paths
// =============================================================================

const learningsFile = state.TddmasterDir + "/learnings.jsonl"

// =============================================================================
// Write
// =============================================================================

// AddLearning appends a learning to the JSONL log.
func AddLearning(root string, learning Learning) error {
	file := filepath.Join(root, learningsFile)

	// Ensure the parent directory exists
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}

	data, err := json.Marshal(learning)
	if err != nil {
		return err
	}
	line := string(data) + "\n"

	var existing []byte
	existing, _ = os.ReadFile(file)
	return os.WriteFile(file, append(existing, []byte(line)...), 0o644)
}

// =============================================================================
// Read
// =============================================================================

// ReadLearnings reads all learnings.
func ReadLearnings(root string) ([]Learning, error) {
	file := filepath.Join(root, learningsFile)

	content, err := os.ReadFile(file)
	if err != nil {
		return []Learning{}, nil
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	var learnings []Learning
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var l Learning
		if err := json.Unmarshal([]byte(line), &l); err != nil {
			continue
		}
		learnings = append(learnings, l)
	}
	if learnings == nil {
		return []Learning{}, nil
	}
	return learnings, nil
}

// RemoveLearning removes a learning by index (0-based). Returns false if index is out of range.
func RemoveLearning(root string, index int) (bool, error) {
	all, err := ReadLearnings(root)
	if err != nil {
		return false, err
	}
	if index < 0 || index >= len(all) {
		return false, nil
	}

	remaining := append(all[:index:index], all[index+1:]...)
	file := filepath.Join(root, learningsFile)

	var sb strings.Builder
	for _, l := range remaining {
		data, err := json.Marshal(l)
		if err != nil {
			return false, err
		}
		sb.Write(data)
		sb.WriteByte('\n')
	}

	return true, os.WriteFile(file, []byte(sb.String()), 0o644)
}

// =============================================================================
// Relevance filtering
// =============================================================================

// GetRelevantLearnings returns learnings relevant to a spec description (max 5).
func GetRelevantLearnings(root string, specDescription string) ([]Learning, error) {
	all, err := ReadLearnings(root)
	if err != nil {
		return nil, err
	}
	if len(all) == 0 {
		return []Learning{}, nil
	}

	descWords := make(map[string]bool)
	for _, w := range strings.Fields(strings.ToLower(specDescription)) {
		if len(w) > 3 {
			descWords[w] = true
		}
	}

	type scored struct {
		learning Learning
		score    int
	}

	var items []scored
	for _, learning := range all {
		score := 0

		switch learning.Severity {
		case "high":
			score += 3
		case "medium":
			score += 1
		}

		if learning.Type == LearningTypeConvention {
			score += 2
		}

		for _, word := range strings.Fields(strings.ToLower(learning.Text)) {
			if len(word) > 3 && descWords[word] {
				score += 2
			}
		}

		if len(learning.Text) < 80 {
			score += 1
		}

		items = append(items, scored{learning: learning, score: score})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].score > items[j].score
	})

	limit := 5
	if len(items) < limit {
		limit = len(items)
	}

	result := make([]Learning, limit)
	for i := 0; i < limit; i++ {
		result[i] = items[i].learning
	}
	return result, nil
}

// FormatLearnings formats learnings for compiler output.
func FormatLearnings(learnings []Learning) []string {
	result := make([]string, 0, len(learnings))
	for _, l := range learnings {
		icon := "\u26A0"
		label := "Dependency"
		switch l.Type {
		case LearningTypeMistake:
			icon = "\u26A0"
			label = "Past mistake"
		case LearningTypeSuccess:
			icon = "\u2713"
			label = "Success"
		case LearningTypeConvention:
			icon = "\u2713"
			label = "Convention"
		}
		result = append(result, icon+" "+label+": "+l.Text+" (from spec: "+l.Spec+")")
	}
	return result
}
