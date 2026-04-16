
package defaults_test

import (
	"testing"

	"github.com/pragmataW/tddmaster/internal/defaults"
)

// =============================================================================
// DefaultConcerns tests
// =============================================================================

func TestDefaultConcerns_Length(t *testing.T) {
	concerns := defaults.DefaultConcerns()
	if len(concerns) != 7 {
		t.Errorf("expected 7 concerns, got %d", len(concerns))
	}
}

func TestDefaultConcerns_IDs(t *testing.T) {
	concerns := defaults.DefaultConcerns()
	expectedIDs := []string{
		"open-source",
		"beautiful-product",
		"long-lived",
		"move-fast",
		"compliance",
		"learning-project",
		"well-engineered",
	}
	for i, expected := range expectedIDs {
		if concerns[i].ID != expected {
			t.Errorf("concerns[%d].ID = %q, want %q", i, concerns[i].ID, expected)
		}
	}
}

func TestDefaultConcerns_Names(t *testing.T) {
	concerns := defaults.DefaultConcerns()
	expectedNames := []string{
		"Open Source",
		"Beautiful Product",
		"Long-Lived",
		"Move Fast",
		"Compliance",
		"Learning Project",
		"Well-Engineered",
	}
	for i, expected := range expectedNames {
		if concerns[i].Name != expected {
			t.Errorf("concerns[%d].Name = %q, want %q", i, concerns[i].Name, expected)
		}
	}
}

func TestDefaultConcerns_NonEmpty(t *testing.T) {
	concerns := defaults.DefaultConcerns()
	for _, c := range concerns {
		if c.ID == "" {
			t.Errorf("concern has empty ID")
		}
		if c.Name == "" {
			t.Errorf("concern %q has empty Name", c.ID)
		}
		if c.Description == "" {
			t.Errorf("concern %q has empty Description", c.ID)
		}
	}
}

func TestDefaultConcerns_ReviewDimensions(t *testing.T) {
	concerns := defaults.DefaultConcerns()
	// Every concern should have at least one review dimension.
	for _, c := range concerns {
		if len(c.ReviewDimensions) == 0 {
			t.Errorf("concern %q has no review dimensions", c.ID)
		}
	}
}

func TestDefaultConcerns_IndependentSlices(t *testing.T) {
	// Each call should return a fresh slice.
	c1 := defaults.DefaultConcerns()
	c2 := defaults.DefaultConcerns()
	if len(c1) != len(c2) {
		t.Errorf("length mismatch: %d vs %d", len(c1), len(c2))
	}
}

// =============================================================================
// DefaultPacks tests
// =============================================================================

func TestDefaultPacks_Keys(t *testing.T) {
	packs := defaults.DefaultPacks()
	expectedKeys := []string{"typescript", "react", "security"}
	for _, key := range expectedKeys {
		if _, ok := packs[key]; !ok {
			t.Errorf("expected pack %q to be present", key)
		}
	}
	if len(packs) != 3 {
		t.Errorf("expected 3 packs, got %d", len(packs))
	}
}

func TestDefaultPacks_TypescriptManifest(t *testing.T) {
	packs := defaults.DefaultPacks()
	ts := packs["typescript"]
	if ts.Manifest.Name != "typescript" {
		t.Errorf("typescript manifest name = %q, want %q", ts.Manifest.Name, "typescript")
	}
	if ts.Manifest.Version != "1.0.0" {
		t.Errorf("typescript manifest version = %q, want %q", ts.Manifest.Version, "1.0.0")
	}
	if ts.Manifest.Description != "TypeScript best practices" {
		t.Errorf("typescript manifest description = %q, want %q", ts.Manifest.Description, "TypeScript best practices")
	}
}

func TestDefaultPacks_ReactManifest(t *testing.T) {
	packs := defaults.DefaultPacks()
	r := packs["react"]
	if r.Manifest.Name != "react" {
		t.Errorf("react manifest name = %q, want %q", r.Manifest.Name, "react")
	}
	if r.Manifest.Version != "1.0.0" {
		t.Errorf("react manifest version = %q, want %q", r.Manifest.Version, "1.0.0")
	}
	if r.Manifest.Description != "React component conventions" {
		t.Errorf("react manifest description = %q, want %q", r.Manifest.Description, "React component conventions")
	}
}

func TestDefaultPacks_SecurityManifest(t *testing.T) {
	packs := defaults.DefaultPacks()
	s := packs["security"]
	if s.Manifest.Name != "security" {
		t.Errorf("security manifest name = %q, want %q", s.Manifest.Name, "security")
	}
	if s.Manifest.Version != "1.0.0" {
		t.Errorf("security manifest version = %q, want %q", s.Manifest.Version, "1.0.0")
	}
	if s.Manifest.Description != "Security audit rules" {
		t.Errorf("security manifest description = %q, want %q", s.Manifest.Description, "Security audit rules")
	}
}

func TestDefaultPacks_TypescriptRules(t *testing.T) {
	packs := defaults.DefaultPacks()
	ts := packs["typescript"]
	expectedRules := map[string]string{
		"use-strict-types": "Prefer explicit types over inference for function params and returns",
		"no-any":           "Never use 'any'. Use 'unknown' when type is genuinely unknown.",
		"prefer-const":     "Use const by default, let only when reassignment is needed",
	}
	if len(ts.RuleContents) != len(expectedRules) {
		t.Errorf("typescript rule count = %d, want %d", len(ts.RuleContents), len(expectedRules))
	}
	for k, v := range expectedRules {
		if ts.RuleContents[k] != v {
			t.Errorf("typescript rule %q = %q, want %q", k, ts.RuleContents[k], v)
		}
	}
}

func TestDefaultPacks_ReactRules(t *testing.T) {
	packs := defaults.DefaultPacks()
	r := packs["react"]
	expectedRules := map[string]string{
		"component-structure":        "One component per file. Name file same as component.",
		"prefer-function-components": "Use function components with hooks. No class components.",
		"state-management":           "Keep state as close to where it's used as possible.",
	}
	if len(r.RuleContents) != len(expectedRules) {
		t.Errorf("react rule count = %d, want %d", len(r.RuleContents), len(expectedRules))
	}
	for k, v := range expectedRules {
		if r.RuleContents[k] != v {
			t.Errorf("react rule %q = %q, want %q", k, r.RuleContents[k], v)
		}
	}
}

func TestDefaultPacks_SecurityRules(t *testing.T) {
	packs := defaults.DefaultPacks()
	s := packs["security"]
	expectedRules := map[string]string{
		"no-secrets-in-code": "Never hardcode API keys, passwords, or tokens",
		"validate-input":     "Validate and sanitize all user input at API boundaries",
		"no-eval":            "Never use eval(), new Function(), or equivalent dynamic code execution",
	}
	if len(s.RuleContents) != len(expectedRules) {
		t.Errorf("security rule count = %d, want %d", len(s.RuleContents), len(expectedRules))
	}
	for k, v := range expectedRules {
		if s.RuleContents[k] != v {
			t.Errorf("security rule %q = %q, want %q", k, s.RuleContents[k], v)
		}
	}
}

func TestDefaultPacks_TypescriptConcerns(t *testing.T) {
	packs := defaults.DefaultPacks()
	ts := packs["typescript"]
	if len(ts.ConcernContents) != 1 {
		t.Errorf("typescript concern count = %d, want 1", len(ts.ConcernContents))
		return
	}
	c := ts.ConcernContents[0]
	if c.ID != "ts-quality" {
		t.Errorf("typescript concern ID = %q, want %q", c.ID, "ts-quality")
	}
	if c.Name != "TypeScript Quality" {
		t.Errorf("typescript concern Name = %q, want %q", c.Name, "TypeScript Quality")
	}
}

func TestDefaultPacks_ReactConcerns(t *testing.T) {
	packs := defaults.DefaultPacks()
	r := packs["react"]
	if len(r.ConcernContents) != 0 {
		t.Errorf("react concern count = %d, want 0", len(r.ConcernContents))
	}
}

func TestDefaultPacks_SecurityConcerns(t *testing.T) {
	packs := defaults.DefaultPacks()
	s := packs["security"]
	if len(s.ConcernContents) != 1 {
		t.Errorf("security concern count = %d, want 1", len(s.ConcernContents))
		return
	}
	c := s.ConcernContents[0]
	if c.ID != "security-audit" {
		t.Errorf("security concern ID = %q, want %q", c.ID, "security-audit")
	}
	if c.Name != "Security Audit" {
		t.Errorf("security concern Name = %q, want %q", c.Name, "Security Audit")
	}
}

func TestDefaultPacks_ManifestTags(t *testing.T) {
	packs := defaults.DefaultPacks()

	tsTags := packs["typescript"].Manifest.Tags
	if len(tsTags) != 2 || tsTags[0] != "typescript" || tsTags[1] != "types" {
		t.Errorf("typescript tags = %v, want [typescript types]", tsTags)
	}

	reactTags := packs["react"].Manifest.Tags
	if len(reactTags) != 2 || reactTags[0] != "react" || reactTags[1] != "frontend" {
		t.Errorf("react tags = %v, want [react frontend]", reactTags)
	}

	secTags := packs["security"].Manifest.Tags
	if len(secTags) != 2 || secTags[0] != "security" || secTags[1] != "audit" {
		t.Errorf("security tags = %v, want [security audit]", secTags)
	}
}

func TestDefaultPacks_ManifestRules(t *testing.T) {
	packs := defaults.DefaultPacks()

	tsRules := packs["typescript"].Manifest.Rules
	if len(tsRules) != 3 {
		t.Errorf("typescript manifest rules count = %d, want 3", len(tsRules))
	}

	reactRulesList := packs["react"].Manifest.Rules
	if len(reactRulesList) != 3 {
		t.Errorf("react manifest rules count = %d, want 3", len(reactRulesList))
	}

	secRules := packs["security"].Manifest.Rules
	if len(secRules) != 3 {
		t.Errorf("security manifest rules count = %d, want 3", len(secRules))
	}
}
