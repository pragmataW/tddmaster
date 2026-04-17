package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"time"
)

// execCmd abstracts the subset of *exec.Cmd used by runners. Tests provide
// a fake implementation; production wraps *exec.Cmd via realExecCmd.
type execCmd interface {
	Run() error
	Start() error
	Wait() error
	SetStdout(io.Writer)
	SetStderr(io.Writer)
	SetDir(string)
	SetEnv([]string)
	ProcessSignal(os.Signal) error
}

// execFactory creates an execCmd for the given context+argv.
type execFactory func(ctx context.Context, name string, args ...string) execCmd

// ClaudeRunner satisfies Runner for the Anthropic `claude` CLI.
type ClaudeRunner struct {
	execFactory  execFactory
	lookPathFunc func(string) (string, error)
}

// NewClaudeRunner returns a *ClaudeRunner wired with defaults.
func NewClaudeRunner() *ClaudeRunner {
	return &ClaudeRunner{
		execFactory:  defaultExecFactory,
		lookPathFunc: exec.LookPath,
	}
}

// realExecCmd wraps *exec.Cmd to satisfy execCmd. Production-only.
type realExecCmd struct {
	cmd *exec.Cmd
}

func (r *realExecCmd) Run() error            { return r.cmd.Run() }
func (r *realExecCmd) Start() error          { return r.cmd.Start() }
func (r *realExecCmd) Wait() error           { return r.cmd.Wait() }
func (r *realExecCmd) SetStdout(w io.Writer) { r.cmd.Stdout = w }
func (r *realExecCmd) SetStderr(w io.Writer) { r.cmd.Stderr = w }
func (r *realExecCmd) SetDir(d string)       { r.cmd.Dir = d }
func (r *realExecCmd) SetEnv(e []string)     { r.cmd.Env = e }
func (r *realExecCmd) ProcessSignal(sig os.Signal) error {
	if r.cmd.Process == nil {
		return fmt.Errorf("runner: process not started")
	}
	return r.cmd.Process.Signal(sig)
}

func defaultExecFactory(ctx context.Context, name string, args ...string) execCmd {
	return &realExecCmd{cmd: exec.CommandContext(ctx, name, args...)}
}

// Name returns the canonical runner identifier.
func (c *ClaudeRunner) Name() string { return "claude-code" }

// Available performs a cheap preflight check for the `claude` binary.
func (c *ClaudeRunner) Available() error {
	_, err := c.lookPathFunc("claude")
	if err != nil {
		return fmt.Errorf("%w: claude", ErrBinaryNotFound)
	}
	return nil
}

// Invoke spawns the claude CLI, streams output, and returns the structured result.
func (c *ClaudeRunner) Invoke(ctx context.Context, req RunRequest) (*RunResult, error) {
	if req.Prompt == "" {
		return nil, fmt.Errorf("%w: prompt is empty", ErrInvalidArgs)
	}

	if _, err := c.lookPathFunc("claude"); err != nil {
		return nil, fmt.Errorf("%w: claude", ErrBinaryNotFound)
	}

	// Build argv: claude -p <prompt> --output-format <fmt> --max-turns <N> [ExtraArgs...]
	outputFormat := req.OutputFormat
	if outputFormat == "" {
		outputFormat = "json"
	}
	maxTurns := req.MaxTurns
	if maxTurns == 0 {
		maxTurns = 10
	}
	args := []string{
		"-p", req.Prompt,
		"--output-format", outputFormat,
		"--max-turns", strconv.Itoa(maxTurns),
	}
	args = append(args, req.ExtraArgs...)

	cmd := c.execFactory(ctx, "claude", args...)

	// Capture + stream: combine writers when caller supplied one.
	var stdoutBuf, stderrBuf bytes.Buffer
	var stdoutW io.Writer = &stdoutBuf
	var stderrW io.Writer = &stderrBuf
	if req.Stdout != nil {
		stdoutW = io.MultiWriter(&stdoutBuf, req.Stdout)
	}
	if req.Stderr != nil {
		stderrW = io.MultiWriter(&stderrBuf, req.Stderr)
	}
	cmd.SetStdout(stdoutW)
	cmd.SetStderr(stderrW)

	if req.WorkDir != "" {
		cmd.SetDir(req.WorkDir)
	}

	// Merge parent env with overrides.
	env := append(os.Environ(), req.Env...)
	cmd.SetEnv(env)

	// Start + watch context for cancel + wait.
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	var runErr error
	select {
	case <-ctx.Done():
		_ = cmd.ProcessSignal(os.Interrupt)
		// Wait briefly for graceful exit, then give up.
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
		return nil, fmt.Errorf("%w: %v", ErrContextCanceled, ctx.Err())
	case runErr = <-done:
	}

	result := &RunResult{
		ExitCode: 0,
		Stdout:   stdoutBuf.Bytes(),
		Stderr:   stderrBuf.Bytes(),
	}
	if runErr != nil {
		// Extract exit code from the error. Tests use a fake error that
		// exposes ExitCode(); production gets *exec.ExitError.
		if coder, ok := runErr.(interface{ ExitCode() int }); ok {
			result.ExitCode = coder.ExitCode()
		} else {
			return result, runErr
		}
	}

	// Best-effort JSON parse of full stdout.
	var parsed map[string]any
	if err := json.Unmarshal(stdoutBuf.Bytes(), &parsed); err == nil {
		result.ParsedJSON = parsed
	}

	return result, nil
}
