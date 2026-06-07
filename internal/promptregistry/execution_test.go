package promptregistry

import (
	"encoding/json"
	"strings"
	"testing"
)

const goldenExecRed = "TDD RED phase active. Spawn the `test-writer` sub-agent. " +
	"It writes FAILING tests only — no implementation, no test execution. " +
	"Pass `edgeCases` from this `next` output verbatim. After the test-writer reports, run `tddmaster spec <name> next` again."

const goldenExecGreen = "TDD GREEN phase active. Spawn the `tddmaster-executor` sub-agent. " +
	"It writes a clean, working implementation that makes the existing failing tests pass. " +
	"It does NOT write new tests and does NOT run tests. " +
	"Submit the executor's status report to `next`. " +
	"Do NOT spawn the verifier yourself — the orchestrator dispatches `tddmaster-verifier` as a separate stage on the next `next` call to run the tests and produce `refactorNotes`."

const goldenExecRefactor = "TDD REFACTOR phase active. " +
	"If `refactorInstructions` is present, spawn `tddmaster-executor` to apply each note verbatim and report `refactorApplied: true`. " +
	"If absent, spawn `tddmaster-verifier` for a regression re-check; tests must still pass."

const goldenExecRefactorApply = "Apply each refactor note verbatim. Do NOT change test behavior — tests must still pass. When finished, report `refactorApplied: true` in your JSON output; the verifier will re-run tests."

const goldenExecVerifyFailed = "Verification FAILED. Fix the failing tests before continuing."

const goldenExecRefactorSkipVerify = "Apply each refactor note verbatim. Tests must still pass. Submit BOTH `refactorApplied: true` AND `completed: [<task-id>]` in the SAME status report — verifier is disabled in this mode, so this single submit advances the task."

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
	if got != goldenExecRed {
		t.Fatalf("Instruction(KeyExecRed): got %q, want %q", got, goldenExecRed)
	}
}

func TestExecutionGoldenStrings_KeyExecGreen(t *testing.T) {
	got, ok := Instruction(KeyExecGreen)
	if !ok {
		t.Fatalf("Instruction(KeyExecGreen): expected ok=true, got false")
	}
	if got != goldenExecGreen {
		t.Fatalf("Instruction(KeyExecGreen): got %q, want %q", got, goldenExecGreen)
	}
}

func TestExecutionGoldenStrings_KeyExecRefactor(t *testing.T) {
	got, ok := Instruction(KeyExecRefactor)
	if !ok {
		t.Fatalf("Instruction(KeyExecRefactor): expected ok=true, got false")
	}
	if got != goldenExecRefactor {
		t.Fatalf("Instruction(KeyExecRefactor): got %q, want %q", got, goldenExecRefactor)
	}
}

func TestExecutionGoldenStrings_KeyExecRefactorApply(t *testing.T) {
	got, ok := Instruction(KeyExecRefactorApply)
	if !ok {
		t.Fatalf("Instruction(KeyExecRefactorApply): expected ok=true, got false")
	}
	if got != goldenExecRefactorApply {
		t.Fatalf("Instruction(KeyExecRefactorApply): got %q, want %q", got, goldenExecRefactorApply)
	}
}

func TestExecutionGoldenStrings_KeyExecVerifyFailed(t *testing.T) {
	got, ok := Instruction(KeyExecVerifyFailed)
	if !ok {
		t.Fatalf("Instruction(KeyExecVerifyFailed): expected ok=true, got false")
	}
	if got != goldenExecVerifyFailed {
		t.Fatalf("Instruction(KeyExecVerifyFailed): got %q, want %q", got, goldenExecVerifyFailed)
	}
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
	if got != goldenExecRefactorSkipVerify {
		t.Fatalf("execRefactorSkipVerifyText: got %q, want %q", got, goldenExecRefactorSkipVerify)
	}
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
