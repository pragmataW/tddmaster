
package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pragmataW/tddmaster/internal/output"
	"github.com/pragmataW/tddmaster/internal/state"
)

// reservedSpecNames cannot be used as spec names.
var reservedSpecNames = map[string]bool{
	"new": true, "list": true, "help": true, "next": true, "approve": true,
	"done": true, "block": true, "reset": true, "cancel": true, "wontfix": true,
	"reopen": true, "revisit": true, "split": true, "ac": true, "task": true,
	"note": true, "review": true, "delegate": true, "followup": true,
}

// slugStopWords are stripped when building a slug from a description.
var slugStopWords = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "or": true, "but": true,
	"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
	"with": true, "by": true, "from": true, "is": true, "it": true, "its": true,
	"this": true, "that": true, "as": true, "be": true, "are": true, "was": true,
	"were": true, "been": true, "being": true, "have": true, "has": true,
	"had": true, "do": true, "does": true, "did": true, "will": true,
	"would": true, "could": true, "should": true, "may": true, "might": true,
	"shall": true, "can": true, "i": true, "we": true, "you": true, "they": true,
	"our": true, "my": true, "so": true, "if": true, "not": true, "no": true,
	"all": true,
}

var pathRe1 = regexp.MustCompile(`(?:~|\.{1,2})?/(?:[\w@.\-]+/)*[\w@.\-]+`)
var pathRe2 = regexp.MustCompile(`(?:[\w@.\-]+/){2,}[\w@.\-]+`)
var nonAlphaNum = regexp.MustCompile(`[^a-z0-9\s\-]`)
var spacesRe = regexp.MustCompile(`\s+`)
var leadTrailDash = regexp.MustCompile(`^-+|-+$`)

// slugFromDescription generates a URL-safe slug from a description string.
func slugFromDescription(desc string) string {
	cleaned := pathRe1.ReplaceAllString(desc, " ")
	cleaned = pathRe2.ReplaceAllString(cleaned, " ")
	cleaned = spacesRe.ReplaceAllString(strings.TrimSpace(cleaned), " ")

	lower := nonAlphaNum.ReplaceAllString(strings.ToLower(cleaned), "")
	words := spacesRe.Split(lower, -1)

	var significant []string
	for _, w := range words {
		if w != "" && !slugStopWords[w] {
			significant = append(significant, w)
			if len(significant) >= 6 {
				break
			}
		}
	}

	slug := strings.Join(significant, "-")
	if len(slug) > 50 {
		truncated := slug[:50]
		if idx := strings.LastIndex(truncated, "-"); idx > 0 {
			truncated = truncated[:idx]
		}
		slug = truncated
	}
	slug = leadTrailDash.ReplaceAllString(slug, "")
	if slug == "" {
		slug = "spec"
	}
	return slug
}

// looksLikeDescription returns true if the value appears to be a description (not a slug).
func looksLikeDescription(value string) bool {
	return strings.Contains(value, " ") || len(value) > 50
}

func newSpecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "spec",
		Short:              "Manage specs",
		Long:               "Manage specs: create, list, and run spec subcommands.",
		RunE:               runSpec,
		DisableFlagParsing: true,
	}
	return cmd
}

func runSpec(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}

	arg1 := args[0]

	switch arg1 {
	case "new":
		return specNew(args[1:])
	case "list":
		return specList(args[1:])
	case "help", "--help", "-h":
		return cmd.Help()
	}

	// Otherwise: args[0] is spec name, args[1] is subcommand
	specName := arg1
	if len(args) < 2 {
		// Show spec status
		root, err := resolveRoot()
		if err != nil {
			return err
		}
		st, err := state.ResolveState(root, &specName)
		if err != nil {
			return writeJSON(map[string]string{"error": err.Error()})
		}
		return writeJSON(map[string]interface{}{
			"spec":  specName,
			"phase": string(st.Phase),
		})
	}

	subcommand := args[1]
	specArgs := append([]string{"--spec=" + specName}, args[2:]...)

	switch subcommand {
	case "next":
		return runNextWithArgs(specArgs)
	case "approve":
		return runApproveWithArgs(specArgs)
	case "done":
		return runDoneWithArgs(specArgs)
	case "block":
		return runBlockWithArgs(specArgs)
	case "reset":
		return runResetWithArgs(specArgs)
	case "cancel":
		return runCancelWithArgs(specArgs)
	case "wontfix":
		return runWontfixWithArgs(specArgs)
	case "reopen":
		return runReopenWithArgs(specArgs)
	case "review":
		return runReviewWithArgs(specArgs)
	case "delegate":
		return runDelegateWithArgs(specArgs)
	case "followup":
		return runFollowupWithArgs(specArgs)
	case "learn":
		return runLearnWithArgs(specArgs)
	case "ac":
		return specAC(specName, args[2:])
	case "task":
		return specTask(specName, args[2:])
	case "note":
		return specNote(specName, args[2:])
	case "revisit":
		return specRevisit(specName, args[2:])
	default:
		return fmt.Errorf("unknown spec subcommand: %s", subcommand)
	}
}

// specNew creates a new spec.
func specNew(args []string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}

	initialized, _ := state.IsInitialized(root)
	if !initialized {
		return fmt.Errorf("tddmaster is not initialized. Run: %s", output.Cmd("init"))
	}

	// Parse positional args: spec new [name] "description" [--from-plan=path]
	var specName string
	var descWords []string
	var planPath string
	var nameConsumed bool

	for _, arg := range args {
		if strings.HasPrefix(arg, "--name=") {
			specName = arg[len("--name="):]
			nameConsumed = true
		} else if strings.HasPrefix(arg, "--from-plan=") {
			planPath = arg[len("--from-plan="):]
		} else if !strings.HasPrefix(arg, "-") {
			descWords = append(descWords, arg)
		}
	}

	if !nameConsumed && len(descWords) > 0 {
		first := descWords[0]
		if looksLikeDescription(first) {
			// First arg is description — auto-generate slug
		} else {
			// First arg is slug name
			specName = first
			descWords = descWords[1:]
		}
	}

	description := strings.Join(descWords, " ")

	// Auto-generate slug if no name
	if specName == "" && description != "" {
		base := slugFromDescription(description)

		// Deduplicate if conflicts exist
		candidate := base
		suffix := 2
		for {
			if reservedSpecNames[candidate] {
				candidate = fmt.Sprintf("%s-%d", base, suffix)
				suffix++
				continue
			}
			dirPath := fmt.Sprintf("%s/%s/specs/%s", root, state.TddmasterDir, candidate)
			if _, err := os.Stat(dirPath); os.IsNotExist(err) {
				break
			}
			candidate = fmt.Sprintf("%s-%d", base, suffix)
			suffix++
		}
		specName = candidate
	}

	if specName == "" {
		return fmt.Errorf("description is required.\nExample: %s spec new \"Add photo upload support\"", output.CmdPrefix())
	}

	if reservedSpecNames[specName] {
		return fmt.Errorf("%q is a reserved name. Choose a different spec name", specName)
	}

	// Validate name
	nameRe := regexp.MustCompile(`^[a-z0-9][a-z0-9\-]*[a-z0-9]$`)
	if len(specName) > 50 || (len(specName) > 1 && !nameRe.MatchString(specName)) ||
		(len(specName) == 1 && !regexp.MustCompile(`^[a-z0-9]$`).MatchString(specName)) {
		return fmt.Errorf("invalid spec name: %q\nMust be lowercase, hyphens, numbers only. Max 50 chars", specName)
	}

	if description == "" {
		return fmt.Errorf("please provide a description:\n  %s spec new \"Add photo upload support\"", output.CmdPrefix())
	}

	// Check plan file
	if planPath != "" {
		info, err := os.Stat(planPath)
		if err != nil {
			return fmt.Errorf("plan file not found: %s", planPath)
		}
		if info.Size() > 50*1024 {
			return fmt.Errorf("plan file too large. Maximum 50KB")
		}
	}

	branch := "spec/" + specName

	// Check if spec already exists
	specDir := fmt.Sprintf("%s/%s/specs/%s", root, state.TddmasterDir, specName)
	if _, err := os.Stat(specDir); err == nil {
		return fmt.Errorf("spec %q already exists. Use a different name or run `%s reset --spec=%s` first",
			specName, output.CmdPrefix(), specName)
	}

	// Create fresh state
	freshState := state.CreateInitialState()
	descPtr := &description
	newState, err := state.StartSpec(freshState, specName, branch, descPtr)
	if err != nil {
		return err
	}

	// Record transition
	user, _ := state.ResolveUser(root)
	userInfo := &state.UserInfo{Name: user.Name, Email: user.Email}
	newState = state.RecordTransition(newState, state.PhaseIdle, state.PhaseDiscovery, userInfo, nil)

	// Inject plan path if provided
	if planPath != "" {
		newState.Discovery.PlanPath = &planPath
	}

	// Create spec directory and save state
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		return fmt.Errorf("create spec dir: %w", err)
	}

	if err := state.WriteSpecState(root, specName, newState); err != nil {
		return fmt.Errorf("write spec state: %w", err)
	}

	// Also update global state
	if err := state.WriteState(root, newState); err != nil {
		return fmt.Errorf("write state: %w", err)
	}

	printErr(fmt.Sprintf("Spec started: %s", specName))
	printErr(fmt.Sprintf("  Directory: %s/specs/%s", state.TddmasterDir, specName))
	printErr(fmt.Sprintf("  Branch:    %s", branch))
	printErr("  Phase:     DISCOVERY")

	fmt.Fprintf(os.Stdout, "Run %s to begin discovery questions.\n",
		output.Cmd(fmt.Sprintf("next --spec=%s", specName)))

	return nil
}

// specList lists all existing specs.
func specList(args []string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}

	specStates, _ := state.ListSpecStates(root)

	type specEntry struct {
		Name      string `json:"name"`
		Phase     string `json:"phase"`
		Iteration int    `json:"iteration"`
	}

	seen := make(map[string]bool)
	var allSpecs []specEntry

	for _, ss := range specStates {
		seen[ss.Name] = true
		allSpecs = append(allSpecs, specEntry{
			Name:      ss.Name,
			Phase:     string(ss.State.Phase),
			Iteration: ss.State.Execution.Iteration,
		})
	}

	// Also pick up spec directories without state files
	specsDir := fmt.Sprintf("%s/%s/specs", root, state.TddmasterDir)
	entries, _ := os.ReadDir(specsDir)
	for _, e := range entries {
		if e.IsDir() && !seen[e.Name()] {
			allSpecs = append(allSpecs, specEntry{
				Name:      e.Name(),
				Phase:     "IDLE",
				Iteration: 0,
			})
		}
	}

	return writeJSON(allSpecs)
}

// specAC adds an acceptance criterion to a spec.
func specAC(specName string, args []string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return fmt.Errorf("usage: tddmaster spec %s ac <add|list> [text]", specName)
	}

	sub := args[0]
	switch sub {
	case "add":
		acText := strings.Join(args[1:], " ")
		if acText == "" {
			return fmt.Errorf("please provide AC text")
		}

		st, err := state.ResolveState(root, &specName)
		if err != nil {
			return err
		}

		user, _ := state.ResolveUser(root)
		userInfo := &state.UserInfo{Name: user.Name, Email: user.Email}
		st = state.AddCustomAC(st, acText, userInfo)

		if err := state.WriteStateAndSpec(root, st); err != nil {
			return err
		}
		printErr(fmt.Sprintf("AC added to spec %s", specName))
		return nil

	case "list":
		st, err := state.ResolveState(root, &specName)
		if err != nil {
			return err
		}
		return writeJSON(st.CustomACs)

	default:
		return fmt.Errorf("unknown ac subcommand: %s (use add or list)", sub)
	}
}

// specTask manages tasks for a spec.
func specTask(specName string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: tddmaster spec %s task <add|list> [text]", specName)
	}
	// Simplified: just output a placeholder
	printErr(fmt.Sprintf("spec task %s: %s", specName, strings.Join(args, " ")))
	return nil
}

// specNote adds a note to a spec.
func specNote(specName string, args []string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return fmt.Errorf("usage: tddmaster spec %s note <add|list> [text]", specName)
	}

	sub := args[0]
	switch sub {
	case "add":
		noteText := strings.Join(args[1:], " ")
		if noteText == "" {
			return fmt.Errorf("please provide note text")
		}

		st, err := state.ResolveState(root, &specName)
		if err != nil {
			return err
		}

		user, _ := state.ResolveUser(root)
		userInfo := &state.UserInfo{Name: user.Name, Email: user.Email}
		st = state.AddSpecNote(st, noteText, userInfo)

		if err := state.WriteStateAndSpec(root, st); err != nil {
			return err
		}
		printErr(fmt.Sprintf("Note added to spec %s", specName))
		return nil

	case "list":
		st, err := state.ResolveState(root, &specName)
		if err != nil {
			return err
		}
		return writeJSON(st.SpecNotes)

	default:
		return fmt.Errorf("unknown note subcommand: %s (use add or list)", sub)
	}
}

// specRevisit transitions from EXECUTING/BLOCKED back to DISCOVERY.
func specRevisit(specName string, args []string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}

	reason := strings.Join(args, " ")
	if reason == "" {
		reason = "manual revisit"
	}

	st, err := state.ResolveState(root, &specName)
	if err != nil {
		return err
	}

	newState, err := state.RevisitSpec(st, reason)
	if err != nil {
		return err
	}

	user, _ := state.ResolveUser(root)
	userInfo := &state.UserInfo{Name: user.Name, Email: user.Email}
	reasonStr := reason
	newState = state.RecordTransition(newState, st.Phase, state.PhaseDiscovery, userInfo, &reasonStr)

	if err := state.WriteStateAndSpec(root, newState); err != nil {
		return err
	}

	printErr(fmt.Sprintf("Spec %s revisited. Back to DISCOVERY.", specName))
	return nil
}

