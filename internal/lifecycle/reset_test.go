package lifecycle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/phasecatalog"
	"github.com/pragmataW/tddmaster/internal/spec"
)

const fixtureSlug = "demo"

func fixtureAnswerKeys() []string {
	return []string{
		"spec_settings",
		"listen_context", "mode", "premises", "status_quo", "ambition",
		"reversibility", "user_impact", "verification", "scope_boundary",
		"edge_cases", "synthesis",
		"tasks_generated", "self_review",
		"refinement_approved",
		"analysis_complete", "analysis_audited", "analysis_findings", "analysis_attempts",
		"rule_proposal", "rule_approved", "rule_applied", "rule_feedback", "rule_attempt",
	}
}

func buildFullFixtureState() spec.State {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	answers := map[string][]spec.Answer{}
	for _, key := range fixtureAnswerKeys() {
		answers[key] = []spec.Answer{{Key: key, Value: "value-of-" + key}}
	}
	return spec.State{
		Version:   1,
		Slug:      fixtureSlug,
		Phase:     string(phasecatalog.PhaseExecution),
		Answers:   answers,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func buildFullFixtureProgress() spec.Progress {
	return spec.Progress{
		Spec:       fixtureSlug,
		Status:     spec.StatusExecuting,
		TaskSeq:    3,
		Iterations: 2,
		Tasks: []spec.Task{
			{
				ID:    "task-1",
				Title: "First task",
				Criteria: []spec.Criterion{
					{ID: "ac-1", Given: "g", When: "w", Then: "t"},
				},
				Done:       true,
				TDDEnabled: true,
				RefactorNotes: []spec.RefactorNote{
					{File: "f.go", Suggestion: "s", Rationale: "r"},
				},
				FailedACReasons: []string{"ac-2: failed"},
				Blocked:         true,
				BlockedReason:   "waiting on dep",
				Exec: &spec.ExecState{
					TDDCycle:    "green",
					Implemented: true,
				},
			},
		},
	}
}

func fixtureCustomSettings() spec.Settings {
	return spec.Settings{
		TDDEnabled:               false,
		SkipVerifierEnabled:      true,
		ImportantTaskGateEnabled: true,
		MinTestCoverage:          42,
		RuleLearningEnabled:      true,
	}
}

func writeFixtureFiles(t *testing.T, root, slug string) {
	t.Helper()
	if err := spec.SaveSettings(root, slug, fixtureCustomSettings()); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	if err := spec.SaveSpecMd(root, slug, "# Custom Spec\n\nBody."); err != nil {
		t.Fatalf("SaveSpecMd: %v", err)
	}
	if err := spec.SaveAnalysis(root, slug, spec.Analysis{
		Verdict: "approved",
		Findings: []spec.Finding{
			{Severity: spec.SeverityBlock, Category: "design", Detail: "issue", Source: spec.SourceAuditor},
		},
	}); err != nil {
		t.Fatalf("SaveAnalysis: %v", err)
	}
	if err := spec.SaveTraceability(root, slug, spec.Traceability{
		Entries: map[string][]spec.TraceEntry{
			"task-1": {{FunctionName: "TestFirst", TaskID: "task-1", CriterionIDs: []string{"ac-1"}, EC: []string{}}},
		},
		Coverage: map[string]map[string]float64{
			"task-1": {"ac-1": 100},
		},
	}); err != nil {
		t.Fatalf("SaveTraceability: %v", err)
	}
	ruleDir := paths.RulesAgentDir(root, "claude")
	if err := os.MkdirAll(ruleDir, 0o755); err != nil {
		t.Fatalf("MkdirAll rules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ruleDir, "global.md"), []byte("# shared rule\n"), 0o644); err != nil {
		t.Fatalf("write global rule: %v", err)
	}
}

func mustLoadSettings(t *testing.T, root, slug string) spec.Settings {
	t.Helper()
	s, err := spec.LoadSettings(root, slug)
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	return s
}

func mustLoadAnalysis(t *testing.T, root, slug string) spec.Analysis {
	t.Helper()
	a, err := spec.LoadAnalysis(root, slug)
	if err != nil {
		t.Fatalf("LoadAnalysis: %v", err)
	}
	return a
}

func mustLoadTraceability(t *testing.T, root, slug string) spec.Traceability {
	t.Helper()
	tr, err := spec.LoadTraceability(root, slug)
	if err != nil {
		t.Fatalf("LoadTraceability: %v", err)
	}
	return tr
}

func assertSpecMdDeleted(t *testing.T, root, slug string) {
	t.Helper()
	if _, err := os.Stat(paths.SpecMd(root, slug)); !os.IsNotExist(err) {
		t.Errorf("expected spec.md to be deleted, stat err = %v", err)
	}
}

func assertSpecMdPreserved(t *testing.T, root, slug string) {
	t.Helper()
	data, err := os.ReadFile(paths.SpecMd(root, slug))
	if err != nil {
		t.Fatalf("expected spec.md to be preserved, read err = %v", err)
	}
	if string(data) != "# Custom Spec\n\nBody." {
		t.Errorf("spec.md content changed unexpectedly: %q", string(data))
	}
}

func assertAnswerKeysAbsent(t *testing.T, state spec.State, keys ...string) {
	t.Helper()
	for _, key := range keys {
		if _, ok := state.Answers[key]; ok {
			t.Errorf("expected answer key %q to be cleared, still present", key)
		}
	}
}

func assertAnswerKeysPresent(t *testing.T, state spec.State, keys ...string) {
	t.Helper()
	for _, key := range keys {
		if _, ok := state.Answers[key]; !ok {
			t.Errorf("expected answer key %q to be preserved, missing", key)
		}
	}
}

func assertRuleFilePreserved(t *testing.T, root string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(paths.RulesAgentDir(root, "claude"), "global.md"))
	if err != nil {
		t.Fatalf("expected global rule file to be preserved, read err = %v", err)
	}
	if string(data) != "# shared rule\n" {
		t.Errorf("global rule file content changed unexpectedly: %q", string(data))
	}
}

func TestResetFrom_ac1_TargetSpecSettings_ClearsEveryPhaseNothingEarlierExists(t *testing.T) {
	root := t.TempDir()
	writeFixtureFiles(t, root, fixtureSlug)
	state := buildFullFixtureState()
	prog := buildFullFixtureProgress()

	warnings, err := ResetFrom(string(phasecatalog.PhaseSettings), &state, &prog, root, fixtureSlug)
	if err != nil {
		t.Fatalf("ResetFrom returned error: %v", err)
	}

	if got := mustLoadSettings(t, root, fixtureSlug); got != spec.DefaultSettings() {
		t.Errorf("expected settings reset to defaults, got %+v", got)
	}
	assertAnswerKeysAbsent(t, state, fixtureAnswerKeys()...)
	assertSpecMdDeleted(t, root, fixtureSlug)
	if len(prog.Tasks) != 0 {
		t.Errorf("expected Tasks emptied, got %d tasks", len(prog.Tasks))
	}
	if prog.TaskSeq != 0 {
		t.Errorf("expected TaskSeq reset to 0, got %d", prog.TaskSeq)
	}
	if prog.Status != spec.StatusDraft {
		t.Errorf("expected Status draft, got %q", prog.Status)
	}
	if a := mustLoadAnalysis(t, root, fixtureSlug); a.Verdict != "" || len(a.Findings) != 0 {
		t.Errorf("expected analysis reset, got %+v", a)
	}
	if tr := mustLoadTraceability(t, root, fixtureSlug); len(tr.Entries) != 0 {
		t.Errorf("expected traceability entries emptied, got %+v", tr.Entries)
	}
	if prog.Iterations != 0 {
		t.Errorf("expected Iterations reset to 0, got %d", prog.Iterations)
	}
	assertRuleFilePreserved(t, root)
	if len(warnings) == 0 {
		t.Errorf("expected at least one warning about untouched rule-learning global files")
	}
}

func TestResetFrom_ac1_TargetDiscovery_PreservesSettingsClearsDownstream(t *testing.T) {
	root := t.TempDir()
	writeFixtureFiles(t, root, fixtureSlug)
	state := buildFullFixtureState()
	prog := buildFullFixtureProgress()

	warnings, err := ResetFrom(string(phasecatalog.PhaseDiscovery), &state, &prog, root, fixtureSlug)
	if err != nil {
		t.Fatalf("ResetFrom returned error: %v", err)
	}

	if got := mustLoadSettings(t, root, fixtureSlug); got != fixtureCustomSettings() {
		t.Errorf("expected settings preserved, got %+v", got)
	}
	assertAnswerKeysPresent(t, state, "spec_settings")
	assertAnswerKeysAbsent(t, state,
		"listen_context", "mode", "premises", "status_quo", "ambition",
		"reversibility", "user_impact", "verification", "scope_boundary",
		"edge_cases", "synthesis",
	)
	assertSpecMdDeleted(t, root, fixtureSlug)
	if len(prog.Tasks) != 0 {
		t.Errorf("expected Tasks emptied, got %d tasks", len(prog.Tasks))
	}
	if len(warnings) == 0 {
		t.Errorf("expected at least one warning about untouched rule-learning global files")
	}
}

func TestResetFrom_ac1_TargetSpecProposal_DeletesSpecMdEmptiesTasksPreservesDiscovery(t *testing.T) {
	root := t.TempDir()
	writeFixtureFiles(t, root, fixtureSlug)
	state := buildFullFixtureState()
	prog := buildFullFixtureProgress()

	warnings, err := ResetFrom(string(phasecatalog.PhaseSpecProposal), &state, &prog, root, fixtureSlug)
	if err != nil {
		t.Fatalf("ResetFrom returned error: %v", err)
	}

	assertAnswerKeysPresent(t, state, "spec_settings", "listen_context", "mode", "synthesis")
	assertSpecMdDeleted(t, root, fixtureSlug)
	if len(prog.Tasks) != 0 {
		t.Errorf("expected Tasks emptied, got %d tasks", len(prog.Tasks))
	}
	if prog.TaskSeq != 0 {
		t.Errorf("expected TaskSeq reset to 0, got %d", prog.TaskSeq)
	}
	if prog.Status != spec.StatusDraft {
		t.Errorf("expected Status draft, got %q", prog.Status)
	}
	assertAnswerKeysAbsent(t, state, "tasks_generated", "self_review")
	if a := mustLoadAnalysis(t, root, fixtureSlug); a.Verdict != "" || len(a.Findings) != 0 {
		t.Errorf("expected analysis reset since downstream of spec-proposal, got %+v", a)
	}
	if len(warnings) == 0 {
		t.Errorf("expected at least one warning about untouched rule-learning global files")
	}
}

func TestResetFrom_ac1_TargetAnalysis_PreservesTaskListClearsVerdict(t *testing.T) {
	root := t.TempDir()
	writeFixtureFiles(t, root, fixtureSlug)
	state := buildFullFixtureState()
	prog := buildFullFixtureProgress()

	warnings, err := ResetFrom(string(phasecatalog.PhaseAnalysis), &state, &prog, root, fixtureSlug)
	if err != nil {
		t.Fatalf("ResetFrom returned error: %v", err)
	}

	assertSpecMdPreserved(t, root, fixtureSlug)
	assertAnswerKeysPresent(t, state, "spec_settings", "listen_context", "tasks_generated", "refinement_approved")
	if len(prog.Tasks) != 1 {
		t.Fatalf("expected task list preserved with 1 task, got %d", len(prog.Tasks))
	}
	if prog.Tasks[0].ID != "task-1" || prog.Tasks[0].Title != "First task" {
		t.Errorf("expected task identity preserved, got %+v", prog.Tasks[0])
	}
	if a := mustLoadAnalysis(t, root, fixtureSlug); a.Verdict != "" || len(a.Findings) != 0 {
		t.Errorf("expected analysis verdict cleared, got %+v", a)
	}
	assertAnswerKeysAbsent(t, state, "analysis_complete", "analysis_audited", "analysis_findings", "analysis_attempts")
	if prog.Tasks[0].Done {
		t.Errorf("expected task Done cleared since execution is downstream of analysis")
	}
	if prog.Tasks[0].Exec != nil {
		t.Errorf("expected task Exec cleared since execution is downstream of analysis")
	}
	if tr := mustLoadTraceability(t, root, fixtureSlug); len(tr.Entries) != 0 {
		t.Errorf("expected traceability emptied since execution is downstream of analysis, got %+v", tr.Entries)
	}
	if len(warnings) == 0 {
		t.Errorf("expected at least one warning about untouched rule-learning global files")
	}
}

func TestResetFrom_ac1_TargetExecution_PreservesTasksAndSpecMdClearsExecFields(t *testing.T) {
	root := t.TempDir()
	writeFixtureFiles(t, root, fixtureSlug)
	state := buildFullFixtureState()
	prog := buildFullFixtureProgress()

	warnings, err := ResetFrom(string(phasecatalog.PhaseExecution), &state, &prog, root, fixtureSlug)
	if err != nil {
		t.Fatalf("ResetFrom returned error: %v", err)
	}

	assertSpecMdPreserved(t, root, fixtureSlug)
	if len(prog.Tasks) != 1 {
		t.Fatalf("expected task list preserved with 1 task, got %d", len(prog.Tasks))
	}
	task := prog.Tasks[0]
	if task.ID != "task-1" || task.Title != "First task" {
		t.Errorf("expected task identity preserved, got %+v", task)
	}
	if task.Done {
		t.Errorf("expected Done cleared to false, got true")
	}
	if task.Exec != nil {
		t.Errorf("expected Exec cleared to nil, got %+v", task.Exec)
	}
	if task.RefactorNotes != nil {
		t.Errorf("expected RefactorNotes cleared to nil, got %+v", task.RefactorNotes)
	}
	if task.FailedACReasons != nil {
		t.Errorf("expected FailedACReasons cleared to nil, got %+v", task.FailedACReasons)
	}
	if task.Blocked {
		t.Errorf("expected Blocked cleared to false, got true")
	}
	if task.BlockedReason != "" {
		t.Errorf("expected BlockedReason cleared to empty, got %q", task.BlockedReason)
	}
	if prog.Status != spec.StatusDraft {
		t.Errorf("expected Status draft, got %q", prog.Status)
	}
	if prog.Iterations != 0 {
		t.Errorf("expected Iterations reset to 0, got %d", prog.Iterations)
	}
	if tr := mustLoadTraceability(t, root, fixtureSlug); len(tr.Entries) != 0 {
		t.Errorf("expected traceability emptied, got %+v", tr.Entries)
	}
	if a := mustLoadAnalysis(t, root, fixtureSlug); a.Verdict != "approved" {
		t.Errorf("expected analysis verdict preserved since analysis is an earlier phase, got %+v", a)
	}
	assertAnswerKeysPresent(t, state, "analysis_complete", "analysis_audited", "analysis_findings", "analysis_attempts", "refinement_approved")
	assertAnswerKeysAbsent(t, state, "rule_proposal", "rule_approved", "rule_applied", "rule_feedback", "rule_attempt")
	if len(warnings) == 0 {
		t.Errorf("expected at least one warning about untouched rule-learning global files")
	}
}

func TestResetFrom_ac1_TargetRuleLearning_WarnsAndDoesNotDeleteGlobalRuleFile(t *testing.T) {
	root := t.TempDir()
	writeFixtureFiles(t, root, fixtureSlug)
	state := buildFullFixtureState()
	prog := buildFullFixtureProgress()

	warnings, err := ResetFrom(string(phasecatalog.PhaseRuleLearning), &state, &prog, root, fixtureSlug)
	if err != nil {
		t.Fatalf("ResetFrom returned error: %v", err)
	}

	if len(warnings) == 0 {
		t.Fatalf("expected a non-empty warning about global rule files not being touched")
	}
	found := false
	for _, w := range warnings {
		if strings.Contains(strings.ToLower(w), "rule") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a warning mentioning rule-learning, got %v", warnings)
	}
	assertRuleFilePreserved(t, root)
	assertAnswerKeysAbsent(t, state, "rule_proposal", "rule_approved", "rule_applied", "rule_feedback", "rule_attempt")

	assertSpecMdPreserved(t, root, fixtureSlug)
	if got := mustLoadSettings(t, root, fixtureSlug); got != fixtureCustomSettings() {
		t.Errorf("expected settings preserved, got %+v", got)
	}
	if len(prog.Tasks) != 1 {
		t.Fatalf("expected task list preserved with 1 task, got %d", len(prog.Tasks))
	}
	if !prog.Tasks[0].Done {
		t.Errorf("expected task Done preserved since execution is an earlier phase, got false")
	}
	if a := mustLoadAnalysis(t, root, fixtureSlug); a.Verdict != "approved" {
		t.Errorf("expected analysis verdict preserved since analysis is an earlier phase, got %+v", a)
	}
}

func TestResetFrom_ac1_UnknownTarget_ReturnsError(t *testing.T) {
	root := t.TempDir()
	writeFixtureFiles(t, root, fixtureSlug)
	state := buildFullFixtureState()
	prog := buildFullFixtureProgress()

	_, err := ResetFrom("not-a-real-phase", &state, &prog, root, fixtureSlug)
	if err == nil {
		t.Fatalf("expected error for unknown target phase, got nil")
	}
}

func TestResetFrom_ec1_CompletedSpecRollbackFromEarliestPhase_ClearsEverything(t *testing.T) {
	root := t.TempDir()
	writeFixtureFiles(t, root, fixtureSlug)
	state := buildFullFixtureState()
	state.Phase = string(engine.PhaseComplete)
	prog := buildFullFixtureProgress()
	prog.Status = spec.StatusCompleted

	warnings, err := ResetFrom(string(phasecatalog.PhaseSettings), &state, &prog, root, fixtureSlug)
	if err != nil {
		t.Fatalf("expected rollback-allowed reset on a completed spec, got error: %v", err)
	}

	if got := mustLoadSettings(t, root, fixtureSlug); got != spec.DefaultSettings() {
		t.Errorf("expected settings reset to defaults, got %+v", got)
	}
	assertAnswerKeysAbsent(t, state, fixtureAnswerKeys()...)
	assertSpecMdDeleted(t, root, fixtureSlug)
	if len(prog.Tasks) != 0 {
		t.Errorf("expected Tasks emptied, got %d tasks", len(prog.Tasks))
	}
	if prog.TaskSeq != 0 {
		t.Errorf("expected TaskSeq reset to 0, got %d", prog.TaskSeq)
	}
	if a := mustLoadAnalysis(t, root, fixtureSlug); a.Verdict != "" || len(a.Findings) != 0 {
		t.Errorf("expected analysis reset, got %+v", a)
	}
	if tr := mustLoadTraceability(t, root, fixtureSlug); len(tr.Entries) != 0 {
		t.Errorf("expected traceability entries emptied, got %+v", tr.Entries)
	}
	assertRuleFilePreserved(t, root)
	if len(warnings) == 0 {
		t.Errorf("expected at least one warning about untouched rule-learning global files")
	}
}
