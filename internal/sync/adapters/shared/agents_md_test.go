package shared

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBuildProtocol_ContainsTaskRecoveryVsSpecTermination verifies that the
// generated AGENTS.md protocol section includes the task recovery heading.
func TestBuildProtocol_ContainsTaskRecoveryVsSpecTermination(t *testing.T) {
	section := BuildProtocol("tddmaster", nil)

	assert.Contains(t, section, "### Task recovery vs spec termination")
	assert.Contains(t, section, "tddmaster undo")
	assert.Contains(t, section, "tddmaster spec <name> task undo <id>")
	assert.Contains(t, section, "tddmaster spec <name> reopen --resume-execution")
}

// TestBuildProtocol_TaskRecoveryExplainsBlastRadii verifies the "blast radii"
// framing is present.
func TestBuildProtocol_TaskRecoveryExplainsBlastRadii(t *testing.T) {
	section := BuildProtocol("tddmaster", nil)

	assert.Contains(t, section, "blast radii")
	assert.Contains(t, section, "task-level, reversible")
	assert.Contains(t, section, "spec-level, use `reopen --resume-execution`")
}

// TestBuildProtocol_CustomPrefix_SubstitutesCorrectly checks that a non-default
// command prefix is substituted in the Task recovery block.
func TestBuildProtocol_CustomPrefix_SubstitutesCorrectly(t *testing.T) {
	section := BuildProtocol("mytool", nil)

	assert.Contains(t, section, "mytool undo")
	assert.Contains(t, section, "mytool spec <name> task undo <id>")
	assert.Contains(t, section, "mytool spec <name> reopen --resume-execution")
	assert.False(t, strings.Contains(section, "tddmaster undo"),
		"default prefix 'tddmaster' must not appear when custom prefix is used")
}

// TestBuildProtocol_TaskRecoveryAppearsBeforeWhySection ensures the task
// recovery block is positioned before the "Why tddmaster calls matter" section.
func TestBuildProtocol_TaskRecoveryAppearsBeforeWhySection(t *testing.T) {
	section := BuildProtocol("tddmaster", nil)

	recoveryIdx := strings.Index(section, "### Task recovery vs spec termination")
	whyIdx := strings.Index(section, "### Why tddmaster calls matter")

	assert.True(t, recoveryIdx != -1, "Task recovery section must be present")
	assert.True(t, whyIdx != -1, "Why section must be present")
	assert.Less(t, recoveryIdx, whyIdx, "Task recovery must appear before Why section")
}

func TestBuildProtocol_DiscoveryRefinementRequiresAnswerRenderingBeforeApproval(t *testing.T) {
	section := BuildProtocol("tddmaster", nil)

	assert.Contains(t, section, "`discoveryReviewData.reviewSummary`")
	assert.Contains(t, section, "`discoveryReviewData.answers`")
	assert.Contains(t, section, "Do NOT show approve/revise/split options until after the answer list is visible to the user.")
}
