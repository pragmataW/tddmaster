package shared

import (
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	conventionsHeader = "## Project Conventions"
	conventionsIntro  = "The following rules are auto-synced from your project and home configuration. They are non-negotiable. Follow them alongside your task-specific instructions below."
	conventionsFooter = "---"
	activeRulesHeader = "### Active Rules"
	sourceSeparator   = "\n\n---\n\n"
	maxImportDepth    = 3
)

var importLineRegex = regexp.MustCompile(`^@([\w./-]+\.md)\s*$`)

// ConventionSources names the project-relative and home-absolute convention
// files an adapter wants injected into its subagent prompts.
type ConventionSources struct {
	ProjectFile string
	HomeFile    string
}

// ReadConventions reads ProjectFile (relative to root) and HomeFile (supports
// "~/" prefix), strips the tddmaster-delimited block from each, recursively
// expands "@foo.md" import directives (max depth 3, cycle-guarded), and joins
// the results with a markdown separator. Returns "" when no source yields
// content.
func ReadConventions(root string, src ConventionSources) string {
	var parts []string

	if src.ProjectFile != "" {
		projectPath := filepath.Join(root, src.ProjectFile)
		if content := loadAndStrip(projectPath); content != "" {
			seen := map[string]bool{absOrSelf(projectPath): true}
			expanded := expandImports(content, filepath.Dir(projectPath), seen, 1)
			parts = append(parts, strings.TrimSpace(expanded))
		}
	}

	if src.HomeFile != "" {
		homePath := resolveHomePath(src.HomeFile)
		if homePath != "" {
			if content := loadAndStrip(homePath); content != "" {
				seen := map[string]bool{absOrSelf(homePath): true}
				expanded := expandImports(content, filepath.Dir(homePath), seen, 1)
				parts = append(parts, strings.TrimSpace(expanded))
			}
		}
	}

	var nonEmpty []string
	for _, p := range parts {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	if len(nonEmpty) == 0 {
		return ""
	}
	return strings.Join(nonEmpty, sourceSeparator)
}

// ConventionsPreamble returns the full preamble block that adapters prepend to
// subagent bodies. It combines ReadConventions output with active rules
// rendered from ctx.Rules (sourced from .tddmaster/rules/). Returns "" when
// inject is false or when both sources are empty.
func ConventionsPreamble(root string, src ConventionSources, rules []string, inject bool) string {
	if !inject {
		return ""
	}

	conventions := ReadConventions(root, src)
	rulesBlock := renderRulesBlock(rules)

	if conventions == "" && rulesBlock == "" {
		return ""
	}

	var body []string
	if conventions != "" {
		body = append(body, conventions)
	}
	if rulesBlock != "" {
		body = append(body, rulesBlock)
	}

	lines := []string{
		conventionsHeader,
		"",
		conventionsIntro,
		"",
		strings.Join(body, "\n\n"),
		"",
		conventionsFooter,
		"",
		"",
	}
	return strings.Join(lines, "\n")
}

func renderRulesBlock(rules []string) string {
	cleaned := make([]string, 0, len(rules))
	for _, r := range rules {
		if strings.TrimSpace(r) != "" {
			cleaned = append(cleaned, r)
		}
	}
	if len(cleaned) == 0 {
		return ""
	}

	lines := []string{activeRulesHeader, ""}
	for _, r := range cleaned {
		if strings.ContainsRune(r, '\n') {
			lines = append(lines, r, "")
			continue
		}
		lines = append(lines, "- "+r)
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

func loadAndStrip(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return stripTddmasterBlock(string(data))
}

func stripTddmasterBlock(content string) string {
	startIdx := strings.Index(content, NosStart)
	if startIdx == -1 {
		return content
	}
	endIdx := strings.Index(content, NosEnd)
	if endIdx == -1 || endIdx < startIdx {
		return content
	}
	before := content[:startIdx]
	after := content[endIdx+len(NosEnd):]
	return strings.TrimRight(before, "\n") + "\n" + strings.TrimLeft(after, "\n")
}

func expandImports(content, baseDir string, seen map[string]bool, depth int) string {
	if depth > maxImportDepth {
		return content
	}

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		match := importLineRegex.FindStringSubmatch(strings.TrimSpace(line))
		if match == nil {
			continue
		}

		refPath := match[1]
		resolved := refPath
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(baseDir, resolved)
		}
		absKey := absOrSelf(resolved)

		if seen[absKey] {
			continue
		}

		data, err := os.ReadFile(resolved)
		if err != nil {
			continue
		}

		childSeen := cloneSeen(seen)
		childSeen[absKey] = true
		nested := stripTddmasterBlock(string(data))
		nested = expandImports(nested, filepath.Dir(resolved), childSeen, depth+1)
		lines[i] = strings.TrimSpace(nested)
	}
	return strings.Join(lines, "\n")
}

func cloneSeen(seen map[string]bool) map[string]bool {
	out := make(map[string]bool, len(seen)+1)
	maps.Copy(out, seen)
	return out
}

func absOrSelf(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}

func resolveHomePath(path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	return path
}
