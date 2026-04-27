package service

import (
	"slices"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/state"
)

const hardcodedTestTask = "Write or update tests for all new and changed behavior"
const hardcodedDocsTask = "Update documentation for all public-facing changes (README, API docs, CHANGELOG)"

func contains(tasks []string, want string) bool {
	return slices.Contains(tasks, want)
}

func indexOf(tasks []string, want string) int {
	for i, t := range tasks {
		if t == want {
			return i
		}
	}
	return -1
}

func indexOfPrefix(tasks []string, prefix string) int {
	for i, t := range tasks {
		if strings.HasPrefix(t, prefix) {
			return i
		}
	}
	return -1
}

func ambitionAnswer(text string) state.DiscoveryAnswer {
	return state.DiscoveryAnswer{QuestionID: "ambition", Answer: text}
}

func TestDeriveTasks_NonTDDMode_AppendsHardcodedTestTask(t *testing.T) {
	answers := []state.DiscoveryAnswer{ambitionAnswer("10-star: Add multiplier helper")}

	got := deriveTasks(answers, nil, false)

	if !contains(got, hardcodedTestTask) {
		t.Fatalf("non-TDD mode must append hard-coded test task, got: %v", got)
	}
	if !contains(got, hardcodedDocsTask) {
		t.Fatalf("non-TDD mode must append docs task, got: %v", got)
	}
}

func TestDeriveTasks_TDDMode_OmitsHardcodedTestTask(t *testing.T) {
	answers := []state.DiscoveryAnswer{ambitionAnswer("10-star: Add multiplier helper")}

	got := deriveTasks(answers, nil, true)

	if contains(got, hardcodedTestTask) {
		t.Fatalf("TDD mode must NOT append hard-coded test task; RGR cycle handles tests. got: %v", got)
	}
}

func TestDeriveTasks_TDDMode_KeepsDocsTask(t *testing.T) {
	answers := []state.DiscoveryAnswer{ambitionAnswer("10-star: Add multiplier helper")}

	got := deriveTasks(answers, nil, true)

	if !contains(got, hardcodedDocsTask) {
		t.Fatalf("TDD mode must keep docs task (no conflict with executor), got: %v", got)
	}
}

func TestDeriveTasks_TDDMode_ReordersDiscoveryTestTasks(t *testing.T) {
	answers := []state.DiscoveryAnswer{
		ambitionAnswer("10-star: Add multiplier helper"),
		{QuestionID: "verification", Answer: "- Add test for negative input\n- Wire up CLI flag"},
	}

	got := deriveTasks(answers, nil, true)

	testIdx := indexOf(got, "Add test for negative input")
	cliIdx := indexOf(got, "Wire up CLI flag")
	if testIdx == -1 || cliIdx == -1 {
		t.Fatalf("discovery items missing from tasks: %v", got)
	}
	if testIdx >= cliIdx {
		t.Fatalf("discovery test item must be reordered before non-test items in TDD mode; got testIdx=%d cliIdx=%d (%v)", testIdx, cliIdx, got)
	}
}

func TestDeriveTasks_TDDMode_NoTestRelatedItems_NoHardcodedTestTask(t *testing.T) {
	answers := []state.DiscoveryAnswer{
		ambitionAnswer("10-star: Add multiplier helper"),
		{QuestionID: "verification", Answer: "- Wire up CLI flag\n- Update changelog manually"},
	}

	got := deriveTasks(answers, nil, true)

	for _, task := range got {
		if isTestTask(task) {
			t.Fatalf("TDD mode with no test-related discovery items must produce a task list with no test tasks at all; got %q in %v", task, got)
		}
	}
	if indexOfPrefix(got, "Wire up CLI flag") == -1 {
		t.Fatalf("CLI task missing: %v", got)
	}
}
