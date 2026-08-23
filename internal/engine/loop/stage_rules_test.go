package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/promptregistry"
	"github.com/pragmataW/tddmaster/internal/rules"
	"github.com/pragmataW/tddmaster/internal/spec"
)

const rulesSectionMarker = promptregistry.SectionRules

var ruleBodyMarkers = []string{"UNIQUE_GLOBAL_MARKER_CONTENT", "UNIQUE_AGENT_MARKER_CONTENT"}

func writeRuleFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func loadRulesFixture(t *testing.T, root string) rules.Set {
	t.Helper()
	s, err := rules.Load(root)
	if err != nil {
		t.Fatalf("rules.Load: %v", err)
	}
	return s
}

func rulesFixtureRoot(t *testing.T, agent string) (string, rules.Set) {
	t.Helper()
	root := t.TempDir()
	rulesBase := filepath.Join(root, ".tddmaster", "rules")
	writeRuleFile(t, filepath.Join(rulesBase, "global.md"), "UNIQUE_GLOBAL_MARKER_CONTENT")
	writeRuleFile(t, filepath.Join(rulesBase, agent, "agent-rule.md"), "UNIQUE_AGENT_MARKER_CONTENT")
	return root, loadRulesFixture(t, root)
}

func ctxWithRules(settings spec.Settings, task spec.Task, state spec.ExecState, r rules.Set) ExecCtx {
	ctx := makeExecCtx(settings, task, state, 0, 3)
	ctx.Rules = r
	return ctx
}

func ctxWithEmptyRules(settings spec.Settings, task spec.Task, state spec.ExecState) ExecCtx {
	return makeExecCtx(settings, task, state, 0, 3)
}

func assertContainsMandatoryBlock(t *testing.T, instruction, path1, path2 string) {
	t.Helper()
	if !strings.Contains(instruction, rulesSectionMarker) {
		t.Errorf("Instruction missing rules section; got:\n%s", instruction)
	}
	for _, p := range []string{path1, path2} {
		if !strings.Contains(instruction, p) {
			t.Errorf("Instruction missing rule path %q; got:\n%s", p, instruction)
		}
	}
	for _, marker := range ruleBodyMarkers {
		if !strings.Contains(instruction, marker) {
			t.Errorf("Instruction missing inlined rule body %q; got:\n%s", marker, instruction)
		}
	}
}

func assertNoRulesBlock(t *testing.T, instruction string) {
	t.Helper()
	if strings.Contains(instruction, rulesSectionMarker) {
		t.Errorf("Instruction must NOT contain a rules section when no rules apply; got:\n%s", instruction)
	}
	for _, marker := range ruleBodyMarkers {
		if strings.Contains(instruction, marker) {
			t.Errorf("Instruction must NOT contain rule body %q when no rules apply; got:\n%s", marker, instruction)
		}
	}
}

func TestAppendRules_RedStage_InjectsTestWriterRules(t *testing.T) {
	root, r := rulesFixtureRoot(t, "test-writer")
	_ = root
	paths := r.For("test-writer")
	if len(paths) < 2 {
		t.Fatalf("fixture setup error: expected 2+ paths, got %v", paths)
	}

	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := ctxWithRules(settings, task, makeExecState("red"), r)

	action := redStage().Prompt(ctx)

	assertContainsMandatoryBlock(t, action.Instruction, paths[0], paths[1])
}

func TestAppendRules_GreenStage_InjectsExecutorRules(t *testing.T) {
	root, r := rulesFixtureRoot(t, "executor")
	_ = root
	paths := r.For("executor")
	if len(paths) < 2 {
		t.Fatalf("fixture setup error: expected 2+ paths, got %v", paths)
	}

	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := ctxWithRules(settings, task, makeExecState("green"), r)

	action := greenStage().Prompt(ctx)

	assertContainsMandatoryBlock(t, action.Instruction, paths[0], paths[1])
}

func TestAppendRules_RefactorApplyBranch_InjectsExecutorRules(t *testing.T) {
	root, r := rulesFixtureRoot(t, "executor")
	_ = root
	paths := r.For("executor")
	if len(paths) < 2 {
		t.Fatalf("fixture setup error: expected 2+ paths, got %v", paths)
	}

	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	st := makeExecState("refactor")
	st.RefactorApplied = false
	ctx := ctxWithRules(settings, task, st, r)

	action := refactorStage().Prompt(ctx)

	assertContainsMandatoryBlock(t, action.Instruction, paths[0], paths[1])
}

func TestAppendRules_RefactorVerifyBranch_InjectsVerifierRulesForVerifierDelegate(t *testing.T) {
	root, r := rulesFixtureRoot(t, "verifier")
	_ = root
	paths := r.For("verifier")
	if len(paths) < 2 {
		t.Fatalf("fixture setup error: expected 2+ paths, got %v", paths)
	}

	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	st := makeExecState("refactor")
	st.RefactorApplied = true
	ctx := ctxWithRules(settings, task, st, r)

	action := refactorStage().Prompt(ctx)

	if action.DelegateAgent != string(promptregistry.AgentVerifier) {
		t.Fatalf("verify branch should delegate to verifier, got %q", action.DelegateAgent)
	}
	assertContainsMandatoryBlock(t, action.Instruction, paths[0], paths[1])
}

func TestAppendRules_ExecutorStage_InjectsExecutorRules(t *testing.T) {
	root, r := rulesFixtureRoot(t, "executor")
	_ = root
	paths := r.For("executor")
	if len(paths) < 2 {
		t.Fatalf("fixture setup error: expected 2+ paths, got %v", paths)
	}

	settings := makeSettings(false, false, false)
	task := makeTask("t-1", false, false)
	ctx := ctxWithRules(settings, task, makeExecState(""), r)

	action := executorStage().Prompt(ctx)

	assertContainsMandatoryBlock(t, action.Instruction, paths[0], paths[1])
}

func TestAppendRules_VerifierStage_InjectsVerifierRules(t *testing.T) {
	root, r := rulesFixtureRoot(t, "verifier")
	_ = root
	paths := r.For("verifier")
	if len(paths) < 2 {
		t.Fatalf("fixture setup error: expected 2+ paths, got %v", paths)
	}

	settings := makeSettings(false, false, false)
	task := makeTask("t-1", false, false)
	ctx := ctxWithRules(settings, task, makeExecState(""), r)

	action := verifierStage().Prompt(ctx)

	assertContainsMandatoryBlock(t, action.Instruction, paths[0], paths[1])
}

func TestAppendRules_GateStage_InjectsPlannerRules(t *testing.T) {
	root, r := rulesFixtureRoot(t, "planner")
	_ = root
	paths := r.For("planner")
	if len(paths) < 2 {
		t.Fatalf("fixture setup error: expected 2+ paths, got %v", paths)
	}

	settings := makeSettings(false, false, true)
	task := makeImportantTask("t-1", false)
	ctx := ctxWithRules(settings, task, makeExecState(""), r)

	action := gateStage().Prompt(ctx)

	assertContainsMandatoryBlock(t, action.Instruction, paths[0], paths[1])
}

func TestAppendRules_GlobalBeforeAgentSpecific_LexicalOrder(t *testing.T) {
	root := t.TempDir()
	rulesBase := filepath.Join(root, ".tddmaster", "rules")
	writeRuleFile(t, filepath.Join(rulesBase, "zzz-global.md"), "content-z")
	writeRuleFile(t, filepath.Join(rulesBase, "aaa-global.md"), "content-a")
	writeRuleFile(t, filepath.Join(rulesBase, "test-writer", "zzz-agent.md"), "content-za")
	writeRuleFile(t, filepath.Join(rulesBase, "test-writer", "aaa-agent.md"), "content-aa")
	r := loadRulesFixture(t, root)

	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := ctxWithRules(settings, task, makeExecState("red"), r)

	action := redStage().Prompt(ctx)

	globalAAA := ".tddmaster/rules/aaa-global.md"
	globalZZZ := ".tddmaster/rules/zzz-global.md"
	agentAAA := ".tddmaster/rules/test-writer/aaa-agent.md"
	agentZZZ := ".tddmaster/rules/test-writer/zzz-agent.md"

	posGlobalAAA := strings.Index(action.Instruction, globalAAA)
	posGlobalZZZ := strings.Index(action.Instruction, globalZZZ)
	posAgentAAA := strings.Index(action.Instruction, agentAAA)
	posAgentZZZ := strings.Index(action.Instruction, agentZZZ)

	if posGlobalAAA < 0 {
		t.Errorf("Instruction missing %q", globalAAA)
	}
	if posGlobalZZZ < 0 {
		t.Errorf("Instruction missing %q", globalZZZ)
	}
	if posAgentAAA < 0 {
		t.Errorf("Instruction missing %q", agentAAA)
	}
	if posAgentZZZ < 0 {
		t.Errorf("Instruction missing %q", agentZZZ)
	}

	if posGlobalAAA > posGlobalZZZ {
		t.Errorf("aaa-global.md must appear before zzz-global.md (lexical order)")
	}
	if posGlobalZZZ > posAgentAAA {
		t.Errorf("global rules must appear before agent-specific rules")
	}
	if posAgentAAA > posAgentZZZ {
		t.Errorf("aaa-agent.md must appear before zzz-agent.md (lexical order)")
	}
}

func TestAppendRules_SmallRules_InlinedWithPathLabel(t *testing.T) {
	uniqueMarker := "UNIQUE_RULE_BODY_XYZ987"
	root := t.TempDir()
	rulesBase := filepath.Join(root, ".tddmaster", "rules")
	writeRuleFile(t, filepath.Join(rulesBase, "test-writer", "small-rule.md"), uniqueMarker)
	r := loadRulesFixture(t, root)

	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := ctxWithRules(settings, task, makeExecState("red"), r)

	action := redStage().Prompt(ctx)

	if !strings.Contains(action.Instruction, uniqueMarker) {
		t.Errorf("Instruction must inline the rule body; got:\n%s", action.Instruction)
	}
	if !strings.Contains(action.Instruction, ".tddmaster/rules/test-writer/small-rule.md") {
		t.Errorf("Instruction must label the inlined body with its rule file path")
	}
}

func TestAppendRules_OversizedRules_FallBackToPathsOnly(t *testing.T) {
	body := strings.Repeat("x", inlineRulesBudget+1)
	root := t.TempDir()
	rulesBase := filepath.Join(root, ".tddmaster", "rules")
	writeRuleFile(t, filepath.Join(rulesBase, "test-writer", "huge-rule.md"), body)
	r := loadRulesFixture(t, root)

	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := ctxWithRules(settings, task, makeExecState("red"), r)

	action := redStage().Prompt(ctx)

	if strings.Contains(action.Instruction, body) {
		t.Error("Instruction must NOT inline a rule file larger than the inline budget")
	}
	if !strings.Contains(action.Instruction, "- .tddmaster/rules/test-writer/huge-rule.md") {
		t.Errorf("Instruction must list the rule path when inlining is skipped; got:\n%s", action.Instruction)
	}
	if !strings.Contains(action.Instruction, promptregistry.RulesTruncatedNote) {
		t.Errorf("Instruction must say WHY the bodies are missing when inlining is skipped; got:\n%s", action.Instruction)
	}
	if !strings.Contains(action.Instruction, promptregistry.RulesInjectionFooter) {
		t.Error("Instruction must instruct the agent to read the listed rule files")
	}
}

func TestAppendRules_EmptySet_RedStage_NoBlock(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := ctxWithEmptyRules(settings, task, makeExecState("red"))

	action := redStage().Prompt(ctx)

	assertNoRulesBlock(t, action.Instruction)
}

func TestAppendRules_EmptySet_GreenStage_NoBlock(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := ctxWithEmptyRules(settings, task, makeExecState("green"))

	action := greenStage().Prompt(ctx)

	assertNoRulesBlock(t, action.Instruction)
}

func TestAppendRules_EmptySet_RefactorApplyBranch_NoBlock(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	st := makeExecState("refactor")
	st.RefactorApplied = false
	ctx := ctxWithEmptyRules(settings, task, st)

	action := refactorStage().Prompt(ctx)

	assertNoRulesBlock(t, action.Instruction)
}

func TestAppendRules_EmptySet_RefactorVerifyBranch_NoBlock(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	st := makeExecState("refactor")
	st.RefactorApplied = true
	ctx := ctxWithEmptyRules(settings, task, st)

	action := refactorStage().Prompt(ctx)

	assertNoRulesBlock(t, action.Instruction)
}

func TestAppendRules_EmptySet_ExecutorStage_NoBlock(t *testing.T) {
	settings := makeSettings(false, false, false)
	task := makeTask("t-1", false, false)
	ctx := ctxWithEmptyRules(settings, task, makeExecState(""))

	action := executorStage().Prompt(ctx)

	assertNoRulesBlock(t, action.Instruction)
}

func TestAppendRules_EmptySet_VerifierStage_NoBlock(t *testing.T) {
	settings := makeSettings(false, false, false)
	task := makeTask("t-1", false, false)
	ctx := ctxWithEmptyRules(settings, task, makeExecState(""))

	action := verifierStage().Prompt(ctx)

	assertNoRulesBlock(t, action.Instruction)
}

func TestAppendRules_EmptySet_GateStage_NoBlock(t *testing.T) {
	settings := makeSettings(false, false, true)
	task := makeImportantTask("t-1", false)
	ctx := ctxWithEmptyRules(settings, task, makeExecState(""))

	action := gateStage().Prompt(ctx)

	assertNoRulesBlock(t, action.Instruction)
}

func TestAppendRules_MissingRulesDir_NoBlock(t *testing.T) {
	root := t.TempDir()
	r := loadRulesFixture(t, root)

	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := ctxWithRules(settings, task, makeExecState("red"), r)

	action := redStage().Prompt(ctx)

	assertNoRulesBlock(t, action.Instruction)
}

func TestAppendRules_AgentWithNoRules_NoBlock(t *testing.T) {
	root := t.TempDir()
	rulesBase := filepath.Join(root, ".tddmaster", "rules")
	writeRuleFile(t, filepath.Join(rulesBase, "executor", "exec-rule.md"), "executor only")
	r := loadRulesFixture(t, root)

	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := ctxWithRules(settings, task, makeExecState("red"), r)

	action := redStage().Prompt(ctx)

	assertNoRulesBlock(t, action.Instruction)
}

func TestAppendRules_EmptySetInstructionByteIdentical_RedStage(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)

	ctxZero := makeExecCtx(settings, task, makeExecState("red"), 0, 3)
	ctxEmpty := ctxWithEmptyRules(settings, task, makeExecState("red"))

	instrZero := redStage().Prompt(ctxZero).Instruction
	instrEmpty := redStage().Prompt(ctxEmpty).Instruction

	if instrZero != instrEmpty {
		t.Errorf("empty rules Set must produce byte-identical instruction to zero-value Set\ngot zero:  %q\ngot empty: %q", instrZero, instrEmpty)
	}
}

func TestAppendRules_EmptySetInstructionByteIdentical_GreenStage(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)

	ctxZero := makeExecCtx(settings, task, makeExecState("green"), 0, 3)
	ctxEmpty := ctxWithEmptyRules(settings, task, makeExecState("green"))

	instrZero := greenStage().Prompt(ctxZero).Instruction
	instrEmpty := greenStage().Prompt(ctxEmpty).Instruction

	if instrZero != instrEmpty {
		t.Errorf("empty rules Set must produce byte-identical instruction to zero-value Set\ngot zero:  %q\ngot empty: %q", instrZero, instrEmpty)
	}
}

func TestAppendRules_EmptySetInstructionByteIdentical_ExecutorStage(t *testing.T) {
	settings := makeSettings(false, false, false)
	task := makeTask("t-1", false, false)

	ctxZero := makeExecCtx(settings, task, makeExecState(""), 0, 3)
	ctxEmpty := ctxWithEmptyRules(settings, task, makeExecState(""))

	instrZero := executorStage().Prompt(ctxZero).Instruction
	instrEmpty := executorStage().Prompt(ctxEmpty).Instruction

	if instrZero != instrEmpty {
		t.Errorf("empty rules Set must produce byte-identical instruction to zero-value Set\ngot zero:  %q\ngot empty: %q", instrZero, instrEmpty)
	}
}

func TestAppendRules_EmptySetInstructionByteIdentical_VerifierStage(t *testing.T) {
	settings := makeSettings(false, false, false)
	task := makeTask("t-1", false, false)

	ctxZero := makeExecCtx(settings, task, makeExecState(""), 0, 3)
	ctxEmpty := ctxWithEmptyRules(settings, task, makeExecState(""))

	instrZero := verifierStage().Prompt(ctxZero).Instruction
	instrEmpty := verifierStage().Prompt(ctxEmpty).Instruction

	if instrZero != instrEmpty {
		t.Errorf("empty rules Set must produce byte-identical instruction to zero-value Set\ngot zero:  %q\ngot empty: %q", instrZero, instrEmpty)
	}
}

func TestAppendRules_EmptySetInstructionByteIdentical_GateStage(t *testing.T) {
	settings := makeSettings(false, false, true)
	task := makeImportantTask("t-1", false)

	ctxZero := makeExecCtx(settings, task, makeExecState(""), 0, 3)
	ctxEmpty := ctxWithEmptyRules(settings, task, makeExecState(""))

	instrZero := gateStage().Prompt(ctxZero).Instruction
	instrEmpty := gateStage().Prompt(ctxEmpty).Instruction

	if instrZero != instrEmpty {
		t.Errorf("empty rules Set must produce byte-identical instruction to zero-value Set\ngot zero:  %q\ngot empty: %q", instrZero, instrEmpty)
	}
}

func TestAppendRules_GlobalOnlyRules_AppliedToAllAgents(t *testing.T) {
	root := t.TempDir()
	rulesBase := filepath.Join(root, ".tddmaster", "rules")
	writeRuleFile(t, filepath.Join(rulesBase, "shared.md"), "global content")
	r := loadRulesFixture(t, root)

	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)

	ctxRed := ctxWithRules(settings, task, makeExecState("red"), r)
	actionRed := redStage().Prompt(ctxRed)
	if !strings.Contains(actionRed.Instruction, ".tddmaster/rules/shared.md") {
		t.Error("red stage: global rule must appear even without agent-specific rules")
	}
	if !strings.Contains(actionRed.Instruction, rulesSectionMarker) {
		t.Errorf("red stage: missing rules section; got:\n%s", actionRed.Instruction)
	}
	if !strings.Contains(actionRed.Instruction, "global content") {
		t.Errorf("red stage: global rule body must be inlined; got:\n%s", actionRed.Instruction)
	}

	ctxGreen := ctxWithRules(settings, task, makeExecState("green"), r)
	actionGreen := greenStage().Prompt(ctxGreen)
	if !strings.Contains(actionGreen.Instruction, ".tddmaster/rules/shared.md") {
		t.Error("green stage: global rule must appear")
	}
}
