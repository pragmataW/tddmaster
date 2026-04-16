
package output_test

import (
	"testing"

	"github.com/pragmataW/tddmaster/internal/output"
)

// mockConfig implements ModeConfig for testing.
type mockConfig struct {
	agentMode *bool
}

func (m *mockConfig) GetAgentMode() *bool {
	return m.agentMode
}

func boolPtr(b bool) *bool { return &b }

// =============================================================================
// DetectMode
// =============================================================================

func TestDetectMode_AgentFlag(t *testing.T) {
	got := output.DetectMode([]string{"--agent", "spec", "list"}, nil)
	if got != output.ModeAgent {
		t.Errorf("DetectMode --agent = %q, want %q", got, output.ModeAgent)
	}
}

func TestDetectMode_HumanFlag(t *testing.T) {
	got := output.DetectMode([]string{"--human"}, nil)
	if got != output.ModeHuman {
		t.Errorf("DetectMode --human = %q, want %q", got, output.ModeHuman)
	}
}

func TestDetectMode_ConfigAgentTrue(t *testing.T) {
	cfg := &mockConfig{agentMode: boolPtr(true)}
	got := output.DetectMode([]string{}, cfg)
	if got != output.ModeAgent {
		t.Errorf("DetectMode config agent=true = %q, want %q", got, output.ModeAgent)
	}
}

func TestDetectMode_ConfigAgentFalse(t *testing.T) {
	cfg := &mockConfig{agentMode: boolPtr(false)}
	got := output.DetectMode([]string{}, cfg)
	if got != output.ModeHuman {
		t.Errorf("DetectMode config agent=false = %q, want %q", got, output.ModeHuman)
	}
}

func TestDetectMode_FlagOverridesConfig(t *testing.T) {
	cfg := &mockConfig{agentMode: boolPtr(false)} // config says human
	got := output.DetectMode([]string{"--agent"}, cfg)
	// flag wins over config
	if got != output.ModeAgent {
		t.Errorf("DetectMode flag overrides config = %q, want %q", got, output.ModeAgent)
	}
}

func TestDetectMode_NilConfig_NoFlag_ReturnsSomething(t *testing.T) {
	// No assertion on exact value — just ensure it doesn't panic
	got := output.DetectMode([]string{}, nil)
	if got != output.ModeAgent && got != output.ModeHuman {
		t.Errorf("DetectMode returned unexpected mode: %q", got)
	}
}

// =============================================================================
// DetectInteraction
// =============================================================================

func TestDetectInteraction_NonInteractiveFlag(t *testing.T) {
	got := output.DetectInteraction([]string{"--non-interactive"})
	if got != output.InteractionNonInteractive {
		t.Errorf("DetectInteraction --non-interactive = %q, want %q", got, output.InteractionNonInteractive)
	}
}

func TestDetectInteraction_NoFlag_ReturnsSomething(t *testing.T) {
	got := output.DetectInteraction([]string{})
	if got != output.InteractionInteractive && got != output.InteractionNonInteractive {
		t.Errorf("DetectInteraction returned unexpected: %q", got)
	}
}

// =============================================================================
// StripModeFlag
// =============================================================================

func TestStripModeFlag_RemovesFlags(t *testing.T) {
	args := []string{"spec", "--agent", "--non-interactive", "list"}
	got := output.StripModeFlag(args)
	want := []string{"spec", "list"}
	if !equal(got, want) {
		t.Errorf("StripModeFlag = %v, want %v", got, want)
	}
}

func TestStripModeFlag_Nil(t *testing.T) {
	got := output.StripModeFlag(nil)
	if len(got) != 0 {
		t.Errorf("StripModeFlag(nil) = %v, want empty", got)
	}
}

func TestStripModeFlag_NoFlags(t *testing.T) {
	args := []string{"spec", "list"}
	got := output.StripModeFlag(args)
	if !equal(got, args) {
		t.Errorf("StripModeFlag no flags = %v, want %v", got, args)
	}
}
