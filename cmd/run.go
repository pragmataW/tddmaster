package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	ctxpkg "github.com/pragmataW/tddmaster/internal/context"
	"github.com/pragmataW/tddmaster/internal/output"
	"github.com/pragmataW/tddmaster/internal/runner"
	"github.com/pragmataW/tddmaster/internal/spec"
	"github.com/pragmataW/tddmaster/internal/state"
)

// runnerSelect is the package-level seam that tests can swap via swapRunnerSelect.
var runnerSelect = runner.Select

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run an agent task",
		Long:  "Autonomous execution loop (Ralph loop). Spawns claude CLI each iteration.",
		RunE:  runRun,
	}
	cmd.Flags().Bool("unattended", false, "Log blocks and continue without pausing")
	cmd.Flags().Int("max-turns", 10, "Max turns per agent process")
	cmd.Flags().Int("max-iterations", 50, "Max loop iterations")
	cmd.Flags().String("spec", "", "Spec name")
	cmd.Flags().String("tool", "", "Coding tool to use (claude-code|codex|opencode). Empty = auto-select from manifest.")
	return cmd
}

func runRun(cmd *cobra.Command, args []string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}

	unattended, _ := cmd.Flags().GetBool("unattended")
	maxTurns, _ := cmd.Flags().GetInt("max-turns")
	maxIterations, _ := cmd.Flags().GetInt("max-iterations")
	specFlag, _ := cmd.Flags().GetString("spec")
	toolFlag, _ := cmd.Flags().GetString("tool")

	// Also parse from args
	for _, arg := range args {
		if strings.HasPrefix(arg, "--spec=") && specFlag == "" {
			specFlag = arg[len("--spec="):]
		}
	}

	initialized, _ := state.IsInitialized(root)
	if !initialized {
		return fmt.Errorf("tddmaster not initialized. Run: %s", output.Cmd("init"))
	}

	var specPtr *string
	if specFlag != "" {
		specPtr = &specFlag
	}

	initialState, err := state.ResolveState(root, specPtr)
	if err != nil {
		return err
	}

	config, _ := state.ReadManifest(root)
	if config == nil {
		return fmt.Errorf("config not found")
	}

	if initialState.Phase != state.PhaseExecuting && initialState.Phase != state.PhaseSpecApproved && initialState.Phase != state.PhaseBlocked {
		return fmt.Errorf("cannot run from phase: %s. Must be in SPEC_APPROVED or EXECUTING to start", initialState.Phase)
	}

	if initialState.Phase == state.PhaseSpecApproved {
		if specApprovedTDDSelectionPending(initialState, config) {
			return fmt.Errorf("spec is approved but TDD task selection is still required; use %s, %s, or %s before running",
				output.Cmd(`next --answer="tdd-all"`),
				output.Cmd(`next --answer="tdd-none"`),
				output.Cmd(`next --answer='{"tddTasks":["task-1"]}'`),
			)
		}

		printErr("Starting execution from approved spec...")
		newState, err := startExecutionFromApproved(initialState, config)
		if err != nil {
			return err
		}
		if err := state.WriteStateAndSpec(root, newState); err != nil {
			return err
		}
	}

	// Set up a cancelable context for this run; install SIGINT/SIGTERM handler.
	var ctx context.Context
	var cancel context.CancelFunc
	if cmd.Context() != nil {
		ctx, cancel = context.WithCancel(cmd.Context())
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	go func() {
		select {
		case <-sigCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	// Select runner once before the loop.
	selectedRunner, err := runnerSelect(config, toolFlag)
	if err != nil {
		if errors.Is(err, runner.ErrRunnerNotFound) {
			// List registered runners would go here; for now surface the error with context.
			return fmt.Errorf("runner not found — check registered runners or install the tool: %w", err)
		}
		return fmt.Errorf("failed to select runner: %w", err)
	}

	printErr(fmt.Sprintf("%s run", output.CmdPrefix()))
	mode := "interactive"
	if unattended {
		mode = "unattended"
	}
	printErr(fmt.Sprintf("Mode: %s, max-turns: %d, max-iterations: %d", mode, maxTurns, maxIterations))
	printErr("")

	loopIteration := 0
	exitCode := 0

	for loopIteration < maxIterations {
		loopIteration++

		st, err := state.ReadState(root)
		if err != nil {
			return err
		}

		if st.Phase == state.PhaseCompleted {
			printErr("")
			printErr("Spec completed!")
			printErr(fmt.Sprintf("  Iterations: %d", st.Execution.Iteration))
			printErr(fmt.Sprintf("  Decisions:  %d", len(st.Decisions)))
			// Mark that we did NOT exhaust iterations — we completed cleanly.
			loopIteration = 0
			break
		}

		if st.Phase == state.PhaseBlocked {
			reason := "Unknown"
			if st.Execution.LastProgress != nil {
				reason = *st.Execution.LastProgress
			}
			printErr("")
			printErr(fmt.Sprintf("Execution blocked: %s", reason))

			if unattended {
				_ = logBlockedFile(root, reason, loopIteration)
				printErr("Logged to .tddmaster/.state/blocked.log. Resolve and re-run.")
				exitCode = 1
				// Mark that we did NOT exhaust iterations.
				loopIteration = 0
				break
			}

			// Interactive: prompt for resolution
			printErr("Enter resolution (or leave empty to stop):")
			var resolution string
			_, _ = fmt.Scanln(&resolution)
			if strings.TrimSpace(resolution) == "" {
				printErr("Stopped by user.")
				loopIteration = 0
				break
			}

			unblocked, err := state.Transition(st, state.PhaseExecuting)
			if err != nil {
				return err
			}
			progress := "Resolved: " + resolution
			unblocked.Execution.LastProgress = &progress
			if err := state.WriteState(root, unblocked); err != nil {
				return err
			}
			continue
		}

		if st.Phase != state.PhaseExecuting {
			printErr(fmt.Sprintf("Unexpected phase: %s. Stopping.", st.Phase))
			exitCode = 1
			loopIteration = 0
			break
		}

		// Build agent prompt
		allConcerns, _ := state.ListConcerns(root)
		activeConcerns := filterConcerns(allConcerns, config.Concerns)
		tier1, hints, tier2Count, _ := loadRulesAndHints(root, st, config)
		parsedSpec, err := loadExecutionSpec(root, st)
		if err != nil {
			return err
		}

		compiled := ctxpkg.Compile(st, activeConcerns, tier1, config, parsedSpec, nil, nil, hints, nil, tier2Count)
		prompt, err := buildAgentPrompt(compiled)
		if err != nil {
			if st.Spec != nil {
				return fmt.Errorf("cannot build execution prompt for spec %q: %w", *st.Spec, err)
			}
			return fmt.Errorf("cannot build execution prompt: %w", err)
		}

		// Log iteration
		debtLen := 0
		if st.Execution.Debt != nil {
			debtLen = len(st.Execution.Debt.Items)
		}
		printErr(fmt.Sprintf("── Iteration %d (execution: %d, debt: %d)", loopIteration, st.Execution.Iteration, debtLen))

		if st.Execution.LastProgress != nil {
			printErr(fmt.Sprintf("  Last: %s", *st.Execution.LastProgress))
		}

		// Invoke runner
		printErr("  Invoking runner...")
		result, invokeErr := selectedRunner.Invoke(ctx, runner.RunRequest{
			Prompt:       prompt,
			MaxTurns:     maxTurns,
			OutputFormat: "json",
			Stdout:       os.Stdout,
			Stderr:       os.Stderr,
		})

		if invokeErr != nil {
			if errors.Is(invokeErr, runner.ErrBinaryNotFound) {
				return fmt.Errorf("runner binary not found — install the CLI tool: %w", invokeErr)
			}
			if errors.Is(invokeErr, runner.ErrContextCanceled) || errors.Is(invokeErr, context.Canceled) {
				return fmt.Errorf("run canceled: %w", invokeErr)
			}
			// Non-fatal: log and continue; tddmaster state file has the real truth.
			printErr(fmt.Sprintf("  Runner error (non-fatal): %v", invokeErr))
		} else if result != nil && result.ExitCode != 0 {
			// Non-zero exit is non-fatal; continue loop.
			printErr(fmt.Sprintf("  Runner exited with code %d (continuing)", result.ExitCode))
		}

		printErr("  Agent exited. Stop hook captured state.")
	}

	if loopIteration >= maxIterations {
		printErr("")
		printErr(fmt.Sprintf("Max iterations (%d) reached. Stopping.", maxIterations))
		exitCode = 2
	}

	if exitCode != 0 {
		return fmt.Errorf("run exited with code %d", exitCode)
	}
	return nil
}

// buildAgentPrompt constructs a prompt string from compiled output.
func buildAgentPrompt(compiled ctxpkg.NextOutput) (string, error) {
	if compiled.ExecutionData == nil {
		return "", fmt.Errorf("executionData missing from compiled output")
	}
	exec := compiled.ExecutionData

	payload, err := json.MarshalIndent(exec, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal executionData: %w", err)
	}
	reportShape, err := buildExpectedReportShape(exec)
	if err != nil {
		return "", fmt.Errorf("marshal expected report shape: %w", err)
	}

	var lines []string

	lines = append(lines, "You are executing the active tddmaster iteration.")
	lines = append(lines, "Read the ordered execution summary first. The raw `executionData` payload later in this prompt is the source of truth.")

	lines = append(lines, "", "Current phase:")
	lines = append(lines, fmt.Sprintf("- %s", compiled.Phase))

	lines = append(lines, "", "Current task:")
	lines = append(lines, formatCurrentTaskSummary(exec))

	lines = append(lines, "", "Files touched or suggested:")
	for _, file := range collectSuggestedFiles(exec) {
		lines = append(lines, "- "+file)
	}

	lines = append(lines, "", "Edge cases:")
	if len(exec.EdgeCases) == 0 {
		lines = append(lines, "- None listed for this iteration.")
	} else {
		for _, edgeCase := range exec.EdgeCases {
			lines = append(lines, "- "+edgeCase)
		}
	}

	if exec.TDDPhase != nil && strings.TrimSpace(*exec.TDDPhase) != "" {
		lines = append(lines, "", "TDD phase:")
		lines = append(lines, "- "+*exec.TDDPhase)
	}

	if instructions := collectVerifierInstructions(exec); len(instructions) > 0 {
		lines = append(lines, "", "Verifier / refactor instructions:")
		for _, instruction := range instructions {
			lines = append(lines, "- "+instruction)
		}
	}

	lines = append(lines, "", "Exact report JSON expected back:")
	lines = append(lines, "```json")
	lines = append(lines, reportShape)
	lines = append(lines, "```")

	if compiled.Behavioral.Tone != "" || compiled.Behavioral.Urgency != nil || len(compiled.Behavioral.Rules) > 0 || len(compiled.Behavioral.OutOfScope) > 0 {
		lines = append(lines, "", "Behavioral guidance:")
		if compiled.Behavioral.Tone != "" {
			lines = append(lines, "- Tone: "+compiled.Behavioral.Tone)
		}
		if compiled.Behavioral.Urgency != nil {
			lines = append(lines, "- Urgency: "+*compiled.Behavioral.Urgency)
		}
		for _, r := range compiled.Behavioral.Rules {
			if strings.ContainsRune(r, '\n') {
				lines = append(lines, r)
				continue
			}
			lines = append(lines, "- "+r)
		}
		if len(compiled.Behavioral.OutOfScope) > 0 {
			lines = append(lines, "Out of scope:")
			for _, item := range compiled.Behavioral.OutOfScope {
				lines = append(lines, "- "+item)
			}
		}
	}

	lines = append(lines, "", "Execution contract (`executionData`):")
	lines = append(lines, "```json")
	lines = append(lines, string(payload))
	lines = append(lines, "```")

	if compiled.Meta.Spec != nil || compiled.Meta.LastProgress != nil || compiled.Meta.ResumeHint != "" {
		lines = append(lines, "", "Resume context:")
		lines = append(lines, fmt.Sprintf("- Phase: %s", compiled.Phase))
		if compiled.Meta.Spec != nil {
			lines = append(lines, fmt.Sprintf("- Spec: %s", *compiled.Meta.Spec))
		}
		lines = append(lines, fmt.Sprintf("- Iteration: %d", compiled.Meta.Iteration))
		if compiled.Meta.LastProgress != nil {
			lines = append(lines, fmt.Sprintf("- Last progress: %s", *compiled.Meta.LastProgress))
		}
		if compiled.Meta.ResumeHint != "" {
			lines = append(lines, fmt.Sprintf("- Resume hint: %s", compiled.Meta.ResumeHint))
		}
	}

	lines = append(lines, "", "Transition commands:")
	lines = append(lines, fmt.Sprintf("- On complete: `%s`", exec.Transition.OnComplete))
	lines = append(lines, fmt.Sprintf("- On blocked: `%s`", exec.Transition.OnBlocked))
	if exec.StatusReportRequired != nil && *exec.StatusReportRequired {
		lines = append(lines, "- Submit a structured status report that matches `executionData.statusReport.reportFormat`.")
	}

	return strings.Join(lines, "\n"), nil
}

func formatCurrentTaskSummary(exec *ctxpkg.ExecutionOutput) string {
	if exec.Task != nil {
		return fmt.Sprintf("- %s: %s", exec.Task.ID, exec.Task.Title)
	}
	if len(exec.BatchTasks) > 0 {
		return fmt.Sprintf("- Batch status review for %s", strings.Join(exec.BatchTasks, ", "))
	}
	if exec.StatusReportRequired != nil && *exec.StatusReportRequired {
		return "- Acceptance/status review for the current iteration."
	}
	return "- No explicit task block provided; follow executionData."
}

func collectSuggestedFiles(exec *ctxpkg.ExecutionOutput) []string {
	seen := make(map[string]bool)
	files := make([]string, 0)
	appendFile := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" || seen[path] {
			return
		}
		seen[path] = true
		files = append(files, path)
	}

	if exec.Task != nil {
		for _, file := range exec.Task.Files {
			appendFile(file)
		}
	}
	if exec.RefactorInstructions != nil {
		for _, note := range exec.RefactorInstructions.Notes {
			appendFile(note.File)
		}
	}
	if len(files) == 0 {
		return []string{"No task-scoped files listed in this iteration."}
	}
	return files
}

func collectVerifierInstructions(exec *ctxpkg.ExecutionOutput) []string {
	items := make([]string, 0, 2)
	if exec.TDDVerificationContext != nil && strings.TrimSpace(exec.TDDVerificationContext.Instruction) != "" {
		items = append(items, "Verifier expectation: "+exec.TDDVerificationContext.Instruction)
	}
	if exec.RefactorInstructions != nil {
		round := fmt.Sprintf("Refactor round %d", exec.RefactorInstructions.Round)
		if exec.RefactorInstructions.MaxRounds > 0 {
			round = fmt.Sprintf("%s of %d", round, exec.RefactorInstructions.MaxRounds)
		}
		items = append(items, round+". "+exec.RefactorInstructions.Instruction)
		for _, note := range exec.RefactorInstructions.Notes {
			noteSummary := note.Suggestion
			if strings.TrimSpace(note.Rationale) != "" {
				noteSummary = fmt.Sprintf("%s (%s)", note.Suggestion, note.Rationale)
			}
			if strings.TrimSpace(note.File) != "" {
				items = append(items, fmt.Sprintf("%s: %s", note.File, noteSummary))
				continue
			}
			items = append(items, noteSummary)
		}
	}
	return items
}

func buildExpectedReportShape(exec *ctxpkg.ExecutionOutput) (string, error) {
	var shape interface{}

	switch {
	case exec.StatusReportRequired != nil && *exec.StatusReportRequired:
		completedID, remainingID := sampleStatusCriterionIDs(exec.StatusReport)
		shape = map[string]interface{}{
			"completed": []string{completedID},
			"remaining": []string{remainingID},
			"blocked":   []string{},
			"na":        []string{},
			"newIssues": []string{},
		}
	case exec.RefactorInstructions != nil:
		shape = map[string]interface{}{
			"refactorApplied": true,
		}
	case exec.TDDPhase != nil && strings.TrimSpace(*exec.TDDPhase) != "":
		shape = sampleTDDVerifierReport(*exec.TDDPhase)
	default:
		completed := []string{}
		if exec.Task != nil && strings.TrimSpace(exec.Task.ID) != "" {
			completed = append(completed, exec.Task.ID)
		}
		shape = map[string]interface{}{
			"completed": completed,
			"remaining": []string{},
			"blocked":   []string{},
		}
	}

	bytes, err := json.MarshalIndent(shape, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func sampleStatusCriterionIDs(status *ctxpkg.StatusReportRequest) (string, string) {
	completedID := "ac-1"
	remainingID := "ac-2"
	if status == nil || len(status.Criteria) == 0 {
		return completedID, remainingID
	}
	completedID = status.Criteria[0].ID
	if len(status.Criteria) > 1 {
		remainingID = status.Criteria[1].ID
	} else {
		remainingID = status.Criteria[0].ID
	}
	return completedID, remainingID
}

func sampleTDDVerifierReport(phase string) map[string]interface{} {
	switch phase {
	case state.TDDCycleRed:
		return map[string]interface{}{
			"passed":   true,
			"phase":    "red",
			"readOnly": true,
			"output":   "<summary>",
			"results": []map[string]string{
				{"id": "ac-1", "status": "PASS", "evidence": "..."},
			},
		}
	case state.TDDCycleGreen:
		return map[string]interface{}{
			"passed":        true,
			"phase":         "green",
			"output":        "<summary>",
			"failedACs":     []string{},
			"refactorNotes": []map[string]string{},
		}
	case state.TDDCycleRefactor:
		return map[string]interface{}{
			"passed":        true,
			"phase":         "refactor",
			"output":        "<summary>",
			"refactorNotes": []map[string]string{},
		}
	default:
		return map[string]interface{}{
			"completed": []string{},
			"remaining": []string{},
			"blocked":   []string{},
		}
	}
}

func loadExecutionSpec(root string, st state.StateFile) (*spec.ParsedSpec, error) {
	if st.Spec == nil || strings.TrimSpace(*st.Spec) == "" {
		return nil, fmt.Errorf("execution state has no active spec; cannot build execution contract")
	}

	parsed, err := spec.ParseSpec(root, *st.Spec)
	if err != nil {
		return nil, fmt.Errorf("parse spec %q: %w", *st.Spec, err)
	}
	if parsed == nil {
		return nil, fmt.Errorf("parse spec %q: empty parsed spec", *st.Spec)
	}
	return parsed, nil
}

// logBlockedFile writes a blocked log entry.
func logBlockedFile(root, reason string, iteration int) error {
	logPath := fmt.Sprintf("%s/%s/.state/blocked.log", root, state.TddmasterDir)
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "iteration=%d reason=%s\n", iteration, reason)
	return err
}
