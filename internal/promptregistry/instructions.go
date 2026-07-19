package promptregistry

import (
	"fmt"

	"github.com/pragmataW/tddmaster/internal/prompts"
)

var instructionMap = make(map[InstructionKey]string)

func init() {
	names := prompts.TemplateNames()
	for _, name := range names {
		rendered, err := prompts.Render(name, prompts.RenderData{})
		if err != nil {
			panic(fmt.Sprintf("promptregistry: failed to render template %q: %v", name, err))
		}
		instructionMap[InstructionKey(name)] = rendered
	}
}

func init() {
	instructionMap[KeySettings] = "Before discovery, configure this spec's settings. Ask the user via the AskUserQuestion tool (multiSelect) which features to enable — never as plain prose. Present the five settings from interactiveOptions with their defaults: TDD/Red-Green-Refactor (default ON), Skip verifier (default OFF), Important task gate (default OFF), Min test coverage (default 80; 0 = disabled), Rule learning (default OFF). A selected toggle means enabled (true); an unselected one means disabled (false). For min test coverage, ask the user for the percentage (0-100) and include it as the number \"minTestCoverage\". Aggregate the selections client-side and submit ALL five as a single JSON object: {\"tddEnabled\":true,\"skipVerifierEnabled\":false,\"importantTaskGateEnabled\":false,\"minTestCoverage\":80,\"ruleLearningEnabled\":false}."
	instructionMap[KeyListenFirst] = "The user just created this spec. Before starting discovery, ask them to share whatever context they have — requirements, notes, tasks, or just a brief description. Say: 'Tell me about this — share as much context as you have.' The shared context is the primary reference for this spec and will be passed verbatim to the test-writer, executor, and verifier sub-agents during task execution. Listen first, then proceed."
	instructionMap[KeyModeSelection] = "Before starting discovery, select the discovery mode via AskUserQuestion. Use the options provided in interactiveOptions — do NOT present them as prose or a numbered list."
	instructionMap[KeyPremiseChallenge] = "Read the spec description. Identify 2-4 premises the spec assumes. Present each premise and ask the user to agree or disagree. Submit as JSON: {\"premises\":[{\"text\":\"...\",\"agreed\":true/false,\"revision\":\"...\"}]}"
	instructionMap[KeySpecTaskGen] = "Generate the task list and acceptance criteria for this spec. Read the discovery answers — especially edge_cases, verification, and scope_boundary — and produce concrete tasks. Each task needs a title and at least one acceptance criterion. Express every acceptance criterion as a Given/When/Then triple: Given the precondition, When the action occurs, Then the observable outcome. The verification discovery answer feeds the Then of each criterion — derive each Then directly from how the user said this work will be verified. Include `linkedEdgeCases` for each task; every edge case from the discovery must be linked to at least one task. Submit as JSON: {\"tasks\":[{\"title\":\"...\",\"ac\":[\"...\"],\"criteria\":[{\"given\":\"...\",\"when\":\"...\",\"then\":\"...\"}],\"linkedEdgeCases\":[\"...\"]}]}."
	instructionMap[KeySelfReview] = "Spec draft is ready. Self-review before presenting to user. Run all 5 checks:\n1. Placeholder scan — no TBD, TODO, or placeholder text remains in any AC.\n2. Consistency — each AC is net and unambiguous; no contradicting criteria.\n3. Scope — no scope leak; each task stays within its declared boundary.\n4. Ambiguity — each AC is verifiable and measurable; no vague success criteria.\n5. Edge cases — every edge case from discovery is linked to at least one task via linkedEdgeCases; no edge case is orphaned. Each task must be atomic: a single deliverable.\nWhen ALL checks pass, submit the literal answer `approve` — do NOT submit a free-text review report; only `approve` is accepted. If any check fails, fix the draft first, then submit `approve`."
	instructionMap[KeyRefinePrompt] = "Show the current tasks and acceptance criteria to the user and ask whether they want changes. For per-task TDD and important flags, ask the user first, then include them. Submit changes with `tddmaster refine <slug> --answer='<json>'` using a payload like: {\"update\":{\"task-1\":{\"tddEnabled\":true,\"important\":false,\"edgeCases\":[\"...\"]}},\"add\":[{\"title\":\"New\",\"criteria\":[{\"given\":\"...\",\"when\":\"...\",\"then\":\"...\"}],\"edgeCases\":[\"...\"],\"dependsOn\":[\"task-1\"]}],\"remove\":[\"task-3\"]}. Acceptance criteria go in \"criteria\" as Given/When/Then objects (\"then\" required); there is no \"ac\" key. dependsOn is a list of task IDs the task depends on; tasks with no dependencies run in parallel. Self-dependencies, unknown task IDs, and dependency cycles are rejected. When removing a task, update the dependsOn of every task that depends on it in the same payload. When satisfied, run `tddmaster next <slug> --answer=\"approve\"`."
	for _, q := range Questions {
		instructionMap[KeyDiscoveryQuestion(q.ID)] = q.Text
	}
}

func Instruction(key InstructionKey) (string, bool) {
	val, ok := instructionMap[key]
	return val, ok
}

func MustInstruction(key InstructionKey) string {
	val, ok := instructionMap[key]
	if !ok {
		panic(fmt.Sprintf("promptregistry: no instruction registered for key %q", key))
	}
	return val
}
