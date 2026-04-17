package adapters

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	statesync "github.com/pragmataW/tddmaster/internal/sync"
)

// TestBuildClaudeSection_ContainsTaskRecovery verifies that the generated
// CLAUDE.md section includes the "Task recovery" heading and the undo command.
func TestBuildClaudeSection_ContainsTaskRecovery(t *testing.T) {
	section := buildClaudeSection(nil, nil, "tddmaster")

	assert.Contains(t, section, "### Task recovery")
	assert.Contains(t, section, "tddmaster undo")
	assert.Contains(t, section, "tddmaster spec <name> task undo <id>")
	assert.Contains(t, section, "tddmaster spec <name> reopen --resume-execution")
}

// TestBuildClaudeSection_TaskRecoveryMentionsDoneAndCancel ensures the block
// warns that done/cancel act on the ENTIRE spec.
func TestBuildClaudeSection_TaskRecoveryMentionsDoneAndCancel(t *testing.T) {
	section := buildClaudeSection(nil, nil, "tddmaster")

	assert.Contains(t, section, "`done` and `cancel` act on the ENTIRE spec")
}

// TestBuildClaudeSection_CustomPrefix_SubstitutesCorrectly checks that a
// non-default command prefix is substituted in the Task recovery block.
func TestBuildClaudeSection_CustomPrefix_SubstitutesCorrectly(t *testing.T) {
	section := buildClaudeSection(nil, nil, "mytool")

	assert.Contains(t, section, "mytool undo")
	assert.Contains(t, section, "mytool spec <name> task undo <id>")
	assert.Contains(t, section, "mytool spec <name> reopen --resume-execution")
	// Default prefix must not appear
	assert.False(t, strings.Contains(section, "tddmaster undo"),
		"default prefix 'tddmaster' must not appear when custom prefix is used")
}

// TestBuildClaudeSection_AllowGit_DoesNotSuppressTaskRecovery ensures the
// Task recovery block is present regardless of the allowGit option.
func TestBuildClaudeSection_AllowGit_DoesNotSuppressTaskRecovery(t *testing.T) {
	opts := &statesync.SyncOptions{AllowGit: true}
	section := buildClaudeSection(nil, opts, "tddmaster")

	assert.Contains(t, section, "### Task recovery")
}
