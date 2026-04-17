package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// openCodeFakeExecCmd — test double for execCmd (OpenCode-specific)
// ---------------------------------------------------------------------------

// openCodeFakeExecCmd is a test-only execCmd implementation for OpenCodeRunner
// tests. It mirrors the fakeExecCmd pattern from codex_test.go exactly.
type openCodeFakeExecCmd struct {
	argv       []string
	stdoutW    io.Writer
	stderrW    io.Writer
	dir        string
	env        []string
	stdoutData []byte
	stderrData []byte
	exitCode   int
	runErr     error
	startDelay time.Duration // non-zero → Start blocks; used for context-cancel test

	mu           sync.Mutex
	signaledWith os.Signal
	started      bool
	done         chan struct{} // closed when Wait becomes unblocked
}

func newOpenCodeFakeExecCmd(stdoutData, stderrData []byte, exitCode int, runErr error) *openCodeFakeExecCmd {
	return &openCodeFakeExecCmd{
		stdoutData: stdoutData,
		stderrData: stderrData,
		exitCode:   exitCode,
		runErr:     runErr,
		done:       make(chan struct{}),
	}
}

func (f *openCodeFakeExecCmd) SetStdout(w io.Writer) { f.stdoutW = w }
func (f *openCodeFakeExecCmd) SetStderr(w io.Writer) { f.stderrW = w }
func (f *openCodeFakeExecCmd) SetDir(dir string)     { f.dir = dir }
func (f *openCodeFakeExecCmd) SetEnv(env []string)   { f.env = env }

func (f *openCodeFakeExecCmd) ProcessSignal(sig os.Signal) error {
	f.mu.Lock()
	f.signaledWith = sig
	f.mu.Unlock()
	// Unblock Wait.
	select {
	case <-f.done:
	default:
		close(f.done)
	}
	return nil
}

func (f *openCodeFakeExecCmd) Run() error {
	defer func() {
		select {
		case <-f.done:
		default:
			close(f.done)
		}
	}()
	if f.startDelay > 0 {
		time.Sleep(f.startDelay)
	}
	if f.stdoutW != nil {
		f.stdoutW.Write(f.stdoutData) //nolint:errcheck
	}
	if f.stderrW != nil {
		f.stderrW.Write(f.stderrData) //nolint:errcheck
	}
	if f.runErr != nil {
		return f.runErr
	}
	if f.exitCode != 0 {
		return &openCodeFakeExitError{code: f.exitCode}
	}
	return nil
}

func (f *openCodeFakeExecCmd) Start() error {
	f.mu.Lock()
	f.started = true
	f.mu.Unlock()
	go func() {
		defer func() {
			select {
			case <-f.done:
			default:
				close(f.done)
			}
		}()
		if f.startDelay > 0 {
			select {
			case <-f.done: // signaled early (context cancel)
				return
			case <-time.After(f.startDelay):
			}
		}
		if f.stdoutW != nil {
			f.stdoutW.Write(f.stdoutData) //nolint:errcheck
		}
		if f.stderrW != nil {
			f.stderrW.Write(f.stderrData) //nolint:errcheck
		}
	}()
	return nil
}

func (f *openCodeFakeExecCmd) Wait() error {
	<-f.done
	if f.runErr != nil {
		return f.runErr
	}
	if f.exitCode != 0 {
		return &openCodeFakeExitError{code: f.exitCode}
	}
	return nil
}

// openCodeFakeExitError simulates exec.ExitError with a configurable exit code.
type openCodeFakeExitError struct {
	code int
}

func (e *openCodeFakeExitError) Error() string { return "exit status " + strconv.Itoa(e.code) }
func (e *openCodeFakeExitError) ExitCode() int { return e.code }

// ---------------------------------------------------------------------------
// buildOpenCodeFakeFactory — captures argv and returns the pre-wired fake
// ---------------------------------------------------------------------------

// buildOpenCodeFakeFactory returns an execFactory that captures the full argv
// (binary name + args) for every spawned command and returns cmd.
func buildOpenCodeFakeFactory(cmd *openCodeFakeExecCmd) (execFactory, *[]string) {
	capturedArgv := &[]string{}
	factory := func(_ context.Context, name string, args ...string) execCmd {
		argv := append([]string{name}, args...)
		*capturedArgv = argv
		cmd.argv = argv
		return cmd
	}
	return factory, capturedArgv
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// newOpenCodeRunnerWithFactory returns a *OpenCodeRunner with injected seams.
func newOpenCodeRunnerWithFactory(t *testing.T, factory execFactory, lookPath func(string) (string, error)) *OpenCodeRunner {
	t.Helper()
	r := NewOpenCodeRunner()
	r.execFactory = factory
	if lookPath != nil {
		r.lookPathFunc = lookPath
	}
	return r
}

// openCodeLookPathFound is a lookPath stub that always succeeds.
func openCodeLookPathFound(_ string) (string, error) {
	return "/Users/pragmata/.opencode/bin/opencode", nil
}

// openCodeLookPathMissing is a lookPath stub that always fails.
func openCodeLookPathMissing(_ string) (string, error) {
	return "", exec.ErrNotFound
}

// openCodeNDJSONFixture builds the NDJSON event fixture described in AC-15.
func openCodeNDJSONFixture() []byte {
	var sb strings.Builder
	sb.WriteString(`{"type":"step_start","timestamp":1,"sessionID":"S-42"}`)
	sb.WriteByte('\n')
	sb.WriteString(`{"type":"tool_use","timestamp":2,"sessionID":"S-42","part":{"id":"t1"}}`)
	sb.WriteByte('\n')
	sb.WriteString(`{"type":"text","timestamp":3,"sessionID":"S-42","part":{"text":"first text"}}`)
	sb.WriteByte('\n')
	sb.WriteString(`{"type":"step_finish","timestamp":4,"sessionID":"S-42"}`)
	sb.WriteByte('\n')
	sb.WriteString(`{"type":"text","timestamp":5,"sessionID":"S-42","part":{"text":"final answer"}}`)
	sb.WriteByte('\n')
	return []byte(sb.String())
}

// ---------------------------------------------------------------------------
// AC-1: Name()
// ---------------------------------------------------------------------------

func TestOpenCodeRunner_Name(t *testing.T) {
	r := NewOpenCodeRunner()
	if got := r.Name(); got != "opencode" {
		t.Fatalf("Name(): want %q, got %q", "opencode", got)
	}
}

// ---------------------------------------------------------------------------
// AC-2: Available() — binary found
// ---------------------------------------------------------------------------

func TestOpenCodeRunner_Available_BinaryFound(t *testing.T) {
	r := NewOpenCodeRunner()
	r.lookPathFunc = openCodeLookPathFound

	if err := r.Available(); err != nil {
		t.Fatalf("Available() with found binary: want nil, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// AC-3 / EC-1: Available() — binary missing
// ---------------------------------------------------------------------------

func TestOpenCodeRunner_Available_BinaryMissing(t *testing.T) {
	// EC-1: CLI binary PATH'te bulunamadı
	r := NewOpenCodeRunner()
	r.lookPathFunc = openCodeLookPathMissing

	err := r.Available()
	if err == nil {
		t.Fatal("Available() with missing binary: want error, got nil")
	}
	if !errors.Is(err, ErrBinaryNotFound) {
		t.Fatalf("Available() with missing binary: want errors.Is(ErrBinaryNotFound), got %v", err)
	}
}

// ---------------------------------------------------------------------------
// AC-4 / EC-2: Default argv shape — run --format json <prompt> (positional last)
// ---------------------------------------------------------------------------

func TestOpenCodeRunner_Invoke_DefaultArgv(t *testing.T) {
	// EC-2: argv/flag uyumsuzluğu — OpenCode uses 'run', positional prompt last, --format json
	fake := newOpenCodeFakeExecCmd(nil, nil, 0, nil)
	factory, capturedArgv := buildOpenCodeFakeFactory(fake)

	r := newOpenCodeRunnerWithFactory(t, factory, openCodeLookPathFound)

	req := RunRequest{Prompt: "hello"}
	_, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}

	// Expected: ["opencode", "run", "--format", "json", "hello"]
	// (in that exact order; positional prompt is always last)
	want := []string{"opencode", "run", "--format", "json", "hello"}
	got := *capturedArgv

	if len(got) != len(want) {
		t.Fatalf("argv length: want %d, got %d\nwant: %v\ngot:  %v", len(want), len(got), want, got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("argv[%d]: want %q, got %q\nfull argv: %v", i, w, got[i], got)
		}
	}
}

// ---------------------------------------------------------------------------
// AC-5 / EC-2: WorkDir — --dir flag AND SetDir
// ---------------------------------------------------------------------------

func TestOpenCodeRunner_Invoke_WorkDir(t *testing.T) {
	fake := newOpenCodeFakeExecCmd(nil, nil, 0, nil)
	factory, capturedArgv := buildOpenCodeFakeFactory(fake)

	r := newOpenCodeRunnerWithFactory(t, factory, openCodeLookPathFound)

	req := RunRequest{Prompt: "hi", WorkDir: "/tmp/x"}
	_, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}

	argv := *capturedArgv

	// --dir /tmp/x must appear before the positional prompt.
	dirIdx := indexOfToken(argv, "--dir")
	if dirIdx == -1 {
		t.Fatalf("argv: --dir flag not found; argv: %v", argv)
	}
	if dirIdx+1 >= len(argv) {
		t.Fatal("argv: --dir flag has no value")
	}
	if argv[dirIdx+1] != "/tmp/x" {
		t.Errorf("--dir value: want %q, got %q", "/tmp/x", argv[dirIdx+1])
	}

	// Positional prompt must be last.
	if argv[len(argv)-1] != "hi" {
		t.Errorf("prompt must be last argv element; argv: %v", argv)
	}

	// --dir must appear before the prompt.
	promptIdx := len(argv) - 1
	if dirIdx >= promptIdx {
		t.Errorf("--dir must appear before prompt; dirIdx=%d promptIdx=%d argv: %v", dirIdx, promptIdx, argv)
	}

	// SetDir must also be called (cwd propagation).
	if fake.dir != "/tmp/x" {
		t.Errorf("SetDir: want %q, got %q", "/tmp/x", fake.dir)
	}
}

// ---------------------------------------------------------------------------
// AC-6 / EC-2: MaxTurns ignored — no turn/time-limit flag for any MaxTurns value
// ---------------------------------------------------------------------------

func TestOpenCodeRunner_Invoke_IgnoresMaxTurns(t *testing.T) {
	// EC-2: OpenCode has no --max-turns flag; MaxTurns must never produce any flag.
	cases := []struct {
		name     string
		maxTurns int
	}{
		{"zero", 0},
		{"one", 1},
		{"five", 5},
		{"hundred", 100},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := newOpenCodeFakeExecCmd(nil, nil, 0, nil)
			factory, capturedArgv := buildOpenCodeFakeFactory(fake)
			r := newOpenCodeRunnerWithFactory(t, factory, openCodeLookPathFound)

			req := RunRequest{Prompt: "task", MaxTurns: tc.maxTurns}
			_, err := r.Invoke(context.Background(), req)
			if err != nil {
				t.Fatalf("MaxTurns=%d: Invoke unexpected error: %v", tc.maxTurns, err)
			}

			argv := *capturedArgv
			// No turn/time-limit related token must appear.
			for _, a := range argv {
				if strings.Contains(a, "max-turns") {
					t.Errorf("MaxTurns=%d: --max-turns token must NOT appear; argv: %v", tc.maxTurns, argv)
				}
				if a == "-c" {
					t.Errorf("MaxTurns=%d: -c flag must NOT appear (no config override); argv: %v", tc.maxTurns, argv)
				}
				if strings.Contains(a, "job_max_runtime_seconds") {
					t.Errorf("MaxTurns=%d: job_max_runtime_seconds must NOT appear; argv: %v", tc.maxTurns, argv)
				}
				if strings.Contains(a, "max-time") {
					t.Errorf("MaxTurns=%d: --max-time token must NOT appear; argv: %v", tc.maxTurns, argv)
				}
			}

			// Base shape must hold: run --format json <prompt>
			want := []string{"opencode", "run", "--format", "json", "task"}
			if len(argv) != len(want) {
				t.Errorf("MaxTurns=%d: argv should be base shape only; want %v, got %v", tc.maxTurns, want, argv)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AC-7 / EC-2: OutputFormat ignored — --format json always; --output-format never
// ---------------------------------------------------------------------------

func TestOpenCodeRunner_Invoke_IgnoresOutputFormat(t *testing.T) {
	// EC-2: RunRequest.OutputFormat is ignored; --format json is always emitted.
	cases := []struct {
		name         string
		outputFormat string
	}{
		{"empty", ""},
		{"text", "text"},
		{"ndjson", "ndjson"},
		{"json", "json"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := newOpenCodeFakeExecCmd(nil, nil, 0, nil)
			factory, capturedArgv := buildOpenCodeFakeFactory(fake)
			r := newOpenCodeRunnerWithFactory(t, factory, openCodeLookPathFound)

			req := RunRequest{Prompt: "hello", OutputFormat: tc.outputFormat}
			_, err := r.Invoke(context.Background(), req)
			if err != nil {
				t.Fatalf("Invoke: unexpected error: %v", err)
			}

			argv := *capturedArgv
			// --format json must always appear as consecutive pair.
			if !containsConsecutive(argv, "--format", "json") {
				t.Errorf("OutputFormat=%q: [\"--format\", \"json\"] must always appear in argv; got: %v", tc.outputFormat, argv)
			}
			// --output-format must never appear.
			if containsToken(argv, "--output-format") {
				t.Errorf("OutputFormat=%q: --output-format must NOT appear in argv; got: %v", tc.outputFormat, argv)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AC-8 / EC-2: SystemPrompt ignored — no --system-prompt or 'system' token in argv
// ---------------------------------------------------------------------------

func TestOpenCodeRunner_Invoke_IgnoresSystemPrompt(t *testing.T) {
	// EC-2: OpenCode has no --system-prompt flag; field is silently ignored.
	fake := newOpenCodeFakeExecCmd(nil, nil, 0, nil)
	factory, capturedArgv := buildOpenCodeFakeFactory(fake)
	r := newOpenCodeRunnerWithFactory(t, factory, openCodeLookPathFound)

	req := RunRequest{Prompt: "hello", SystemPrompt: "you are helpful"}
	_, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}

	argv := *capturedArgv
	for _, a := range argv {
		if strings.Contains(strings.ToLower(a), "system") {
			t.Errorf("SystemPrompt must be ignored; unexpected token %q in argv: %v", a, argv)
		}
	}

	// argv must be byte-identical to a call without SystemPrompt.
	fake2 := newOpenCodeFakeExecCmd(nil, nil, 0, nil)
	factory2, capturedArgv2 := buildOpenCodeFakeFactory(fake2)
	r2 := newOpenCodeRunnerWithFactory(t, factory2, openCodeLookPathFound)

	req2 := RunRequest{Prompt: "hello"}
	_, err2 := r2.Invoke(context.Background(), req2)
	if err2 != nil {
		t.Fatalf("Invoke (no SystemPrompt): unexpected error: %v", err2)
	}

	argv2 := *capturedArgv2
	if len(argv) != len(argv2) {
		t.Errorf("SystemPrompt must not change argv length: with=%v without=%v", argv, argv2)
	}
	for i := range argv {
		if i < len(argv2) && argv[i] != argv2[i] {
			t.Errorf("argv[%d] differs: with SystemPrompt=%q without=%q", i, argv[i], argv2[i])
		}
	}
}

// ---------------------------------------------------------------------------
// AC-9: ExtraArgs — appended after adapter flags, before positional prompt
// ---------------------------------------------------------------------------

func TestOpenCodeRunner_Invoke_ExtraArgs(t *testing.T) {
	fake := newOpenCodeFakeExecCmd(nil, nil, 0, nil)
	factory, capturedArgv := buildOpenCodeFakeFactory(fake)

	r := newOpenCodeRunnerWithFactory(t, factory, openCodeLookPathFound)

	req := RunRequest{
		Prompt:    "do work",
		ExtraArgs: []string{"--model", "anthropic/claude-opus-4-7"},
	}
	_, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}

	argv := *capturedArgv
	promptIdx := len(argv) - 1

	// The prompt must still be last.
	if argv[promptIdx] != "do work" {
		t.Errorf("prompt must be last argv element; argv: %v", argv)
	}

	// --model must appear before the prompt.
	modelIdx := indexOfToken(argv, "--model")
	if modelIdx == -1 {
		t.Fatalf("argv: --model not found; argv: %v", argv)
	}
	if modelIdx >= promptIdx {
		t.Errorf("ExtraArgs must appear before prompt; modelIdx=%d promptIdx=%d argv: %v", modelIdx, promptIdx, argv)
	}
	if modelIdx+1 >= len(argv) || argv[modelIdx+1] != "anthropic/claude-opus-4-7" {
		t.Errorf("ExtraArgs value: want anthropic/claude-opus-4-7 after --model; argv: %v", argv)
	}

	// Adapter-owned flags must also be present.
	if !containsConsecutive(argv, "--format", "json") {
		t.Errorf("[\"--format\", \"json\"] must always be present; argv: %v", argv)
	}
}

// ---------------------------------------------------------------------------
// AC-10: ExtraArgs with --agent plan — passed through verbatim
// ---------------------------------------------------------------------------

func TestOpenCodeRunner_Invoke_ExtraArgs_AgentPlan(t *testing.T) {
	fake := newOpenCodeFakeExecCmd(nil, nil, 0, nil)
	factory, capturedArgv := buildOpenCodeFakeFactory(fake)

	r := newOpenCodeRunnerWithFactory(t, factory, openCodeLookPathFound)

	req := RunRequest{
		Prompt:    "plan this",
		ExtraArgs: []string{"--agent", "plan"},
	}
	_, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}

	argv := *capturedArgv

	// --agent plan must appear verbatim (adapter does NOT strip or override).
	if !containsConsecutive(argv, "--agent", "plan") {
		t.Errorf("ExtraArgs [--agent plan] must be passed through verbatim; argv: %v", argv)
	}

	// Prompt must still be last.
	if argv[len(argv)-1] != "plan this" {
		t.Errorf("prompt must be last argv element; argv: %v", argv)
	}
}

// ---------------------------------------------------------------------------
// AC-11 / EC-2: No auto --agent flag — adapter must NOT inject --agent if not in ExtraArgs
// ---------------------------------------------------------------------------

func TestOpenCodeRunner_Invoke_NoAutoAgentFlag(t *testing.T) {
	// EC-2: The adapter must NOT auto-inject --agent build; rely on CLI default.
	fake := newOpenCodeFakeExecCmd(nil, nil, 0, nil)
	factory, capturedArgv := buildOpenCodeFakeFactory(fake)

	r := newOpenCodeRunnerWithFactory(t, factory, openCodeLookPathFound)

	req := RunRequest{Prompt: "hello"}
	_, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}

	argv := *capturedArgv
	if containsToken(argv, "--agent") {
		t.Errorf("adapter must NOT auto-inject --agent flag; argv: %v", argv)
	}
}

// ---------------------------------------------------------------------------
// AC-12: Stdout streamed to caller writer AND captured in RunResult
// ---------------------------------------------------------------------------

func TestOpenCodeRunner_Invoke_StreamsStdout(t *testing.T) {
	stdoutData := []byte("stream data")
	fake := newOpenCodeFakeExecCmd(stdoutData, nil, 0, nil)
	factory, _ := buildOpenCodeFakeFactory(fake)
	r := newOpenCodeRunnerWithFactory(t, factory, openCodeLookPathFound)

	callerBuf := &bytes.Buffer{}
	req := RunRequest{Prompt: "hello", Stdout: callerBuf}
	result, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}

	// Caller's writer must receive the bytes.
	if !bytes.Equal(callerBuf.Bytes(), stdoutData) {
		t.Errorf("caller stdout: want %q, got %q", stdoutData, callerBuf.Bytes())
	}
	// RunResult.Stdout must also have them.
	if !bytes.Contains(result.Stdout, stdoutData) {
		t.Errorf("RunResult.Stdout: want to contain %q, got %q", stdoutData, result.Stdout)
	}
}

// ---------------------------------------------------------------------------
// AC-13: Stderr streamed to caller writer AND captured in RunResult
// ---------------------------------------------------------------------------

func TestOpenCodeRunner_Invoke_StreamsStderr(t *testing.T) {
	stderrData := []byte("error output")
	fake := newOpenCodeFakeExecCmd(nil, stderrData, 0, nil)
	factory, _ := buildOpenCodeFakeFactory(fake)
	r := newOpenCodeRunnerWithFactory(t, factory, openCodeLookPathFound)

	callerBuf := &bytes.Buffer{}
	req := RunRequest{Prompt: "hello", Stderr: callerBuf}
	result, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}

	// Caller's writer must receive the bytes.
	if !bytes.Equal(callerBuf.Bytes(), stderrData) {
		t.Errorf("caller stderr: want %q, got %q", stderrData, callerBuf.Bytes())
	}
	// RunResult.Stderr must also have them.
	if !bytes.Contains(result.Stderr, stderrData) {
		t.Errorf("RunResult.Stderr: want to contain %q, got %q", stderrData, result.Stderr)
	}
}

// ---------------------------------------------------------------------------
// AC-14: Env merged onto parent env
// ---------------------------------------------------------------------------

func TestOpenCodeRunner_Invoke_MergesEnv(t *testing.T) {
	fake := newOpenCodeFakeExecCmd(nil, nil, 0, nil)
	factory, _ := buildOpenCodeFakeFactory(fake)
	r := newOpenCodeRunnerWithFactory(t, factory, openCodeLookPathFound)

	req := RunRequest{Prompt: "hello", Env: []string{"FOO=bar"}}
	_, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}

	found := false
	for _, e := range fake.env {
		if e == "FOO=bar" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Env: want FOO=bar in process env; got: %v", fake.env)
	}

	// Parent env must also be present (merged, not replaced).
	if len(fake.env) < len(os.Environ())+1 {
		t.Errorf("Env: expected merged env (parent + overrides); len=%d", len(fake.env))
	}
}

// ---------------------------------------------------------------------------
// AC-15: NDJSON parsing — last text event's part.text + sessionID
// ---------------------------------------------------------------------------

func TestOpenCodeRunner_Invoke_ParsesNDJSON(t *testing.T) {
	// Fixture from spec:
	// line1: step_start  (sessionID="S-42")
	// line2: tool_use    (sessionID="S-42")
	// line3: text        (part.text="first text", sessionID="S-42")
	// line4: step_finish (sessionID="S-42")
	// line5: text        (part.text="final answer", sessionID="S-42")  ← LAST text wins
	stream := openCodeNDJSONFixture()

	fake := newOpenCodeFakeExecCmd(stream, nil, 0, nil)
	factory, _ := buildOpenCodeFakeFactory(fake)
	r := newOpenCodeRunnerWithFactory(t, factory, openCodeLookPathFound)

	req := RunRequest{Prompt: "hello"}
	result, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}

	if result.ParsedJSON == nil {
		t.Fatal("ParsedJSON: want non-nil map from NDJSON stream, got nil")
	}

	// result must be the LAST text event's part.text.
	gotResult, ok := result.ParsedJSON["result"]
	if !ok {
		t.Fatalf("ParsedJSON: missing key \"result\"; map: %v", result.ParsedJSON)
	}
	if gotResult != "final answer" {
		t.Errorf("ParsedJSON[\"result\"]: want %q (last text wins), got %q", "final answer", gotResult)
	}

	// session_id must be captured.
	gotSessionID := result.ParsedJSON["session_id"]
	if gotSessionID != "S-42" {
		t.Errorf("ParsedJSON[\"session_id\"]: want %q, got %v", "S-42", gotSessionID)
	}
}

// ---------------------------------------------------------------------------
// AC-16 / EC-4: NDJSON parse failure — ParsedJSON nil, err nil; raw Stdout preserved
// ---------------------------------------------------------------------------

func TestOpenCodeRunner_Invoke_NDJSONParseFailure_NilParsedJSON(t *testing.T) {
	// EC-4: JSON parse failure — graceful fallback, ParsedJSON nil, no error.
	invalidOutput := []byte("not json at all\x00\xff garbage")

	fake := newOpenCodeFakeExecCmd(invalidOutput, nil, 0, nil)
	factory, _ := buildOpenCodeFakeFactory(fake)
	r := newOpenCodeRunnerWithFactory(t, factory, openCodeLookPathFound)

	req := RunRequest{Prompt: "hello"}
	result, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke with invalid NDJSON: want nil error, got %v", err)
	}
	if result.ParsedJSON != nil {
		t.Errorf("ParsedJSON: want nil on parse failure, got %v", result.ParsedJSON)
	}
	// Raw stdout must still be captured.
	if !bytes.Contains(result.Stdout, invalidOutput) {
		t.Errorf("RunResult.Stdout: want to contain raw bytes, got %q", result.Stdout)
	}
}

// ---------------------------------------------------------------------------
// AC-17 / EC-4: NDJSON with no text event — ParsedJSON nil
// ---------------------------------------------------------------------------

func TestOpenCodeRunner_Invoke_NDJSONNoTextEvent_NilParsedJSON(t *testing.T) {
	// Stream contains only non-text events; no text event → ParsedJSON must be nil.
	stream := []byte(
		`{"type":"step_start","timestamp":1,"sessionID":"S-99"}` + "\n" +
			`{"type":"tool_use","timestamp":2,"sessionID":"S-99","part":{"id":"t1"}}` + "\n" +
			`{"type":"error","timestamp":3,"sessionID":"S-99","error":"something went wrong"}` + "\n",
	)

	fake := newOpenCodeFakeExecCmd(stream, nil, 0, nil)
	factory, _ := buildOpenCodeFakeFactory(fake)
	r := newOpenCodeRunnerWithFactory(t, factory, openCodeLookPathFound)

	req := RunRequest{Prompt: "hello"}
	result, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}
	if result.ParsedJSON != nil {
		t.Errorf("ParsedJSON: want nil when no text event found; got %v", result.ParsedJSON)
	}
}

// ---------------------------------------------------------------------------
// AC-18 / EC-3: Non-zero exit — ExitCode set, err nil, Stdout/Stderr captured
// ---------------------------------------------------------------------------

func TestOpenCodeRunner_Invoke_NonZeroExit(t *testing.T) {
	// EC-3: Non-zero exit code — Invoke returns nil error, caller decides fatality.
	stderrData := []byte("opencode fatal error")
	fake := newOpenCodeFakeExecCmd(nil, stderrData, 2, nil)
	factory, _ := buildOpenCodeFakeFactory(fake)
	r := newOpenCodeRunnerWithFactory(t, factory, openCodeLookPathFound)

	req := RunRequest{Prompt: "hello"}
	result, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke with non-zero exit: want nil error, got %v", err)
	}
	if result.ExitCode != 2 {
		t.Errorf("ExitCode: want 2, got %d", result.ExitCode)
	}
	if !bytes.Contains(result.Stderr, stderrData) {
		t.Errorf("RunResult.Stderr: want to contain %q, got %q", stderrData, result.Stderr)
	}
}

// ---------------------------------------------------------------------------
// AC-19 / EC-6: Context cancel — SIGINT sent, ErrContextCanceled returned
// ---------------------------------------------------------------------------

func TestOpenCodeRunner_Invoke_ContextCancel(t *testing.T) {
	// EC-6: context cancel + SIGINT + grace period.
	fake := newOpenCodeFakeExecCmd(nil, nil, 0, nil)
	fake.startDelay = 5 * time.Second // simulate slow process
	factory, _ := buildOpenCodeFakeFactory(fake)
	r := newOpenCodeRunnerWithFactory(t, factory, openCodeLookPathFound)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req := RunRequest{Prompt: "hello"}
	_, err := r.Invoke(ctx, req)

	if err == nil {
		t.Fatal("Invoke with canceled context: want error, got nil")
	}
	if !errors.Is(err, ErrContextCanceled) {
		t.Errorf("Invoke with canceled context: want errors.Is(ErrContextCanceled), got %v", err)
	}

	fake.mu.Lock()
	sig := fake.signaledWith
	fake.mu.Unlock()
	if sig == nil {
		t.Error("context cancel: ProcessSignal was not called on the child process")
	}
	if sig != os.Interrupt {
		t.Errorf("context cancel: want os.Interrupt signal, got %v", sig)
	}
}

// ---------------------------------------------------------------------------
// AC-20: Empty prompt — ErrInvalidArgs, no process spawned, lookPathFunc not called
// ---------------------------------------------------------------------------

func TestOpenCodeRunner_Invoke_EmptyPrompt(t *testing.T) {
	spawned := false
	factory := func(_ context.Context, _ string, _ ...string) execCmd {
		spawned = true
		return newOpenCodeFakeExecCmd(nil, nil, 0, nil)
	}
	lookPathCalled := false
	lookPath := func(_ string) (string, error) {
		lookPathCalled = true
		return "/usr/bin/opencode", nil
	}
	r := newOpenCodeRunnerWithFactory(t, factory, lookPath)

	req := RunRequest{Prompt: ""}
	_, err := r.Invoke(context.Background(), req)
	if err == nil {
		t.Fatal("Invoke with empty prompt: want error, got nil")
	}
	if !errors.Is(err, ErrInvalidArgs) {
		t.Errorf("Invoke with empty prompt: want errors.Is(ErrInvalidArgs), got %v", err)
	}
	if spawned {
		t.Error("Invoke with empty prompt: no process should be spawned")
	}
	if lookPathCalled {
		t.Error("Invoke with empty prompt: lookPathFunc must NOT be called (fast-fail on empty prompt)")
	}
}

// ---------------------------------------------------------------------------
// AC-21 / EC-1: Binary not found via Invoke path — ErrBinaryNotFound, no process spawned
// ---------------------------------------------------------------------------

func TestOpenCodeRunner_Invoke_BinaryNotFound(t *testing.T) {
	// EC-1: binary missing → ErrBinaryNotFound, no process spawned.
	spawned := false
	factory := func(_ context.Context, _ string, _ ...string) execCmd {
		spawned = true
		return newOpenCodeFakeExecCmd(nil, nil, 0, nil)
	}
	r := newOpenCodeRunnerWithFactory(t, factory, openCodeLookPathMissing)

	req := RunRequest{Prompt: "hello"}
	_, err := r.Invoke(context.Background(), req)
	if err == nil {
		t.Fatal("Invoke with missing binary: want error, got nil")
	}
	if !errors.Is(err, ErrBinaryNotFound) {
		t.Errorf("Invoke with missing binary: want errors.Is(ErrBinaryNotFound), got %v", err)
	}
	if spawned {
		t.Error("Invoke with missing binary: no process should be spawned")
	}
}

// ---------------------------------------------------------------------------
// Additional: NewOpenCodeRunner wires defaults
// ---------------------------------------------------------------------------

func TestNewOpenCodeRunner_Defaults(t *testing.T) {
	r := NewOpenCodeRunner()
	if r == nil {
		t.Fatal("NewOpenCodeRunner() returned nil")
	}
	if r.execFactory == nil {
		t.Error("NewOpenCodeRunner(): execFactory must not be nil")
	}
	if r.lookPathFunc == nil {
		t.Error("NewOpenCodeRunner(): lookPathFunc must not be nil")
	}
}

// ---------------------------------------------------------------------------
// Additional: session_id omitted from ParsedJSON when empty
// ---------------------------------------------------------------------------

func TestOpenCodeRunner_Invoke_ParsesNDJSON_EmptySessionID(t *testing.T) {
	// Events without sessionID field → session_id must be omitted from ParsedJSON.
	stream := []byte(
		`{"type":"text","timestamp":1,"part":{"text":"answer without session"}}` + "\n",
	)

	fake := newOpenCodeFakeExecCmd(stream, nil, 0, nil)
	factory, _ := buildOpenCodeFakeFactory(fake)
	r := newOpenCodeRunnerWithFactory(t, factory, openCodeLookPathFound)

	req := RunRequest{Prompt: "hello"}
	result, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}
	if result.ParsedJSON == nil {
		t.Fatal("ParsedJSON: want non-nil (text event found), got nil")
	}
	if result.ParsedJSON["result"] != "answer without session" {
		t.Errorf("ParsedJSON[\"result\"]: want %q, got %v", "answer without session", result.ParsedJSON["result"])
	}
	// session_id must be absent when empty.
	if _, hasSession := result.ParsedJSON["session_id"]; hasSession {
		t.Errorf("ParsedJSON[\"session_id\"]: must be omitted when sessionID is empty; map: %v", result.ParsedJSON)
	}
}

// ---------------------------------------------------------------------------
// Additional: prompt ordering — ExtraArgs before positional prompt
// ---------------------------------------------------------------------------

func TestOpenCodeRunner_Invoke_ExtraArgs_BeforePrompt(t *testing.T) {
	// Verify ExtraArgs come after adapter flags (--format json) but before the prompt.
	// This is the strict ordering contract: run --format json [ExtraArgs...] <prompt>
	fake := newOpenCodeFakeExecCmd(nil, nil, 0, nil)
	factory, capturedArgv := buildOpenCodeFakeFactory(fake)
	r := newOpenCodeRunnerWithFactory(t, factory, openCodeLookPathFound)

	req := RunRequest{
		Prompt:    "final prompt",
		ExtraArgs: []string{"--thinking", "--variant", "high"},
	}
	_, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}

	argv := *capturedArgv

	// Prompt must be absolutely last.
	if argv[len(argv)-1] != "final prompt" {
		t.Errorf("prompt must be absolutely last; argv: %v", argv)
	}

	// --format json must appear before ExtraArgs.
	formatIdx := indexOfToken(argv, "--format")
	thinkingIdx := indexOfToken(argv, "--thinking")
	if formatIdx == -1 {
		t.Fatalf("--format not found; argv: %v", argv)
	}
	if thinkingIdx == -1 {
		t.Fatalf("--thinking not found; argv: %v", argv)
	}
	if formatIdx >= thinkingIdx {
		t.Errorf("--format must appear before ExtraArgs; formatIdx=%d thinkingIdx=%d argv: %v", formatIdx, thinkingIdx, argv)
	}

	// Verify full structure: run is first token after binary name.
	if len(argv) < 2 || argv[1] != "run" {
		t.Errorf("argv[1] must be \"run\"; argv: %v", argv)
	}
}

// Ensure fmt is used (imported for openCodeNDJSONFixture-like helpers).
var _ = fmt.Sprintf
