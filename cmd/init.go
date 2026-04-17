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

func pickCodingTools(detected []state.CodingToolId) []state.CodingToolId {
	allTools := []struct {
		Value state.CodingToolId
		Label string
	}{
		{Value: "claude-code", Label: "Claude Code"},
		{Value: "opencode", Label: "OpenCode"},
		{Value: "codex", Label: "Codex CLI"},
	}

	detectedSet := make(map[state.CodingToolId]bool)
	for _, d := range detected {
		detectedSet[d] = true
	}

	var options []huh.Option[state.CodingToolId]
	for _, t := range allTools {
		opt := huh.NewOption(t.Label, t.Value)
		if detectedSet[t.Value] {
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
		return detected // on cancel, use detected
	}

	if len(selected) == 0 {
		return detected
	}

	return selected
}

func pickConcerns(allConcerns []state.ConcernDefinition) []string {
	var options []huh.Option[string]
	for _, c := range allConcerns {
		desc := c.Description
		if len(desc) > 60 {
			desc = desc[:60]
		}
		options = append(options, huh.NewOption(c.Name+" — "+desc, c.ID))
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
		return nil // on cancel, no concerns
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
	return cmd
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

	var codingTools []state.CodingToolId
	if len(parsedTools) > 0 {
		codingTools = parsedTools
	} else if alreadyInitialized {
		existing, _ := state.ReadManifest(root)
		if existing != nil {
			codingTools = existing.Tools
		} else {
			codingTools = detectedTools
		}
	} else if nonInteractive {
		codingTools = detectedTools
	} else {
		codingTools = pickCodingTools(detectedTools)
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
	} else if alreadyInitialized {
		existing, _ := state.ReadManifest(root)
		if existing != nil {
			selectedConcernIds = existing.Concerns
		}
	} else if !nonInteractive {
		selectedConcernIds = pickConcerns(allConcerns)
	}
	// else: no concerns selected (non-interactive without explicit concerns)

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
	tddMode := true
	if !nonInteractive {
		tddEnabled := true
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Enable TDD (test-first) workflow?").
					Description("When enabled, test tasks run before implementation tasks.").
					Value(&tddEnabled),
			),
		)
		if err := form.Run(); err == nil {
			tddMode = tddEnabled
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
	config.Tdd = &state.Manifest{TddMode: tddMode}

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
		synced, _ := statesync.SyncAll(root, codingTools, &config)
		fmt.Fprintf(os.Stderr, "Synced %d tool(s)\n", len(synced))
	}

	fmt.Fprintf(os.Stderr, "Done. %d tool(s), %d concern(s).\n",
		len(codingTools), len(selectedConcernIds))
	fmt.Fprintf(os.Stderr, "Start a spec with: %s\n", output.Cmd(`spec new "..."`))

	return nil
}
