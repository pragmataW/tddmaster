package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	ctxpkg "github.com/pragmataW/tddmaster/internal/context"
	"github.com/pragmataW/tddmaster/internal/detect"
	"github.com/pragmataW/tddmaster/internal/output"
	"github.com/pragmataW/tddmaster/internal/state"
	statesync "github.com/pragmataW/tddmaster/internal/sync"
	_ "github.com/pragmataW/tddmaster/internal/sync/adapters"
)

func pickCodingTools(preselected []state.CodingToolId) []state.CodingToolId {
	allTools := []struct {
		Value state.CodingToolId
		Label string
	}{
		{Value: "claude-code", Label: "Claude Code"},
		{Value: "opencode", Label: "OpenCode"},
		{Value: "codex", Label: "Codex CLI"},
	}

	preselectedSet := make(map[state.CodingToolId]bool)
	for _, d := range preselected {
		preselectedSet[d] = true
	}

	var options []huh.Option[state.CodingToolId]
	for _, t := range allTools {
		opt := huh.NewOption(t.Label, t.Value)
		if preselectedSet[t.Value] {
			opt = opt.Selected(true)
		}
		options = append(options, opt)
	}

	var selected []state.CodingToolId
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[state.CodingToolId]().
				Title("Select coding tools (space to toggle, enter to confirm)").
				Options(options...).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		return preselected // on cancel, keep preselected
	}

	if len(selected) == 0 {
		return preselected
	}

	return selected
}

func pickConcerns(allConcerns []state.ConcernDefinition, preselected []string) []string {
	preselectedSet := make(map[string]bool, len(preselected))
	for _, id := range preselected {
		preselectedSet[id] = true
	}

	var options []huh.Option[string]
	for _, c := range allConcerns {
		desc := c.Description
		if len(desc) > 60 {
			desc = desc[:60]
		}
		opt := huh.NewOption(c.Name+" — "+desc, c.ID)
		if preselectedSet[c.ID] {
			opt = opt.Selected(true)
		}
		options = append(options, opt)
	}

	var selected []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("What kind of project is this? (space to toggle, enter to confirm)").
				Options(options...).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		return preselected // on cancel, keep preselected
	}

	return selected
}

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new tddmaster project",
		Long:  "Initialize and scaffold .tddmaster/ in the current project. Idempotent — safe to re-run.",
		RunE:  runInit,
	}
	cmd.Flags().String("concerns", "", "Comma-separated concern IDs to enable")
	cmd.Flags().String("tools", "", "Comma-separated tool IDs to enable")
	cmd.Flags().Bool("non-interactive", false, "Skip interactive prompts")
	cmd.Flags().Bool("skip-verify", false, "Skip verifier sub-agent (GREEN-only if TDD enabled)")
	cmd.Flags().Bool("tdd-enabled", true, "Enable TDD (test-first) workflow")
	return cmd
}

func resolveTddSettings(cmd *cobra.Command, existing *state.NosManifest) (tddMode, skipVerify bool) {
	// defaults
	tddMode = true
	skipVerify = false
	// existing manifest
	if existing != nil && existing.Tdd != nil {
		tddMode = existing.Tdd.TddMode
		skipVerify = existing.Tdd.SkipVerify
	}
	// flags override when explicitly set
	if cmd.Flags().Changed("tdd-enabled") {
		tddMode, _ = cmd.Flags().GetBool("tdd-enabled")
	}
	if cmd.Flags().Changed("skip-verify") {
		skipVerify, _ = cmd.Flags().GetBool("skip-verify")
	}
	return
}

func runInit(cmd *cobra.Command, args []string) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}

	// Parse flags
	concernsFlag, _ := cmd.Flags().GetString("concerns")
	toolsFlag, _ := cmd.Flags().GetString("tools")
	nonInteractive, _ := cmd.Flags().GetBool("non-interactive")

	var parsedConcerns []string
	if concernsFlag != "" {
		for _, c := range strings.Split(concernsFlag, ",") {
			if t := strings.TrimSpace(c); t != "" {
				parsedConcerns = append(parsedConcerns, t)
			}
		}
	}

	var parsedTools []state.CodingToolId
	if toolsFlag != "" {
		validTools := map[string]bool{
			"claude-code": true, "opencode": true, "codex": true,
		}
		for _, t := range strings.Split(toolsFlag, ",") {
			t = strings.TrimSpace(t)
			if validTools[t] {
				parsedTools = append(parsedTools, state.CodingToolId(t))
			}
		}
	}

	alreadyInitialized, _ := state.IsInitialized(root)

	fmt.Fprintln(os.Stderr, "tddmaster init")

	// Step 1: Scaffold directories
	if err := state.ScaffoldDir(root); err != nil {
		return fmt.Errorf("scaffold: %w", err)
	}
	fmt.Fprintln(os.Stderr, "Config: .tddmaster/")

	// Step 2: Detect project
	project := detect.DetectProject(root)
	fmt.Fprintf(os.Stderr, "Project scanned (%d languages, %d frameworks)\n",
		len(project.Languages), len(project.Frameworks))

	// Step 3: Detect coding tools
	detectedTools := detect.DetectCodingTools(root)
	fmt.Fprintf(os.Stderr, "%d coding tool(s) detected\n", len(detectedTools))

	// Read existing manifest once for the alreadyInitialized branch (tools, concerns, TDD settings).
	var existingManifest *state.NosManifest
	if alreadyInitialized {
		var rerr error
		existingManifest, rerr = state.ReadManifest(root)
		if rerr != nil {
			fmt.Fprintf(os.Stderr, "warning: manifest read failed: %v\n", rerr)
		}
	}

	var codingTools []state.CodingToolId
	if len(parsedTools) > 0 {
		codingTools = parsedTools
	} else if nonInteractive {
		if alreadyInitialized && existingManifest != nil && len(existingManifest.Tools) > 0 {
			codingTools = existingManifest.Tools
		} else {
			codingTools = detectedTools
		}
	} else {
		// Always show the picker so users can add/remove tools on every sync.
		// Preselect from manifest if initialized, otherwise from filesystem detection.
		preselected := detectedTools
		if alreadyInitialized && existingManifest != nil && len(existingManifest.Tools) > 0 {
			preselected = existingManifest.Tools
		}
		codingTools = pickCodingTools(preselected)
	}

	// Step 4: Load default concerns
	allConcerns := ctxpkg.LoadDefaultConcerns()

	var selectedConcernIds []string
	if len(parsedConcerns) > 0 {
		canonicalIDs := make(map[string]bool)
		for _, c := range allConcerns {
			canonicalIDs[c.ID] = true
		}
		for _, id := range parsedConcerns {
			if canonicalIDs[id] {
				selectedConcernIds = append(selectedConcernIds, id)
			}
		}
	} else if nonInteractive {
		if alreadyInitialized && existingManifest != nil {
			selectedConcernIds = existingManifest.Concerns
		}
	} else {
		// Always show the picker so users can adjust concerns on every sync.
		// Preselect from manifest if initialized.
		var preselected []string
		if alreadyInitialized && existingManifest != nil {
			preselected = existingManifest.Concerns
		}
		selectedConcernIds = pickConcerns(allConcerns, preselected)
	}

	// Step 5: Write selected concerns to disk
	selectedConcerns := make([]state.ConcernDefinition, 0)
	for _, c := range allConcerns {
		for _, id := range selectedConcernIds {
			if c.ID == id {
				selectedConcerns = append(selectedConcerns, c)
				break
			}
		}
	}
	for _, concern := range selectedConcerns {
		if err := state.WriteConcern(root, concern); err != nil {
			return fmt.Errorf("write concern %s: %w", concern.ID, err)
		}
	}

	// Step 6: Ask TDD mode preference
	tddMode, skipVerify := resolveTddSettings(cmd, existingManifest)

	if !nonInteractive {
		tddEnabled := tddMode
		skipVerifyVal := skipVerify
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Enable TDD (test-first) workflow?").
					Description("When enabled, test tasks run before implementation tasks.").
					Value(&tddEnabled),
				huh.NewConfirm().
					Title("Skip verifier sub-agent?").
					Description("When enabled, verification step is skipped (GREEN-only mode).").
					Value(&skipVerifyVal),
			),
		)
		if err := form.Run(); err == nil {
			tddMode = tddEnabled
			skipVerify = skipVerifyVal
		}
	}

	// Step 7: Write manifest
	projectTraits := state.ProjectTraits{
		Languages:  project.Languages,
		Frameworks: project.Frameworks,
		CI:         project.CI,
	}

	config := state.CreateInitialManifest(selectedConcernIds, codingTools, projectTraits)
	config.Command = output.CmdPrefix()
	tdd := &state.Manifest{TddMode: tddMode, SkipVerify: skipVerify}
	if existingManifest != nil && existingManifest.Tdd != nil {
		prev := existingManifest.Tdd
		tdd.TestRunner = prev.TestRunner
		tdd.MaxVerificationRetries = prev.MaxVerificationRetries
		tdd.MaxRefactorRounds = prev.MaxRefactorRounds
		tdd.InjectProjectConventions = prev.InjectProjectConventions
	}
	config.Tdd = tdd

	if err := state.WriteManifest(root, config); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	// Step 7: Write initial state if not already initialized
	if !alreadyInitialized {
		initialState := state.CreateInitialState()
		if err := state.WriteState(root, initialState); err != nil {
			return fmt.Errorf("write state: %w", err)
		}
	}

	// Step 8: Sync tool files
	if len(codingTools) > 0 {
		synced, syncErr := statesync.SyncAll(root, codingTools, &config)
		if syncErr != nil {
			return fmt.Errorf("sync: %w", syncErr)
		}
		fmt.Fprintf(os.Stderr, "Synced %d tool(s)\n", len(synced))
	} else {
		fmt.Fprintln(os.Stderr, "warning: no coding tools — skipping adapter sync (no files written)")
	}

	fmt.Fprintf(os.Stderr, "Done. %d tool(s), %d concern(s).\n",
		len(codingTools), len(selectedConcernIds))
	fmt.Fprintf(os.Stderr, "Start a spec with: %s\n", output.Cmd(`spec new "..."`))

	return nil
}
