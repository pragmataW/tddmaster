
package bridge

import (
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
