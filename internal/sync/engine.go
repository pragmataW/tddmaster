
package sync

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pragmataW/tddmaster/internal/state"
)

// =============================================================================
// ScopedRule — a rule with optional phase/file scope
// =============================================================================

// ScopedRule is a rule loaded from .tddmaster/rules/*.md with optional frontmatter.
type ScopedRule struct {
	Text      string
	Phases    []string
	AppliesTo []string
}

// =============================================================================
// Rule loading
// =============================================================================

// LoadRules loads all rules from .tddmaster/rules/ as plain strings (backward compat).
func LoadRules(root string) ([]string, error) {
	scoped, err := LoadScopedRules(root)
	if err != nil {
		return nil, err
	}
	texts := make([]string, len(scoped))
	for i, r := range scoped {
		texts[i] = r.Text
	}
	return texts, nil
}

// LoadScopedRules loads all rules from .tddmaster/rules/ with scope metadata.
func LoadScopedRules(root string) ([]ScopedRule, error) {
	rulesDir := filepath.Join(root, state.TddmasterDir, "rules")
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		return []ScopedRule{}, nil // No rules directory yet
	}

	var rules []ScopedRule
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".md") && !strings.HasSuffix(name, ".txt") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(rulesDir, name))
		if err != nil {
			continue
		}
		meta, body := parseFrontmatter(string(data))
		body = strings.TrimSpace(body)
		if body == "" {
			continue
		}

		r := ScopedRule{Text: body}
		if phases, ok := meta["phases"]; ok {
			r.Phases = phases
		}
		if appliesTo, ok := meta["applies_to"]; ok {
			r.AppliesTo = appliesTo
		}
		rules = append(rules, r)
	}
	return rules, nil
}

// parseFrontmatter parses YAML-like frontmatter from a rule file.
// Returns a map of string-list values and the body after the frontmatter.
func parseFrontmatter(content string) (meta map[string][]string, body string) {
	meta = make(map[string][]string)
	content = strings.TrimSpace(content)

	if !strings.HasPrefix(content, "---") {
		return meta, content
	}

	rest := content[3:]
	endIdx := strings.Index(rest, "---")
	if endIdx == -1 {
		return meta, content
	}

	yamlBlock := strings.TrimSpace(rest[:endIdx])
	body = strings.TrimSpace(rest[endIdx+3:])

	for _, line := range strings.Split(yamlBlock, "\n") {
		colonIdx := strings.IndexByte(line, ':')
		if colonIdx == -1 {
			continue
		}
		key := strings.TrimSpace(line[:colonIdx])
		val := strings.TrimSpace(line[colonIdx+1:])

		// Parse array syntax: [a, b, c]
		if strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]") {
			inner := val[1 : len(val)-1]
			parts := strings.Split(inner, ",")
			var items []string
			for _, p := range parts {
				p = strings.TrimSpace(p)
				p = strings.Trim(p, `"'`)
				if p != "" {
					items = append(items, p)
				}
			}
			meta[key] = items
		} else {
			val = strings.Trim(val, `"'`)
			if val != "" {
				meta[key] = []string{val}
			}
		}
	}

	return meta, body
}

// =============================================================================
// Two-tier rule splitting
// =============================================================================

// SplitByTier splits scoped rules into tier1 (compile-time, no file scope) and tier2 count.
func SplitByTier(rules []ScopedRule, currentPhase state.Phase) (tier1 []string, tier2Count int) {
	phase := string(currentPhase)
	for _, r := range rules {
		// Phase filter
		if len(r.Phases) > 0 {
			matched := false
			for _, p := range r.Phases {
				if strings.EqualFold(p, phase) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		// Tier split: rules with appliesTo → tier2 (hook-time)
		if len(r.AppliesTo) > 0 {
			tier2Count++
		} else {
			tier1 = append(tier1, r.Text)
		}
	}
	return tier1, tier2Count
}

// FilterRules filters scoped rules by current phase and optional file patterns.
func FilterRules(rules []ScopedRule, currentPhase string, currentFiles []string) []string {
	var result []string
	for _, r := range rules {
		// Phase filter
		if len(r.Phases) > 0 {
			matched := false
			for _, p := range r.Phases {
				if strings.EqualFold(p, currentPhase) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		// File filter
		if len(r.AppliesTo) > 0 && len(currentFiles) > 0 {
			matched := false
			for _, pat := range r.AppliesTo {
				for _, f := range currentFiles {
					if matchFilePattern(f, pat) {
						matched = true
						break
					}
				}
				if matched {
					break
				}
			}
			if !matched {
				continue
			}
		}
		result = append(result, r.Text)
	}
	return result
}

// matchFilePattern matches a single file path against a glob pattern.
func matchFilePattern(filePath, pattern string) bool {
	if strings.HasPrefix(pattern, "*.") {
		ext := pattern[1:] // e.g. ".go"
		return strings.HasSuffix(filePath, ext)
	}
	return strings.Contains(filePath, strings.ReplaceAll(pattern, "*", ""))
}

// GetTier2RulesForFile returns tier2 rules that match a specific file path.
func GetTier2RulesForFile(rules []ScopedRule, currentPhase, filePath string) []string {
	var result []string
	for _, r := range rules {
		// Phase filter
		if len(r.Phases) > 0 {
			matched := false
			for _, p := range r.Phases {
				if strings.EqualFold(p, currentPhase) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		// Must have file scope
		if len(r.AppliesTo) == 0 {
			continue
		}
		// Match file
		for _, pat := range r.AppliesTo {
			if matchFilePattern(filePath, pat) {
				result = append(result, r.Text)
				break
			}
		}
	}
	return result
}

// =============================================================================
// Interaction hints
// =============================================================================

// defaultInteractionHints are the Claude Code defaults.
var defaultInteractionHints = InteractionHints{
	HasAskUserTool:        true,
	OptionPresentation:    "tool",
	HasSubAgentDelegation: true,
	SubAgentMethod:        "task",
}

// ResolveInteractionHints resolves the interaction hints for the primary active tool.
// Falls back to Claude Code defaults if no tools are configured or tool ID is unknown.
func ResolveInteractionHints(tools []state.CodingToolId) *InteractionHints {
	if len(tools) == 0 {
		result := defaultInteractionHints
		return &result
	}

	primaryID := tools[0]
	for _, a := range Adapters {
		if a.ID() == primaryID {
			caps := a.Capabilities()
			hints := caps.Interaction
			return &hints
		}
	}

	result := defaultInteractionHints
	return &result
}

// =============================================================================
// Adapter Registry
// =============================================================================

// Adapters is the registry of all known tool adapters.
var Adapters []ToolAdapter

// RegisterAdapter registers a ToolAdapter into the global registry.
func RegisterAdapter(a ToolAdapter) {
	Adapters = append(Adapters, a)
}

// =============================================================================
// SyncAll
// =============================================================================

// SyncAll syncs all coding tool configuration files for the given tools.
// Returns the list of tool IDs that were synced.
func SyncAll(root string, tools []state.CodingToolId, config *state.NosManifest) ([]state.CodingToolId, error) {
	rules, err := LoadRules(root)
	if err != nil {
		return nil, err
	}

	syncOptions := &SyncOptions{AllowGit: false}
	commandPrefix := "tddmaster"

	if config != nil {
		syncOptions.AllowGit = config.AllowGit
		if config.Command != "" {
			commandPrefix = config.Command
		}
	}

	// Load TDD manifest for TestRunner and TddMode settings.
	// Errors are non-fatal: if the manifest is absent or unparseable, adapters
	// receive a nil Manifest and fall back to their built-in defaults.
	var tddManifest *state.Manifest
	if m, merr := state.LoadManifest(root); merr == nil {
		tddManifest = &m
	}

	var synced []state.CodingToolId

	for _, toolID := range tools {
		var found ToolAdapter
		for _, a := range Adapters {
			if a.ID() == toolID {
				found = a
				break
			}
		}
		if found == nil {
			continue
		}

		ctx := SyncContext{Root: root, Rules: rules, CommandPrefix: commandPrefix, Manifest: tddManifest}

		if err := found.SyncRules(ctx, syncOptions); err != nil {
			return nil, err
		}

		caps := found.Capabilities()

		if caps.Hooks {
			if err := found.SyncHooks(ctx, syncOptions); err != nil {
				return nil, err
			}
		}

		if caps.Agents {
			if err := found.SyncAgents(ctx, syncOptions); err != nil {
				return nil, err
			}
		}

		if caps.Specs {
			specsDir := filepath.Join(root, state.TddmasterDir, "specs")
			entries, readErr := os.ReadDir(specsDir)
			if readErr == nil {
				for _, entry := range entries {
					if entry.IsDir() {
						specPath := filepath.Join(specsDir, entry.Name(), "spec.md")
						if err := found.SyncSpecs(ctx, specPath); err != nil {
							return nil, err
						}
					}
				}
			}
			// If specsDir doesn't exist, skip silently
		}

		if caps.Mcp {
			if err := found.SyncMcp(ctx); err != nil {
				return nil, err
			}
		}

		synced = append(synced, toolID)
	}

	// Preserve the "hooks" marker in the synced list for backward compatibility
	for _, t := range tools {
		if t == state.CodingToolClaudeCode {
			synced = append(synced, "hooks")
			break
		}
	}

	return synced, nil
}
