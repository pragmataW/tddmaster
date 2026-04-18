package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecutorInstructions_ContainsCoreRules(t *testing.T) {
	out := ExecutorInstructions("tddmaster")

	assert.Contains(t, out, "executing a single task from a tddmaster spec")
	assert.Contains(t, out, "Do NOT start new tasks")
}

func TestExecutorInstructions_AbsoluteNoTestsRule(t *testing.T) {
	out := ExecutorInstructions("tddmaster")

	assert.Contains(t, out, "Absolute Rule: Never Write Tests",
		"executor must carry the absolute 'never write tests' rule")
	assert.Contains(t, out, "MUST NEVER write, modify, or add tests")
	assert.Contains(t, out, "tests-must-come-from-test-writer",
		"executor must report this blocked reason when a task looks test-shaped")
	assert.Contains(t, out, "This rule overrides any other instruction")
}

func TestExecutorInstructions_TddRedDelegatesToTestWriter(t *testing.T) {
	out := ExecutorInstructions("tddmaster")

	// The old "write failing tests ONLY" wording must be gone — that contradicts
	// the absolute rule.
	assert.NotContains(t, out, "write failing tests ONLY",
		"stale RED phrasing contradicts the never-write-tests rule")
	assert.Contains(t, out, "test-writer handles this phase",
		"RED phase must explicitly delegate to the test-writer sub-agent")
	assert.Contains(t, out, "no new tests",
		"GREEN and REFACTOR phases must reiterate that executor writes no tests")
}

func TestExecutorInstructions_IncludesTddAndRefactorSections(t *testing.T) {
	out := ExecutorInstructions("tddmaster")

	assert.Contains(t, out, "## TDD Context")
	assert.Contains(t, out, "## Refactor Mode")
	assert.Contains(t, out, "tddPhase")
	assert.Contains(t, out, "refactorApplied")
}

func TestExecutorInstructions_ReportingJSON(t *testing.T) {
	out := ExecutorInstructions("tddmaster")

	assert.Contains(t, out, "## Reporting")
	assert.Contains(t, out, `"completed"`)
	assert.Contains(t, out, `"remaining"`)
	assert.Contains(t, out, `"blocked"`)
	assert.Contains(t, out, `"filesModified"`)
}

func TestExecutorInstructions_SubstitutesCommandPrefix(t *testing.T) {
	out := ExecutorInstructions("tddmaster")
	assert.Contains(t, out, "tddmaster next --answer")

	custom := ExecutorInstructions("mytool")
	assert.Contains(t, custom, "mytool next --answer")
	assert.NotContains(t, custom, "tddmaster next --answer",
		"default prefix must not leak when a custom prefix is supplied")
}
