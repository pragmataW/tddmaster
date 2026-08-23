package promptregistry

import (
	"encoding/json"
	"strings"
	"testing"
)

func requireContains(t *testing.T, name, got string, want []string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Fatalf("%s: missing %q in %q", name, w, got)
		}
	}
}

func requireAbsent(t *testing.T, name, got string, unwanted []string) {
	t.Helper()
	for _, u := range unwanted {
		if strings.Contains(got, u) {
			t.Fatalf("%s: must not contain %q, got %q", name, u, got)
		}
	}
}

func TestExecutionKeysResolve(t *testing.T) {
	keys := []InstructionKey{
		KeyExecRed,
		KeyExecGreen,
		KeyExecRefactor,
		KeyExecRefactorApply,
		KeyExecExecutor,
		KeyExecExecutorSkipVerify,
		KeyExecVerifier,
		KeyExecGate,
		KeyExecVerifyFailed,
	}
	for _, k := range keys {
		t.Run(string(k), func(t *testing.T) {
			got, ok := Instruction(k)
			if !ok {
				t.Fatalf("Instruction(%q): expected ok=true, got false", k)
			}
			if got == "" {
				t.Fatalf("Instruction(%q): expected non-empty string, got empty", k)
			}
		})
	}
}

func TestExecutionKeys_AreDistinct(t *testing.T) {
	keys := []InstructionKey{
		KeyExecRed,
		KeyExecGreen,
		KeyExecRefactor,
		KeyExecRefactorApply,
		KeyExecExecutor,
		KeyExecExecutorSkipVerify,
		KeyExecVerifier,
		KeyExecGate,
		KeyExecVerifyFailed,
	}
	seen := make(map[InstructionKey]bool)
	for _, k := range keys {
		if seen[k] {
			t.Fatalf("duplicate execution key: %q", k)
		}
		seen[k] = true
	}
}

func TestExecutionKeys_DoNotCollideWithDiscoveryKeys(t *testing.T) {
	execKeys := []InstructionKey{
		KeyExecRed,
		KeyExecGreen,
		KeyExecRefactor,
		KeyExecRefactorApply,
		KeyExecExecutor,
		KeyExecExecutorSkipVerify,
		KeyExecVerifier,
		KeyExecGate,
		KeyExecVerifyFailed,
	}
	discoveryKeys := []InstructionKey{
		KeyListenFirst,
		KeyModeSelection,
		KeyPremiseChallenge,
		KeySpecTaskGen,
		KeySelfReview,
		KeyRefinePrompt,
	}
	for _, ek := range execKeys {
		for _, dk := range discoveryKeys {
			if ek == dk {
				t.Fatalf("execution key %q collides with discovery key %q", ek, dk)
			}
		}
	}
}

func TestExecutionGoldenStrings_KeyExecRed(t *testing.T) {
	got, ok := Instruction(KeyExecRed)
	if !ok {
		t.Fatalf("Instruction(KeyExecRed): expected ok=true, got false")
	}
	requireContains(t, "KeyExecRed", got, []string{
		"RED", "FAILING tests ONLY",
		"traceability", "testFilePath", "functionName", "taskId",
		"ac-N", "ec-N", NameCriteria, NameEdgeCases,
	})
	requireAbsent(t, "KeyExecRed", got, []string{"AC-<n>", "EC-<n>", "coverage", "Spawn"})
}

func TestExecutionGoldenStrings_KeyExecGreen(t *testing.T) {
	got, ok := Instruction(KeyExecGreen)
	if !ok {
		t.Fatalf("Instruction(KeyExecGreen): expected ok=true, got false")
	}
	requireContains(t, "KeyExecGreen", got, []string{
		"GREEN", "Do NOT write new tests", "filesModified",
	})
	requireAbsent(t, "KeyExecGreen", got, []string{"Spawn", "Submit the executor"})
}

func TestExecutionInstructions_AreAddressedToTheSubAgent(t *testing.T) {
	subAgentKeys := []InstructionKey{
		KeyExecRed, KeyExecGreen, KeyExecRefactor, KeyExecRefactorApply,
		KeyExecRefactorSkipVerify, KeyExecExecutor, KeyExecExecutorSkipVerify, KeyExecVerifier,
	}
	orchestratorOnly := []string{"Spawn", "sub-agent", "orchestrator", "next <slug>", "AskUserQuestion"}
	for _, key := range subAgentKeys {
		got := MustInstruction(key)
		requireAbsent(t, string(key), got, orchestratorOnly)
	}
}

func TestExecutionGoldenStrings_KeyExecRefactor(t *testing.T) {
	got, ok := Instruction(KeyExecRefactor)
	if !ok {
		t.Fatalf("Instruction(KeyExecRefactor): expected ok=true, got false")
	}
	requireContains(t, "KeyExecRefactor", got, []string{
		"REFACTOR", "regression", "Re-run the full test suite",
	})
	requireAbsent(t, "KeyExecRefactor", got, []string{"refactorInstructions"})
}

func TestExecutionGoldenStrings_KeyExecRefactorApply(t *testing.T) {
	got, ok := Instruction(KeyExecRefactorApply)
	if !ok {
		t.Fatalf("Instruction(KeyExecRefactorApply): expected ok=true, got false")
	}
	requireContains(t, "KeyExecRefactorApply", got, []string{
		"REFACTOR", NameRefactorNotes, "verbatim", "refactorApplied",
	})
}

func TestExecutionGoldenStrings_KeyExecVerifyFailed(t *testing.T) {
	got, ok := Instruction(KeyExecVerifyFailed)
	if !ok {
		t.Fatalf("Instruction(KeyExecVerifyFailed): expected ok=true, got false")
	}
	requireContains(t, "KeyExecVerifyFailed", got, []string{"FAILED"})
}

func TestExecutionGoldenStrings_KeyExecExecutor_NonEmptyAndMeaningful(t *testing.T) {
	got, ok := Instruction(KeyExecExecutor)
	if !ok {
		t.Fatalf("Instruction(KeyExecExecutor): expected ok=true, got false")
	}
	if got == "" {
		t.Fatalf("Instruction(KeyExecExecutor): expected non-empty string, got empty")
	}
	if !strings.Contains(strings.ToLower(got), "executor") && !strings.Contains(strings.ToLower(got), "implement") {
		t.Fatalf("Instruction(KeyExecExecutor): expected to mention executor or implementation, got %q", got)
	}
}

func TestExecutionGoldenStrings_KeyExecExecutorSkipVerify_NonEmptyAndDisablesVerifier(t *testing.T) {
	got, ok := Instruction(KeyExecExecutorSkipVerify)
	if !ok {
		t.Fatalf("Instruction(KeyExecExecutorSkipVerify): expected ok=true, got false")
	}
	if got == "" {
		t.Fatalf("Instruction(KeyExecExecutorSkipVerify): expected non-empty string, got empty")
	}
	low := strings.ToLower(got)
	if !strings.Contains(low, "disabled") || !strings.Contains(low, "verif") {
		t.Fatalf("Instruction(KeyExecExecutorSkipVerify): expected to state verifier disabled, got %q", got)
	}
	if !strings.Contains(low, "completed") {
		t.Fatalf("Instruction(KeyExecExecutorSkipVerify): expected to mention completed array submit, got %q", got)
	}
}

func TestExecutionGoldenStrings_KeyExecVerifier_NonEmptyAndMeaningful(t *testing.T) {
	got, ok := Instruction(KeyExecVerifier)
	if !ok {
		t.Fatalf("Instruction(KeyExecVerifier): expected ok=true, got false")
	}
	if got == "" {
		t.Fatalf("Instruction(KeyExecVerifier): expected non-empty string, got empty")
	}
	if !strings.Contains(strings.ToLower(got), "verif") {
		t.Fatalf("Instruction(KeyExecVerifier): expected to mention verification, got %q", got)
	}
}

func TestExecutionGoldenStrings_KeyExecGate_NonEmptyAndMeaningful(t *testing.T) {
	got, ok := Instruction(KeyExecGate)
	if !ok {
		t.Fatalf("Instruction(KeyExecGate): expected ok=true, got false")
	}
	if got == "" {
		t.Fatalf("Instruction(KeyExecGate): expected non-empty string, got empty")
	}
	if !strings.Contains(strings.ToLower(got), "plan") && !strings.Contains(strings.ToLower(got), "gate") {
		t.Fatalf("Instruction(KeyExecGate): expected to mention plan or gate, got %q", got)
	}
}

func TestReportExamplesValidJSON_Executor(t *testing.T) {
	var v any
	if err := json.Unmarshal([]byte(ReportExampleExecutor), &v); err != nil {
		t.Fatalf("ReportExampleExecutor is not valid JSON: %v", err)
	}
}

func TestReportExamplesValidJSON_Verifier(t *testing.T) {
	var v any
	if err := json.Unmarshal([]byte(ReportExampleVerifier), &v); err != nil {
		t.Fatalf("ReportExampleVerifier is not valid JSON: %v", err)
	}
}

func TestReportExamplesValidJSON_Planner(t *testing.T) {
	var v any
	if err := json.Unmarshal([]byte(ReportExamplePlanner), &v); err != nil {
		t.Fatalf("ReportExamplePlanner is not valid JSON: %v", err)
	}
}

func TestReportExamplesValidJSON_TestWriter(t *testing.T) {
	var v any
	if err := json.Unmarshal([]byte(ReportExampleTestWriter), &v); err != nil {
		t.Fatalf("ReportExampleTestWriter is not valid JSON: %v", err)
	}
}

func TestReportExampleExecutor_ContainsRequiredFields(t *testing.T) {
	var m map[string]any
	if err := json.Unmarshal([]byte(ReportExampleExecutor), &m); err != nil {
		t.Fatalf("ReportExampleExecutor is not valid JSON: %v", err)
	}
	for _, field := range []string{"completed", "remaining", "blocked"} {
		if _, ok := m[field]; !ok {
			t.Fatalf("ReportExampleExecutor: missing required field %q", field)
		}
	}
}

func TestReportExampleVerifier_ContainsRequiredFields(t *testing.T) {
	var m map[string]any
	if err := json.Unmarshal([]byte(ReportExampleVerifier), &m); err != nil {
		t.Fatalf("ReportExampleVerifier is not valid JSON: %v", err)
	}
	for _, field := range []string{"passed", "refactorNotes", "failedACs"} {
		if _, ok := m[field]; !ok {
			t.Fatalf("ReportExampleVerifier: missing required field %q", field)
		}
	}
}

func TestReportExamplePlanner_ContainsRequiredFields(t *testing.T) {
	var m map[string]any
	if err := json.Unmarshal([]byte(ReportExamplePlanner), &m); err != nil {
		t.Fatalf("ReportExamplePlanner is not valid JSON: %v", err)
	}
	planRaw, ok := m["plan"]
	if !ok {
		t.Fatalf("ReportExamplePlanner: missing top-level field \"plan\"")
	}
	plan, ok := planRaw.(map[string]any)
	if !ok {
		t.Fatalf("ReportExamplePlanner: \"plan\" field is not an object")
	}
	for _, field := range []string{"touchedFiles", "approach"} {
		if _, ok := plan[field]; !ok {
			t.Fatalf("ReportExamplePlanner: missing required field \"plan.%s\"", field)
		}
	}
}

func TestReportExampleTestWriter_ContainsRequiredFields(t *testing.T) {
	var m map[string]any
	if err := json.Unmarshal([]byte(ReportExampleTestWriter), &m); err != nil {
		t.Fatalf("ReportExampleTestWriter is not valid JSON: %v", err)
	}
	for _, field := range []string{"testsWritten", "filesModified"} {
		if _, ok := m[field]; !ok {
			t.Fatalf("ReportExampleTestWriter: missing required field %q", field)
		}
	}
}

func TestExecutionGoldenStrings_ExecRefactorSkipVerifyText_MatchesConstant(t *testing.T) {
	got := execRefactorSkipVerifyText
	if got != MustInstruction(KeyExecRefactorSkipVerify) {
		t.Fatalf("execRefactorSkipVerifyText not registered under KeyExecRefactorSkipVerify: got %q", got)
	}
	requireContains(t, "execRefactorSkipVerifyText", got, []string{
		NameRefactorNotes, "verbatim", "refactorApplied", "completed", "SAME status report",
	})
}

func TestExecutionKeyValues_ExactStrings(t *testing.T) {
	cases := []struct {
		name string
		key  InstructionKey
		want string
	}{
		{"KeyExecRed", KeyExecRed, "execution:red"},
		{"KeyExecGreen", KeyExecGreen, "execution:green"},
		{"KeyExecRefactor", KeyExecRefactor, "execution:refactor"},
		{"KeyExecRefactorApply", KeyExecRefactorApply, "execution:refactor-apply"},
		{"KeyExecExecutor", KeyExecExecutor, "execution:executor"},
		{"KeyExecVerifier", KeyExecVerifier, "execution:verifier"},
		{"KeyExecGate", KeyExecGate, "execution:gate"},
		{"KeyExecVerifyFailed", KeyExecVerifyFailed, "execution:verify-failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if string(tc.key) != tc.want {
				t.Fatalf("%s: got %q, want %q", tc.name, string(tc.key), tc.want)
			}
		})
	}
}
