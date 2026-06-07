package loop

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/promptregistry"
	"github.com/pragmataW/tddmaster/internal/prompts"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func TestStageReport_UnmarshalExecutor(t *testing.T) {
	var report StageReport
	if err := json.Unmarshal([]byte(promptregistry.ReportExampleExecutor), &report); err != nil {
		t.Fatalf("unmarshal executor report: %v", err)
	}
	if len(report.Completed) == 0 {
		t.Error("expected Completed to be non-empty")
	}
	if report.Completed[0] != "task-1" {
		t.Errorf("expected Completed[0] = %q, got %q", "task-1", report.Completed[0])
	}
	if len(report.FilesModified) == 0 {
		t.Error("expected FilesModified to be non-empty")
	}
	if report.FilesModified[0] != "internal/foo/bar.go" {
		t.Errorf("expected FilesModified[0] = %q, got %q", "internal/foo/bar.go", report.FilesModified[0])
	}
	if report.Phase != "green" {
		t.Errorf("expected Phase = %q, got %q", "green", report.Phase)
	}
}

func TestStageReport_UnmarshalVerifier(t *testing.T) {
	var report StageReport
	if err := json.Unmarshal([]byte(promptregistry.ReportExampleVerifier), &report); err != nil {
		t.Fatalf("unmarshal verifier report: %v", err)
	}
	if !report.Passed {
		t.Error("expected Passed to be true")
	}
	if report.FailedACs == nil {
		t.Error("expected FailedACs to be non-nil (empty slice)")
	}
	if report.UncoveredEdgeCases == nil {
		t.Error("expected UncoveredEdgeCases to be non-nil (empty slice)")
	}
}

func TestStageReport_RefactorNotes_FromArrayJSON(t *testing.T) {
	raw := `{"passed":true,"refactorNotes":[{"file":"x.go","suggestion":"consider extracting validation logic","rationale":"r"}]}`
	var report StageReport
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(report.RefactorNotes) == 0 {
		t.Error("expected RefactorNotes to be non-empty")
	}
}

func TestStageReport_UnmarshalPlanner(t *testing.T) {
	var report StageReport
	if err := json.Unmarshal([]byte(promptregistry.ReportExamplePlanner), &report); err != nil {
		t.Fatalf("unmarshal planner report: %v", err)
	}
	if report.Plan == nil {
		t.Fatal("expected Plan to be non-nil")
	}
	if len(report.Plan.TouchedFiles) == 0 {
		t.Error("expected Plan.TouchedFiles to be non-empty")
	}
	if report.Plan.TouchedFiles[0] != "internal/foo/bar.go" {
		t.Errorf("expected TouchedFiles[0] = %q, got %q", "internal/foo/bar.go", report.Plan.TouchedFiles[0])
	}
	if len(report.Plan.Assumptions) == 0 {
		t.Error("expected Plan.Assumptions to be non-empty")
	}
	if report.Plan.Assumptions[0] != "existing tests cover happy path" {
		t.Errorf("expected Assumptions[0] = %q, got %q", "existing tests cover happy path", report.Plan.Assumptions[0])
	}
	if report.Plan.Approach == "" {
		t.Error("expected Plan.Approach to be non-empty")
	}
}

func TestStageReport_UnmarshalTestWriter(t *testing.T) {
	var report StageReport
	if err := json.Unmarshal([]byte(promptregistry.ReportExampleTestWriter), &report); err != nil {
		t.Fatalf("unmarshal test writer report: %v", err)
	}
	if len(report.TestsWritten) == 0 {
		t.Error("expected TestsWritten to be non-empty")
	}
	if report.TestsWritten[0] != "TestFoo_HappyPath" {
		t.Errorf("expected TestsWritten[0] = %q, got %q", "TestFoo_HappyPath", report.TestsWritten[0])
	}
	if report.TestsWritten[1] != "TestFoo_EdgeCase" {
		t.Errorf("expected TestsWritten[1] = %q, got %q", "TestFoo_EdgeCase", report.TestsWritten[1])
	}
}

func TestStageReport_RefactorNotesPresent_Method(t *testing.T) {
	empty := StageReport{}
	if empty.RefactorNotesPresent() {
		t.Error("expected RefactorNotesPresent() = false for nil slice")
	}

	withNotes := StageReport{RefactorNotes: []RefactorNote{{Suggestion: "do X"}}}
	if !withNotes.RefactorNotesPresent() {
		t.Error("expected RefactorNotesPresent() = true for non-empty slice")
	}
}

func TestExecCtx_UserContextField(t *testing.T) {
	ctx := ExecCtx{
		UserContext: "some context",
	}
	if ctx.UserContext != "some context" {
		t.Errorf("expected UserContext = %q, got %q", "some context", ctx.UserContext)
	}
}

func TestStageReport_PlannerWithAccepted(t *testing.T) {
	raw := `{"plan":{"taskId":"t1","touchedFiles":["a.go"],"assumptions":["x"],"approach":"y","designPatterns":["dp"],"bestPractices":["bp"]},"accepted":true}`
	var report StageReport
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !report.Accepted {
		t.Error("expected Accepted = true")
	}
	if report.Plan == nil {
		t.Fatal("expected Plan non-nil")
	}
}

func TestStageReport_PlannerWithFeedback(t *testing.T) {
	raw := `{"planFeedback":"needs more detail"}`
	var report StageReport
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if report.PlanFeedback != "needs more detail" {
		t.Errorf("expected PlanFeedback = %q, got %q", "needs more detail", report.PlanFeedback)
	}
}

func TestStageReport_BlockedField(t *testing.T) {
	raw := `{"blocked":["task-2"],"completed":[]}`
	var report StageReport
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(report.Blocked) == 0 {
		t.Error("expected Blocked to be non-empty")
	}
	if report.Blocked[0] != "task-2" {
		t.Errorf("expected Blocked[0] = %q, got %q", "task-2", report.Blocked[0])
	}
}

func TestStageReport_RefactorApplied(t *testing.T) {
	raw := `{"refactorApplied":true}`
	var report StageReport
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !report.RefactorApplied {
		t.Error("expected RefactorApplied = true")
	}
}

func TestStageReport_PlanNilWhenAbsent(t *testing.T) {
	raw := `{"passed":true}`
	var report StageReport
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if report.Plan != nil {
		t.Error("expected Plan to be nil when not present in JSON")
	}
}

func TestStageReport_PlanRoundTrip(t *testing.T) {
	original := StageReport{
		Plan: &spec.TaskPlan{
			TaskID:         "t1",
			TouchedFiles:   []string{"a.go"},
			Assumptions:    []string{"assume x"},
			DesignPatterns: []string{"strategy"},
			BestPractices:  []string{"srp"},
			Approach:       "extend y",
		},
		Accepted: true,
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got StageReport
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Plan == nil {
		t.Fatal("expected Plan non-nil after round-trip")
	}
	if got.Plan.TaskID != "t1" {
		t.Errorf("TaskID mismatch: got %q, want %q", got.Plan.TaskID, "t1")
	}
	if len(got.Plan.TouchedFiles) != 1 || got.Plan.TouchedFiles[0] != "a.go" {
		t.Errorf("TouchedFiles mismatch: got %v", got.Plan.TouchedFiles)
	}
}

func TestStageReport_EffectivePassed_PassedAndNilSlices_ReturnsTrue(t *testing.T) {
	r := StageReport{Passed: true, FailedACs: nil, UncoveredEdgeCases: nil}
	if !r.EffectivePassed() {
		t.Error("EffectivePassed: got false, want true (passed=true, nil slices)")
	}
}

func TestStageReport_EffectivePassed_PassedAndEmptySlices_ReturnsTrue(t *testing.T) {
	r := StageReport{Passed: true, FailedACs: []string{}, UncoveredEdgeCases: []string{}}
	if !r.EffectivePassed() {
		t.Error("EffectivePassed: got false, want true (passed=true, empty slices)")
	}
}

func TestStageReport_EffectivePassed_PassedWithFailedACs_ReturnsFalse(t *testing.T) {
	r := StageReport{Passed: true, FailedACs: []string{"ac-1"}, UncoveredEdgeCases: nil}
	if r.EffectivePassed() {
		t.Error("EffectivePassed: got true, want false (passed=true but failedACs not empty)")
	}
}

func TestStageReport_EffectivePassed_PassedWithUncoveredEdgeCases_ReturnsFalse(t *testing.T) {
	r := StageReport{Passed: true, FailedACs: nil, UncoveredEdgeCases: []string{"ec-2"}}
	if r.EffectivePassed() {
		t.Error("EffectivePassed: got true, want false (passed=true but uncoveredEdgeCases not empty)")
	}
}

func TestStageReport_EffectivePassed_NotPassed_ReturnsFalse(t *testing.T) {
	r := StageReport{Passed: false, FailedACs: nil, UncoveredEdgeCases: nil}
	if r.EffectivePassed() {
		t.Error("EffectivePassed: got true, want false (passed=false)")
	}
}

func TestRefactorNote_JSONTags(t *testing.T) {
	note := RefactorNote{
		File:       "internal/foo/bar.go",
		Suggestion: "extract validation",
		Rationale:  "reused in two places",
	}
	data, err := json.Marshal(note)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	raw := string(data)
	if !strings.Contains(raw, `"file"`) {
		t.Error("expected json tag 'file' in marshaled output")
	}
	if !strings.Contains(raw, `"suggestion"`) {
		t.Error("expected json tag 'suggestion' in marshaled output")
	}
	if !strings.Contains(raw, `"rationale"`) {
		t.Error("expected json tag 'rationale' in marshaled output")
	}

	var got RefactorNote
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.File != "internal/foo/bar.go" {
		t.Errorf("File: got %q, want %q", got.File, "internal/foo/bar.go")
	}
	if got.Suggestion != "extract validation" {
		t.Errorf("Suggestion: got %q, want %q", got.Suggestion, "extract validation")
	}
	if got.Rationale != "reused in two places" {
		t.Errorf("Rationale: got %q, want %q", got.Rationale, "reused in two places")
	}
}

func TestStageReport_RefactorNotes_ArrayOfObjects(t *testing.T) {
	raw := `{"refactorNotes":[{"file":"a.go","suggestion":"s","rationale":"r"}]}`
	var report StageReport
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(report.RefactorNotes) != 1 {
		t.Fatalf("expected 1 RefactorNote, got %d", len(report.RefactorNotes))
	}
	if report.RefactorNotes[0].File != "a.go" {
		t.Errorf("File: got %q, want %q", report.RefactorNotes[0].File, "a.go")
	}
	if report.RefactorNotes[0].Suggestion != "s" {
		t.Errorf("Suggestion: got %q, want %q", report.RefactorNotes[0].Suggestion, "s")
	}
	if report.RefactorNotes[0].Rationale != "r" {
		t.Errorf("Rationale: got %q, want %q", report.RefactorNotes[0].Rationale, "r")
	}
}

func TestVerifierSchema_UnmarshalsIntoStageReport(t *testing.T) {
	var report StageReport
	if err := json.Unmarshal([]byte(promptregistry.ReportExampleVerifier), &report); err != nil {
		t.Fatalf("unmarshal ReportExampleVerifier into StageReport: %v", err)
	}
	if !report.Passed {
		t.Error("expected Passed = true")
	}
	if len(report.RefactorNotes) < 1 {
		t.Fatalf("expected at least 1 RefactorNote, got %d", len(report.RefactorNotes))
	}
	if report.RefactorNotes[0].File == "" {
		t.Error("expected RefactorNotes[0].File to be non-empty")
	}
	if report.RefactorNotes[0].Suggestion == "" {
		t.Error("expected RefactorNotes[0].Suggestion to be non-empty")
	}
	if report.RefactorNotes[0].Rationale == "" {
		t.Error("expected RefactorNotes[0].Rationale to be non-empty")
	}
	if report.Phase != "green" {
		t.Errorf("expected Phase = %q, got %q", "green", report.Phase)
	}
}

func TestVerifierTmpl_SchemaMatchesStageReportTags(t *testing.T) {
	out, err := prompts.Render("verifier", prompts.RenderData{})
	if err != nil {
		t.Fatalf("Render verifier: %v", err)
	}
	for _, tag := range []string{"passed", "phase", "failedACs", "uncoveredEdgeCases", "refactorNotes"} {
		if !strings.Contains(out, `"`+tag+`"`) {
			t.Errorf("verifier template output missing JSON field %q", tag)
		}
	}
}
