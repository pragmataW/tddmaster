package cmd

import (
	"context"
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

	if initialState.Phase != state.PhaseExecuting && initialState.Phase != state.PhaseSpecApproved && initialState.Phase != state.PhaseBlocked {
		return fmt.Errorf("cannot run from phase: %s. Must be in SPEC_APPROVED or EXECUTING to start", initialState.Phase)
	}

	if initialState.Phase == state.PhaseSpecApproved {
		printErr("Starting execution from approved spec...")
		newState, err := state.StartExecution(initialState)
		if err != nil {
			return err
		}
		if err := state.WriteState(root, newState); err != nil {
			return err
		}
	}

	config, _ := state.ReadManifest(root)
	if config == nil {
		return fmt.Errorf("config not found")
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

		compiled := ctxpkg.Compile(st, activeConcerns, tier1, config, nil, nil, nil, hints, nil, tier2Count)
		prompt := buildAgentPrompt(compiled)

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
func buildAgentPrompt(compiled ctxpkg.NextOutput) string {
	var lines []string

	lines = append(lines, compiled.Meta.ResumeHint)
	lines = append(lines, "")

	if compiled.Meta.Spec != nil {
		lines = append(lines, fmt.Sprintf("Working on spec: %s", *compiled.Meta.Spec))
		lines = append(lines, "")
	}

	if compiled.Behavioral.Rules != nil {
		lines = append(lines, "Rules:")
		for _, r := range compiled.Behavioral.Rules {
			lines = append(lines, "- "+r)
		}
		lines = append(lines, "")
	}

	prefix := output.CmdPrefix()
	lines = append(lines, fmt.Sprintf(`When done, report progress: %s next --answer="your progress"`, prefix))
	lines = append(lines, fmt.Sprintf(`If blocked, run: %s block "reason"`, prefix))
	lines = append(lines, fmt.Sprintf("When all tasks are complete: %s done", prefix))

	return strings.Join(lines, "\n")
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
