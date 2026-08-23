package phases

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/phasecatalog"
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

func TestAnalysisDriver_InstructIncludesTaskDependsOn(t *testing.T) {
	root := t.TempDir()
	tasks := cleanTasks()
	tasks[1].DependsOn = []string{"task-1"}
	seedAnalysisSpec(t, root, "s", tasks)
	ctx := buildAnalysisCtx(t, root, "s")

	action, _ := AnalysisDriver().Next(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"})
	if !strings.Contains(action.Instruction, "depends on: task-1") {
		t.Error("instruction must contain 'depends on: task-1' when a task declares a dependency")
	}
}

func TestAnalysisDriver_InstructOmitsDependsOn_WhenNone(t *testing.T) {
	root := t.TempDir()
	seedAnalysisSpec(t, root, "s", cleanTasks())
	ctx := buildAnalysisCtx(t, root, "s")

	action, _ := AnalysisDriver().Next(ctx, &engine.PhaseDef{ID: "cross-artifact-analysis"})
	if strings.Contains(action.Instruction, "depends on:") {
		t.Error("instruction must not contain 'depends on:' when no task declares a dependency")
	}
}

func seedAnalysisSpecWithAnswers(t *testing.T, root, slug string, tasks []spec.Task, answers map[string]string) *engine.Context {
	t.Helper()
	seedAnalysisSpec(t, root, slug, tasks)
	ctx := buildAnalysisCtx(t, root, slug)
	for key, value := range answers {
		if err := ctx.SetAnswer(key, value); err != nil {
			t.Fatalf("SetAnswer %s: %v", key, err)
		}
	}
	return buildAnalysisCtx(t, root, slug)
}

func TestAuditorInstruction_CarriesFullGivenWhenThen(t *testing.T) {
	root := t.TempDir()
	tasks := []spec.Task{{
		ID:    "task-1",
		Title: "Discount",
		Criteria: []spec.Criterion{{
			ID:    "ac-1",
			Given: "a cart totalling 200 with SAVE10 applied",
			When:  "DiscountedTotal is called",
			Then:  "it returns 180",
		}},
	}}
	ctx := seedAnalysisSpecWithAnswers(t, root, "audit-gwt", tasks, nil)

	got := buildAuditorInstruction(ctx, tasks, nil)
	for _, want := range []string{
		"a cart totalling 200 with SAVE10 applied",
		"DiscountedTotal is called",
		"it returns 180",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("auditor prompt must carry %q, got:\n%s", want, got)
		}
	}
}

func TestAuditorInstruction_CarriesContextAndRejectedPremises(t *testing.T) {
	root := t.TempDir()
	tasks := []spec.Task{{
		ID:       "task-1",
		Title:    "Coupons",
		Criteria: []spec.Criterion{{ID: "ac-1", Then: "it works"}},
	}}
	answers := map[string]string{
		"listen_context": "Add percentage coupons to the cart package only.",
		"verification":   "go test ./... with table-driven unit tests",
		"premises":       `{"premises":[{"text":"Coupons may stack","agreed":false},{"text":"Codes come from a map","agreed":true}]}`,
	}
	ctx := seedAnalysisSpecWithAnswers(t, root, "audit-context", tasks, answers)

	got := buildAuditorInstruction(ctx, tasks, nil)
	if !strings.Contains(got, "Add percentage coupons to the cart package only.") {
		t.Fatalf("auditor prompt must carry the listen-first context, got:\n%s", got)
	}
	if !strings.Contains(got, "go test ./... with table-driven unit tests") {
		t.Fatalf("auditor prompt must carry the verification method, got:\n%s", got)
	}
	if !strings.Contains(got, "Coupons may stack -> REJECTED") {
		t.Fatalf("auditor prompt must carry rejected premises, got:\n%s", got)
	}
	if strings.Contains(got, "Codes come from a map") {
		t.Fatalf("accepted premises must not be listed as challenged, got:\n%s", got)
	}
}

func TestAnalysis_InfoOnlyFindings_AreSurfacedWhenThePhaseCloses(t *testing.T) {
	root := t.TempDir()
	slug := "audit-info-only"
	tasks := []spec.Task{{ID: "task-1", Title: "one", Criteria: []spec.Criterion{{ID: "ac-1", Given: "a cart", When: "checkout runs", Then: "it works"}}}}
	seedAnalysisSpec(t, root, slug, tasks)
	ctx := buildAnalysisCtx(t, root, slug)

	verdict := `{"verdict":"clean","findings":[{"severity":"info","category":"note","taskId":"task-1","detail":"rounding is pinned by ac-1","suggestion":"none","source":"auditor"}]}`
	action, err := ctx.Submit([]byte(verdict))
	if err != nil {
		t.Fatalf("Submit auditor verdict: %v", err)
	}
	if action.Action != engine.ActionNotify {
		t.Fatalf("advisory findings must be surfaced as a notify, got %q", action.Action)
	}
	if !strings.Contains(action.Instruction, "rounding is pinned by ac-1") {
		t.Fatalf("notify must carry the finding detail, got:\n%s", action.Instruction)
	}
}

func TestAnalysis_NoFindings_ClosesSilently(t *testing.T) {
	root := t.TempDir()
	slug := "audit-clean"
	tasks := []spec.Task{{ID: "task-1", Title: "one", Criteria: []spec.Criterion{{ID: "ac-1", Given: "a cart", When: "checkout runs", Then: "it works"}}}}
	seedAnalysisSpec(t, root, slug, tasks)
	ctx := buildAnalysisCtx(t, root, slug)

	action, err := ctx.Submit([]byte(`{"verdict":"clean","findings":[]}`))
	if err != nil {
		t.Fatalf("Submit auditor verdict: %v", err)
	}
	if action.Action == engine.ActionNotify {
		t.Fatalf("a clean audit must not emit an advisory notify, got %q", action.Instruction)
	}
}

func TestAnalysis_ReturnToRefinement_RewindsPhaseAndClearsApproval(t *testing.T) {
	root := t.TempDir()
	slug := "r2r"
	seedAnalysisSpec(t, root, slug, tasksWithCriteria())

	ctx := buildAnalysisCtx(t, root, slug)
	if err := ctx.SetAnswer(refinementApprovedKey, "approve"); err != nil {
		t.Fatalf("SetAnswer: %v", err)
	}
	d := &analysisDriver{}
	verdict := `{"verdict":"issues","findings":[{"severity":"warn","category":"scope-gap","taskId":"task-1","detail":"gap"}]}`
	if _, _, err := d.Submit(ctx, nil, []byte(verdict)); err != nil {
		t.Fatalf("auditor Submit: %v", err)
	}

	if _, _, err := d.Submit(ctx, nil, []byte(optReturnToRefinement)); err != nil {
		t.Fatalf("return-to-refinement Submit: %v", err)
	}

	state, err := spec.LoadState(root, slug)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if state.Phase != string(phasecatalog.PhaseRefinement) {
		t.Fatalf("phase = %q, want refinement", state.Phase)
	}
	for _, key := range []string{refinementApprovedKey, answerKeyAudited, answerKeyFindings, answerKeyComplete} {
		if entries, ok := state.Answers[key]; ok && len(entries) > 0 && entries[0].Value != "" {
			t.Fatalf("answer %q should be cleared, got %q", key, entries[0].Value)
		}
	}
}

func TestAnalysis_DecisionGate_CarriesExpectedInput(t *testing.T) {
	root := t.TempDir()
	slug := "gate-input"
	seedAnalysisSpec(t, root, slug, tasksWithCriteria())

	ctx := buildAnalysisCtx(t, root, slug)
	d := &analysisDriver{}
	verdict := `{"verdict":"issues","findings":[{"severity":"warn","category":"scope-gap","taskId":"task-1","detail":"gap"}]}`
	if _, _, err := d.Submit(ctx, nil, []byte(verdict)); err != nil {
		t.Fatalf("auditor Submit: %v", err)
	}

	action, done := d.Next(ctx, nil)
	if done {
		t.Fatal("expected the phase to pause on a warn finding")
	}
	if action.ExpectedInput.Format == "" {
		t.Fatal("decision gate must declare an input format")
	}
	if action.ExpectedInput.Example == "" {
		t.Fatal("decision gate must carry an example answer")
	}
	if got := action.CommandMap[optReturnToRefinement]; !strings.Contains(got, "--answer='return-to-refinement'") {
		t.Fatalf("return-to-refinement command should take no payload, got %q", got)
	}
}
