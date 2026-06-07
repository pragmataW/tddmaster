package promptregistry

import (
	"testing"

	"github.com/pragmataW/tddmaster/internal/prompts"
)

func TestInstructionKey_IsDistinctType(t *testing.T) {
	var k InstructionKey = "some-key"
	var _ InstructionKey = k
}

func TestAgentRegistryKey_IsDistinctType(t *testing.T) {
	var k AgentRegistryKey = "some-agent"
	var _ AgentRegistryKey = k
}

func TestAgentRegistryKey_TypeDistinctFromInstructionKey(t *testing.T) {
	var ik InstructionKey = "x"
	var ak AgentRegistryKey = "x"
	_ = ik
	_ = ak
}

func TestAgentConstants_Executor(t *testing.T) {
	if AgentExecutor != AgentRegistryKey("tddmaster-executor") {
		t.Fatalf("AgentExecutor: expected %q, got %q", "tddmaster-executor", AgentExecutor)
	}
}

func TestAgentConstants_Verifier(t *testing.T) {
	if AgentVerifier != AgentRegistryKey("tddmaster-verifier") {
		t.Fatalf("AgentVerifier: expected %q, got %q", "tddmaster-verifier", AgentVerifier)
	}
}

func TestAgentConstants_Planner(t *testing.T) {
	if AgentPlanner != AgentRegistryKey("tddmaster-planner") {
		t.Fatalf("AgentPlanner: expected %q, got %q", "tddmaster-planner", AgentPlanner)
	}
}

func TestAgentConstants_TestWriter(t *testing.T) {
	if AgentTestWriter != AgentRegistryKey("tddmaster-test-writer") {
		t.Fatalf("AgentTestWriter: expected %q, got %q", "tddmaster-test-writer", AgentTestWriter)
	}
}

func TestInstruction_KnownKey_ReturnsNonEmptyStringAndTrue(t *testing.T) {
	names := prompts.TemplateNames()
	if len(names) == 0 {
		t.Fatal("prompts.TemplateNames() returned no templates; cannot validate registry")
	}
	key := InstructionKey(names[0])
	val, ok := Instruction(key)
	if !ok {
		t.Fatalf("Instruction(%q): expected ok=true, got false", key)
	}
	if val == "" {
		t.Fatalf("Instruction(%q): expected non-empty string, got empty", key)
	}
}

func TestInstruction_UnknownKey_ReturnsFalse(t *testing.T) {
	val, ok := Instruction(InstructionKey("__does_not_exist__"))
	if ok {
		t.Fatalf("Instruction unknown key: expected ok=false, got true")
	}
	if val != "" {
		t.Fatalf("Instruction unknown key: expected empty string, got %q", val)
	}
}

func TestInstruction_ExecutorKey_MatchesPromptsPackage(t *testing.T) {
	const name = "executor"
	key := InstructionKey(name)
	got, ok := Instruction(key)
	if !ok {
		t.Fatalf("Instruction(%q): expected ok=true, got false", key)
	}
	want, err := prompts.Render(name, prompts.RenderData{})
	if err != nil {
		t.Fatalf("prompts.Render(%q): unexpected error: %v", name, err)
	}
	if got != want {
		t.Fatalf("Instruction(%q): value does not match prompts.Render output\ngot:  %q\nwant: %q", key, got, want)
	}
}

func TestInstruction_VerifierKey_MatchesPromptsPackage(t *testing.T) {
	const name = "verifier"
	key := InstructionKey(name)
	got, ok := Instruction(key)
	if !ok {
		t.Fatalf("Instruction(%q): expected ok=true, got false", key)
	}
	want, err := prompts.Render(name, prompts.RenderData{})
	if err != nil {
		t.Fatalf("prompts.Render(%q): unexpected error: %v", name, err)
	}
	if got != want {
		t.Fatalf("Instruction(%q): value does not match prompts.Render output\ngot:  %q\nwant: %q", key, got, want)
	}
}

func TestInstruction_PlannerKey_MatchesPromptsPackage(t *testing.T) {
	const name = "planner"
	key := InstructionKey(name)
	got, ok := Instruction(key)
	if !ok {
		t.Fatalf("Instruction(%q): expected ok=true, got false", key)
	}
	want, err := prompts.Render(name, prompts.RenderData{})
	if err != nil {
		t.Fatalf("prompts.Render(%q): unexpected error: %v", name, err)
	}
	if got != want {
		t.Fatalf("Instruction(%q): value does not match prompts.Render output\ngot:  %q\nwant: %q", key, got, want)
	}
}

func TestInstruction_TestWriterKey_MatchesPromptsPackage(t *testing.T) {
	const name = "test-writer"
	key := InstructionKey(name)
	got, ok := Instruction(key)
	if !ok {
		t.Fatalf("Instruction(%q): expected ok=true, got false", key)
	}
	want, err := prompts.Render(name, prompts.RenderData{})
	if err != nil {
		t.Fatalf("prompts.Render(%q): unexpected error: %v", name, err)
	}
	if got != want {
		t.Fatalf("Instruction(%q): value does not match prompts.Render output\ngot:  %q\nwant: %q", key, got, want)
	}
}

func TestInstruction_AllKnownKeys_HaveNonEmptyValues(t *testing.T) {
	names := prompts.TemplateNames()
	for _, name := range names {
		key := InstructionKey(name)
		val, ok := Instruction(key)
		if !ok {
			t.Errorf("Instruction(%q): expected ok=true, got false", key)
			continue
		}
		if val == "" {
			t.Errorf("Instruction(%q): expected non-empty string, got empty", key)
		}
	}
}

func TestInstruction_NoDuplicateValues(t *testing.T) {
	names := prompts.TemplateNames()
	seen := make(map[string]InstructionKey)
	for _, name := range names {
		key := InstructionKey(name)
		val, ok := Instruction(key)
		if !ok {
			continue
		}
		if prev, exists := seen[val]; exists {
			t.Errorf("duplicate instruction value for keys %q and %q", prev, key)
		}
		seen[val] = key
	}
}
