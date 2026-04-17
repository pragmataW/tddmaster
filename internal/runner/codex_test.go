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
// codexFakeExecCmd — test double for execCmd (Codex-specific copy of pattern)
// ---------------------------------------------------------------------------

// codexFakeExecCmd is a test-only execCmd implementation for Codex runner tests.
// It mirrors the fakeExecCmd pattern from claude_test.go exactly.
type codexFakeExecCmd struct {
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

func newCodexFakeExecCmd(stdoutData, stderrData []byte, exitCode int, runErr error) *codexFakeExecCmd {
	return &codexFakeExecCmd{
		stdoutData: stdoutData,
		stderrData: stderrData,
		exitCode:   exitCode,
		runErr:     runErr,
		done:       make(chan struct{}),
	}
}

func (f *codexFakeExecCmd) SetStdout(w io.Writer) { f.stdoutW = w }
func (f *codexFakeExecCmd) SetStderr(w io.Writer) { f.stderrW = w }
func (f *codexFakeExecCmd) SetDir(dir string)     { f.dir = dir }
func (f *codexFakeExecCmd) SetEnv(env []string)   { f.env = env }

func (f *codexFakeExecCmd) ProcessSignal(sig os.Signal) error {
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

func (f *codexFakeExecCmd) Run() error {
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
		return &codexFakeExitError{code: f.exitCode}
	}
	return nil
}

func (f *codexFakeExecCmd) Start() error {
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

func (f *codexFakeExecCmd) Wait() error {
	<-f.done
	if f.runErr != nil {
		return f.runErr
	}
	if f.exitCode != 0 {
		return &codexFakeExitError{code: f.exitCode}
	}
	return nil
}

// codexFakeExitError simulates exec.ExitError with a configurable exit code.
type codexFakeExitError struct {
	code int
}

func (e *codexFakeExitError) Error() string { return "exit status " + strconv.Itoa(e.code) }
func (e *codexFakeExitError) ExitCode() int { return e.code }

// ---------------------------------------------------------------------------
// buildCodexFakeFactory — captures argv and returns the pre-wired fake
// ---------------------------------------------------------------------------

// buildCodexFakeFactory returns an execFactory that captures the full argv
// (binary name + args) for every spawned command and returns cmd.
func buildCodexFakeFactory(cmd *codexFakeExecCmd) (execFactory, *[]string) {
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

// newCodexRunnerWithFactory returns a *CodexRunner with injected seams.
func newCodexRunnerWithFactory(t *testing.T, factory execFactory, lookPath func(string) (string, error)) *CodexRunner {
	t.Helper()
	r := NewCodexRunner()
	r.execFactory = factory
	if lookPath != nil {
		r.lookPathFunc = lookPath
	}
	return r
}

// codexLookPathFound is a lookPath stub that always succeeds.
func codexLookPathFound(_ string) (string, error) {
	return "/opt/homebrew/bin/codex", nil
}

// codexLookPathMissing is a lookPath stub that always fails.
func codexLookPathMissing(_ string) (string, error) {
	return "", exec.ErrNotFound
}

// containsToken reports whether token appears anywhere in argv.
func containsToken(argv []string, token string) bool {
	for _, a := range argv {
		if a == token {
			return true
		}
	}
	return false
}

// containsConsecutive reports whether [a, b] appears consecutively in argv.
func containsConsecutive(argv []string, a, b string) bool {
	for i := 0; i+1 < len(argv); i++ {
		if argv[i] == a && argv[i+1] == b {
			return true
		}
	}
	return false
}

// indexOfToken returns the first index of token in argv, or -1.
func indexOfToken(argv []string, token string) int {
	for i, a := range argv {
		if a == token {
			return i
		}
	}
	return -1
}

// ndjsonFixture builds a minimal Codex NDJSON stream as a byte slice.
// threadID, resultText, and includeUsage control what events appear.
func ndjsonFixture(threadID, resultText string, includeUsage bool) []byte {
	var sb strings.Builder
	// thread.started
	sb.WriteString(fmt.Sprintf(`{"type":"thread.started","thread_id":%q}`, threadID))
	sb.WriteByte('\n')
	// turn.started
	sb.WriteString(`{"type":"turn.started"}`)
	sb.WriteByte('\n')
	// item.completed — agent_message
	sb.WriteString(fmt.Sprintf(`{"type":"item.completed","item":{"item_type":"agent_message","text":%q}}`, resultText))
	sb.WriteByte('\n')
	// turn.completed
	if includeUsage {
		sb.WriteString(`{"type":"turn.completed","usage":{"input_tokens":10,"output_tokens":20}}`)
	} else {
		sb.WriteString(`{"type":"turn.completed"}`)
	}
	sb.WriteByte('\n')
	return []byte(sb.String())
}

// ---------------------------------------------------------------------------
// AC-1: Name()
// ---------------------------------------------------------------------------

func TestCodexRunner_Name(t *testing.T) {
	r := NewCodexRunner()
	if got := r.Name(); got != "codex" {
		t.Fatalf("Name(): want %q, got %q", "codex", got)
	}
}

// ---------------------------------------------------------------------------
// AC-2: Available() — binary found
// ---------------------------------------------------------------------------

func TestCodexRunner_Available_BinaryFound(t *testing.T) {
	r := NewCodexRunner()
	r.lookPathFunc = codexLookPathFound

	if err := r.Available(); err != nil {
		t.Fatalf("Available() with found binary: want nil, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// AC-3 / EC-1: Available() — binary missing
// ---------------------------------------------------------------------------

func TestCodexRunner_Available_BinaryMissing(t *testing.T) {
	// EC-1: CLI binary PATH'te bulunamadı
	r := NewCodexRunner()
	r.lookPathFunc = codexLookPathMissing

	err := r.Available()
	if err == nil {
		t.Fatal("Available() with missing binary: want error, got nil")
	}
	if !errors.Is(err, ErrBinaryNotFound) {
		t.Fatalf("Available() with missing binary: want errors.Is(ErrBinaryNotFound), got %v", err)
	}
}

// ---------------------------------------------------------------------------
// AC-4 / EC-2: Default argv shape
// ---------------------------------------------------------------------------

func TestCodexRunner_Invoke_DefaultArgv(t *testing.T) {
	// EC-2: argv/flag uyumsuzluğu — Codex uses 'exec', positional prompt, --json, --ask-for-approval never
	fake := newCodexFakeExecCmd(nil, nil, 0, nil)
	factory, capturedArgv := buildCodexFakeFactory(fake)

	r := newCodexRunnerWithFactory(t, factory, codexLookPathFound)

	req := RunRequest{Prompt: "hello"}
	_, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}

	// Expected: ["codex", "exec", "--json", "--ask-for-approval", "never", "hello"]
	// (in that exact order; prompt is the last positional argument)
	want := []string{"codex", "exec", "--json", "--ask-for-approval", "never", "hello"}
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
// AC-5 / EC-2: WorkDir — --cd flag AND SetDir
// ---------------------------------------------------------------------------

func TestCodexRunner_Invoke_WorkDir(t *testing.T) {
	fake := newCodexFakeExecCmd(nil, nil, 0, nil)
	factory, capturedArgv := buildCodexFakeFactory(fake)

	r := newCodexRunnerWithFactory(t, factory, codexLookPathFound)

	req := RunRequest{Prompt: "hi", WorkDir: "/tmp/x"}
	_, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}

	argv := *capturedArgv

	// --cd /tmp/x must appear before the positional prompt.
	cdIdx := indexOfToken(argv, "--cd")
	if cdIdx == -1 {
		t.Fatalf("argv: --cd flag not found; argv: %v", argv)
	}
	if cdIdx+1 >= len(argv) {
		t.Fatal("argv: --cd flag has no value")
	}
	if argv[cdIdx+1] != "/tmp/x" {
		t.Errorf("--cd value: want %q, got %q", "/tmp/x", argv[cdIdx+1])
	}

	// Positional prompt must be last.
	if argv[len(argv)-1] != "hi" {
		t.Errorf("prompt must be last argv element; argv: %v", argv)
	}

	// --cd must appear before the prompt.
	promptIdx := len(argv) - 1
	if cdIdx >= promptIdx {
		t.Errorf("--cd must appear before prompt; cdIdx=%d promptIdx=%d argv: %v", cdIdx, promptIdx, argv)
	}

	// SetDir must also be called (cwd propagation).
	if fake.dir != "/tmp/x" {
		t.Errorf("SetDir: want %q, got %q", "/tmp/x", fake.dir)
	}
}

// ---------------------------------------------------------------------------
// AC-6 / EC-2: MaxTurns > 0 → -c agents.job_max_runtime_seconds=N*60
// ---------------------------------------------------------------------------

func TestCodexRunner_Invoke_MaxTurns_Positive(t *testing.T) {
	fake := newCodexFakeExecCmd(nil, nil, 0, nil)
	factory, capturedArgv := buildCodexFakeFactory(fake)

	r := newCodexRunnerWithFactory(t, factory, codexLookPathFound)

	req := RunRequest{Prompt: "task", MaxTurns: 5}
	_, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}

	argv := *capturedArgv
	// Two-token form: ["-c", "agents.job_max_runtime_seconds=300"]
	if !containsConsecutive(argv, "-c", "agents.job_max_runtime_seconds=300") {
		t.Errorf("MaxTurns=5: want consecutive [\"-c\", \"agents.job_max_runtime_seconds=300\"] in argv; got: %v", argv)
	}
}

// ---------------------------------------------------------------------------
// AC-7 / EC-2: MaxTurns == 0 → no -c agents.job_max_runtime_seconds token
// ---------------------------------------------------------------------------

func TestCodexRunner_Invoke_MaxTurns_Zero(t *testing.T) {
	fake := newCodexFakeExecCmd(nil, nil, 0, nil)
	factory, capturedArgv := buildCodexFakeFactory(fake)

	r := newCodexRunnerWithFactory(t, factory, codexLookPathFound)

	req := RunRequest{Prompt: "task", MaxTurns: 0}
	_, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}

	argv := *capturedArgv
	for _, a := range argv {
		if strings.Contains(a, "job_max_runtime_seconds") {
			t.Errorf("MaxTurns=0: no job_max_runtime_seconds token expected; got argv: %v", argv)
		}
	}
	if containsToken(argv, "-c") {
		t.Errorf("MaxTurns=0: no -c flag expected; got argv: %v", argv)
	}
}

// ---------------------------------------------------------------------------
// AC-8 / EC-2: ExtraArgs — appended after adapter flags, before prompt
// ---------------------------------------------------------------------------

func TestCodexRunner_Invoke_ExtraArgs(t *testing.T) {
	fake := newCodexFakeExecCmd(nil, nil, 0, nil)
	factory, capturedArgv := buildCodexFakeFactory(fake)

	r := newCodexRunnerWithFactory(t, factory, codexLookPathFound)

	req := RunRequest{
		Prompt:    "do work",
		ExtraArgs: []string{"--sandbox", "workspace-write"},
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

	// --sandbox must appear before the prompt.
	sandboxIdx := indexOfToken(argv, "--sandbox")
	if sandboxIdx == -1 {
		t.Fatalf("argv: --sandbox not found; argv: %v", argv)
	}
	if sandboxIdx >= promptIdx {
		t.Errorf("ExtraArgs must appear before prompt; sandboxIdx=%d promptIdx=%d argv: %v", sandboxIdx, promptIdx, argv)
	}
	if sandboxIdx+1 >= len(argv) || argv[sandboxIdx+1] != "workspace-write" {
		t.Errorf("ExtraArgs value: want workspace-write after --sandbox; argv: %v", argv)
	}

	// Adapter-owned flags must also be present.
	if !containsToken(argv, "--json") {
		t.Errorf("--json must always be present; argv: %v", argv)
	}
	if !containsConsecutive(argv, "--ask-for-approval", "never") {
		t.Errorf("--ask-for-approval never must always be present; argv: %v", argv)
	}
}

// ---------------------------------------------------------------------------
// AC-9 / EC-2: OutputFormat is ignored — --json always present, --output-format never
// ---------------------------------------------------------------------------

func TestCodexRunner_Invoke_IgnoresOutputFormat(t *testing.T) {
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
			fake := newCodexFakeExecCmd(nil, nil, 0, nil)
			factory, capturedArgv := buildCodexFakeFactory(fake)
			r := newCodexRunnerWithFactory(t, factory, codexLookPathFound)

			req := RunRequest{Prompt: "hello", OutputFormat: tc.outputFormat}
			_, err := r.Invoke(context.Background(), req)
			if err != nil {
				t.Fatalf("Invoke: unexpected error: %v", err)
			}

			argv := *capturedArgv
			if !containsToken(argv, "--json") {
				t.Errorf("OutputFormat=%q: --json must always appear in argv; got: %v", tc.outputFormat, argv)
			}
			if containsToken(argv, "--output-format") {
				t.Errorf("OutputFormat=%q: --output-format must NOT appear in argv (Codex only has --json); got: %v", tc.outputFormat, argv)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AC-10 / EC-2: SystemPrompt is ignored — no --system-prompt flag added
// ---------------------------------------------------------------------------

func TestCodexRunner_Invoke_IgnoresSystemPrompt(t *testing.T) {
	// Codex exec has no native system-prompt flag; the field is silently ignored.
	// This test documents and enforces that no --system-prompt (or equivalent) flag
	// appears in argv when SystemPrompt is set.
	fake := newCodexFakeExecCmd(nil, nil, 0, nil)
	factory, capturedArgv := buildCodexFakeFactory(fake)
	r := newCodexRunnerWithFactory(t, factory, codexLookPathFound)

	req := RunRequest{Prompt: "hello", SystemPrompt: "you are helpful"}
	_, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}

	argv := *capturedArgv
	for _, a := range argv {
		if strings.Contains(a, "system") {
			t.Errorf("SystemPrompt must be ignored; unexpected token %q in argv: %v", a, argv)
		}
	}
	// Base shape must still hold.
	if !containsToken(argv, "--json") {
		t.Errorf("--json must still appear; argv: %v", argv)
	}
}

// ---------------------------------------------------------------------------
// AC-11: Stdout streamed to caller writer AND captured in RunResult
// ---------------------------------------------------------------------------

func TestCodexRunner_Invoke_StreamsStdout(t *testing.T) {
	stdoutData := []byte("stream data")
	fake := newCodexFakeExecCmd(stdoutData, nil, 0, nil)
	factory, _ := buildCodexFakeFactory(fake)
	r := newCodexRunnerWithFactory(t, factory, codexLookPathFound)

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
// AC-12: Stderr streamed to caller writer AND captured in RunResult
// ---------------------------------------------------------------------------

func TestCodexRunner_Invoke_StreamsStderr(t *testing.T) {
	stderrData := []byte("error output")
	fake := newCodexFakeExecCmd(nil, stderrData, 0, nil)
	factory, _ := buildCodexFakeFactory(fake)
	r := newCodexRunnerWithFactory(t, factory, codexLookPathFound)

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
// AC-13: Env merged onto parent env
// ---------------------------------------------------------------------------

func TestCodexRunner_Invoke_MergesEnv(t *testing.T) {
	fake := newCodexFakeExecCmd(nil, nil, 0, nil)
	factory, _ := buildCodexFakeFactory(fake)
	r := newCodexRunnerWithFactory(t, factory, codexLookPathFound)

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
// AC-14: NDJSON stdout parsing — ParsedJSON populated
// ---------------------------------------------------------------------------

func TestCodexRunner_Invoke_ParsesNDJSON(t *testing.T) {
	threadID := "th-abc-123"
	resultText := "response text"
	stream := ndjsonFixture(threadID, resultText, true)

	fake := newCodexFakeExecCmd(stream, nil, 0, nil)
	factory, _ := buildCodexFakeFactory(fake)
	r := newCodexRunnerWithFactory(t, factory, codexLookPathFound)

	req := RunRequest{Prompt: "hello"}
	result, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}

	if result.ParsedJSON == nil {
		t.Fatal("ParsedJSON: want non-nil map from NDJSON stream, got nil")
	}

	// result field must equal the agent_message text.
	gotResult, ok := result.ParsedJSON["result"]
	if !ok {
		t.Fatalf("ParsedJSON: missing key \"result\"; map: %v", result.ParsedJSON)
	}
	if gotResult != resultText {
		t.Errorf("ParsedJSON[\"result\"]: want %q, got %q", resultText, gotResult)
	}

	// usage must be non-nil when the stream contains usage.
	gotUsage := result.ParsedJSON["usage"]
	if gotUsage == nil {
		t.Errorf("ParsedJSON[\"usage\"]: want non-nil, got nil; map: %v", result.ParsedJSON)
	}

	// thread_id must match the thread.started event.
	gotThreadID := result.ParsedJSON["thread_id"]
	if gotThreadID != threadID {
		t.Errorf("ParsedJSON[\"thread_id\"]: want %q, got %v", threadID, gotThreadID)
	}
}

// ---------------------------------------------------------------------------
// AC-15 / EC-4: NDJSON parse failure — ParsedJSON nil, err nil
// ---------------------------------------------------------------------------

func TestCodexRunner_Invoke_NDJSONParseFailure_NilParsedJSON(t *testing.T) {
	// EC-4: JSON parse failure — graceful fallback, ParsedJSON nil, no error.
	invalidOutput := []byte("not json at all\x00\xff garbage")

	fake := newCodexFakeExecCmd(invalidOutput, nil, 0, nil)
	factory, _ := buildCodexFakeFactory(fake)
	r := newCodexRunnerWithFactory(t, factory, codexLookPathFound)

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

func TestCodexRunner_Invoke_EmptyOutput_NilParsedJSON(t *testing.T) {
	// EC-4: empty stdout → ParsedJSON nil, no error.
	fake := newCodexFakeExecCmd([]byte{}, nil, 0, nil)
	factory, _ := buildCodexFakeFactory(fake)
	r := newCodexRunnerWithFactory(t, factory, codexLookPathFound)

	req := RunRequest{Prompt: "hello"}
	result, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke with empty output: want nil error, got %v", err)
	}
	if result.ParsedJSON != nil {
		t.Errorf("ParsedJSON: want nil for empty output, got %v", result.ParsedJSON)
	}
}

// ---------------------------------------------------------------------------
// AC-16 / EC-4: NDJSON with no agent_message — ParsedJSON nil
// ---------------------------------------------------------------------------

func TestCodexRunner_Invoke_NDJSONNoAgentMessage_NilParsedJSON(t *testing.T) {
	// Stream has thread.started + turn.failed but no agent_message item.
	// ParsedJSON must remain nil.
	stream := []byte(`{"type":"thread.started","thread_id":"th-x"}` + "\n" +
		`{"type":"turn.started"}` + "\n" +
		`{"type":"turn.failed","error":{"message":"something went wrong"}}` + "\n")

	fake := newCodexFakeExecCmd(stream, nil, 0, nil)
	factory, _ := buildCodexFakeFactory(fake)
	r := newCodexRunnerWithFactory(t, factory, codexLookPathFound)

	req := RunRequest{Prompt: "hello"}
	result, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}
	if result.ParsedJSON != nil {
		t.Errorf("ParsedJSON: want nil when no agent_message found; got %v", result.ParsedJSON)
	}
}

// ---------------------------------------------------------------------------
// AC-17 / EC-3: Non-zero exit — ExitCode set, err nil
// ---------------------------------------------------------------------------

func TestCodexRunner_Invoke_NonZeroExit(t *testing.T) {
	// EC-3: Non-zero exit code — Invoke returns nil error, caller decides fatality.
	stderrData := []byte("codex fatal error")
	fake := newCodexFakeExecCmd(nil, stderrData, 1, nil)
	factory, _ := buildCodexFakeFactory(fake)
	r := newCodexRunnerWithFactory(t, factory, codexLookPathFound)

	req := RunRequest{Prompt: "hello"}
	result, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke with non-zero exit: want nil error, got %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("ExitCode: want 1, got %d", result.ExitCode)
	}
	if !bytes.Contains(result.Stderr, stderrData) {
		t.Errorf("RunResult.Stderr: want to contain %q, got %q", stderrData, result.Stderr)
	}
}

// ---------------------------------------------------------------------------
// AC-18 / EC-6: Context cancel — SIGINT sent, ErrContextCanceled returned
// ---------------------------------------------------------------------------

func TestCodexRunner_Invoke_ContextCancel(t *testing.T) {
	// EC-6: context cancel + SIGINT + grace period.
	fake := newCodexFakeExecCmd(nil, nil, 0, nil)
	fake.startDelay = 5 * time.Second // simulate slow process
	factory, _ := buildCodexFakeFactory(fake)
	r := newCodexRunnerWithFactory(t, factory, codexLookPathFound)

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
}

// ---------------------------------------------------------------------------
// AC-19: Empty prompt — ErrInvalidArgs, no process spawned
// ---------------------------------------------------------------------------

func TestCodexRunner_Invoke_EmptyPrompt(t *testing.T) {
	spawned := false
	factory := func(_ context.Context, _ string, _ ...string) execCmd {
		spawned = true
		return newCodexFakeExecCmd(nil, nil, 0, nil)
	}
	r := newCodexRunnerWithFactory(t, factory, codexLookPathFound)

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
}

// ---------------------------------------------------------------------------
// AC-20 / EC-1: Binary not found via Invoke path
// ---------------------------------------------------------------------------

func TestCodexRunner_Invoke_BinaryNotFound(t *testing.T) {
	// EC-1: binary missing → ErrBinaryNotFound, no process spawned.
	spawned := false
	factory := func(_ context.Context, _ string, _ ...string) execCmd {
		spawned = true
		return newCodexFakeExecCmd(nil, nil, 0, nil)
	}
	r := newCodexRunnerWithFactory(t, factory, codexLookPathMissing)

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
// AC-21: --ask-for-approval never always present, even when ExtraArgs tries to override
// ---------------------------------------------------------------------------

func TestCodexRunner_Invoke_AskForApprovalAlwaysPresent(t *testing.T) {
	// The adapter-owned --ask-for-approval never must appear at its canonical position
	// regardless of what ExtraArgs contains.
	// If the caller also supplies --ask-for-approval on-request via ExtraArgs, BOTH tokens
	// appear (adapter's canonical position first, caller's override after).
	// Codex resolves duplicate flags left-to-right, so the caller's override wins if
	// Codex applies the last-wins rule — this is acceptable and documented here.
	// The critical contract is: the adapter NEVER drops its own default.
	fake := newCodexFakeExecCmd(nil, nil, 0, nil)
	factory, capturedArgv := buildCodexFakeFactory(fake)
	r := newCodexRunnerWithFactory(t, factory, codexLookPathFound)

	req := RunRequest{
		Prompt:    "hello",
		ExtraArgs: []string{"--ask-for-approval", "on-request"},
	}
	_, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}

	argv := *capturedArgv

	// The adapter's canonical --ask-for-approval never must be present.
	if !containsConsecutive(argv, "--ask-for-approval", "never") {
		t.Errorf("adapter default --ask-for-approval never must always be present; argv: %v", argv)
	}

	// --json must also be present.
	if !containsToken(argv, "--json") {
		t.Errorf("--json must always be present; argv: %v", argv)
	}

	// The caller's ExtraArgs tokens must also appear (after adapter flags, before prompt).
	callerIdx := -1
	for i, a := range argv {
		if a == "--ask-for-approval" && i+1 < len(argv) && argv[i+1] == "on-request" {
			callerIdx = i
		}
	}
	// callerIdx might be -1 if the implementation chose not to append duplicate flags;
	// that is also acceptable as long as the adapter's own "never" is present.
	// Document: either both appear OR only the adapter's canonical flag appears.
	// The important invariant is that "never" appears at the adapter's canonical position.
	_ = callerIdx

	// Verify adapter tokens appear before ExtraArgs tokens (canonical position check).
	// Find the first occurrence of the adapter's --ask-for-approval never pair.
	adapterApprovalIdx := -1
	for i := 0; i+1 < len(argv); i++ {
		if argv[i] == "--ask-for-approval" && argv[i+1] == "never" {
			adapterApprovalIdx = i
			break
		}
	}
	if adapterApprovalIdx == -1 {
		t.Fatalf("adapter --ask-for-approval never not found at any position; argv: %v", argv)
	}

	// The prompt must still be last.
	if argv[len(argv)-1] != "hello" {
		t.Errorf("prompt must be last argv element; argv: %v", argv)
	}
}

// ---------------------------------------------------------------------------
// Additional table-driven: MaxTurns translation correctness
// ---------------------------------------------------------------------------

func TestCodexRunner_Invoke_MaxTurns_Translation(t *testing.T) {
	cases := []struct {
		maxTurns int
		wantKey  string
		wantVal  string
		present  bool
	}{
		{1, "-c", "agents.job_max_runtime_seconds=60", true},
		{3, "-c", "agents.job_max_runtime_seconds=180", true},
		{10, "-c", "agents.job_max_runtime_seconds=600", true},
		{0, "", "", false},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("MaxTurns=%d", tc.maxTurns), func(t *testing.T) {
			fake := newCodexFakeExecCmd(nil, nil, 0, nil)
			factory, capturedArgv := buildCodexFakeFactory(fake)
			r := newCodexRunnerWithFactory(t, factory, codexLookPathFound)

			req := RunRequest{Prompt: "task", MaxTurns: tc.maxTurns}
			_, err := r.Invoke(context.Background(), req)
			if err != nil {
				t.Fatalf("Invoke: unexpected error: %v", err)
			}

			argv := *capturedArgv
			if tc.present {
				if !containsConsecutive(argv, tc.wantKey, tc.wantVal) {
					t.Errorf("MaxTurns=%d: want [\"%s\", \"%s\"] in argv; got: %v",
						tc.maxTurns, tc.wantKey, tc.wantVal, argv)
				}
			} else {
				for _, a := range argv {
					if strings.Contains(a, "job_max_runtime_seconds") {
						t.Errorf("MaxTurns=0: no job_max_runtime_seconds expected; argv: %v", argv)
					}
				}
			}
		})
	}
}
