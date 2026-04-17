package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"testing"

	"github.com/pragmataW/tddmaster/internal/runner"
	"github.com/pragmataW/tddmaster/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// AC-1: Compile-time public API contract guard
// =============================================================================

// CallAgent must keep this exact signature forever.  A compile error here
// means the public contract has been broken — GREEN phase must not change it.
var _ func(string, string) (*BridgeResult, error) = CallAgent

// =============================================================================
// BridgeResult struct
// =============================================================================

func TestBridgeResult_Fields(t *testing.T) {
	r := BridgeResult{Text: "hello", Provider: "claude-cli"}
	assert.Equal(t, "hello", r.Text)
	assert.Equal(t, "claude-cli", r.Provider)
}

// =============================================================================
// callViaClaude — JSON parsing logic
// =============================================================================

// claudeOutput is used to parse JSON output from `claude -p`.
type claudeOutput struct {
	Result  string `json:"result"`
	Message *struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	} `json:"message"`
}

// parseClaudeOutput is a helper that exercises the same JSON parsing path
// used inside callViaClaude, allowing unit testing without spawning a process.
func parseClaudeOutput(raw string) string {
	var parsed claudeOutput
	if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
		text := parsed.Result
		if text == "" && parsed.Message != nil && len(parsed.Message.Content) > 0 {
			text = parsed.Message.Content[0].Text
		}
		if text == "" {
			text = raw
		}
		return text
	}
	return raw
}

func TestParseClaudeOutput_ResultField(t *testing.T) {
	raw := `{"result": "hello from claude"}`
	text := parseClaudeOutput(raw)
	assert.Equal(t, "hello from claude", text)
}

func TestParseClaudeOutput_MessageContentFallback(t *testing.T) {
	raw := `{"message": {"content": [{"text": "content text"}]}}`
	text := parseClaudeOutput(raw)
	assert.Equal(t, "content text", text)
}

func TestParseClaudeOutput_RawFallbackOnInvalidJSON(t *testing.T) {
	raw := "not json at all"
	text := parseClaudeOutput(raw)
	assert.Equal(t, "not json at all", text)
}

func TestParseClaudeOutput_EmptyResultFallsBackToRaw(t *testing.T) {
	// Valid JSON but result is empty and no message — fallback to raw
	raw := `{"result": ""}`
	text := parseClaudeOutput(raw)
	// result is "" so falls back to raw
	assert.Equal(t, raw, text)
}

func TestParseClaudeOutput_ResultTakesPrecedenceOverMessage(t *testing.T) {
	raw := `{"result": "from-result", "message": {"content": [{"text": "from-message"}]}}`
	text := parseClaudeOutput(raw)
	assert.Equal(t, "from-result", text)
}

// =============================================================================
// CallAgent — returns nil when no provider is available
// =============================================================================

func TestCallAgent_ReturnsNilWhenClaudeUnavailable(t *testing.T) {
	// On a machine without `claude` in PATH this call will fail to spawn and
	// fall through to the nil return. On a machine that has claude installed,
	// this test is skipped to avoid real network calls.
	_, err := exec.LookPath("claude")
	if err == nil {
		t.Skip("claude CLI is available; skipping nil-fallback test")
	}

	result, callErr := CallAgent("test prompt", "")
	require.NoError(t, callErr)
	assert.Nil(t, result)
}

// =============================================================================
// Task-8 migration tests
// =============================================================================
//
// These tests reference package-level seam variables that do NOT yet exist in
// bridge.go:
//
//   var (
//       bridgeRunnerSelect  = runner.Select
//       bridgeReadManifest  = state.ReadManifest
//       bridgeResolveRoot   = state.ResolveProjectRoot
//   )
//
// The file intentionally fails to compile until GREEN phase adds those seams.
// That is the RED phase.

// ---------------------------------------------------------------------------
// bridgeFakeRunner — test double implementing runner.Runner for bridge tests
// ---------------------------------------------------------------------------

type bridgeFakeRunner struct {
	name     string
	invokeFn func(ctx context.Context, req runner.RunRequest) (*runner.RunResult, error)

	// recorded invocation state
	lastCtx context.Context //nolint:containedctx
	lastReq runner.RunRequest
	invoked bool
}

func (f *bridgeFakeRunner) Name() string     { return f.name }
func (f *bridgeFakeRunner) Available() error { return nil }
func (f *bridgeFakeRunner) Invoke(ctx context.Context, req runner.RunRequest) (*runner.RunResult, error) {
	f.invoked = true
	f.lastCtx = ctx
	f.lastReq = req
	if f.invokeFn != nil {
		return f.invokeFn(ctx, req)
	}
	return &runner.RunResult{ExitCode: 0}, nil
}

// swapBridgeSeams replaces the package-level seam vars and restores them in
// t.Cleanup.  Pass nil for any fn to keep the current value unchanged.
func swapBridgeSeams(
	t *testing.T,
	selectFn func(*state.NosManifest, string) (runner.Runner, error),
	readManifestFn func(string) (*state.NosManifest, error),
	resolveRootFn func() (state.ResolveProjectRootResult, error),
) {
	t.Helper()

	origSelect := bridgeRunnerSelect
	origRead := bridgeReadManifest
	origRoot := bridgeResolveRoot

	if selectFn != nil {
		bridgeRunnerSelect = selectFn
	}
	if readManifestFn != nil {
		bridgeReadManifest = readManifestFn
	}
	if resolveRootFn != nil {
		bridgeResolveRoot = resolveRootFn
	}

	t.Cleanup(func() {
		bridgeRunnerSelect = origSelect
		bridgeReadManifest = origRead
		bridgeResolveRoot = origRoot
	})
}

// stubResolveRoot returns a helper that always resolves to a non-empty root
// without touching the file system.
func stubResolveRoot(root string) func() (state.ResolveProjectRootResult, error) {
	return func() (state.ResolveProjectRootResult, error) {
		return state.ResolveProjectRootResult{Root: root, Found: true}, nil
	}
}

// stubReadManifest returns a helper that returns a minimal NosManifest with a
// single "claude-code" tool entry.
func stubReadManifest(m *state.NosManifest) func(string) (*state.NosManifest, error) {
	return func(_ string) (*state.NosManifest, error) {
		return m, nil
	}
}

// minimalManifest builds a NosManifest with a single tools entry.
func minimalManifest(tool state.CodingToolId) *state.NosManifest {
	m := state.CreateInitialManifest([]string{}, []state.CodingToolId{tool}, state.ProjectTraits{})
	return &m
}

// ---------------------------------------------------------------------------
// AC-2: CallAgent delegates to runner.Invoke via the selected runner
// ---------------------------------------------------------------------------

func TestCallAgent_DelegatesToRunnerInvoke(t *testing.T) {
	fake := &bridgeFakeRunner{
		name: "claude-code",
		invokeFn: func(_ context.Context, _ runner.RunRequest) (*runner.RunResult, error) {
			return &runner.RunResult{
				ParsedJSON: map[string]any{"result": "hello"},
			}, nil
		},
	}

	swapBridgeSeams(t,
		func(_ *state.NosManifest, _ string) (runner.Runner, error) { return fake, nil },
		stubReadManifest(minimalManifest("claude-code")),
		stubResolveRoot(t.TempDir()),
	)

	result, err := CallAgent("test prompt", "")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, fake.invoked, "Invoke must be called exactly once")
	assert.Equal(t, "test prompt", fake.lastReq.Prompt)
	assert.Equal(t, 1, fake.lastReq.MaxTurns)
}

// ---------------------------------------------------------------------------
// AC-3: System prompt forwarded when non-empty
// ---------------------------------------------------------------------------

func TestCallAgent_SystemPromptForwarded(t *testing.T) {
	fake := &bridgeFakeRunner{
		name: "claude-code",
		invokeFn: func(_ context.Context, _ runner.RunRequest) (*runner.RunResult, error) {
			return &runner.RunResult{ParsedJSON: map[string]any{"result": "ok"}}, nil
		},
	}

	swapBridgeSeams(t,
		func(_ *state.NosManifest, _ string) (runner.Runner, error) { return fake, nil },
		stubReadManifest(minimalManifest("claude-code")),
		stubResolveRoot(t.TempDir()),
	)

	_, err := CallAgent("prompt", "you are a helper")

	require.NoError(t, err)
	assert.Equal(t, "you are a helper", fake.lastReq.SystemPrompt)
}

// ---------------------------------------------------------------------------
// AC-4: Empty system prompt — RunRequest.SystemPrompt is empty
// ---------------------------------------------------------------------------

func TestCallAgent_EmptySystemPrompt_NotSet(t *testing.T) {
	fake := &bridgeFakeRunner{
		name: "claude-code",
		invokeFn: func(_ context.Context, _ runner.RunRequest) (*runner.RunResult, error) {
			return &runner.RunResult{ParsedJSON: map[string]any{"result": "ok"}}, nil
		},
	}

	swapBridgeSeams(t,
		func(_ *state.NosManifest, _ string) (runner.Runner, error) { return fake, nil },
		stubReadManifest(minimalManifest("claude-code")),
		stubResolveRoot(t.TempDir()),
	)

	_, err := CallAgent("prompt", "")

	require.NoError(t, err)
	assert.Empty(t, fake.lastReq.SystemPrompt, "SystemPrompt must be empty when system arg is \"\"")
	// No extra args should carry a --system-prompt flag either.
	for _, arg := range fake.lastReq.ExtraArgs {
		assert.NotEqual(t, "--system-prompt", arg, "ExtraArgs must not inject --system-prompt when system is empty")
	}
}

// ---------------------------------------------------------------------------
// AC-5: OutputFormat == "json"
// ---------------------------------------------------------------------------

func TestCallAgent_OutputFormatIsJSON(t *testing.T) {
	fake := &bridgeFakeRunner{
		name: "claude-code",
		invokeFn: func(_ context.Context, _ runner.RunRequest) (*runner.RunResult, error) {
			return &runner.RunResult{ParsedJSON: map[string]any{"result": "ok"}}, nil
		},
	}

	swapBridgeSeams(t,
		func(_ *state.NosManifest, _ string) (runner.Runner, error) { return fake, nil },
		stubReadManifest(minimalManifest("claude-code")),
		stubResolveRoot(t.TempDir()),
	)

	_, err := CallAgent("prompt", "")

	require.NoError(t, err)
	assert.Equal(t, "json", fake.lastReq.OutputFormat)
}

// ---------------------------------------------------------------------------
// AC-6: Result parsing — "result" field; Provider == runner Name()
// EC-4 mapping
// ---------------------------------------------------------------------------

func TestCallAgent_ParsedJSON_ResultField_ProviderIsRunnerName(t *testing.T) {
	fake := &bridgeFakeRunner{
		name: "claude-code",
		invokeFn: func(_ context.Context, _ runner.RunRequest) (*runner.RunResult, error) {
			return &runner.RunResult{
				ParsedJSON: map[string]any{"result": "hello"},
			}, nil
		},
	}

	swapBridgeSeams(t,
		func(_ *state.NosManifest, _ string) (runner.Runner, error) { return fake, nil },
		stubReadManifest(minimalManifest("claude-code")),
		stubResolveRoot(t.TempDir()),
	)

	result, err := CallAgent("prompt", "")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "hello", result.Text)
	// Provider now reflects the actual runner used, NOT the hardcoded "claude-cli".
	assert.Equal(t, "claude-code", result.Provider)
}

// ---------------------------------------------------------------------------
// AC-7: Result parsing — message.content[0].text fallback
// EC-4 mapping
// ---------------------------------------------------------------------------

func TestCallAgent_ParsedJSON_MessageContentFallback(t *testing.T) {
	fake := &bridgeFakeRunner{
		name: "claude-code",
		invokeFn: func(_ context.Context, _ runner.RunRequest) (*runner.RunResult, error) {
			return &runner.RunResult{
				ParsedJSON: map[string]any{
					"message": map[string]any{
						"content": []any{
							map[string]any{"text": "msg text"},
						},
					},
				},
			}, nil
		},
	}

	swapBridgeSeams(t,
		func(_ *state.NosManifest, _ string) (runner.Runner, error) { return fake, nil },
		stubReadManifest(minimalManifest("claude-code")),
		stubResolveRoot(t.TempDir()),
	)

	result, err := CallAgent("prompt", "")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "msg text", result.Text)
}

// ---------------------------------------------------------------------------
// AC-8: Raw text fallback when ParsedJSON is nil (EC-4)
// ---------------------------------------------------------------------------

func TestCallAgent_RawTextFallback_WhenParsedJSONNil(t *testing.T) {
	fake := &bridgeFakeRunner{
		name: "claude-code",
		invokeFn: func(_ context.Context, _ runner.RunRequest) (*runner.RunResult, error) {
			return &runner.RunResult{
				ParsedJSON: nil,
				Stdout:     []byte("plain text"),
			}, nil
		},
	}

	swapBridgeSeams(t,
		func(_ *state.NosManifest, _ string) (runner.Runner, error) { return fake, nil },
		stubReadManifest(minimalManifest("claude-code")),
		stubResolveRoot(t.TempDir()),
	)

	result, err := CallAgent("prompt", "")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "plain text", result.Text)
	assert.Equal(t, "claude-code", result.Provider)
}

// ---------------------------------------------------------------------------
// AC-9: Empty Stdout and nil ParsedJSON → (nil, nil)
// ---------------------------------------------------------------------------

func TestCallAgent_EmptyStdoutAndNilParsedJSON_ReturnsNilNil(t *testing.T) {
	fake := &bridgeFakeRunner{
		name: "claude-code",
		invokeFn: func(_ context.Context, _ runner.RunRequest) (*runner.RunResult, error) {
			return &runner.RunResult{
				ParsedJSON: nil,
				Stdout:     []byte(""),
			}, nil
		},
	}

	swapBridgeSeams(t,
		func(_ *state.NosManifest, _ string) (runner.Runner, error) { return fake, nil },
		stubReadManifest(minimalManifest("claude-code")),
		stubResolveRoot(t.TempDir()),
	)

	result, err := CallAgent("prompt", "")

	require.NoError(t, err)
	assert.Nil(t, result, "must return (nil, nil) when there is nothing to parse")
}

// ---------------------------------------------------------------------------
// AC-10: Invoke returns ErrBinaryNotFound → graceful (nil, nil) (EC-1)
// ---------------------------------------------------------------------------

func TestCallAgent_InvokeReturnsBinaryNotFound_ReturnsNilNil(t *testing.T) {
	fake := &bridgeFakeRunner{
		name: "claude-code",
		invokeFn: func(_ context.Context, _ runner.RunRequest) (*runner.RunResult, error) {
			return nil, fmt.Errorf("%w: claude", runner.ErrBinaryNotFound)
		},
	}

	swapBridgeSeams(t,
		func(_ *state.NosManifest, _ string) (runner.Runner, error) { return fake, nil },
		stubReadManifest(minimalManifest("claude-code")),
		stubResolveRoot(t.TempDir()),
	)

	result, err := CallAgent("prompt", "")

	require.NoError(t, err)
	assert.Nil(t, result, "ErrBinaryNotFound must cause graceful (nil, nil) return")
}

// ---------------------------------------------------------------------------
// AC-11: bridgeRunnerSelect returns ErrRunnerNotFound → graceful (nil, nil)
// ---------------------------------------------------------------------------

func TestCallAgent_RunnerSelectReturnsNotFound_ReturnsNilNil(t *testing.T) {
	swapBridgeSeams(t,
		func(_ *state.NosManifest, _ string) (runner.Runner, error) {
			return nil, fmt.Errorf("%w: no runner", runner.ErrRunnerNotFound)
		},
		stubReadManifest(minimalManifest("claude-code")),
		stubResolveRoot(t.TempDir()),
	)

	result, err := CallAgent("prompt", "")

	require.NoError(t, err)
	assert.Nil(t, result, "ErrRunnerNotFound must cause graceful (nil, nil) return")
}

// ---------------------------------------------------------------------------
// AC-12: Invoke returns generic error → graceful (nil, nil)
// ---------------------------------------------------------------------------

func TestCallAgent_InvokeReturnsGenericError_ReturnsNilNil(t *testing.T) {
	fake := &bridgeFakeRunner{
		name: "claude-code",
		invokeFn: func(_ context.Context, _ runner.RunRequest) (*runner.RunResult, error) {
			return nil, errors.New("some unexpected internal error")
		},
	}

	swapBridgeSeams(t,
		func(_ *state.NosManifest, _ string) (runner.Runner, error) { return fake, nil },
		stubReadManifest(minimalManifest("claude-code")),
		stubResolveRoot(t.TempDir()),
	)

	result, err := CallAgent("prompt", "")

	require.NoError(t, err)
	assert.Nil(t, result, "generic Invoke error must cause graceful (nil, nil) return")
}

// ---------------------------------------------------------------------------
// AC-13: Manifest load failure → graceful (nil, nil)
// ---------------------------------------------------------------------------

func TestCallAgent_ManifestLoadFailure_ReturnsNilNil(t *testing.T) {
	swapBridgeSeams(t,
		nil, // don't swap select — it won't be reached
		func(_ string) (*state.NosManifest, error) {
			return nil, errors.New("disk read error")
		},
		stubResolveRoot(t.TempDir()),
	)

	result, err := CallAgent("prompt", "")

	require.NoError(t, err)
	assert.Nil(t, result, "manifest load failure must cause graceful (nil, nil) return")
}

// ---------------------------------------------------------------------------
// AC-14: toolFlag is always "" at the bridge layer (manifest-driven only)
// ---------------------------------------------------------------------------

func TestCallAgent_ToolFlagIsAlwaysEmpty(t *testing.T) {
	var capturedToolFlag string

	fake := &bridgeFakeRunner{
		name: "claude-code",
		invokeFn: func(_ context.Context, _ runner.RunRequest) (*runner.RunResult, error) {
			return &runner.RunResult{ParsedJSON: map[string]any{"result": "ok"}}, nil
		},
	}

	swapBridgeSeams(t,
		func(m *state.NosManifest, toolFlag string) (runner.Runner, error) {
			capturedToolFlag = toolFlag
			return fake, nil
		},
		stubReadManifest(minimalManifest("claude-code")),
		stubResolveRoot(t.TempDir()),
	)

	_, err := CallAgent("prompt", "")

	require.NoError(t, err)
	assert.Equal(t, "", capturedToolFlag,
		"bridge never passes a toolFlag — selection must be purely manifest-driven")
}

// ---------------------------------------------------------------------------
// AC-15: Context threading — ctx passed to Invoke has a non-nil Done channel (EC-6)
// ---------------------------------------------------------------------------

func TestCallAgent_ContextThreadedIntoInvoke_HasDoneChannel(t *testing.T) {
	var capturedCtx context.Context

	fake := &bridgeFakeRunner{
		name: "claude-code",
		invokeFn: func(ctx context.Context, _ runner.RunRequest) (*runner.RunResult, error) {
			capturedCtx = ctx
			return &runner.RunResult{ParsedJSON: map[string]any{"result": "ok"}}, nil
		},
	}

	swapBridgeSeams(t,
		func(_ *state.NosManifest, _ string) (runner.Runner, error) { return fake, nil },
		stubReadManifest(minimalManifest("claude-code")),
		stubResolveRoot(t.TempDir()),
	)

	_, err := CallAgent("prompt", "")

	require.NoError(t, err)
	require.NotNil(t, capturedCtx, "Invoke must receive a non-nil context")
	assert.NotNil(t, capturedCtx.Done(),
		"ctx passed to Invoke must be cancelable (Done != nil) for SIGINT/EC-6 propagation")
}
