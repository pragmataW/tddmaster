
// Package bridge orchestrates AI calls for validation and spec generation.
//
// Tries claude CLI first (equivalent to @eser/ai registry with claude-code
// provider), falls back to manual (returns nil — caller handles).
package bridge

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strings"
)

// =============================================================================
// Bridge Result
// =============================================================================

// BridgeResult holds the text response and the provider that produced it.
type BridgeResult struct {
	Text     string
	Provider string
}

// =============================================================================
// Bridge
// =============================================================================

// CallAgent tries available AI providers in order and returns the first
// successful result. The fallback chain mirrors the TS implementation:
//
//  1. claude CLI (covers both the @eser/ai registry and raw CLI paths)
//  2. Returns nil — caller handles manual fallback.
func CallAgent(prompt string, system string) (*BridgeResult, error) {
	// Try claude CLI (covers @eser/ai claude-code + raw claude CLI paths)
	result, err := callViaClaude(prompt, system)
	if err == nil && result != nil {
		return result, nil
	}

	// Manual — return nil, caller handles
	return nil, nil
}

// =============================================================================
// Claude CLI Spawn
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

// callViaClaude spawns the `claude` CLI and captures its output.
// It mirrors the TS callViaClaude function: runs
//
//	claude -p <prompt> --output-format json --max-turns 1
//
// and attempts JSON parsing, falling back to raw text.
func callViaClaude(prompt string, system string) (*BridgeResult, error) {
	args := []string{"-p", prompt, "--output-format", "json", "--max-turns", "1"}
	if system != "" {
		args = append(args, "--system-prompt", system)
	}

	cmd := exec.Command("claude", args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	raw := strings.TrimSpace(stdout.String())
	if raw == "" {
		return nil, nil
	}

	// Attempt JSON parse — mirrors TS try/catch around JSON.parse
	var parsed claudeOutput
	if jsonErr := json.Unmarshal([]byte(raw), &parsed); jsonErr == nil {
		text := parsed.Result
		if text == "" && parsed.Message != nil && len(parsed.Message.Content) > 0 {
			text = parsed.Message.Content[0].Text
		}
		if text == "" {
			text = raw
		}
		return &BridgeResult{Text: text, Provider: "claude-cli"}, nil
	}

	// JSON parse failed — return raw text (same as TS fallback)
	return &BridgeResult{Text: raw, Provider: "claude-cli"}, nil
}
