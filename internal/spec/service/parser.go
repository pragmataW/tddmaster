// Package service implements the spec business logic: parsing spec.md,
// rendering new spec documents, deriving task and edge-case lists from
// discovery state, and updating status artifacts on disk. Pure data
// shapes live in the sibling model package.
package service

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pragmataW/tddmaster/internal/spec/model"
	"github.com/pragmataW/tddmaster/internal/state"
)

var (
	taskLineRe   = regexp.MustCompile(`(?i)^-\s*\[[ x]\]\s*(task-\d+):\s*(.+)$`)
	filesLineRe  = regexp.MustCompile(`(?i)^Files?:\s*(.+)$`)
	coversLineRe = regexp.MustCompile(`(?i)^Covers?:\s*(.+)$`)
)

// ParseSpec reads a spec.md file from disk and returns structured data.
func ParseSpec(root, specName string) (*model.ParsedSpec, error) {
	specPath := filepath.Join(root, state.Paths{}.SpecFile(specName))
	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil, err
	}
	return parseContent(specName, string(data)), nil
}

// parseContent parses spec markdown content. Pure function, used both by
// ParseSpec (disk) and Generate (after render) to derive task IDs.
func parseContent(specName, content string) *model.ParsedSpec {
	var tasks []model.ParsedTask
	var outOfScope []string
	var verification []string

	currentSection := ""

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "## ") {
			currentSection = strings.ToLower(strings.TrimSpace(trimmed[3:]))
			continue
		}

		if strings.HasPrefix(currentSection, "tasks") {
			if m := taskLineRe.FindStringSubmatch(trimmed); m != nil {
				tasks = append(tasks, model.ParsedTask{
					ID:    m[1],
					Title: strings.TrimSpace(m[2]),
				})
			}

			if m := filesLineRe.FindStringSubmatch(trimmed); m != nil && len(tasks) > 0 {
				last := &tasks[len(tasks)-1]
				var fileList []string
				for _, f := range strings.Split(m[1], ",") {
					f = strings.TrimSpace(f)
					f = strings.TrimPrefix(f, "`")
					f = strings.TrimSuffix(f, "`")
					if f != "" {
						fileList = append(fileList, f)
					}
				}
				if len(fileList) > 0 {
					last.Files = fileList
				}
			}

			if m := coversLineRe.FindStringSubmatch(trimmed); m != nil && len(tasks) > 0 {
				last := &tasks[len(tasks)-1]
				var coverList []string
				for _, c := range strings.Split(m[1], ",") {
					c = strings.TrimSpace(c)
					if c != "" {
						coverList = append(coverList, strings.ToUpper(c))
					}
				}
				if len(coverList) > 0 {
					last.Covers = coverList
				}
			}
		}

		if strings.HasPrefix(currentSection, "out of scope") {
			if strings.HasPrefix(trimmed, "- ") {
				outOfScope = append(outOfScope, strings.TrimSpace(trimmed[2:]))
			}
		}

		if strings.HasPrefix(currentSection, "verification") {
			if strings.HasPrefix(trimmed, "- ") {
				verification = append(verification, strings.TrimSpace(trimmed[2:]))
			}
		}
	}

	if tasks == nil {
		tasks = []model.ParsedTask{}
	}
	if outOfScope == nil {
		outOfScope = []string{}
	}
	if verification == nil {
		verification = []string{}
	}

	return &model.ParsedSpec{
		Name:         specName,
		Tasks:        tasks,
		OutOfScope:   outOfScope,
		Verification: verification,
	}
}
