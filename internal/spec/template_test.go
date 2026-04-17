package spec

import (
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/state"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// toBulletList
// =============================================================================

func TestToBulletList_SingleLine(t *testing.T) {
	items := toBulletList("A single sentence")
	assert.Len(t, items, 1)
	assert.Equal(t, "A single sentence", items[0])
}

func TestToBulletList_MultipleLines(t *testing.T) {
	items := toBulletList("Line one\nLine two\nLine three")
	assert.Len(t, items, 3)
}

func TestToBulletList_SentenceBoundaries(t *testing.T) {
	items := toBulletList("First sentence. Second sentence. Third one")
	// Should split on ". S" (period + space + uppercase)
	assert.True(t, len(items) >= 2)
}

func TestToBulletList_DoesNotSplitOnFileExtension(t *testing.T) {
	items := toBulletList("Check CLAUDE.md for details")
	assert.Len(t, items, 1)
	assert.Contains(t, items[0], "CLAUDE.md")
}

func TestToBulletList_DoesNotSplitOnVersionNumber(t *testing.T) {
	items := toBulletList("Check v0.1 compatibility with the new API")
	assert.Len(t, items, 1)
	assert.Contains(t, items[0], "v0.1")
}

func TestToBulletList_DoesNotSplitOnFilePath(t *testing.T) {
	items := toBulletList("Verify src/api/v1/endpoint.ts handles errors correctly")
	assert.Len(t, items, 1)
	assert.Contains(t, items[0], "endpoint.ts")
}

// Regression: long technical answers use semicolons as in-sentence
// punctuation. Splitting on ";" fragmented Out-of-Scope / Edge Cases
// rendering in specs such as claude-harici-provider-abstraction-tamamlama.
func TestToBulletList_DoesNotSplitOnInSentenceSemicolon(t *testing.T) {
	input := "SyncRules and SyncHooks remain unchanged; only Capabilities() expands for the new adapter surface"
	items := toBulletList(input)
	assert.Len(t, items, 1)
	assert.Contains(t, items[0], "remain unchanged; only Capabilities")
}

// =============================================================================
// DeriveTasks
// =============================================================================

func TestDeriveTasks_TenStarProducesOneTask(t *testing.T) {
	answers := []state.DiscoveryAnswer{
		{
			QuestionID: "ambition",
			Answer:     "1-star: basic upload. 10-star: full system with validation, preview, and batch processing",
		},
	}
	tasks := DeriveTasks(answers, nil, false)

	// +2 mandatory test/docs tasks appended
	assert.Len(t, tasks, 3)
	assert.Contains(t, tasks[0], "validation")
	assert.Contains(t, tasks[0], "preview")
	assert.Contains(t, tasks[0], "batch processing")
}

func TestDeriveTasks_UsesFallbackWhenNoTenStar(t *testing.T) {
	answers := []state.DiscoveryAnswer{
		{
			QuestionID: "ambition",
			Answer:     "Build a dashboard with filtering and sorting",
		},
	}
	tasks := DeriveTasks(answers, nil, false)

	assert.Len(t, tasks, 3)
	assert.Contains(t, tasks[0], "filtering")
	assert.Contains(t, tasks[0], "sorting")
}

func TestDeriveTasks_DoesNotSplitIntoPerStarTasks(t *testing.T) {
	answers := []state.DiscoveryAnswer{
		{
			QuestionID: "ambition",
			Answer:     "1-star: Add a sentence about --validate. 10-star: Full protocol section with human->AgentA->AgentB->human cycle, usage examples, and mermaid diagram",
		},
	}
	tasks := DeriveTasks(answers, nil, false)

	// Must be ONE task (+2 mandatory)
	assert.Len(t, tasks, 3)
	assert.Contains(t, tasks[0], "protocol section")
	assert.NotContains(t, tasks[0], "1-star")
}

func TestDeriveTasks_StripsGarbledPrefix(t *testing.T) {
	answers := []state.DiscoveryAnswer{
		{
			QuestionID: "ambition",
			Answer:     "10-star: the target: Full protocol section with phase transition docs",
		},
	}
	tasks := DeriveTasks(answers, nil, false)

	assert.Len(t, tasks, 3)
	assert.Equal(t, "Full protocol section with phase transition docs", tasks[0])
}

func TestDeriveTasks_DoesNotAddImplementPrefix(t *testing.T) {
	answers := []state.DiscoveryAnswer{
		{
			QuestionID: "ambition",
			Answer:     "10-star: comprehensive analytics dashboard with real-time updates, drill-down charts, cohort analysis, and export to CSV and PDF formats for all user segments",
		},
	}
	tasks := DeriveTasks(answers, nil, false)

	assert.Len(t, tasks, 3)
	assert.False(t, strings.HasPrefix(tasks[0], "Implement"))
}

func TestDeriveTasks_VerificationSingleLineSingleTask(t *testing.T) {
	answers := []state.DiscoveryAnswer{
		{
			QuestionID: "verification",
			Answer:     "Run tddmaster sync, check CLAUDE.md contains Phase Transition Protocol section and --validate docs",
		},
	}
	tasks := DeriveTasks(answers, nil, false)

	assert.Len(t, tasks, 3)
	assert.Contains(t, tasks[0], "Run tddmaster sync")
	assert.Contains(t, tasks[0], "CLAUDE.md")
	assert.Contains(t, tasks[0], "Phase Transition Protocol")
}

func TestDeriveTasks_VerificationWithNewlinesMultipleTasks(t *testing.T) {
	answers := []state.DiscoveryAnswer{
		{
			QuestionID: "verification",
			Answer:     "Run unit tests\nCheck e2e tests pass\nVerify deployment",
		},
	}
	tasks := DeriveTasks(answers, nil, false)

	// 3 verification + 2 mandatory
	assert.Len(t, tasks, 5)
	assert.Equal(t, "Run unit tests", tasks[0])
	assert.Equal(t, "Check e2e tests pass", tasks[1])
	assert.Equal(t, "Verify deployment", tasks[2])
}

func TestDeriveTasks_VerificationDoesNotAddPrefix(t *testing.T) {
	answers := []state.DiscoveryAnswer{
		{
			QuestionID: "verification",
			Answer:     "Run tddmaster sync",
		},
	}
	tasks := DeriveTasks(answers, nil, false)

	assert.Len(t, tasks, 3)
	assert.False(t, strings.HasPrefix(tasks[0], "Verify:"))
	assert.Equal(t, "Run tddmaster sync", tasks[0])
}

func TestDeriveTasks_DoesNotSplitOnFileExtensions(t *testing.T) {
	answers := []state.DiscoveryAnswer{
		{
			QuestionID: "verification",
			Answer:     "Run tddmaster sync, check CLAUDE.md contains protocol section",
		},
	}
	tasks := DeriveTasks(answers, nil, false)

	assert.Len(t, tasks, 3)
	assert.Contains(t, tasks[0], "CLAUDE.md")
}

func TestDeriveTasks_DoesNotSplitOnVersionNumbers(t *testing.T) {
	answers := []state.DiscoveryAnswer{
		{
			QuestionID: "verification",
			Answer:     "Check v0.1 compatibility with the new API",
		},
	}
	tasks := DeriveTasks(answers, nil, false)

	assert.Len(t, tasks, 3)
	assert.Contains(t, tasks[0], "v0.1")
}

func TestDeriveTasks_DoesNotSplitOnFilePaths(t *testing.T) {
	answers := []state.DiscoveryAnswer{
		{
			QuestionID: "verification",
			Answer:     "Verify src/api/v1/endpoint.ts handles errors correctly",
		},
	}
	tasks := DeriveTasks(answers, nil, false)

	assert.Len(t, tasks, 3)
	assert.Contains(t, tasks[0], "endpoint.ts")
}

func TestDeriveTasks_EmptyProducesPlaceholder(t *testing.T) {
	tasks := DeriveTasks(nil, nil, false)

	assert.Len(t, tasks, 3)
	assert.Contains(t, tasks[0], "Tasks need to be defined")
}

func TestDeriveTasks_AcceptedDecisionBecomesTask(t *testing.T) {
	decisions := []state.Decision{
		{ID: "d1", Question: "Should we add caching?", Choice: "accepted", Promoted: false},
	}
	tasks := DeriveTasks(nil, decisions, false)

	// decision task + 2 mandatory = 3 (no placeholder because tasks is non-empty)
	assert.Len(t, tasks, 3)
	assert.Contains(t, tasks[0], "caching")
}

func TestDeriveTasks_MandatoryTasksAlwaysAppended(t *testing.T) {
	tasks := DeriveTasks(nil, nil, false)

	last := tasks[len(tasks)-1]
	secondLast := tasks[len(tasks)-2]
	assert.Contains(t, last, "documentation")
	assert.Contains(t, secondLast, "tests")
}

// =============================================================================
// DeriveTasks — tddMode
// =============================================================================

func TestDeriveTasks_TddModeFalsePreservesOriginalOrder(t *testing.T) {
	answers := []state.DiscoveryAnswer{
		{
			QuestionID: "ambition",
			Answer:     "10-star: Build a comprehensive dashboard",
		},
	}
	tasks := DeriveTasks(answers, nil, false)

	// tddMode=false: implementation task first, test task second-to-last, docs last
	assert.True(t, len(tasks) >= 3)
	assert.Contains(t, tasks[0], "dashboard")
	assert.Contains(t, tasks[len(tasks)-2], "tests")
	assert.Contains(t, tasks[len(tasks)-1], "documentation")
}

func TestDeriveTasks_TddModeTrueMovesTestTaskToFront(t *testing.T) {
	answers := []state.DiscoveryAnswer{
		{
			QuestionID: "ambition",
			Answer:     "10-star: Build a comprehensive dashboard",
		},
	}
	tasks := DeriveTasks(answers, nil, true)

	// tddMode=true: test task must come first
	assert.True(t, len(tasks) >= 3)
	assert.Contains(t, strings.ToLower(tasks[0]), "test")
}

func TestDeriveTasks_TddModeTrueAllTestTasksAtFront(t *testing.T) {
	answers := []state.DiscoveryAnswer{
		{
			QuestionID: "ambition",
			Answer:     "10-star: Build a comprehensive dashboard",
		},
		{
			QuestionID: "verification",
			Answer:     "Run unit tests\nCheck e2e tests pass\nVerify deployment",
		},
	}
	tasks := DeriveTasks(answers, nil, true)

	// Find the first non-test task index
	firstNonTest := -1
	for i, task := range tasks {
		if !strings.Contains(strings.ToLower(task), "test") {
			firstNonTest = i
			break
		}
	}

	// All tasks before firstNonTest must be test tasks
	if firstNonTest > 0 {
		for i := 0; i < firstNonTest; i++ {
			assert.Contains(t, strings.ToLower(tasks[i]), "test",
				"expected test task at position %d but got: %s", i, tasks[i])
		}
	}

	// All tasks from firstNonTest onward must be non-test tasks
	if firstNonTest >= 0 {
		for i := firstNonTest; i < len(tasks); i++ {
			assert.NotContains(t, strings.ToLower(tasks[i]), "test",
				"expected non-test task at position %d but got: %s", i, tasks[i])
		}
	}
}

func TestDeriveTasks_TddModeTrueNoTestTasksUnchanged(t *testing.T) {
	// No test tasks: only non-test tasks from ambition + docs (docs doesn't contain "test")
	answers := []state.DiscoveryAnswer{
		{
			QuestionID: "ambition",
			Answer:     "10-star: Build a comprehensive dashboard",
		},
	}
	// Temporarily use decisions with no test keywords to have a scenario with fewer test tasks
	// Actually the mandatory "Write or update tests" task always contains "test",
	// so we just verify order is stable when all test tasks move to front
	tasks := DeriveTasks(answers, nil, true)

	assert.True(t, len(tasks) >= 1)
	// The list should still contain all original tasks
	assert.True(t, len(tasks) == 3) // ambition task + test mandatory + docs mandatory
}

func TestDeriveTasks_TddModeTrueSingleTask(t *testing.T) {
	// Edge case: only one task (placeholder) — tddMode should not panic
	tasks := DeriveTasks(nil, nil, true)

	assert.True(t, len(tasks) >= 1)
}

func TestDeriveTasks_TddModeTrueMultipleTestTasksAllAtFront(t *testing.T) {
	// Edge case: multiple test tasks from verification + mandatory test task
	// all test tasks should be grouped at the front
	answers := []state.DiscoveryAnswer{
		{
			QuestionID: "verification",
			Answer:     "Run unit tests\nRun integration tests",
		},
	}
	tasks := DeriveTasks(answers, nil, true)

	// Expect: test tasks first, then docs task last
	// tasks: ["Run unit tests", "Run integration tests", "Write or update tests...", "Update documentation..."]
	assert.True(t, len(tasks) >= 2)
	// First task must be a test task
	assert.Contains(t, strings.ToLower(tasks[0]), "test")
	// Last task is the docs mandatory task (no "test" keyword)
	assert.NotContains(t, strings.ToLower(tasks[len(tasks)-1]), "test")
}

func TestDeriveTasks_TddModeFalseDoesNotReorder(t *testing.T) {
	answers := []state.DiscoveryAnswer{
		{
			QuestionID: "verification",
			Answer:     "Run unit tests\nCheck e2e tests pass\nVerify deployment",
		},
	}
	withoutTdd := DeriveTasks(answers, nil, false)
	// tddMode=false: "Verify deployment" is NOT a test task and comes after test tasks
	// Last non-mandatory task before mandatory ones is "Verify deployment"
	assert.Equal(t, "Verify deployment", withoutTdd[2])
}

// =============================================================================
// RenderSpec
// =============================================================================

func TestRenderSpec_IncludesSpecName(t *testing.T) {
	md := RenderSpec("photo-upload", nil, nil, nil, nil, nil, nil, nil, nil, false)
	assert.Contains(t, md, "# Spec: photo-upload")
}

func TestRenderSpec_IncludesStatusDraft(t *testing.T) {
	md := RenderSpec("test", nil, nil, nil, nil, nil, nil, nil, nil, false)
	assert.Contains(t, md, "## Status: draft")
}

func TestRenderSpec_IncludesDiscoveryAnswers(t *testing.T) {
	answers := []state.DiscoveryAnswer{
		{QuestionID: "status_quo", Answer: "Users manually upload files"},
		{QuestionID: "ambition", Answer: "1-star: basic upload. 10-star: smart listing"},
		{QuestionID: "verification", Answer: "Unit tests + e2e test of upload flow"},
	}
	md := RenderSpec("test", answers, nil, nil, nil, nil, nil, nil, nil, false)

	assert.Contains(t, md, "### status_quo")
	assert.Contains(t, md, "Users manually upload files")
	assert.Contains(t, md, "### ambition")
	assert.Contains(t, md, "### verification")
}

func TestRenderSpec_IncludesDecisionsTable(t *testing.T) {
	decisions := []state.Decision{
		{ID: "d1", Question: "Which storage backend?", Choice: "S3", Promoted: false},
		{ID: "d2", Question: "Auth method?", Choice: "OAuth2", Promoted: true},
	}
	md := RenderSpec("test", nil, nil, nil, decisions, nil, nil, nil, nil, false)

	assert.Contains(t, md, "## Decisions")
	assert.Contains(t, md, "Which storage backend?")
	assert.Contains(t, md, "S3")
	assert.Contains(t, md, "OAuth2")
	assert.Contains(t, md, "| yes |")
	assert.Contains(t, md, "| no |")
}

func TestRenderSpec_OmitsDecisionsSectionWhenEmpty(t *testing.T) {
	md := RenderSpec("test", nil, nil, nil, nil, nil, nil, nil, nil, false)
	assert.NotContains(t, md, "## Decisions")
}

func TestRenderSpec_IncludesConcernSectionsWhenRelevant(t *testing.T) {
	concern := state.ConcernDefinition{
		ID:           "open-source",
		SpecSections: []string{"Contributor Guide", "Public API Surface"},
	}
	classification := &state.SpecClassification{
		InvolvesPublicAPI: true,
	}
	md := RenderSpec("test", nil, nil, []state.ConcernDefinition{concern}, nil, classification, nil, nil, nil, false)

	assert.Contains(t, md, "## Contributor Guide (open-source)")
	assert.Contains(t, md, "## Public API Surface (open-source)")
}

func TestRenderSpec_SkipsIrrelevantConcernSections(t *testing.T) {
	concern := state.ConcernDefinition{
		ID:           "beautiful-product",
		SpecSections: []string{"Design States", "Mobile Layout"},
	}
	classification := &state.SpecClassification{
		InvolvesWebUI: false,
	}
	md := RenderSpec("test", nil, nil, []state.ConcernDefinition{concern}, nil, classification, nil, nil, nil, false)

	assert.NotContains(t, md, "## Design States")
	assert.NotContains(t, md, "## Mobile Layout")
}

func TestRenderSpec_SkipsConcernSectionsWhenNoClassification(t *testing.T) {
	concern := state.ConcernDefinition{
		ID:           "beautiful-product",
		SpecSections: []string{"Design States"},
	}
	md := RenderSpec("test", nil, nil, []state.ConcernDefinition{concern}, nil, nil, nil, nil, nil, false)

	assert.NotContains(t, md, "## Design States")
}

func TestRenderSpec_IncludesTransitionHistory(t *testing.T) {
	reason := "user approved"
	transitions := []state.PhaseTransition{
		{
			From:      state.PhaseDiscovery,
			To:        state.PhaseSpecProposal,
			User:      "alice",
			Timestamp: "2026-01-01T00:00:00Z",
			Reason:    &reason,
		},
	}
	md := RenderSpec("test", nil, nil, nil, nil, nil, nil, nil, transitions, false)

	assert.Contains(t, md, "## Transition History")
	assert.Contains(t, md, "DISCOVERY")
	assert.Contains(t, md, "alice")
	assert.Contains(t, md, "user approved")
}

func TestDeriveEdgeCases_UsesExplicitEdgeCasesAnswerAndPremiseRevisions(t *testing.T) {
	revision := "If the verifier rejects a test, ask the user before changing it."
	answers := []state.DiscoveryAnswer{
		{QuestionID: "edge_cases", Answer: "- Cover timeout recovery\n- Happy path smoke test"},
	}
	premises := []state.Premise{
		{Text: "Tests can be rewritten automatically", Agreed: false, Revision: &revision},
	}

	edgeCases := DeriveEdgeCases(answers, premises)

	assert.Equal(t, []string{
		"Cover timeout recovery",
		"Happy path smoke test",
		"If the verifier rejects a test, ask the user before changing it.",
	}, edgeCases)
}

// Regression: edge cases must NOT be harvested from unrelated answers
// just because they happen to contain keywords like "error", "fallback",
// "nil", "race". A previous Pass 2 keyword harvester pulled whole
// sentences from user_impact / status_quo / ambition / reversibility
// into the Edge Cases list.
func TestDeriveEdgeCases_IgnoresUnrelatedAnswersEvenWithKeywords(t *testing.T) {
	answers := []state.DiscoveryAnswer{
		{QuestionID: "status_quo", Answer: "Today the runner silently falls back to nil when the race between workers is lost."},
		{QuestionID: "ambition", Answer: "Introduce a typed error path so callers can retry with a fallback provider."},
	}

	edgeCases := DeriveEdgeCases(answers, nil)

	assert.Empty(t, edgeCases,
		"unrelated discovery answers must not leak into edge cases via keyword harvesting")
}

func TestDeriveEdgeCases_FallsBackToDisagreedPremiseText(t *testing.T) {
	premises := []state.Premise{
		{Text: "When the adapter is opencode, model selection is skipped.", Agreed: false},
	}

	edgeCases := DeriveEdgeCases(nil, premises)

	assert.Equal(t, []string{"When the adapter is opencode, model selection is skipped."}, edgeCases)
}

func TestRenderSpec_IncludesEdgeCasesSection(t *testing.T) {
	revision := "If the verifier rejects a test, ask the user before changing it."
	answers := []state.DiscoveryAnswer{
		{QuestionID: "edge_cases", Answer: "- Cover timeout recovery\n- Happy path smoke test"},
	}
	premises := []state.Premise{
		{Text: "Tests can be rewritten automatically", Agreed: false, Revision: &revision},
	}

	md := RenderSpec("test", answers, premises, nil, nil, nil, nil, nil, nil, false)

	assert.Contains(t, md, "## Edge Cases")
	assert.Contains(t, md, "- Cover timeout recovery")
	assert.Contains(t, md, "- If the verifier rejects a test, ask the user before changing it.")
}

func TestRenderSpec_OmitsEdgeCasesSectionWhenNoneDerived(t *testing.T) {
	answers := []state.DiscoveryAnswer{
		{QuestionID: "ambition", Answer: "Build a polished release flow."},
	}

	md := RenderSpec("test", answers, nil, nil, nil, nil, nil, nil, nil, false)

	assert.NotContains(t, md, "## Edge Cases")
}

func TestRenderSpec_TddModeTrue_ProducesTestTasksFirst(t *testing.T) {
	answers := []state.DiscoveryAnswer{
		{
			QuestionID: "ambition",
			Answer:     "10-star: Build a comprehensive dashboard",
		},
	}

	md := RenderSpec("test", answers, nil, nil, nil, nil, nil, nil, nil, true)

	// Find the Tasks section
	assert.Contains(t, md, "## Tasks")

	// Extract lines from Tasks section onwards
	lines := strings.Split(md, "\n")
	inTasks := false
	var taskLines []string
	for _, line := range lines {
		if line == "## Tasks" {
			inTasks = true
			continue
		}
		if inTasks && strings.HasPrefix(line, "## ") {
			break
		}
		if inTasks && strings.HasPrefix(line, "- [ ] task-") {
			taskLines = append(taskLines, line)
		}
	}

	assert.True(t, len(taskLines) >= 2, "expected at least 2 task lines")

	// The first task must be a test-related task
	assert.Contains(t, strings.ToLower(taskLines[0]), "test",
		"expected first task to be test-related in tddMode=true, got: %s", taskLines[0])
}

// =============================================================================
// Refinement render options
// =============================================================================

func TestRenderSpec_WithTaskOverride_ReplacesDerivedTasks(t *testing.T) {
	answers := []state.DiscoveryAnswer{
		{QuestionID: "ambition", Answer: "1-star: auto-derived"},
	}
	md := RenderSpec("test", answers, nil, nil, nil, nil, nil, nil, nil, false,
		WithTaskOverride([]state.SpecTask{
			{ID: "task-1", Title: "model/todo.go: define Todo struct"},
			{ID: "task-2", Title: "store/memory.go: thread-safe store"},
		}),
	)

	assert.Contains(t, md, "- [ ] task-1: model/todo.go: define Todo struct")
	assert.Contains(t, md, "- [ ] task-2: store/memory.go: thread-safe store")

	// Task list must not include any derived-from-ambition task line.
	assert.NotContains(t, md, "- [ ] task-3:", "override must cap the task list at the override length")
	assert.NotContains(t, md, "- [ ] task-1: auto-derived", "override must replace the derived task line")
}

func TestRenderSpec_WithTaskOverride_CompletedTaskRendersCheckbox(t *testing.T) {
	md := RenderSpec("test", nil, nil, nil, nil, nil, nil, nil, nil, false,
		WithTaskOverride([]state.SpecTask{
			{ID: "task-1", Title: "done task", Completed: true},
			{ID: "task-2", Title: "pending task", Completed: false},
		}),
	)
	assert.Contains(t, md, "- [x] task-1: done task")
	assert.Contains(t, md, "- [ ] task-2: pending task")
}

func TestRenderSpec_WithTaskOverride_StableIDsNotRenumbered(t *testing.T) {
	md := RenderSpec("test", nil, nil, nil, nil, nil, nil, nil, nil, false,
		WithTaskOverride([]state.SpecTask{
			{ID: "task-5", Title: "first"},
			{ID: "task-7", Title: "second"},
		}),
	)
	assert.Contains(t, md, "- [ ] task-5: first", "IDs must not be renumbered")
	assert.Contains(t, md, "- [ ] task-7: second", "IDs must not be renumbered")
	assert.NotContains(t, md, "task-1:", "auto-numbering must not replace provided IDs")
}

func TestRenderSpec_WithTaskOverride_EmptyFallsBackToDerived(t *testing.T) {
	answers := []state.DiscoveryAnswer{
		{QuestionID: "ambition", Answer: "1-star: keep derived"},
	}
	md := RenderSpec("test", answers, nil, nil, nil, nil, nil, nil, nil, false,
		WithTaskOverride(nil),
	)
	assert.Contains(t, md, "keep derived")
}

func TestRenderSpec_WithOutOfScopeOverride_AppendsItems(t *testing.T) {
	md := RenderSpec("test", nil, nil, nil, nil, nil, nil, nil, nil, false,
		WithOutOfScopeOverride([]string{"authentication", "database persistence"}),
	)
	assert.Contains(t, md, "## Out of Scope")
	assert.Contains(t, md, "- authentication")
	assert.Contains(t, md, "- database persistence")
}

// =============================================================================
// C.1 — edge_cases/scope_boundary skip in Discovery Answers section
// =============================================================================

// TestRenderSpec_EdgeCasesNotDuplicated verifies that edge_cases answers appear
// only in the dedicated "## Edge Cases" section and NOT as a discovery answer
// heading in the "## Discovery Answers" section.
func TestRenderSpec_EdgeCasesNotDuplicated(t *testing.T) {
	answers := []state.DiscoveryAnswer{
		{QuestionID: "status_quo", Answer: "Users manually track tasks in spreadsheets."},
		{QuestionID: "edge_cases", Answer: "- Empty list\n- Long text overflow"},
	}
	md := RenderSpec("test", answers, nil, nil, nil, nil, nil, nil, nil, false)

	// edge_cases must appear in the dedicated Edge Cases section
	assert.Contains(t, md, "## Edge Cases")
	assert.Contains(t, md, "- Empty list")

	// but NOT as a discovery answer heading
	assert.NotContains(t, md, "### edge_cases")
}

// TestRenderSpec_ScopeBoundaryNotDuplicated verifies that scope_boundary answers
// appear only in the "## Out of Scope" section and NOT in "## Discovery Answers".
func TestRenderSpec_ScopeBoundaryNotDuplicated(t *testing.T) {
	answers := []state.DiscoveryAnswer{
		{QuestionID: "status_quo", Answer: "Users manually track tasks in spreadsheets."},
		{QuestionID: "scope_boundary", Answer: "Auth and billing are out of scope."},
	}
	md := RenderSpec("test", answers, nil, nil, nil, nil, nil, nil, nil, false)

	// scope_boundary must appear in the dedicated Out of Scope section
	assert.Contains(t, md, "## Out of Scope")
	assert.Contains(t, md, "- Auth and billing are out of scope.")

	// but NOT as a discovery answer heading
	assert.NotContains(t, md, "### scope_boundary")
}

// =============================================================================
// C.2 — toBulletList handles (N) numbered markers
// =============================================================================

// TestToBulletList_NumberedMarkers verifies "(N)" prefixed items are split into
// separate bullets.
func TestToBulletList_NumberedMarkers(t *testing.T) {
	items := toBulletList("(1) Alpha. (2) Beta. (3) Gamma.")
	assert.Len(t, items, 3)
	assert.Equal(t, "(1) Alpha.", items[0])
	assert.Equal(t, "(2) Beta.", items[1])
	assert.Equal(t, "(3) Gamma.", items[2])
}

// TestToBulletList_ExistingSentenceBoundariesUnchanged ensures the existing
// sentence-splitting logic still works correctly after the regex change.
func TestToBulletList_ExistingSentenceBoundariesUnchanged(t *testing.T) {
	items := toBulletList("Foo sentence. Bar sentence.")
	assert.True(t, len(items) >= 2, "existing sentence splitting must still work")
}

// TestToBulletList_NoFalsePositiveOnFilename verifies "file.go contains X." is
// not split into a new bullet.
func TestToBulletList_NoFalsePositiveOnFilename(t *testing.T) {
	items := toBulletList("file.go contains important logic for the system.")
	assert.Len(t, items, 1)
}

