package loop

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/promptregistry"
	"github.com/pragmataW/tddmaster/internal/prompts"
	"github.com/pragmataW/tddmaster/internal/spec"
)

var sectionTokenRe = regexp.MustCompile(`(?m)^=== [A-Z][A-Z ()a-z-]* ===$`)

func fullExecCtx(cycle string) ExecCtx {
	return ExecCtx{
		Settings: spec.Settings{
			TDDEnabled:               true,
			ImportantTaskGateEnabled: true,
			MinTestCoverage:          80,
		},
		Slug: "demo-spec",
		Task: spec.Task{
			ID:         "task-1",
			Title:      "Add token rotation",
			TDDEnabled: true,
			Important:  true,
			DependsOn:  []string{"task-0"},
			Criteria: []spec.Criterion{
				{ID: "ac-1", Given: "a stale token", When: "refresh runs", Then: "a new token is issued"},
			},
			EdgeCases: []string{"empty token", "concurrent refresh"},
		},
		State: spec.ExecState{
			TDDCycle:          cycle,
			Plan:              &spec.TaskPlan{Approach: "extend the store", TouchedFiles: []string{"internal/auth/store.go"}},
			LastModifiedFiles: []string{"internal/auth/store.go"},
			LastFailedACs:     []string{"ac-1"},
			LastUncoveredEC:   []string{"ec-2"},
			RefactorNotes:     []spec.RefactorNote{{File: "internal/auth/store.go", Suggestion: "extract validation"}},
			LastCoverage:      map[string]float64{"internal/auth/store.go": 40},
		},
		Mode:        "ship-fast",
		UserContext: "rotate tokens without logging anybody out",
		ProjectCommands: []ProjectCommand{
			{Label: "build", Value: "go build ./..."},
			{Label: "test", Value: "go test ./..."},
		},
		ScopeBoundary:    "do not touch the billing module",
		VerificationHint: "integration test against the staging IdP",
		Intent: SpecIntent{
			StatusQuo:     "operators rotate tokens by hand",
			Ambition:      "1-star: a manual command. 10-star: fully automatic rotation",
			Reversibility: "the storage format change is irreversible once written",
			UserImpact:    "existing sessions must keep working",
			ChallengedPremises: []string{
				"tokens can be rotated offline -> REVISED: rotation must happen live",
			},
		},
		Overview: []TaskOverview{
			{ID: "task-0", Title: "Token store scaffolding", Status: "done"},
			{ID: "task-1", Title: "Add token rotation", Status: "ready", Self: true},
			{ID: "task-2", Title: "Audit log", Status: "ready"},
		},
		Trace: []TraceRef{
			{TestFilePath: "internal/auth/store_test.go", FunctionName: "TestRotate_ac1", AC: []string{"ac-1"}, EC: []string{"ec-1"}},
		},
		Siblings: []SiblingTask{{ID: "task-2", Title: "Audit log", Files: []string{"internal/audit/log.go"}}},
	}
}

func TestPromptContract_EveryStageCarriesTaskIdentity(t *testing.T) {
	cases := []struct {
		stage Stage
		cycle string
	}{
		{gateStage(), cycleRed},
		{redStage(), cycleRed},
		{greenStage(), cycleGreen},
		{refactorStage(), cycleRefactor},
		{executorStage(), cycleRefactor},
		{verifierStage(), cycleGreen},
	}

	for _, tc := range cases {
		t.Run(tc.stage.ID(), func(t *testing.T) {
			ctx := fullExecCtx(tc.cycle)
			got := tc.stage.Prompt(ctx).Instruction

			required := []string{
				promptregistry.SectionTask,
				promptregistry.SectionSpecOverview,
				promptregistry.SectionSpecContext,
				promptregistry.SectionSpecIntent,
				promptregistry.SectionCriteria,
				promptregistry.SectionEdgeCases,
				promptregistry.SectionOutOfScope,
				promptregistry.SectionVerification,
				promptregistry.SectionInstruction,
			}
			for _, section := range required {
				if !strings.Contains(got, section) {
					t.Fatalf("stage %s: missing section %q in prompt:\n%s", tc.stage.ID(), section, got)
				}
			}

			for _, want := range []string{ctx.Slug, ctx.Task.ID, ctx.Task.Title, ctx.ScopeBoundary, ctx.UserContext} {
				if !strings.Contains(got, want) {
					t.Fatalf("stage %s: missing %q in prompt:\n%s", tc.stage.ID(), want, got)
				}
			}
		})
	}
}

func TestPromptContract_StageSectionMatrix(t *testing.T) {
	cases := []struct {
		name  string
		stage Stage
		cycle string
		want  []string
	}{
		{"red", redStage(), cycleRed, []string{
			promptregistry.SectionVerification,
			promptregistry.SectionTraceability,
			promptregistry.SectionLastFailure,
			promptregistry.SectionParallelTasks,
		}},
		{"green", greenStage(), cycleGreen, []string{
			promptregistry.SectionVerification,
			promptregistry.SectionChangedFiles,
			promptregistry.SectionTraceability,
			promptregistry.SectionLastFailure,
			promptregistry.SectionParallelTasks,
		}},
		{"refactor", refactorStage(), cycleRefactor, []string{
			promptregistry.SectionVerification,
			promptregistry.SectionChangedFiles,
			promptregistry.SectionTraceability,
			promptregistry.SectionRefactorNotes,
			promptregistry.SectionParallelTasks,
		}},
		{"executor", executorStage(), cycleGreen, []string{
			promptregistry.SectionVerification,
			promptregistry.SectionLastFailure,
			promptregistry.SectionParallelTasks,
		}},
		{"verifier", verifierStage(), cycleGreen, []string{
			promptregistry.SectionVerification,
			promptregistry.SectionChangedFiles,
			promptregistry.SectionTraceability,
			promptregistry.SectionParallelTasks,
			promptregistry.SectionCoverage,
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.stage.Prompt(fullExecCtx(tc.cycle)).Instruction
			for _, section := range tc.want {
				if !strings.Contains(got, section) {
					t.Fatalf("stage %s: missing section %q in prompt:\n%s", tc.name, section, got)
				}
			}
		})
	}
}

func TestPromptContract_RedTellsTestWriterNotToWriteProductionCode(t *testing.T) {
	got := redStage().Prompt(fullExecCtx(cycleRed)).Instruction
	if !strings.Contains(got, promptregistry.RedVerifyFailedNote) {
		t.Fatalf("red prompt does not carry the test-writer-specific failure note:\n%s", got)
	}
	if strings.Contains(got, instructionFor(promptregistry.KeyExecVerifyFailed)) {
		t.Fatalf("red prompt reuses the executor failure note, which orders production-code fixes:\n%s", got)
	}
}

func TestPromptContract_GreenNamesTheRedTestFiles(t *testing.T) {
	got := greenStage().Prompt(fullExecCtx(cycleGreen)).Instruction
	for _, want := range []string{promptregistry.GreenChangedFilesNote, "internal/auth/store.go"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in green prompt:\n%s", want, got)
		}
	}
}

func TestPromptContract_GreenRetryStillNamesTheRedTestFiles(t *testing.T) {
	ctx := fullExecCtx(cycleGreen)
	ctx.State.TestFiles = []string{"internal/auth/store_test.go"}
	ctx.State.LastModifiedFiles = []string{"internal/auth/store.go"}

	got := greenStage().Prompt(ctx).Instruction
	for _, want := range []string{"internal/auth/store_test.go", "internal/auth/store.go"} {
		if !strings.Contains(got, want) {
			t.Fatalf("green retry prompt dropped %q; the failing tests must stay visible:\n%s", want, got)
		}
	}
}

func TestPromptContract_GreenDoesNotRepeatAFileListedTwice(t *testing.T) {
	ctx := fullExecCtx(cycleGreen)
	ctx.State.TestFiles = []string{"internal/auth/store_test.go"}
	ctx.State.LastModifiedFiles = []string{"internal/auth/store_test.go", "internal/auth/store.go"}

	got := greenStage().Prompt(ctx).Instruction
	if n := strings.Count(got, "- internal/auth/store_test.go\n"); n != 1 {
		t.Fatalf("test file listed %d times in green changed-files, want 1:\n%s", n, got)
	}
}

func TestPromptContract_RedReportPersistsTestFilesAcrossGreen(t *testing.T) {
	ctx := fullExecCtx(cycleRed)
	ctx.State.TestFiles = nil
	ctx.State.LastModifiedFiles = nil

	afterRed, err := redStage().OnReport(ctx, StageReport{
		TestsWritten:  []string{"TestRotate_ac1"},
		FilesModified: []string{"internal/auth/store_test.go"},
	})
	if err != nil {
		t.Fatalf("red OnReport: %v", err)
	}

	afterGreen, err := greenStage().OnReport(afterRed, StageReport{
		FilesModified: []string{"internal/auth/store.go"},
	})
	if err != nil {
		t.Fatalf("green OnReport: %v", err)
	}
	if !reflect.DeepEqual(afterGreen.State.TestFiles, []string{"internal/auth/store_test.go"}) {
		t.Fatalf("GREEN overwrote the RED test file list: %v", afterGreen.State.TestFiles)
	}
}

func TestPromptContract_ChallengedPremisesReachEveryStage(t *testing.T) {
	for _, st := range allStages() {
		cycle := cycleGreen
		if st.ID() == StageIDRefactor {
			cycle = cycleRefactor
		}
		got := st.Prompt(fullExecCtx(cycle)).Instruction
		for _, want := range []string{
			promptregistry.ChallengedPremisesLabel,
			"tokens can be rotated offline -> REVISED: rotation must happen live",
		} {
			if !strings.Contains(got, want) {
				t.Fatalf("stage %s: missing rejected premise %q:\n%s", st.ID(), want, got)
			}
		}
	}
}

func TestPromptContract_DiscoveryModeReachesEveryStage(t *testing.T) {
	for _, st := range allStages() {
		cycle := cycleGreen
		if st.ID() == StageIDRefactor {
			cycle = cycleRefactor
		}
		got := st.Prompt(fullExecCtx(cycle)).Instruction
		if !strings.Contains(got, "discoveryMode: ship-fast") {
			t.Fatalf("stage %s: missing discoveryMode in the task header:\n%s", st.ID(), got)
		}
		if !strings.Contains(got, promptregistry.ModeDescription("ship-fast")) {
			t.Fatalf("stage %s: discoveryMode carries no explanation:\n%s", st.ID(), got)
		}
	}
}

func TestPromptContract_ProjectCommandsReachTestRunningStages(t *testing.T) {
	verifierPrompt := verifierStage().Prompt(fullExecCtx(cycleGreen)).Instruction
	for _, want := range []string{promptregistry.SectionProjectCmds, "test: go test ./..."} {
		if !strings.Contains(verifierPrompt, want) {
			t.Fatalf("verifier prompt missing %q:\n%s", want, verifierPrompt)
		}
	}

	ctx := fullExecCtx(cycleGreen)
	ctx.Settings.TDDEnabled = false
	ctx.Task.TDDEnabled = false
	ctx.Settings.SkipVerifierEnabled = true
	if got := executorStage().Prompt(ctx).Instruction; !strings.Contains(got, promptregistry.SectionProjectCmds) {
		t.Fatalf("executor runs the tests itself when the verifier is skipped, so it needs the commands:\n%s", got)
	}
}

func TestPromptContract_ProjectCommandsStayOutOfStagesThatMustNotRunTests(t *testing.T) {
	for _, tc := range []struct {
		stage Stage
		cycle string
	}{
		{redStage(), cycleRed},
		{greenStage(), cycleGreen},
		{gateStage(), cycleRed},
	} {
		got := tc.stage.Prompt(fullExecCtx(tc.cycle)).Instruction
		if strings.Contains(got, promptregistry.SectionProjectCmds) {
			t.Fatalf("stage %s is forbidden from running tests but carries project commands:\n%s", tc.stage.ID(), got)
		}
	}
}

func TestPromptContract_SpecIntentReachesEveryStage(t *testing.T) {
	for _, st := range allStages() {
		cycle := cycleGreen
		if st.ID() == StageIDRefactor {
			cycle = cycleRefactor
		}
		got := st.Prompt(fullExecCtx(cycle)).Instruction
		for _, want := range []string{
			"operators rotate tokens by hand",
			"the storage format change is irreversible once written",
			"existing sessions must keep working",
		} {
			if !strings.Contains(got, want) {
				t.Fatalf("stage %s: missing discovery answer %q:\n%s", st.ID(), want, got)
			}
		}
	}
}

func TestPromptContract_SpecOverviewMarksTheOwningTask(t *testing.T) {
	got := greenStage().Prompt(fullExecCtx(cycleGreen)).Instruction
	for _, want := range []string{
		"[done] task-0: Token store scaffolding",
		"[ready] task-1: Add token rotation  <- YOU",
		"[ready] task-2: Audit log",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing overview row %q in green prompt:\n%s", want, got)
		}
	}
}

func TestPromptContract_TraceabilityReachesTestAwareStages(t *testing.T) {
	for _, tc := range []struct {
		stage Stage
		cycle string
	}{
		{redStage(), cycleRed},
		{greenStage(), cycleGreen},
		{refactorStage(), cycleRefactor},
		{verifierStage(), cycleGreen},
	} {
		got := tc.stage.Prompt(fullExecCtx(tc.cycle)).Instruction
		if !strings.Contains(got, "internal/auth/store_test.go::TestRotate_ac1 covers ac-1, ec-1") {
			t.Fatalf("stage %s: traceability row missing:\n%s", tc.stage.ID(), got)
		}
	}
}

func TestPromptContract_ReportExamplesUseLowercaseIDs(t *testing.T) {
	for _, example := range []string{
		promptregistry.ReportExampleTestWriter,
		promptregistry.ReportExampleVerifier,
	} {
		for _, stale := range []string{`"AC-`, `"EC-`} {
			if strings.Contains(example, stale) {
				t.Fatalf("report example uses uppercase id %q, contradicting the lowercase rule: %s", stale, example)
			}
		}
	}
}

func TestPromptContract_UncoveredEdgeCasesReturnToRed(t *testing.T) {
	ctx := fullExecCtx(cycleGreen)
	ctx.State.Implemented = true
	ctx.State.LastFailedACs = nil
	ctx.State.LastUncoveredEC = nil

	newCtx, err := verifierStage().OnReport(ctx, StageReport{
		Passed:             true,
		Phase:              cycleGreen,
		UncoveredEdgeCases: []string{"ec-2"},
	})
	if err != nil {
		t.Fatalf("OnReport: %v", err)
	}
	if newCtx.State.TDDCycle != cycleRed {
		t.Fatalf("uncovered edge cases must route back to RED (test-writer), got cycle %q", newCtx.State.TDDCycle)
	}
	if newCtx.State.Implemented {
		t.Fatal("implemented flag must be cleared when verification fails")
	}
}

func TestPromptContract_RedReportSeedsChangedFilesForGreen(t *testing.T) {
	ctx := fullExecCtx(cycleRed)
	ctx.State.LastModifiedFiles = nil

	newCtx, err := redStage().OnReport(ctx, StageReport{
		TestsWritten:  []string{"TestRotate_ac1"},
		FilesModified: []string{"internal/auth/store_test.go"},
	})
	if err != nil {
		t.Fatalf("OnReport: %v", err)
	}
	if !reflect.DeepEqual(newCtx.State.LastModifiedFiles, []string{"internal/auth/store_test.go"}) {
		t.Fatalf("RED must persist its test files for GREEN, got %v", newCtx.State.LastModifiedFiles)
	}
}

func TestPromptContract_InstructionSectionIsLast(t *testing.T) {
	for _, st := range allStages() {
		ctx := fullExecCtx(cycleGreen)
		if st.ID() == StageIDRefactor {
			ctx = fullExecCtx(cycleRefactor)
		}
		got := st.Prompt(ctx).Instruction
		tokens := sectionTokenRe.FindAllString(got, -1)
		if len(tokens) == 0 {
			t.Fatalf("stage %s: no section tokens in prompt:\n%s", st.ID(), got)
		}
		if last := tokens[len(tokens)-1]; last != promptregistry.SectionInstruction {
			t.Fatalf("stage %s: last section is %q, want %q", st.ID(), last, promptregistry.SectionInstruction)
		}
	}
}

func TestPromptContract_EdgeCasesCarryCanonicalIDs(t *testing.T) {
	got := redStage().Prompt(fullExecCtx(cycleRed)).Instruction
	for _, want := range []string{"[ec-1] empty token", "[ec-2] concurrent refresh", "[ac-1]"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in red prompt:\n%s", want, got)
		}
	}
}

func TestPromptContract_VerifierReceivesChangedFilesAndPlan(t *testing.T) {
	got := verifierStage().Prompt(fullExecCtx(cycleGreen)).Instruction
	for _, want := range []string{
		promptregistry.SectionChangedFiles,
		promptregistry.SectionApprovedPlan,
		promptregistry.SectionVerification,
		"internal/auth/store.go",
		"integration test against the staging IdP",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in verifier prompt:\n%s", want, got)
		}
	}
}

func TestPromptContract_GateReceivesFullTaskScope(t *testing.T) {
	ctx := fullExecCtx(cycleRed)
	ctx.State.PlanFeedback = "touched files are too broad"
	ctx.State.PlanAttempts = 2
	got := gateStage().Prompt(ctx).Instruction

	for _, want := range []string{
		promptregistry.SectionPriorFeedback,
		promptregistry.SectionParallelTasks,
		"touched files are too broad",
		"attemptCount: 2",
		"Add token rotation",
		"a new token is issued",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in gate prompt:\n%s", want, got)
		}
	}
}

func TestPromptContract_SiblingFilesAreNamedAsOffLimits(t *testing.T) {
	got := greenStage().Prompt(fullExecCtx(cycleGreen)).Instruction
	for _, want := range []string{promptregistry.SectionParallelTasks, "task-2", "internal/audit/log.go"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in green prompt:\n%s", want, got)
		}
	}
}

func TestPromptContract_EveryEmittedSectionIsDeclared(t *testing.T) {
	declared := make(map[string]bool, len(promptregistry.AllSections))
	for _, s := range promptregistry.AllSections {
		declared[s] = true
	}

	for _, st := range allStages() {
		for _, cycle := range []string{cycleRed, cycleGreen, cycleRefactor} {
			got := st.Prompt(fullExecCtx(cycle)).Instruction
			for _, token := range sectionTokenRe.FindAllString(got, -1) {
				if !declared[token] {
					t.Fatalf("stage %s cycle %s: emits undeclared section %q", st.ID(), cycle, token)
				}
			}
		}
	}
}

func TestPromptContract_TemplatesOnlyReferenceDeclaredSections(t *testing.T) {
	declared := make(map[string]bool, len(promptregistry.AllSections))
	for _, s := range promptregistry.AllSections {
		declared[s] = true
	}

	for _, name := range prompts.TemplateNames() {
		rendered, err := prompts.Render(name, prompts.RenderData{})
		if err != nil {
			t.Fatalf("render %s: %v", name, err)
		}
		for _, token := range sectionTokenRe.FindAllString(rendered, -1) {
			if !declared[token] {
				t.Fatalf("template %s references undeclared section %q", name, token)
			}
		}
	}
}

func TestPromptContract_TestWriterTemplateMatchesTraceabilitySchema(t *testing.T) {
	rendered, err := prompts.Render("test-writer", prompts.RenderData{})
	if err != nil {
		t.Fatalf("render test-writer: %v", err)
	}

	entry := reflect.TypeOf(TraceReportEntry{})
	for i := range entry.NumField() {
		tag := strings.Split(entry.Field(i).Tag.Get("json"), ",")[0]
		if tag == "" || tag == "-" {
			continue
		}
		if !strings.Contains(rendered, tag) {
			t.Fatalf("test-writer template does not mention required traceability field %q", tag)
		}
	}

	for _, stale := range []string{`"test":`, `"id":`, `"given":`} {
		if strings.Contains(rendered, stale) {
			t.Fatalf("test-writer template still documents the stale traceability key %q", stale)
		}
	}
}

func TestPromptContract_TemplatesAreLanguageAgnostic(t *testing.T) {
	for _, name := range prompts.TemplateNames() {
		if name == "claude_md" {
			continue
		}
		rendered, err := prompts.Render(name, prompts.RenderData{})
		if err != nil {
			t.Fatalf("render %s: %v", name, err)
		}
		lines := strings.Split(rendered, "\n")
		for i, line := range lines {
			if !strings.Contains(line, "go build ./...") && !strings.Contains(line, "go test ./...") {
				continue
			}
			if strings.Contains(line, "go.mod") {
				continue
			}
			t.Fatalf("template %s line %d hardcodes a Go command outside a detection table: %s", name, i+1, line)
		}
	}
}
