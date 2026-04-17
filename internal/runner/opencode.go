package runner

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

const (
	openCodeScannerInitBufSize = 64 * 1024        // 64 KiB initial buffer
	openCodeScannerMaxLineSize = 10 * 1024 * 1024 // 10 MiB max NDJSON line
)

// OpenCodeRunner satisfies Runner for the `opencode` CLI.
type OpenCodeRunner struct {
	execFactory  execFactory
	lookPathFunc func(string) (string, error)
}

// NewOpenCodeRunner returns a *OpenCodeRunner wired with defaults.
func NewOpenCodeRunner() *OpenCodeRunner {
	return &OpenCodeRunner{
		execFactory:  defaultExecFactory,
		lookPathFunc: exec.LookPath,
	}
}

// Name returns the canonical runner identifier.
func (o *OpenCodeRunner) Name() string { return "opencode" }

// Available performs a cheap preflight check for the `opencode` binary.
func (o *OpenCodeRunner) Available() error {
	_, err := o.lookPathFunc("opencode")
	if err != nil {
		return fmt.Errorf("%w: opencode", ErrBinaryNotFound)
	}
	return nil
}

// Invoke spawns the opencode CLI, streams output, and returns the structured result.
func (o *OpenCodeRunner) Invoke(ctx context.Context, req RunRequest) (*RunResult, error) {
	// Empty-prompt check must precede lookPath (AC-20).
	if req.Prompt == "" {
		return nil, fmt.Errorf("%w: prompt is empty", ErrInvalidArgs)
	}

	if _, err := o.lookPathFunc("opencode"); err != nil {
		return nil, fmt.Errorf("%w: opencode", ErrBinaryNotFound)
	}

	// Build argv:
	// opencode run --format json [--dir <WorkDir>] [ExtraArgs...] <Prompt>
	args := []string{
		"run",
		"--format", "json",
	}

	if req.WorkDir != "" {
		args = append(args, "--dir", req.WorkDir)
	}

	// MaxTurns, SystemPrompt, OutputFormat are intentionally ignored.
	// ExtraArgs appended verbatim before positional prompt.
	args = append(args, req.ExtraArgs...)
	args = append(args, req.Prompt)

	cmd := o.execFactory(ctx, "opencode", args...)

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
		Stdout: stdoutBuf.Bytes(),
		Stderr: stderrBuf.Bytes(),
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

	if parsed := o.parseOpenCodeNDJSON(stdoutBuf.Bytes()); parsed != nil {
		result.ParsedJSON = parsed
	}

	return result, nil
}

// parseOpenCodeNDJSON scans NDJSON output from the opencode CLI and extracts the
// last text event's part.text along with any sessionID metadata.
// Returns nil if no text event was found or the data could not be decoded.
func (o *OpenCodeRunner) parseOpenCodeNDJSON(data []byte) map[string]any {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, openCodeScannerInitBufSize), openCodeScannerMaxLineSize)

	var lastText string
	var sessionID string
	foundText := false

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var evt map[string]any
		if err := json.Unmarshal(line, &evt); err != nil {
			continue
		}

		eventType, _ := evt["type"].(string)

		// Track last seen sessionID across all events.
		if sid, ok := evt["sessionID"].(string); ok && sid != "" {
			sessionID = sid
		}

		if eventType == "text" {
			if part, ok := evt["part"].(map[string]any); ok {
				if txt, ok := part["text"].(string); ok {
					lastText = txt
					foundText = true
				}
			}
		}
	}

	if !foundText {
		return nil
	}

	parsed := map[string]any{"result": lastText}
	if sessionID != "" {
		parsed["session_id"] = sessionID
	}
	return parsed
}
