// Package bridge orchestrates AI calls for validation and spec generation.
//
// Tries claude CLI first (equivalent to @eser/ai registry with claude-code
// provider), falls back to manual (returns nil — caller handles).
package bridge

import (
	"context"
	"strings"

	"github.com/pragmataW/tddmaster/internal/runner"
	"github.com/pragmataW/tddmaster/internal/state"
)

// =============================================================================
// Package-level seam variables (swappable in tests)
// =============================================================================

var (
	bridgeRunnerSelect = runner.Select
	bridgeReadManifest = state.ReadManifest
	bridgeResolveRoot  = state.ResolveProjectRoot
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
//  1. Resolve project root — graceful on error.
//  2. Load manifest — graceful on error.
//  3. Select runner via registry — graceful on ErrRunnerNotFound.
//  4. Invoke runner — graceful on any error.
//  5. Returns nil — caller handles manual fallback.
func CallAgent(prompt string, system string) (*BridgeResult, error) {
	// 1. Resolve project root — if error, return (nil, nil) gracefully.
	rootResult, err := bridgeResolveRoot()
	if err != nil || rootResult.Root == "" {
		return nil, nil
	}

	// 2. Load manifest — if error, return (nil, nil) gracefully.
	manifest, err := bridgeReadManifest(rootResult.Root)
	if err != nil || manifest == nil {
		return nil, nil
	}

	// 3. Select runner — graceful on ErrRunnerNotFound (return nil, nil).
	selectedRunner, err := bridgeRunnerSelect(manifest, "")
	if err != nil || selectedRunner == nil {
		return nil, nil
	}

	// 4. Build RunRequest.
	req := runner.RunRequest{
		Prompt:       prompt,
		SystemPrompt: system,
		MaxTurns:     1,
		OutputFormat: "json",
	}

	// 5. Invoke with cancelable context.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result, err := selectedRunner.Invoke(ctx, req)
	if err != nil {
		// All errors (binary not found, generic) → graceful fallback.
		return nil, nil
	}
	if result == nil {
		return nil, nil
	}

	// 6. Parse result.
	text := extractTextFromResult(result)
	if text == "" {
		return nil, nil
	}
	return &BridgeResult{Text: text, Provider: selectedRunner.Name()}, nil
}

// extractTextFromResult extracts the text content from a RunResult.
// Prefers ParsedJSON.result; then ParsedJSON.message.content[0].text; then raw Stdout.
func extractTextFromResult(result *runner.RunResult) string {
	if result.ParsedJSON != nil {
		if v, ok := result.ParsedJSON["result"].(string); ok && v != "" {
			return v
		}
		// message.content[0].text
		if msg, ok := result.ParsedJSON["message"].(map[string]any); ok {
			if content, ok := msg["content"].([]any); ok && len(content) > 0 {
				if first, ok := content[0].(map[string]any); ok {
					if t, ok := first["text"].(string); ok && t != "" {
						return t
					}
				}
			}
		}
	}
	// Raw Stdout fallback (EC-4).
	raw := strings.TrimSpace(string(result.Stdout))
	return raw
}
