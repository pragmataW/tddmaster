
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pragmataW/tddmaster/internal/output"
	statesync "github.com/pragmataW/tddmaster/internal/sync"
	"github.com/pragmataW/tddmaster/internal/state"
)

func newRuleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rule",
		Short: "Manage rules",
		RunE:  runRule,
	}
}

func runRule(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		printErr(fmt.Sprintf("Usage: %s rule <add \"rule text\" | list | promote \"decision\">", output.CmdPrefix()))
		return nil
	}

	switch args[0] {
	case "add":
		return ruleAdd(args[1:])
	case "list":
		return ruleList()
	case "promote":
		return ruleAdd(args[1:]) // promote = add as rule
	default:
		printErr(fmt.Sprintf("Usage: %s rule <add \"rule text\" | list | promote \"decision\">", output.CmdPrefix()))
		return nil
	}
}

func ruleAdd(args []string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}

	var phases []string
	var appliesTo []string
	var textParts []string

	for _, arg := range args {
		if strings.HasPrefix(arg, "--phases=") {
			raw := arg[len("--phases="):]
			for _, p := range strings.Split(raw, ",") {
				if t := strings.TrimSpace(p); t != "" {
					phases = append(phases, t)
				}
			}
		} else if strings.HasPrefix(arg, "--applies-to=") {
			raw := arg[len("--applies-to="):]
			for _, p := range strings.Split(raw, ",") {
				t := strings.Trim(strings.TrimSpace(p), `"'`)
				if t != "" {
					appliesTo = append(appliesTo, t)
				}
			}
		} else if !strings.HasPrefix(arg, "-") {
			textParts = append(textParts, arg)
		}
	}

	ruleText := strings.Join(textParts, " ")
	if strings.TrimSpace(ruleText) == "" {
		return fmt.Errorf("please provide a rule: %s", output.Cmd(`rule add "Use Deno Tests for all tests"`))
	}

	// Slugify for filename
	slug := strings.ToLower(ruleText)
	slug = strings.NewReplacer("[", "", "]", "", "(", "", ")", "", "\"", "", "'", "").Replace(slug)
	slugRe := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, slug)
	// Collapse multiple dashes
	for strings.Contains(slugRe, "--") {
		slugRe = strings.ReplaceAll(slugRe, "--", "-")
	}
	slugRe = strings.Trim(slugRe, "-")
	if len(slugRe) > 50 {
		slugRe = slugRe[:50]
	}

	// Build file content
	var content strings.Builder
	if len(phases) > 0 || len(appliesTo) > 0 {
		content.WriteString("---\n")
		if len(phases) > 0 {
			content.WriteString(fmt.Sprintf("phases: [%s]\n", strings.Join(phases, ", ")))
		}
		if len(appliesTo) > 0 {
			quotedTo := make([]string, len(appliesTo))
			for i, p := range appliesTo {
				quotedTo[i] = fmt.Sprintf("%q", p)
			}
			content.WriteString(fmt.Sprintf("applies_to: [%s]\n", strings.Join(quotedTo, ", ")))
		}
		content.WriteString("---\n")
	}
	content.WriteString(ruleText + "\n")

	rulesDir := filepath.Join(root, state.TddmasterDir, "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		return err
	}

	filePath := filepath.Join(rulesDir, slugRe+".md")
	if err := os.WriteFile(filePath, []byte(content.String()), 0o644); err != nil {
		return err
	}

	var scope []string
	if len(phases) > 0 {
		scope = append(scope, strings.Join(phases, ", "))
	}
	if len(appliesTo) > 0 {
		scope = append(scope, strings.Join(appliesTo, ", "))
	}
	scopeLabel := ""
	if len(scope) > 0 {
		scopeLabel = fmt.Sprintf(" [%s]", strings.Join(scope, "; "))
	}

	printErr(fmt.Sprintf("Rule added: %s%s", ruleText, scopeLabel))

	// Auto-sync
	config, _ := state.ReadManifest(root)
	if config != nil && len(config.Tools) > 0 {
		_, _ = statesync.SyncAll(root, config.Tools, config)
		printErr("  Tool files synced.")
	}

	return nil
}

func ruleList() error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}

	rules, _ := statesync.LoadScopedRules(root)

	printErr("Rules\n")
	if len(rules) == 0 {
		printErr(fmt.Sprintf("  No rules yet. Add one with: %s", output.Cmd(`rule add "..."`)))
		return nil
	}

	for i, r := range rules {
		var scope []string
		if len(r.Phases) > 0 {
			scope = append(scope, strings.Join(r.Phases, ", "))
		} else {
			scope = append(scope, "all phases")
		}
		if len(r.AppliesTo) > 0 {
			scope = append(scope, strings.Join(r.AppliesTo, ", "))
		} else {
			scope = append(scope, "all files")
		}
		printErr(fmt.Sprintf("  %d. %s [%s]", i+1, r.Text, strings.Join(scope, ", ")))
	}

	return nil
}
