
package context

import (
	"os"
	"strings"
)

// =============================================================================
// Types
// =============================================================================

// FolderRule is a rule scoped to a specific folder from .folder-rules.md.
type FolderRule struct {
	Folder string `json:"folder"`
	Rule   string `json:"rule"`
}

// =============================================================================
// Reader
// =============================================================================

// parseRulesFile parses a .folder-rules.md file — each bullet or non-empty line is a rule.
func parseRulesFile(content string) []string {
	var rules []string
	for _, line := range strings.Split(content, "\n") {
		// Remove bullet markers
		line = strings.TrimPrefix(line, "-")
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			rules = append(rules, line)
		}
	}
	return rules
}

// collectRulesForFile walks up from a file path collecting .folder-rules.md rules.
func collectRulesForFile(root, filePath string) []FolderRule {
	var rules []FolderRule

	// Normalize: remove root prefix to get relative path
	relative := filePath
	if strings.HasPrefix(filePath, root) {
		relative = filePath[len(root):]
		if strings.HasPrefix(relative, "/") {
			relative = relative[1:]
		}
	}

	// Walk up directory chain
	parts := strings.Split(relative, "/")
	if len(parts) > 0 {
		parts = parts[:len(parts)-1] // remove filename
	}

	for i := len(parts); i >= 0; i-- {
		var dir string
		var folderLabel string
		if i == 0 {
			dir = root
			folderLabel = "."
		} else {
			dir = root + "/" + strings.Join(parts[:i], "/")
			folderLabel = strings.Join(parts[:i], "/")
		}

		rulesFile := dir + "/.folder-rules.md"
		data, err := os.ReadFile(rulesFile)
		if err != nil {
			continue
		}

		parsed := parseRulesFile(string(data))
		for _, rule := range parsed {
			rules = append(rules, FolderRule{Folder: folderLabel, Rule: rule})
		}
	}

	return rules
}

// CollectFolderRules collects folder rules for all touched files.
// Deduplicates: same folder+rule pair only appears once.
func CollectFolderRules(root string, touchedFiles []string) []FolderRule {
	seen := make(map[string]bool)
	var allRules []FolderRule

	for _, file := range touchedFiles {
		fileRules := collectRulesForFile(root, file)
		for _, fr := range fileRules {
			key := fr.Folder + "::" + fr.Rule
			if !seen[key] {
				seen[key] = true
				allRules = append(allRules, fr)
			}
		}
	}

	return allRules
}
