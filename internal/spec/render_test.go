package spec

import (
	"os"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/paths"
)

func buildFullState() State {
	return State{
		Version: 1,
		Slug:    "my-spec",
		Phase:   PhaseInitial,
		Answers: map[string][]Answer{
			"premises": {
				{Key: "premises", Value: "We assume REST over gRPC."},
			},
			"scope_boundary": {
				{Key: "scope_boundary", Value: "No UI changes."},
			},
			"edge_cases": {
				{Key: "edge_cases", Value: "Empty payload must return 400."},
			},
			"verification": {
				{Key: "verification", Value: "Run integration tests."},
			},
			"context": {
				{Key: "context", Value: "Background context here."},
			},
			"approach": {
				{Key: "approach", Value: "Use layered architecture."},
			},
		},
	}
}

func buildFullProgress() Progress {
	return Progress{
		Spec:   "my-spec",
		Status: StatusExecuting,
		Tasks: []Task{
			{
				ID:         "task-1",
				Title:      "Bootstrap",
				AC:         []string{"ac line one", "ac line two"},
				Done:       false,
				TDDEnabled: true,
				Important:  true,
			},
			{
				ID:         "task-2",
				Title:      "Other title",
				AC:         []string{},
				Done:       true,
				TDDEnabled: false,
				Important:  false,
			},
		},
	}
}

func TestRenderSpecMd_GoldenFullSpec(t *testing.T) {
	st := buildFullState()
	pr := buildFullProgress()

	got := RenderSpecMd("my-spec", st, pr)

	want := "# Spec: my-spec\n\n" +
		"## Status\n" +
		"executing\n\n" +
		"## Discovery Answers\n" +
		"### approach\n" +
		"Use layered architecture.\n\n" +
		"### context\n" +
		"Background context here.\n\n" +
		"## Decisions\n" +
		"We assume REST over gRPC.\n\n" +
		"## Out of Scope\n" +
		"No UI changes.\n\n" +
		"## Edge Cases\n" +
		"- Empty payload must return 400\n\n" +
		"## Tasks\n" +
		"- [ ] task-1: Bootstrap (TDD) (important)\n" +
		"  - ac line one\n" +
		"  - ac line two\n" +
		"- [x] task-2: Other title\n\n" +
		"## Verification\n" +
		"Run integration tests.\n"

	if got != want {
		t.Errorf("RenderSpecMd golden mismatch.\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}

func TestRenderSpecMd_SlugInHeader(t *testing.T) {
	st := State{
		Version: 1,
		Slug:    "custom-slug",
		Phase:   PhaseInitial,
		Answers: map[string][]Answer{},
	}
	pr := Progress{Spec: "custom-slug", Status: StatusDraft, Tasks: []Task{}}

	got := RenderSpecMd("custom-slug", st, pr)

	if !strings.HasPrefix(got, "# Spec: custom-slug\n") {
		t.Errorf("expected header '# Spec: custom-slug', got first line: %q", strings.SplitN(got, "\n", 2)[0])
	}
}


func TestRenderSpecMd_HiddenKeysOmitted(t *testing.T) {
	st := State{
		Version: 1,
		Slug:    "s",
		Phase:   PhaseInitial,
		Answers: map[string][]Answer{
			"mode":            {{Key: "mode", Value: "full"}},
			"listen_context":  {{Key: "listen_context", Value: "ctx"}},
			"self_review":     {{Key: "self_review", Value: "approve"}},
			"synthesis":       {{Key: "synthesis", Value: "approve"}},
			"tasks_generated": {{Key: "tasks_generated", Value: "{}"}},
			"status_quo":      {{Key: "status_quo", Value: "current state"}},
		},
	}
	pr := Progress{Spec: "s", Status: StatusDraft, Tasks: []Task{}}

	got := RenderSpecMd("s", st, pr)

	for _, k := range []string{"### mode", "### listen_context", "### self_review", "### synthesis", "### tasks_generated"} {
		if strings.Contains(got, k) {
			t.Errorf("hidden key %q must not appear, got:\n%s", k, got)
		}
	}
	if !strings.Contains(got, "### status_quo\ncurrent state") {
		t.Errorf("expected status_quo kept in Discovery Answers, got:\n%s", got)
	}
}

func TestRenderSpecMd_DecisionsJSONAsBullets(t *testing.T) {
	st := State{
		Version: 1,
		Slug:    "s",
		Phase:   PhaseInitial,
		Answers: map[string][]Answer{
			"premises": {{Key: "premises", Value: `{"premises":[{"text":"In-memory only","agreed":true},{"text":"Single user","agreed":false,"revision":"allow two"}]}`}},
		},
	}
	pr := Progress{Spec: "s", Status: StatusDraft, Tasks: []Task{}}

	got := RenderSpecMd("s", st, pr)

	if !strings.Contains(got, "## Decisions\n- In-memory only\n- Single user (revize: allow two)\n") {
		t.Errorf("expected premises rendered as bullets, got:\n%s", got)
	}
	if strings.Contains(got, "{\"premises\"") {
		t.Errorf("raw JSON must not appear in Decisions, got:\n%s", got)
	}
}

func TestRenderSpecMd_EdgeCasesNumberedSplitToBullets(t *testing.T) {
	st := State{
		Version: 1,
		Slug:    "s",
		Phase:   PhaseInitial,
		Answers: map[string][]Answer{
			"edge_cases": {{Key: "edge_cases", Value: "Edge cases: (1) bad ID errors. (2) empty title rejected. (3) empty list message."}},
		},
	}
	pr := Progress{Spec: "s", Status: StatusDraft, Tasks: []Task{}}

	got := RenderSpecMd("s", st, pr)

	want := "## Edge Cases\n- bad ID errors\n- empty title rejected\n- empty list message\n"
	if !strings.Contains(got, want) {
		t.Errorf("expected edge cases split into bullets, got:\n%s", got)
	}
}

func TestRenderSpecMd_MissingSpecialAnswers_NoneRendered(t *testing.T) {
	st := State{
		Version: 1,
		Slug:    "s",
		Phase:   PhaseInitial,
		Answers: map[string][]Answer{},
	}
	pr := Progress{Spec: "s", Status: StatusDraft, Tasks: []Task{}}

	got := RenderSpecMd("s", st, pr)

	for _, section := range []string{"## Decisions\n_None_", "## Out of Scope\n_None_", "## Edge Cases\n_None_", "## Verification\n_None_"} {
		if !strings.Contains(got, section) {
			t.Errorf("expected %q in output, got:\n%s", section, got)
		}
	}
}

func TestRenderSpecMd_EmptySpecialAnswerValues_NoneRendered(t *testing.T) {
	st := State{
		Version: 1,
		Slug:    "s",
		Phase:   PhaseInitial,
		Answers: map[string][]Answer{
			"premises":       {{Key: "premises", Value: ""}},
			"scope_boundary": {{Key: "scope_boundary", Value: ""}},
			"edge_cases":     {{Key: "edge_cases", Value: ""}},
			"verification":   {{Key: "verification", Value: ""}},
		},
	}
	pr := Progress{Spec: "s", Status: StatusDraft, Tasks: []Task{}}

	got := RenderSpecMd("s", st, pr)

	for _, section := range []string{"## Decisions\n_None_", "## Out of Scope\n_None_", "## Edge Cases\n_None_", "## Verification\n_None_"} {
		if !strings.Contains(got, section) {
			t.Errorf("expected %q in output when value is empty, got:\n%s", section, got)
		}
	}
}

func TestRenderSpecMd_EmptyTasks_NoneRendered(t *testing.T) {
	st := State{Version: 1, Slug: "s", Phase: PhaseInitial, Answers: map[string][]Answer{}}
	pr := Progress{Spec: "s", Status: StatusDraft, Tasks: []Task{}}

	got := RenderSpecMd("s", st, pr)

	if !strings.Contains(got, "## Tasks\n_None_") {
		t.Errorf("expected '## Tasks\\n_None_' for empty task list, got:\n%s", got)
	}
}

func TestRenderSpecMd_TaskDone_CheckboxX(t *testing.T) {
	st := State{Version: 1, Slug: "s", Phase: PhaseInitial, Answers: map[string][]Answer{}}
	pr := Progress{
		Spec:   "s",
		Status: StatusDraft,
		Tasks: []Task{
			{ID: "task-1", Title: "Done task", Done: true},
		},
	}

	got := RenderSpecMd("s", st, pr)

	if !strings.Contains(got, "- [x] task-1: Done task\n") {
		t.Errorf("expected '[x]' for done task, got:\n%s", got)
	}
}

func TestRenderSpecMd_TaskNotDone_EmptyCheckbox(t *testing.T) {
	st := State{Version: 1, Slug: "s", Phase: PhaseInitial, Answers: map[string][]Answer{}}
	pr := Progress{
		Spec:   "s",
		Status: StatusDraft,
		Tasks: []Task{
			{ID: "task-1", Title: "Open task", Done: false},
		},
	}

	got := RenderSpecMd("s", st, pr)

	if !strings.Contains(got, "- [ ] task-1: Open task\n") {
		t.Errorf("expected '[ ]' for incomplete task, got:\n%s", got)
	}
}

func TestRenderSpecMd_TaskTDDEnabled_AppendsTDDTag(t *testing.T) {
	st := State{Version: 1, Slug: "s", Phase: PhaseInitial, Answers: map[string][]Answer{}}
	pr := Progress{
		Spec:   "s",
		Status: StatusDraft,
		Tasks: []Task{
			{ID: "task-1", Title: "T", Done: false, TDDEnabled: true, Important: false},
		},
	}

	got := RenderSpecMd("s", st, pr)

	if !strings.Contains(got, "- [ ] task-1: T (TDD)\n") {
		t.Errorf("expected '(TDD)' tag for TDDEnabled task, got:\n%s", got)
	}
}

func TestRenderSpecMd_TaskImportant_AppendsImportantTag(t *testing.T) {
	st := State{Version: 1, Slug: "s", Phase: PhaseInitial, Answers: map[string][]Answer{}}
	pr := Progress{
		Spec:   "s",
		Status: StatusDraft,
		Tasks: []Task{
			{ID: "task-1", Title: "T", Done: false, TDDEnabled: false, Important: true},
		},
	}

	got := RenderSpecMd("s", st, pr)

	if !strings.Contains(got, "- [ ] task-1: T (important)\n") {
		t.Errorf("expected '(important)' tag for Important task, got:\n%s", got)
	}
}

func TestRenderSpecMd_TaskBothTags_TDDBeforeImportant(t *testing.T) {
	st := State{Version: 1, Slug: "s", Phase: PhaseInitial, Answers: map[string][]Answer{}}
	pr := Progress{
		Spec:   "s",
		Status: StatusDraft,
		Tasks: []Task{
			{ID: "task-1", Title: "T", Done: false, TDDEnabled: true, Important: true},
		},
	}

	got := RenderSpecMd("s", st, pr)

	if !strings.Contains(got, "- [ ] task-1: T (TDD) (important)\n") {
		t.Errorf("expected both '(TDD) (important)' tags in that order, got:\n%s", got)
	}
}

func TestRenderSpecMd_TaskNoTags(t *testing.T) {
	st := State{Version: 1, Slug: "s", Phase: PhaseInitial, Answers: map[string][]Answer{}}
	pr := Progress{
		Spec:   "s",
		Status: StatusDraft,
		Tasks: []Task{
			{ID: "task-1", Title: "Plain", Done: false, TDDEnabled: false, Important: false},
		},
	}

	got := RenderSpecMd("s", st, pr)

	if strings.Contains(got, "(TDD)") || strings.Contains(got, "(important)") {
		t.Errorf("expected no tags for plain task, got:\n%s", got)
	}
	if !strings.Contains(got, "- [ ] task-1: Plain\n") {
		t.Errorf("expected plain task line, got:\n%s", got)
	}
}

func TestRenderSpecMd_TaskACSubBullets(t *testing.T) {
	st := State{Version: 1, Slug: "s", Phase: PhaseInitial, Answers: map[string][]Answer{}}
	pr := Progress{
		Spec:   "s",
		Status: StatusDraft,
		Tasks: []Task{
			{ID: "task-1", Title: "T", Done: false, AC: []string{"first ac", "second ac"}},
		},
	}

	got := RenderSpecMd("s", st, pr)

	if !strings.Contains(got, "  - first ac\n") {
		t.Errorf("expected AC sub-bullet '  - first ac', got:\n%s", got)
	}
	if !strings.Contains(got, "  - second ac\n") {
		t.Errorf("expected AC sub-bullet '  - second ac', got:\n%s", got)
	}
}

func TestRenderSpecMd_DiscoveryAnswersSortedLexicographic(t *testing.T) {
	st := State{
		Version: 1,
		Slug:    "s",
		Phase:   PhaseInitial,
		Answers: map[string][]Answer{
			"zebra":   {{Key: "zebra", Value: "last"}},
			"apple":   {{Key: "apple", Value: "first"}},
			"mango":   {{Key: "mango", Value: "middle"}},
		},
	}
	pr := Progress{Spec: "s", Status: StatusDraft, Tasks: []Task{}}

	got := RenderSpecMd("s", st, pr)

	appleIdx := strings.Index(got, "### apple")
	mangoIdx := strings.Index(got, "### mango")
	zebraIdx := strings.Index(got, "### zebra")

	if appleIdx == -1 || mangoIdx == -1 || zebraIdx == -1 {
		t.Fatalf("expected all discovery keys in output, got:\n%s", got)
	}
	if !(appleIdx < mangoIdx && mangoIdx < zebraIdx) {
		t.Errorf("expected discovery keys in sorted order (apple < mango < zebra), got positions: apple=%d mango=%d zebra=%d\n%s", appleIdx, mangoIdx, zebraIdx, got)
	}
}

func TestRenderSpecMd_SpecialKeysNotInDiscoveryAnswers(t *testing.T) {
	st := State{
		Version: 1,
		Slug:    "s",
		Phase:   PhaseInitial,
		Answers: map[string][]Answer{
			"premises":       {{Key: "premises", Value: "p"}},
			"scope_boundary": {{Key: "scope_boundary", Value: "sb"}},
			"edge_cases":     {{Key: "edge_cases", Value: "ec"}},
			"verification":   {{Key: "verification", Value: "v"}},
			"context":        {{Key: "context", Value: "ctx"}},
		},
	}
	pr := Progress{Spec: "s", Status: StatusDraft, Tasks: []Task{}}

	got := RenderSpecMd("s", st, pr)

	for _, specialKey := range []string{"### premises", "### scope_boundary", "### edge_cases", "### verification"} {
		if strings.Contains(got, specialKey) {
			t.Errorf("special key %q must not appear in ## Discovery Answers, got:\n%s", specialKey, got)
		}
	}

	if !strings.Contains(got, "### context") {
		t.Errorf("expected non-special key 'context' in Discovery Answers, got:\n%s", got)
	}
}

func TestRenderSpecMd_NoNonSpecialKeys_DiscoveryNone(t *testing.T) {
	st := State{
		Version: 1,
		Slug:    "s",
		Phase:   PhaseInitial,
		Answers: map[string][]Answer{
			"premises": {{Key: "premises", Value: "p"}},
		},
	}
	pr := Progress{Spec: "s", Status: StatusDraft, Tasks: []Task{}}

	got := RenderSpecMd("s", st, pr)

	if !strings.Contains(got, "## Discovery Answers\n_None_") {
		t.Errorf("expected Discovery Answers to show '_None_' when no non-special keys, got:\n%s", got)
	}
}

func TestRenderSpecMd_MultipleAnswerValuesJoinedByNewline(t *testing.T) {
	st := State{
		Version: 1,
		Slug:    "s",
		Phase:   PhaseInitial,
		Answers: map[string][]Answer{
			"premises": {
				{Key: "premises", Value: "line one"},
				{Key: "premises", Value: "line two"},
			},
		},
	}
	pr := Progress{Spec: "s", Status: StatusDraft, Tasks: []Task{}}

	got := RenderSpecMd("s", st, pr)

	if !strings.Contains(got, "## Decisions\nline one\nline two") {
		t.Errorf("expected multi-answer values joined by newline in Decisions, got:\n%s", got)
	}
}

func TestRenderSpecMd_Idempotent(t *testing.T) {
	st := buildFullState()
	pr := buildFullProgress()

	first := RenderSpecMd("my-spec", st, pr)
	second := RenderSpecMd("my-spec", st, pr)

	if first != second {
		t.Errorf("RenderSpecMd must be idempotent: first and second calls differ.\nfirst:\n%s\n\nsecond:\n%s", first, second)
	}
}

func TestRenderSpecMd_StatusSection_Executing(t *testing.T) {
	st := State{Version: 1, Slug: "s", Phase: PhaseInitial, Answers: map[string][]Answer{}}
	pr := Progress{Spec: "s", Status: StatusExecuting, Tasks: []Task{}}

	got := RenderSpecMd("s", st, pr)

	if !strings.Contains(got, "## Status\n") {
		t.Errorf("expected '## Status' section in output, got:\n%s", got)
	}
	if !strings.Contains(got, StatusExecuting) {
		t.Errorf("expected status value %q in output, got:\n%s", StatusExecuting, got)
	}
}

func TestRenderSpecMd_StatusSection_Completed(t *testing.T) {
	st := State{Version: 1, Slug: "s", Phase: PhaseInitial, Answers: map[string][]Answer{}}
	pr := Progress{Spec: "s", Status: StatusCompleted, Tasks: []Task{}}

	got := RenderSpecMd("s", st, pr)

	if !strings.Contains(got, "## Status\n") {
		t.Errorf("expected '## Status' section in output, got:\n%s", got)
	}
	if !strings.Contains(got, StatusCompleted) {
		t.Errorf("expected status value %q in output, got:\n%s", StatusCompleted, got)
	}
}

func TestRenderSpecMd_DoneCheckbox(t *testing.T) {
	st := State{Version: 1, Slug: "s", Phase: PhaseInitial, Answers: map[string][]Answer{}}
	pr := Progress{
		Spec:   "s",
		Status: StatusExecuting,
		Tasks: []Task{
			{ID: "t1", Title: "done task", Done: true},
			{ID: "t2", Title: "pending task", Done: false},
		},
	}

	got := RenderSpecMd("s", st, pr)

	if !strings.Contains(got, "- [x] t1: done task\n") {
		t.Errorf("expected '[x]' checkbox for done task, got:\n%s", got)
	}
	if !strings.Contains(got, "- [ ] t2: pending task\n") {
		t.Errorf("expected '[ ]' checkbox for pending task, got:\n%s", got)
	}
}

func TestRenderSpecMd_DeterministicMapKeyOrder(t *testing.T) {
	answers := map[string][]Answer{
		"zzz": {{Key: "zzz", Value: "z-val"}},
		"aaa": {{Key: "aaa", Value: "a-val"}},
		"mmm": {{Key: "mmm", Value: "m-val"}},
	}

	results := make([]string, 5)
	for i := range results {
		st := State{Version: 1, Slug: "s", Phase: PhaseInitial, Answers: answers}
		pr := Progress{Spec: "s", Status: StatusDraft, Tasks: []Task{}}
		results[i] = RenderSpecMd("s", st, pr)
	}

	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			t.Errorf("RenderSpecMd is not deterministic: call 0 and call %d differ.\ncall 0:\n%s\n\ncall %d:\n%s", i, results[0], i, results[i])
		}
	}

	got := results[0]
	aaaIdx := strings.Index(got, "### aaa")
	mmmIdx := strings.Index(got, "### mmm")
	zzzIdx := strings.Index(got, "### zzz")

	if !(aaaIdx < mmmIdx && mmmIdx < zzzIdx) {
		t.Errorf("keys not in sorted order across repeated calls: aaa=%d mmm=%d zzz=%d", aaaIdx, mmmIdx, zzzIdx)
	}
}

func TestSaveSpecMd_WritesContentToCorrectPath(t *testing.T) {
	root := t.TempDir()
	slug := "test-spec"
	content := "# Spec: test-spec\n\n## Status\ndraft\n"

	err := SaveSpecMd(root, slug, content)
	if err != nil {
		t.Fatalf("SaveSpecMd returned error: %v", err)
	}

	data, err := os.ReadFile(paths.SpecMd(root, slug))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	if string(data) != content {
		t.Errorf("file content mismatch.\nwant: %q\ngot:  %q", content, string(data))
	}
}

func TestSaveSpecMd_FileMode0644(t *testing.T) {
	root := t.TempDir()
	slug := "perm-spec"

	err := SaveSpecMd(root, slug, "content")
	if err != nil {
		t.Fatalf("SaveSpecMd returned error: %v", err)
	}

	info, err := os.Stat(paths.SpecMd(root, slug))
	if err != nil {
		t.Fatalf("Stat returned error: %v", err)
	}

	if info.Mode().Perm() != 0o644 {
		t.Errorf("expected file mode 0644, got %o", info.Mode().Perm())
	}
}

func TestSaveSpecMd_CreatesSpecDirIfAbsent(t *testing.T) {
	root := t.TempDir()
	slug := "new-spec"

	err := SaveSpecMd(root, slug, "hello")
	if err != nil {
		t.Fatalf("SaveSpecMd returned error: %v", err)
	}

	if _, err := os.Stat(paths.SpecDir(root, slug)); err != nil {
		t.Errorf("expected spec dir to exist after SaveSpecMd, got: %v", err)
	}
}

func TestParseEdgeCases_Exported_NumberedInput(t *testing.T) {
	input := "Edge cases: (1) bad ID errors. (2) empty title rejected. (3) empty list message."
	got := ParseEdgeCases(input)
	want := []string{"bad ID errors", "empty title rejected", "empty list message"}
	if len(got) != len(want) {
		t.Fatalf("ParseEdgeCases len = %d, want %d; got %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ParseEdgeCases[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestParseEdgeCases_Exported_MultilineInput(t *testing.T) {
	input := "nil payload\nempty list\nlong string"
	got := ParseEdgeCases(input)
	want := []string{"nil payload", "empty list", "long string"}
	if len(got) != len(want) {
		t.Fatalf("ParseEdgeCases multiline len = %d, want %d; got %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ParseEdgeCases[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestParseEdgeCases_Exported_EmptyString_ReturnsNil(t *testing.T) {
	got := ParseEdgeCases("")
	if got != nil {
		t.Errorf("ParseEdgeCases(\"\") = %v, want nil", got)
	}
}

func TestSaveSpecMd_OverwritesExistingFile(t *testing.T) {
	root := t.TempDir()
	slug := "overwrite-spec"

	if err := SaveSpecMd(root, slug, "original content"); err != nil {
		t.Fatalf("first SaveSpecMd returned error: %v", err)
	}

	if err := SaveSpecMd(root, slug, "updated content"); err != nil {
		t.Fatalf("second SaveSpecMd returned error: %v", err)
	}

	data, err := os.ReadFile(paths.SpecMd(root, slug))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	if string(data) != "updated content" {
		t.Errorf("expected overwritten content, got: %q", string(data))
	}
}
