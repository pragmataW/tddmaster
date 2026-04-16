
// Package state provides types and state management for tddmaster.
package state

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// =============================================================================
// Manifest — lightweight schema for manifest.yml fields used by the TDD workflow
// =============================================================================

// Manifest represents the TDD-relevant fields parsed from (or written to) a
// manifest.yml file.  It intentionally has a smaller surface area than
// NosManifest so that callers only deal with what the TDD workflow cares about.
//
// YAML tags use camelCase to match the TypeScript-side manifest schema.
type Manifest struct {
	// TddMode enables TDD workflow enforcement when true.
	// YAML key: tddMode, default: false.
	TddMode bool `json:"tddMode" yaml:"tddMode"`

	// TestRunner is the shell command used to run the test suite, e.g. "go test ./...".
	// When omitted in YAML the field is nil; callers should treat nil as "not configured".
	// YAML key: testRunner, nullable.
	TestRunner *string `json:"testRunner,omitempty" yaml:"testRunner"`

	// MaxVerificationRetries is the maximum number of times the verifier will
	// retry a failing verification step before giving up.
	// Default: 3.
	MaxVerificationRetries int `json:"maxVerificationRetries" yaml:"maxVerificationRetries"`

	// MaxRefactorRounds caps the number of verifier→executor refactor rounds per task.
	// Default: 3.
	MaxRefactorRounds int `json:"maxRefactorRounds,omitempty" yaml:"maxRefactorRounds,omitempty"`
}

// ParseManifest deserializes a Manifest from raw YAML bytes.
// Unknown keys are ignored; missing keys receive their zero values
// (TddMode defaults to false, TestRunner defaults to nil).
func ParseManifest(data []byte) (Manifest, error) {
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

// MarshalManifest serializes a Manifest to YAML bytes.
func MarshalManifest(m Manifest) ([]byte, error) {
	return yaml.Marshal(m)
}

// LoadManifest reads TDD-specific settings from <root>/.tddmaster/manifest.yml.
// It looks for the "tdd" key inside the "tddmaster" section of the YAML document;
// if no "tdd" key is present the function returns a zero-value Manifest (no error).
// File-not-found and parse errors are returned to the caller.
func LoadManifest(root string) (Manifest, error) {
	filePath := filepath.Join(root, TddmasterDir, "manifest.yml")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return Manifest{}, err
	}

	// Parse the top-level document as a map so we can extract the tddmaster section.
	var doc map[string]yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return Manifest{}, err
	}

	// Look for tddmaster.tdd first (canonical location).
	tddmasterNode, ok := doc["tddmaster"]
	if ok {
		var section map[string]yaml.Node
		if err := tddmasterNode.Decode(&section); err == nil {
			if tddNode, found := section["tdd"]; found {
				var m Manifest
				if err := tddNode.Decode(&m); err != nil {
					return Manifest{}, err
				}
				return m, nil
			}
		}
	}

	// Fallback: top-level "tdd" key for backward compatibility.
	tddNode, ok := doc["tdd"]
	if !ok {
		// No "tdd" key — return zero value (TddMode false, TestRunner nil).
		return Manifest{}, nil
	}

	var m Manifest
	if err := tddNode.Decode(&m); err != nil {
		return Manifest{}, err
	}
	return m, nil
}
