package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// fakeExecCmd — test double for execCmd interface
// ---------------------------------------------------------------------------

// fakeExecCmd is a test-only implementation of the execCmd interface.
// Each test that needs process-level control constructs its own instance.
type fakeExecCmd struct {
	argv       []string
	stdoutW    io.Writer
	stderrW    io.Writer
	dir        string
	env        []string
	stdoutData []byte
	stderrData []byte
	exitCode   int
	runErr     error
	startDelay time.Duration // used by cancel test to simulate a slow process

	mu           sync.Mutex
	signaledWith os.Signal
	started      bool
	done         chan struct{} // closed when Run/Wait completes
}

func newFakeExecCmd(stdoutData, stderrData []byte, exitCode int, runErr error) *fakeExecCmd {
	return &fakeExecCmd{
		stdoutData: stdoutData,
		stderrData: stderrData,
		exitCode:   exitCode,
		runErr:     runErr,
		done:       make(chan struct{}),
	}
}

func (f *fakeExecCmd) SetStdout(w io.Writer) { f.stdoutW = w }
func (f *fakeExecCmd) SetStderr(w io.Writer) { f.stderrW = w }
func (f *fakeExecCmd) SetDir(dir string)     { f.dir = dir }
func (f *fakeExecCmd) SetEnv(env []string)   { f.env = env }

func (f *fakeExecCmd) ProcessSignal(sig os.Signal) error {
	f.mu.Lock()
	f.signaledWith = sig
	f.mu.Unlock()
	// Unblock any waiting Start+Wait pair
	select {
	case <-f.done:
	default:
		close(f.done)
	}
	return nil
}

func (f *fakeExecCmd) Run() error {
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
		return &fakeExitError{code: f.exitCode}
	}
	return nil
}

func (f *fakeExecCmd) Start() error {
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
			case <-f.done: // signaled early
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

func (f *fakeExecCmd) Wait() error {
	<-f.done
	if f.runErr != nil {
		return f.runErr
	}
	if f.exitCode != 0 {
		return &fakeExitError{code: f.exitCode}
	}
	return nil
}

// fakeExitError simulates exec.ExitError with a configurable exit code.
type fakeExitError struct {
	code int
}

func (e *fakeExitError) Error() string {
	return "exit status " + strconv.Itoa(e.code)
}

func (e *fakeExitError) ExitCode() int {
	return e.code
}

// ---------------------------------------------------------------------------
// fakeExecFactory — captures argv and returns a pre-wired fakeExecCmd
// ---------------------------------------------------------------------------

// buildFakeFactory returns an execFactory that captures the argv on each call
// and returns the given fakeExecCmd instance.
func buildFakeFactory(cmd *fakeExecCmd) (execFactory, *[]string) {
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

// newClaudeRunnerWithFactory returns a ClaudeRunner with injected seams.
func newClaudeRunnerWithFactory(t *testing.T, factory execFactory, lookPath func(string) (string, error)) *ClaudeRunner {
	t.Helper()
	r := NewClaudeRunner()
	r.execFactory = factory
	if lookPath != nil {
		r.lookPathFunc = lookPath
	}
	return r
}

// lookPathFound is a lookPath stub that always succeeds.
func lookPathFound(_ string) (string, error) {
	return "/usr/bin/claude", nil
}

// lookPathMissing is a lookPath stub that returns exec.ErrNotFound.
func lookPathMissing(_ string) (string, error) {
	return "", exec.ErrNotFound
}

// ---------------------------------------------------------------------------
// TestClaudeRunner_Name
// ---------------------------------------------------------------------------

func TestClaudeRunner_Name(t *testing.T) {
	r := NewClaudeRunner()
	if got := r.Name(); got != "claude-code" {
		t.Fatalf("Name(): want %q, got %q", "claude-code", got)
	}
}

// ---------------------------------------------------------------------------
// TestClaudeRunner_Available_BinaryFound
// ---------------------------------------------------------------------------

func TestClaudeRunner_Available_BinaryFound(t *testing.T) {
	r := NewClaudeRunner()
	r.lookPathFunc = lookPathFound

	if err := r.Available(); err != nil {
		t.Fatalf("Available() with found binary: want nil, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestClaudeRunner_Available_BinaryMissing — EC-1
// ---------------------------------------------------------------------------

func TestClaudeRunner_Available_BinaryMissing(t *testing.T) {
	r := NewClaudeRunner()
	r.lookPathFunc = lookPathMissing

	err := r.Available()
	if err == nil {
		t.Fatal("Available() with missing binary: want error, got nil")
	}
	if !errors.Is(err, ErrBinaryNotFound) {
		t.Fatalf("Available() with missing binary: want errors.Is(ErrBinaryNotFound), got %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestClaudeRunner_Invoke_BuildsCorrectArgv — EC-2
// ---------------------------------------------------------------------------

func TestClaudeRunner_Invoke_BuildsCorrectArgv(t *testing.T) {
	fake := newFakeExecCmd([]byte(`{"type":"result","result":"ok"}`), nil, 0, nil)
	factory, capturedArgv := buildFakeFactory(fake)

	r := newClaudeRunnerWithFactory(t, factory, lookPathFound)

	req := RunRequest{
		Prompt:       "hello",
		MaxTurns:     5,
		OutputFormat: "json",
	}
	_, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}

	want := []string{"claude", "-p", "hello", "--output-format", "json", "--max-turns", "5"}
	gotArgv := *capturedArgv
	if len(gotArgv) != len(want) {
		t.Fatalf("argv length: want %d, got %d\nwant: %v\ngot:  %v", len(want), len(gotArgv), want, gotArgv)
	}
	for i, w := range want {
		if gotArgv[i] != w {
			t.Errorf("argv[%d]: want %q, got %q", i, w, gotArgv[i])
		}
	}
}

// ---------------------------------------------------------------------------
// TestClaudeRunner_Invoke_DefaultMaxTurns
// ---------------------------------------------------------------------------

func TestClaudeRunner_Invoke_DefaultMaxTurns(t *testing.T) {
	fake := newFakeExecCmd([]byte(`{}`), nil, 0, nil)
	factory, capturedArgv := buildFakeFactory(fake)

	r := newClaudeRunnerWithFactory(t, factory, lookPathFound)

	req := RunRequest{
		Prompt:       "hello",
		MaxTurns:     0, // zero → adapter default
		OutputFormat: "json",
	}
	_, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}

	argv := *capturedArgv
	maxTurnsIdx := -1
	for i, arg := range argv {
		if arg == "--max-turns" {
			maxTurnsIdx = i
			break
		}
	}
	if maxTurnsIdx == -1 {
		t.Fatal("argv: --max-turns flag not found")
	}
	if maxTurnsIdx+1 >= len(argv) {
		t.Fatal("argv: --max-turns flag has no value")
	}
	if argv[maxTurnsIdx+1] != "10" {
		t.Errorf("default max-turns: want %q, got %q", "10", argv[maxTurnsIdx+1])
	}
}

// ---------------------------------------------------------------------------
// TestClaudeRunner_Invoke_DefaultOutputFormat
// ---------------------------------------------------------------------------

func TestClaudeRunner_Invoke_DefaultOutputFormat(t *testing.T) {
	fake := newFakeExecCmd([]byte(`{}`), nil, 0, nil)
	factory, capturedArgv := buildFakeFactory(fake)

	r := newClaudeRunnerWithFactory(t, factory, lookPathFound)

	req := RunRequest{
		Prompt:       "hello",
		OutputFormat: "", // empty → adapter default "json"
	}
	_, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}

	argv := *capturedArgv
	fmtIdx := -1
	for i, arg := range argv {
		if arg == "--output-format" {
			fmtIdx = i
			break
		}
	}
	if fmtIdx == -1 {
		t.Fatal("argv: --output-format flag not found")
	}
	if fmtIdx+1 >= len(argv) {
		t.Fatal("argv: --output-format flag has no value")
	}
	if argv[fmtIdx+1] != "json" {
		t.Errorf("default output-format: want %q, got %q", "json", argv[fmtIdx+1])
	}
}

// ---------------------------------------------------------------------------
// TestClaudeRunner_Invoke_AppendsExtraArgs
// ---------------------------------------------------------------------------

func TestClaudeRunner_Invoke_AppendsExtraArgs(t *testing.T) {
	fake := newFakeExecCmd([]byte(`{}`), nil, 0, nil)
	factory, capturedArgv := buildFakeFactory(fake)

	r := newClaudeRunnerWithFactory(t, factory, lookPathFound)

	req := RunRequest{
		Prompt:       "hello",
		OutputFormat: "json",
		MaxTurns:     1,
		ExtraArgs:    []string{"--custom-flag", "value"},
	}
	_, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}

	argv := *capturedArgv
	// Find the position of --output-format and --max-turns (adapter-owned) to ensure
	// ExtraArgs come after them.
	lastAdapterFlagIdx := -1
	for i, arg := range argv {
		if arg == "--output-format" || arg == "--max-turns" || arg == "-p" {
			if i > lastAdapterFlagIdx {
				lastAdapterFlagIdx = i + 1 // include the value
			}
		}
	}

	customFlagIdx := -1
	for i, arg := range argv {
		if arg == "--custom-flag" {
			customFlagIdx = i
			break
		}
	}
	if customFlagIdx == -1 {
		t.Fatal("argv: --custom-flag not found in argv")
	}
	if customFlagIdx <= lastAdapterFlagIdx {
		t.Errorf("ExtraArgs must appear AFTER adapter-owned flags; --custom-flag at index %d, last adapter flag value at index %d\nargv: %v", customFlagIdx, lastAdapterFlagIdx, argv)
	}
	if customFlagIdx+1 >= len(argv) || argv[customFlagIdx+1] != "value" {
		t.Errorf("ExtraArgs value: want %q after --custom-flag, got argv: %v", "value", argv)
	}
}

// ---------------------------------------------------------------------------
// TestClaudeRunner_Invoke_PassesWorkDir
// ---------------------------------------------------------------------------

func TestClaudeRunner_Invoke_PassesWorkDir(t *testing.T) {
	fake := newFakeExecCmd([]byte(`{}`), nil, 0, nil)
	factory, _ := buildFakeFactory(fake)

	r := newClaudeRunnerWithFactory(t, factory, lookPathFound)

	req := RunRequest{
		Prompt:  "hello",
		WorkDir: "/tmp",
	}
	_, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}
	if fake.dir != "/tmp" {
		t.Errorf("WorkDir: want %q, got %q", "/tmp", fake.dir)
	}
}

// ---------------------------------------------------------------------------
// TestClaudeRunner_Invoke_PassesEnv
// ---------------------------------------------------------------------------

func TestClaudeRunner_Invoke_PassesEnv(t *testing.T) {
	fake := newFakeExecCmd([]byte(`{}`), nil, 0, nil)
	factory, _ := buildFakeFactory(fake)

	r := newClaudeRunnerWithFactory(t, factory, lookPathFound)

	req := RunRequest{
		Prompt: "hello",
		Env:    []string{"FOO=bar"},
	}
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
		t.Errorf("Env: want FOO=bar in process env %v", fake.env)
	}
}

// ---------------------------------------------------------------------------
// TestClaudeRunner_Invoke_StreamsStdoutAndStderr
// ---------------------------------------------------------------------------

func TestClaudeRunner_Invoke_StreamsStdoutAndStderr(t *testing.T) {
	stdoutData := []byte("hello")
	stderrData := []byte("err")
	fake := newFakeExecCmd(stdoutData, stderrData, 0, nil)
	factory, _ := buildFakeFactory(fake)

	r := newClaudeRunnerWithFactory(t, factory, lookPathFound)

	callerStdout := &bytes.Buffer{}
	callerStderr := &bytes.Buffer{}
	req := RunRequest{
		Prompt: "hello",
		Stdout: callerStdout,
		Stderr: callerStderr,
	}
	result, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}

	// Caller's writers must receive the streamed bytes.
	if callerStdout.String() != "hello" {
		t.Errorf("caller stdout: want %q, got %q", "hello", callerStdout.String())
	}
	if callerStderr.String() != "err" {
		t.Errorf("caller stderr: want %q, got %q", "err", callerStderr.String())
	}

	// RunResult must also capture the bytes.
	if !bytes.Contains(result.Stdout, stdoutData) {
		t.Errorf("RunResult.Stdout: want to contain %q, got %q", stdoutData, result.Stdout)
	}
	if !bytes.Contains(result.Stderr, stderrData) {
		t.Errorf("RunResult.Stderr: want to contain %q, got %q", stderrData, result.Stderr)
	}
}

// ---------------------------------------------------------------------------
// TestClaudeRunner_Invoke_ParsesFinalJSON
// ---------------------------------------------------------------------------

func TestClaudeRunner_Invoke_ParsesFinalJSON(t *testing.T) {
	payload := []byte(`{"type":"result","result":"ok"}`)
	fake := newFakeExecCmd(payload, nil, 0, nil)
	factory, _ := buildFakeFactory(fake)

	r := newClaudeRunnerWithFactory(t, factory, lookPathFound)

	req := RunRequest{Prompt: "hello"}
	result, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}
	if result.ParsedJSON == nil {
		t.Fatal("ParsedJSON: want non-nil map, got nil")
	}

	// Verify round-trip: re-encode ParsedJSON and check key presence.
	var want map[string]any
	if jerr := json.Unmarshal(payload, &want); jerr != nil {
		t.Fatalf("test setup: cannot unmarshal payload: %v", jerr)
	}
	for k, wv := range want {
		gv, ok := result.ParsedJSON[k]
		if !ok {
			t.Errorf("ParsedJSON: missing key %q", k)
			continue
		}
		// Compare as JSON-marshaled strings to avoid interface{} type comparison pitfalls.
		wantStr := func(v any) string {
			b, _ := json.Marshal(v)
			return string(b)
		}
		if wantStr(wv) != wantStr(gv) {
			t.Errorf("ParsedJSON[%q]: want %v, got %v", k, wv, gv)
		}
	}
}

// ---------------------------------------------------------------------------
// TestClaudeRunner_Invoke_JSONParseFailure — EC-4
// ---------------------------------------------------------------------------

func TestClaudeRunner_Invoke_JSONParseFailure(t *testing.T) {
	rawOutput := []byte("plain text output")
	fake := newFakeExecCmd(rawOutput, nil, 0, nil)
	factory, _ := buildFakeFactory(fake)

	r := newClaudeRunnerWithFactory(t, factory, lookPathFound)

	req := RunRequest{Prompt: "hello"}
	result, err := r.Invoke(context.Background(), req)
	// Invoke must return nil error — graceful fallback.
	if err != nil {
		t.Fatalf("Invoke with non-JSON output: want nil error, got %v", err)
	}
	// ParsedJSON must be nil when JSON parse fails.
	if result.ParsedJSON != nil {
		t.Errorf("ParsedJSON: want nil on parse failure, got %v", result.ParsedJSON)
	}
	// Raw stdout bytes must still be captured.
	if !bytes.Contains(result.Stdout, rawOutput) {
		t.Errorf("RunResult.Stdout: want to contain %q, got %q", rawOutput, result.Stdout)
	}
}

// ---------------------------------------------------------------------------
// TestClaudeRunner_Invoke_NonZeroExit — EC-3
// ---------------------------------------------------------------------------

func TestClaudeRunner_Invoke_NonZeroExit(t *testing.T) {
	stderrData := []byte("boom")
	fake := newFakeExecCmd(nil, stderrData, 2, nil)
	factory, _ := buildFakeFactory(fake)

	r := newClaudeRunnerWithFactory(t, factory, lookPathFound)

	req := RunRequest{Prompt: "hello"}
	result, err := r.Invoke(context.Background(), req)
	// Per contract: Invoke returns nil error on non-zero exit — caller decides fatality.
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
// TestClaudeRunner_Invoke_ContextCancel — EC-6
// ---------------------------------------------------------------------------

func TestClaudeRunner_Invoke_ContextCancel(t *testing.T) {
	// Slow process that blocks for longer than the test timeout.
	fake := newFakeExecCmd(nil, nil, 0, nil)
	fake.startDelay = 5 * time.Second
	factory, _ := buildFakeFactory(fake)

	r := newClaudeRunnerWithFactory(t, factory, lookPathFound)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req := RunRequest{Prompt: "hello"}
	_, err := r.Invoke(ctx, req)

	// Invoke must return an error wrapping ErrContextCanceled.
	if err == nil {
		t.Fatal("Invoke with canceled context: want error, got nil")
	}
	if !errors.Is(err, ErrContextCanceled) {
		t.Errorf("Invoke with canceled context: want errors.Is(ErrContextCanceled), got %v", err)
	}

	// The fake must have received a signal (os.Interrupt or os.Kill).
	fake.mu.Lock()
	sig := fake.signaledWith
	fake.mu.Unlock()
	if sig == nil {
		t.Error("context cancel: ProcessSignal was not called on the child process")
	}
}

// ---------------------------------------------------------------------------
// TestClaudeRunner_Invoke_EmptyPrompt
// ---------------------------------------------------------------------------

func TestClaudeRunner_Invoke_EmptyPrompt(t *testing.T) {
	spawned := false
	factory := func(_ context.Context, _ string, _ ...string) execCmd {
		spawned = true
		return newFakeExecCmd(nil, nil, 0, nil)
	}

	r := newClaudeRunnerWithFactory(t, factory, lookPathFound)

	req := RunRequest{Prompt: ""} // empty prompt
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
}

// ---------------------------------------------------------------------------
// TestClaudeRunner_Invoke_BinaryNotFound — EC-1 via Invoke path
// ---------------------------------------------------------------------------

func TestClaudeRunner_Invoke_BinaryNotFound(t *testing.T) {
	spawned := false
	factory := func(_ context.Context, _ string, _ ...string) execCmd {
		spawned = true
		return newFakeExecCmd(nil, nil, 0, nil)
	}

	r := newClaudeRunnerWithFactory(t, factory, lookPathMissing)

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
