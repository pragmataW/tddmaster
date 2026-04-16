
// Package spec provides spec file reading, writing, and updating for tddmaster.
package spec

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pragmataW/tddmaster/internal/state"
)

// =============================================================================
// Types
// =============================================================================

// ParsedTask represents a single task extracted from a spec.md file.
type ParsedTask struct {
	ID     string
	Title  string
	Files  []string
	Covers []string // EC IDs this task covers, e.g. ["EC-1","EC-3"]
}

// ParsedSpec represents the structured content of a spec.md file.
type ParsedSpec struct {
	Name        string
	Tasks       []ParsedTask
	OutOfScope  []string
	Verification []string
}

// =============================================================================
// Parser
// =============================================================================

// ParseSpec reads a spec.md file from disk and returns structured data.
func ParseSpec(root, specName string) (*ParsedSpec, error) {
	specPath := filepath.Join(root, state.Paths{}.SpecFile(specName))

	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil, err
	}

	parsed := ParseSpecContent(specName, string(data))
	return parsed, nil
}

var taskLineRe = regexp.MustCompile(`(?i)^-\s*\[[ x]\]\s*(task-\d+):\s*(.+)$`)
var filesLineRe = regexp.MustCompile(`(?i)^Files?:\s*(.+)$`)
var coversLineRe = regexp.MustCompile(`(?i)^Covers?:\s*(.+)$`)

// ParseSpecContent parses spec markdown content (pure function).
func ParseSpecContent(specName, content string) *ParsedSpec {
	var tasks []ParsedTask
	var outOfScope []string
	var verification []string

	currentSection := ""

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)

		// Track current section
		if strings.HasPrefix(trimmed, "## ") {
			currentSection = strings.ToLower(strings.TrimSpace(trimmed[3:]))
			continue
		}

		// Parse task checkboxes: - [ ] task-N: Title
		if strings.HasPrefix(currentSection, "tasks") {
			if m := taskLineRe.FindStringSubmatch(trimmed); m != nil {
				tasks = append(tasks, ParsedTask{
					ID:    m[1],
					Title: strings.TrimSpace(m[2]),
				})
			}

			// Parse file hints: Files: `path/to/file.ts`, `path/to/other.ts`
			if m := filesLineRe.FindStringSubmatch(trimmed); m != nil && len(tasks) > 0 {
				last := &tasks[len(tasks)-1]
				parts := strings.Split(m[1], ",")
				var fileList []string
				for _, f := range parts {
					f = strings.TrimSpace(f)
					f = strings.TrimPrefix(f, "`")
					f = strings.TrimSuffix(f, "`")
					if len(f) > 0 {
						fileList = append(fileList, f)
					}
				}
				if len(fileList) > 0 {
					last.Files = fileList
				}
			}

			// Parse edge case coverage: Covers: EC-1, EC-3
			if m := coversLineRe.FindStringSubmatch(trimmed); m != nil && len(tasks) > 0 {
				last := &tasks[len(tasks)-1]
				parts := strings.Split(m[1], ",")
				var coverList []string
				for _, c := range parts {
					c = strings.TrimSpace(c)
					if len(c) > 0 {
						coverList = append(coverList, strings.ToUpper(c))
					}
				}
				if len(coverList) > 0 {
					last.Covers = coverList
				}
			}
		}

		// Parse out of scope items
		if strings.HasPrefix(currentSection, "out of scope") {
			if strings.HasPrefix(trimmed, "- ") {
				outOfScope = append(outOfScope, strings.TrimSpace(trimmed[2:]))
			}
		}

		// Parse verification items
		if strings.HasPrefix(currentSection, "verification") {
			if strings.HasPrefix(trimmed, "- ") {
				verification = append(verification, strings.TrimSpace(trimmed[2:]))
			}
		}
	}

	if tasks == nil {
		tasks = []ParsedTask{}
	}
	if outOfScope == nil {
		outOfScope = []string{}
	}
	if verification == nil {
		verification = []string{}
	}

	return &ParsedSpec{
		Name:        specName,
		Tasks:       tasks,
		OutOfScope:  outOfScope,
		Verification: verification,
	}
}

// FindNextTask returns the first task not in completedIDs.
func FindNextTask(tasks []ParsedTask, completedIDs []string) *ParsedTask {
	completed := make(map[string]bool, len(completedIDs))
	for _, id := range completedIDs {
		completed[id] = true
	}

	for i := range tasks {
		if !completed[tasks[i].ID] {
			return &tasks[i]
		}
	}

	return nil
}
