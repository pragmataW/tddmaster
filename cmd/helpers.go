package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/state"
	statesync "github.com/pragmataW/tddmaster/internal/sync"
)

// writeJSON writes any value as indented JSON to stdout.
func writeJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(data))
	return err
}

// printErr prints a message to stderr.
func printErr(msg string) {
	fmt.Fprintln(os.Stderr, msg)
}

// resolveRoot resolves the project root from cwd or env.
func resolveRoot() (string, error) {
	result, err := state.ResolveProjectRoot()
	if err != nil {
		return "", err
	}
	return result.Root, nil
}

// loadRulesAndHints loads tier1 rules and interaction hints for a given state.
func loadRulesAndHints(root string, st state.StateFile, config *state.NosManifest) ([]string, *model.InteractionHints, int, error) {
	scoped, err := statesync.LoadScopedRules(root)
	if err != nil {
		return nil, nil, 0, err
	}

	tier1, tier2Count := statesync.SplitByTier(scoped, st.Phase)

	var hints *model.InteractionHints
	if config != nil {
		h := statesync.ResolveInteractionHints(config.Tools)
		if h != nil {
			hints = &model.InteractionHints{
				HasAskUserTool:        h.HasAskUserTool,
				OptionPresentation:    h.OptionPresentation,
				HasSubAgentDelegation: h.HasSubAgentDelegation,
				SubAgentMethod:        h.SubAgentMethod,
				AskUserStrategy:       h.AskUserStrategy,
			}
		}
	}

	return tier1, hints, tier2Count, nil
}

// compiledToMap converts a NextOutput to a map via JSON round-trip.
func compiledToMap(c model.NextOutput) map[string]interface{} {
	data, err := json.Marshal(c)
	if err != nil {
		return map[string]interface{}{}
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return map[string]interface{}{}
	}
	return m
}

// mergeMap merges src into dst (src values take precedence).
func mergeMap(dst, src map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range dst {
		result[k] = v
	}
	for k, v := range src {
		result[k] = v
	}
	return result
}

// specDirExists checks if a spec's directory exists on disk.
func specDirExists(root, specName string) bool {
	dirPath := fmt.Sprintf("%s/%s/specs/%s", root, state.TddmasterDir, specName)
	_, err := os.Stat(dirPath)
	return err == nil
}

// marshalJSON and unmarshalJSON are thin wrappers for use in helpers.
func marshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func unmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
