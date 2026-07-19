package phases

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func seedAnalysisSpec(t *testing.T, root, slug string, tasks []spec.Task) {
	t.Helper()
	writeDiscoveryManifest(t, root)
	state := spec.State{
		Version: 1,
		Slug:    slug,
		Phase:   "cross-artifact-analysis",
		Answers: map[string][]spec.Answer{},
	}
	if err := spec.SaveState(root, slug, state); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	if err := spec.SaveSettings(root, slug, spec.DefaultSettings()); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	pr := spec.Progress{Spec: slug, Status: spec.StatusDraft, Tasks: tasks}
	if err := spec.SaveProgress(root, slug, pr); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
}

func buildAnalysisCtx(t *testing.T, root, slug string) *engine.Context {
	t.Helper()
	defs := []engine.PhaseDef{{ID: "cross-artifact-analysis", Driver: AnalysisDriver()}}
	ctx, err := engine.Build(root, slug, defs)
	if err != nil {
		t.Fatalf("engine.Build: %v", err)
	}
	return ctx
}

func tasksWithCriteria() []spec.Task {
	return []spec.Task{
		{
			ID:    "task-1",
			Title: "Alpha",
			Criteria: []spec.Criterion{
				{ID: "ac-1", Then: "system validates input"},
			},
		},
		{
			ID:    "task-2",
			Title: "Beta",
			Criteria: []spec.Criterion{
				{ID: "ac-1", Then: "system returns result"},
			},
		},
	}
}

// cleanTasks returns tasks that produce zero linter findings: every criterion
// has a non-empty When and Then and none duplicate one another.
func cleanTasks() []spec.Task {
	return []spec.Task{
		{
			ID:    "task-1",
			Title: "Alpha",
			Criteria: []spec.Criterion{
				{ID: "ac-1", Given: "input given", When: "validate is called", Then: "system validates input"},
			},
		},
		{
			ID:    "task-2",
			Title: "Beta",
			Criteria: []spec.Criterion{
				{ID: "ac-1", Given: "a request", When: "process is called", Then: "system returns result"},
			},
		},
	}
}

func tasksWithNoCriteria() []spec.Task {
	return []spec.Task{
		{ID: "task-1", Title: "Alpha", Criteria: nil},
	}
}

func TestAnalysisDriver_ReturnsNonNilDriver(t *testing.T) {
	d := AnalysisDriver()
	if d == nil {
		t.Fatal("AnalysisDriver() returned nil")
	}
}

func TestAnalysisDriver_ConcreteTypeIsAnalysisDriver(t *testing.T) {
	d := AnalysisDriver()
	if _, ok := d.(*analysisDriver); !ok {
		t.Fatalf("AnalysisDriver() returned %T, want *analysisDriver", d)
	}
}

func TestAnalysisDriver_ImplementsEngineDriverInterface(t *testing.T) {
	var _ engine.Driver = AnalysisDriver()
}

func TestAnalysisDriver_FirstNext_EmitsAuditorInstruct(t *testing.T) {
	root := t.TempDir()
	seedAnalysisSpec(t, root, "s", tasksWithCriteria())
	ctx := buildAnalysisCtx(t, root, "s")

	action, phaseDone := AnalysisDriver().Next(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"})
	if phaseDone {
		t.Fatal("phaseDone must be false on first Next (audit not yet done)")
	}
	if action.Action != engine.ActionInstruct {
		t.Fatalf("action = %q, want %q", action.Action, engine.ActionInstruct)
	}
	if action.DelegateAgent != "tddmaster-auditor" {
		t.Fatalf("DelegateAgent = %q, want %q", action.DelegateAgent, "tddmaster-auditor")
	}
}

func TestAnalysisDriver_InstructIncludesLinterFindingsAndTasks(t *testing.T) {
	root := t.TempDir()
	tasks := tasksWithNoCriteria()
	seedAnalysisSpec(t, root, "s", tasks)
	ctx := buildAnalysisCtx(t, root, "s")

	action, _ := AnalysisDriver().Next(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"})
	if !strings.Contains(action.Instruction, "task-1") {
		t.Error("instruction must contain task id 'task-1'")
	}
	if !strings.Contains(action.Instruction, "task-no-ac") {
		t.Error("instruction must contain BuildLint finding category 'task-no-ac' for task with no criteria")
	}
}

func TestAnalysisDriver_AuditSubmit_MergesAndSavesAnalysis(t *testing.T) {
	root := t.TempDir()
	tasks := tasksWithCriteria()
	seedAnalysisSpec(t, root, "s", tasks)
	ctx := buildAnalysisCtx(t, root, "s")

	auditorJSON := `{"verdict":"issues","findings":[{"severity":"warn","category":"missing-scope","taskId":"task-1","detail":"scope unclear","source":"auditor"}]}`
	_, _, err := AnalysisDriver().Submit(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"}, []byte(auditorJSON))
	if err != nil {
		t.Fatalf("Submit returned unexpected error: %v", err)
	}

	saved, loadErr := spec.LoadAnalysis(root, "s")
	if loadErr != nil {
		t.Fatalf("LoadAnalysis: %v", loadErr)
	}
	if len(saved.Findings) == 0 {
		t.Fatal("merged findings must not be empty after audit submit")
	}

	hasAuditorFinding := false
	for _, f := range saved.Findings {
		if f.Source == "auditor" {
			hasAuditorFinding = true
		}
	}
	if !hasAuditorFinding {
		t.Error("merged analysis must include auditor findings")
	}

	analysisPath := paths.SpecAnalysis(root, "s")
	if _, statErr := os.Stat(analysisPath); os.IsNotExist(statErr) {
		t.Fatal("analysis.json must exist on disk after Submit")
	}
}

func TestAnalysisDriver_NoActionable_CompletesToExecution(t *testing.T) {
	root := t.TempDir()
	tasks := cleanTasks()
	seedAnalysisSpec(t, root, "s", tasks)
	ctx := buildAnalysisCtx(t, root, "s")

	auditorJSON := `{"verdict":"clean","findings":[]}`
	_, phaseDone, err := AnalysisDriver().Submit(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"}, []byte(auditorJSON))
	if err != nil {
		t.Fatalf("Submit returned unexpected error: %v", err)
	}
	if !phaseDone {
		if !ctx.HasAnswer("analysis_complete") {
			action, done := AnalysisDriver().Next(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"})
			if !done {
				t.Fatalf("expected phase done after clean audit, got action=%q", action.Action)
			}
		}
	}
}

func TestAnalysisDriver_InfoOnly_CompletesToExecution(t *testing.T) {
	root := t.TempDir()
	tasks := cleanTasks()
	seedAnalysisSpec(t, root, "s", tasks)
	ctx := buildAnalysisCtx(t, root, "s")

	auditorJSON := `{"verdict":"clean","findings":[{"severity":"info","category":"note","taskId":"task-1","detail":"advisory only","source":"auditor"}]}`
	_, phaseDone, err := AnalysisDriver().Submit(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"}, []byte(auditorJSON))
	if err != nil {
		t.Fatalf("Submit returned unexpected error: %v", err)
	}
	if !phaseDone && !ctx.HasAnswer("analysis_complete") {
		action, done := AnalysisDriver().Next(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"})
		if !done {
			t.Fatalf("info-only findings must pass through, got action=%q", action.Action)
		}
	}
}

func TestAnalysisDriver_WarnFinding_OpensGate(t *testing.T) {
	root := t.TempDir()
	tasks := cleanTasks()
	seedAnalysisSpec(t, root, "s", tasks)
	ctx := buildAnalysisCtx(t, root, "s")

	auditorJSON := `{"verdict":"issues","findings":[{"severity":"warn","category":"overlap","taskId":"task-1","detail":"tasks overlap","source":"auditor"}]}`
	if _, _, err := AnalysisDriver().Submit(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"}, []byte(auditorJSON)); err != nil {
		t.Fatalf("Submit returned unexpected error: %v", err)
	}

	action, phaseDone := AnalysisDriver().Next(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"})
	if phaseDone {
		t.Fatal("phaseDone must be false when a non-info (warn) finding exists")
	}
	if action.Action != engine.ActionAsk {
		t.Fatalf("action = %q, want %q on warn finding", action.Action, engine.ActionAsk)
	}
	if ctx.HasAnswer("analysis_complete") {
		t.Fatal("analysis_complete must not be set while a warn finding awaits user decision")
	}
}

func TestAnalysisDriver_BlockFinding_OpensGate(t *testing.T) {
	root := t.TempDir()
	tasks := tasksWithCriteria()
	seedAnalysisSpec(t, root, "s", tasks)
	ctx := buildAnalysisCtx(t, root, "s")

	auditorJSON := `{"verdict":"block","findings":[{"severity":"block","category":"critical-gap","taskId":"task-1","detail":"missing edge case","source":"auditor"}]}`
	_, _, err := AnalysisDriver().Submit(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"}, []byte(auditorJSON))
	if err != nil {
		t.Fatalf("Submit returned unexpected error: %v", err)
	}

	action, phaseDone := AnalysisDriver().Next(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"})
	if phaseDone {
		t.Fatal("phaseDone must be false when block finding exists")
	}
	if action.Action != engine.ActionAsk {
		t.Fatalf("action = %q, want %q on block", action.Action, engine.ActionAsk)
	}
	if len(action.InteractiveOptions) != 3 {
		t.Fatalf("expected 3 interactive options, got %d", len(action.InteractiveOptions))
	}

	labels := make([]string, len(action.InteractiveOptions))
	for i, o := range action.InteractiveOptions {
		labels[i] = o.Label
	}
	hasRefinement, hasAccept, hasEdit := false, false, false
	for _, l := range labels {
		lower := strings.ToLower(l)
		if strings.Contains(lower, "refinement") {
			hasRefinement = true
		}
		if strings.Contains(lower, "accept") {
			hasAccept = true
		}
		if strings.Contains(lower, "edit") {
			hasEdit = true
		}
	}
	if !hasRefinement {
		t.Errorf("interactive options must include return-to-refinement option, got %v", labels)
	}
	if !hasAccept {
		t.Errorf("interactive options must include accept-anyway option, got %v", labels)
	}
	if !hasEdit {
		t.Errorf("interactive options must include edit option, got %v", labels)
	}
	if len(action.CommandMap) == 0 && action.ExpectedInput.SubmitCmd == "" {
		t.Error("either CommandMap or ExpectedInput.SubmitCmd must be non-empty at gate")
	}
}

func TestAnalysisDriver_GateAccept_Completes(t *testing.T) {
	root := t.TempDir()
	tasks := tasksWithCriteria()
	seedAnalysisSpec(t, root, "s", tasks)
	ctx := buildAnalysisCtx(t, root, "s")

	auditorJSON := `{"verdict":"block","findings":[{"severity":"block","category":"critical-gap","taskId":"task-1","detail":"missing edge case","source":"auditor"}]}`
	if _, _, err := AnalysisDriver().Submit(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"}, []byte(auditorJSON)); err != nil {
		t.Fatalf("Submit audit: %v", err)
	}

	_, phaseDone, err := AnalysisDriver().Submit(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"}, []byte(`accept-anyway`))
	if err != nil {
		t.Fatalf("Submit gate accept: %v", err)
	}
	if !phaseDone {
		if !ctx.HasAnswer("analysis_complete") {
			t.Fatal("expected done=true after accept-anyway")
		}
	}
}

func TestAnalysisDriver_GateEdit_AppliesRefineAndReAudits(t *testing.T) {
	root := t.TempDir()
	tasks := tasksWithCriteria()
	seedAnalysisSpec(t, root, "s", tasks)
	ctx := buildAnalysisCtx(t, root, "s")

	auditorJSON := `{"verdict":"block","findings":[{"severity":"block","category":"critical-gap","taskId":"task-1","detail":"missing edge case","source":"auditor"}]}`
	if _, _, err := AnalysisDriver().Submit(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"}, []byte(auditorJSON)); err != nil {
		t.Fatalf("Submit audit: %v", err)
	}

	newTitle := "Alpha Updated"
	refinePayload := map[string]interface{}{
		"update": map[string]interface{}{
			"task-1": map[string]interface{}{
				"title": newTitle,
				"criteria": []map[string]interface{}{
					{"then": "system validates input thoroughly"},
				},
			},
		},
	}
	refineJSON, _ := json.Marshal(refinePayload)

	editPayload := map[string]interface{}{
		"action":  "edit",
		"payload": json.RawMessage(refineJSON),
	}
	editJSON, _ := json.Marshal(editPayload)

	_, phaseDone, err := AnalysisDriver().Submit(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"}, editJSON)
	if err != nil {
		t.Fatalf("Submit gate edit: %v", err)
	}
	if phaseDone {
		t.Fatal("phaseDone must be false after edit (must re-audit)")
	}

	action, done := AnalysisDriver().Next(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"})
	if done {
		t.Fatal("next after edit must not be done (re-audit expected)")
	}
	if action.Action != engine.ActionInstruct {
		t.Fatalf("after edit, next action = %q, want %q (re-audit instruct)", action.Action, engine.ActionInstruct)
	}
	if action.DelegateAgent != "tddmaster-auditor" {
		t.Fatalf("after edit, DelegateAgent = %q, want %q", action.DelegateAgent, "tddmaster-auditor")
	}

	pr := ctx.Progress()
	found := false
	for _, task := range pr.Tasks {
		if task.ID == "task-1" && task.Title == newTitle {
			found = true
		}
	}
	if !found {
		t.Error("ApplyRefinement must have updated task-1 title in progress")
	}
}

func TestAnalysisDriver_MalformedAuditorJSON_CleanError(t *testing.T) {
	root := t.TempDir()
	tasks := tasksWithCriteria()
	seedAnalysisSpec(t, root, "s", tasks)
	ctx := buildAnalysisCtx(t, root, "s")

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Submit panicked on malformed JSON: %v", r)
		}
	}()

	action, phaseDone, err := AnalysisDriver().Submit(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"}, []byte(`{not valid json`))
	if err == nil && action.Action != engine.ActionError {
		t.Fatal("Submit with malformed auditor JSON must return non-nil error or ActionError")
	}
	if phaseDone {
		t.Fatal("phaseDone must be false after malformed JSON")
	}
	if ctx.HasAnswer("analysis_complete") {
		t.Fatal("analysis_complete must not be set after malformed JSON")
	}
	if ctx.HasAnswer("analysis_audited") {
		t.Fatal("analysis_audited must not be set after malformed JSON")
	}
}

func TestAnalysisDriver_ReAuditLoop_Terminates(t *testing.T) {
	root := t.TempDir()
	tasks := tasksWithCriteria()
	seedAnalysisSpec(t, root, "s", tasks)
	ctx := buildAnalysisCtx(t, root, "s")

	blockAuditJSON := `{"verdict":"block","findings":[{"severity":"block","category":"critical-gap","taskId":"task-1","detail":"missing edge case","source":"auditor"}]}`

	noopEditPayload := map[string]interface{}{
		"action":  "edit",
		"payload": map[string]interface{}{"update": map[string]interface{}{}},
	}
	noopEditJSON, _ := json.Marshal(noopEditPayload)

	const maxAttempts = 10
	terminated := false
	for i := 0; i < maxAttempts; i++ {
		if _, _, err := AnalysisDriver().Submit(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"}, []byte(blockAuditJSON)); err != nil {
			break
		}

		action, done := AnalysisDriver().Next(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"})
		if done {
			terminated = true
			break
		}
		if action.Action != engine.ActionAsk {
			break
		}

		_, done2, err := AnalysisDriver().Submit(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"}, noopEditJSON)
		if err != nil {
			terminated = true
			break
		}
		if done2 {
			terminated = true
			break
		}

		action2, done3 := AnalysisDriver().Next(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"})
		if done3 {
			terminated = true
			break
		}
		if action2.Action == engine.ActionError {
			terminated = true
			break
		}
	}

	if !terminated {
		t.Fatal("re-audit loop must terminate within bounded cap; detected potential infinite loop")
	}
}

func TestAnalysisDriver_LintBlockSurvivesCleanAuditor(t *testing.T) {
	root := t.TempDir()
	tasks := tasksWithNoCriteria()
	seedAnalysisSpec(t, root, "s", tasks)
	ctx := buildAnalysisCtx(t, root, "s")

	action, phaseDone := AnalysisDriver().Next(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"})
	if phaseDone {
		t.Fatal("phaseDone must be false on first Next (audit not yet done)")
	}
	if action.Action != engine.ActionInstruct {
		t.Fatalf("action = %q, want %q", action.Action, engine.ActionInstruct)
	}

	auditorJSON := `{"verdict":"clean","findings":[]}`
	_, submitDone, err := AnalysisDriver().Submit(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"}, []byte(auditorJSON))
	if err != nil {
		t.Fatalf("Submit returned unexpected error: %v", err)
	}
	if submitDone {
		t.Fatal("Submit must return done=false: lint block survives the merge and keeps the phase open")
	}

	saved, loadErr := spec.LoadAnalysis(root, "s")
	if loadErr != nil {
		t.Fatalf("LoadAnalysis: %v", loadErr)
	}
	hasTaskNoAcBlock := false
	for _, f := range saved.Findings {
		if f.Category == "task-no-ac" && f.IsBlock() {
			hasTaskNoAcBlock = true
		}
	}
	if !hasTaskNoAcBlock {
		t.Fatal("merged analysis must contain the task-no-ac block finding despite the clean auditor verdict")
	}

	gateAction, gateDone := AnalysisDriver().Next(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"})
	if gateDone {
		t.Fatal("phaseDone must be false when the lint block finding exists")
	}
	if gateAction.Action != engine.ActionAsk {
		t.Fatalf("action = %q, want %q (lint block must open the gate)", gateAction.Action, engine.ActionAsk)
	}
}

func TestAnalysisDriver_EditKeepsCriterionIDsStable(t *testing.T) {
	root := t.TempDir()
	tasks := []spec.Task{
		{
			ID:    "task-1",
			Title: "Alpha",
			Criteria: []spec.Criterion{
				{ID: "ac-1", Then: "system validates input"},
				{ID: "ac-2", Then: "system logs result"},
			},
		},
	}
	seedAnalysisSpec(t, root, "s", tasks)
	ctx := buildAnalysisCtx(t, root, "s")

	blockAuditJSON := `{"verdict":"block","findings":[{"severity":"block","category":"critical-gap","taskId":"task-1","detail":"needs more","source":"auditor"}]}`
	if _, _, err := AnalysisDriver().Submit(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"}, []byte(blockAuditJSON)); err != nil {
		t.Fatalf("Submit audit: %v", err)
	}

	newCriteria := []map[string]interface{}{
		{"id": "ac-1", "then": "system validates input strictly"},
		{"id": "ac-2", "then": "system logs result verbosely"},
	}
	refinePayload := map[string]interface{}{
		"update": map[string]interface{}{
			"task-1": map[string]interface{}{
				"criteria": newCriteria,
			},
		},
	}
	refineJSON, _ := json.Marshal(refinePayload)
	editPayload := map[string]interface{}{
		"action":  "edit",
		"payload": json.RawMessage(refineJSON),
	}
	editJSON, _ := json.Marshal(editPayload)

	if _, _, err := AnalysisDriver().Submit(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"}, editJSON); err != nil {
		t.Fatalf("Submit gate edit: %v", err)
	}

	pr := ctx.Progress()
	for _, task := range pr.Tasks {
		if task.ID != "task-1" {
			continue
		}
		ids := make(map[string]bool)
		for _, c := range task.Criteria {
			if ids[c.ID] {
				t.Errorf("duplicate criterion id %q after refinement", c.ID)
			}
			ids[c.ID] = true
			if c.ID != "ac-1" && c.ID != "ac-2" {
				t.Errorf("criterion id changed after refinement: got %q, expected ac-1 or ac-2", c.ID)
			}
		}
	}
}

func TestAnalysisDriver_InstructIncludesTaskExecPlanTouchedFiles(t *testing.T) {
	root := t.TempDir()
	tasks := cleanTasks()
	tasks[0].Exec = &spec.ExecState{
		Plan: &spec.TaskPlan{
			TaskID:       "task-1",
			TouchedFiles: []string{"internal/auth/login.go", "internal/auth/session.go"},
		},
	}
	seedAnalysisSpec(t, root, "s", tasks)
	ctx := buildAnalysisCtx(t, root, "s")

	action, _ := AnalysisDriver().Next(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"})
	if !strings.Contains(action.Instruction, "approved touched files:") {
		t.Error("instruction must contain 'approved touched files:' when a task has an approved plan")
	}
	if !strings.Contains(action.Instruction, "internal/auth/login.go") {
		t.Error("instruction must list touched file 'internal/auth/login.go'")
	}
	if !strings.Contains(action.Instruction, "internal/auth/session.go") {
		t.Error("instruction must list touched file 'internal/auth/session.go'")
	}
}

func TestAnalysisDriver_InstructOmitsTouchedFiles_WhenNoPlan(t *testing.T) {
	root := t.TempDir()
	seedAnalysisSpec(t, root, "s", cleanTasks())
	ctx := buildAnalysisCtx(t, root, "s")

	action, _ := AnalysisDriver().Next(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"})
	if strings.Contains(action.Instruction, "approved touched files:") {
		t.Error("instruction must not contain 'approved touched files:' when no task has a plan")
	}
}
