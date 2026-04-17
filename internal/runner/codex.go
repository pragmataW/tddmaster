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
	"strconv"
	"time"
)

const (
	codexScannerInitBufSize = 64 * 1024        // 64 KiB initial buffer
	codexScannerMaxLineSize = 10 * 1024 * 1024 // 10 MiB max NDJSON line
)

// CodexRunner satisfies Runner for the OpenAI `codex` CLI.
type CodexRunner struct {
	execFactory  execFactory
	lookPathFunc func(string) (string, error)
}

// NewCodexRunner returns a *CodexRunner wired with defaults.
func NewCodexRunner() *CodexRunner {
	return &CodexRunner{
		execFactory:  defaultExecFactory,
		lookPathFunc: exec.LookPath,
	}
}

// Name returns the canonical runner identifier.
func (c *CodexRunner) Name() string { return "codex" }

// Available performs a cheap preflight check for the `codex` binary.
func (c *CodexRunner) Available() error {
	_, err := c.lookPathFunc("codex")
	if err != nil {
		return fmt.Errorf("%w: codex", ErrBinaryNotFound)
	}
	return nil
}

// Invoke spawns the codex CLI, streams output, and returns the structured result.
func (c *CodexRunner) Invoke(ctx context.Context, req RunRequest) (*RunResult, error) {
	if req.Prompt == "" {
		return nil, fmt.Errorf("%w: prompt is empty", ErrInvalidArgs)
	}

	if _, err := c.lookPathFunc("codex"); err != nil {
		return nil, fmt.Errorf("%w: codex", ErrBinaryNotFound)
	}

	// Build argv:
	// codex exec --json --ask-for-approval never [--cd <WorkDir>] [-c agents.job_max_runtime_seconds=<N*60>] [ExtraArgs...] <Prompt>
	args := []string{
		"exec",
		"--json",
		"--ask-for-approval", "never",
	}

	if req.WorkDir != "" {
		args = append(args, "--cd", req.WorkDir)
	}

	if req.MaxTurns > 0 {
		seconds := req.MaxTurns * 60
		args = append(args, "-c", "agents.job_max_runtime_seconds="+strconv.Itoa(seconds))
	}

	args = append(args, req.ExtraArgs...)
	args = append(args, req.Prompt)

	cmd := c.execFactory(ctx, "codex", args...)

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

	if parsed := c.parseCodexNDJSON(stdoutBuf.Bytes()); parsed != nil {
		result.ParsedJSON = parsed
	}

	return result, nil
}

// parseCodexNDJSON scans NDJSON output from the codex CLI and extracts the
// last agent_message along with any usage and thread_id metadata.
// Returns nil if no agent_message was found or the data could not be decoded.
func (c *CodexRunner) parseCodexNDJSON(data []byte) map[string]any {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, codexScannerInitBufSize), codexScannerMaxLineSize) // allow big lines

	var lastAgentMessage string
	var usage any
	var threadID string
	foundAgentMessage := false

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
		switch eventType {
		case "thread.started":
			if tid, ok := evt["thread_id"].(string); ok {
				threadID = tid
			}
		case "turn.completed":
			if u, ok := evt["usage"]; ok {
				usage = u
			}
		case "item.completed":
			if item, ok := evt["item"].(map[string]any); ok {
				itype, _ := item["item_type"].(string)
				if itype == "agent_message" {
					if txt, ok := item["text"].(string); ok {
						lastAgentMessage = txt
						foundAgentMessage = true
					}
				}
			}
		}
	}

	if !foundAgentMessage {
		return nil
	}

	parsed := map[string]any{
		"result": lastAgentMessage,
	}
	if usage != nil {
		parsed["usage"] = usage
	}
	if threadID != "" {
		parsed["thread_id"] = threadID
	}
	return parsed
}
