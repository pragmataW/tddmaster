package promptregistry

import (
	"strings"
	"testing"
)

func TestDiscoveryKeys_AreInstructionKeyType(t *testing.T) {
	keys := []InstructionKey{
		KeyListenFirst,
		KeyModeSelection,
		KeyPremiseChallenge,
		KeySpecTaskGen,
		KeySelfReview,
		KeyRefinePrompt,
	}
	for _, k := range keys {
		var _ InstructionKey = k
	}
}

func TestDiscoveryKeys_AreDistinct(t *testing.T) {
	keys := []InstructionKey{
		KeyListenFirst,
		KeyModeSelection,
		KeyPremiseChallenge,
		KeySpecTaskGen,
		KeySelfReview,
		KeyRefinePrompt,
	}
	seen := make(map[InstructionKey]bool)
	for _, k := range keys {
		if seen[k] {
			t.Fatalf("duplicate discovery key: %q", k)
		}
		seen[k] = true
	}
}

func TestKeyDiscoveryQuestion_ReturnsDistinctKeyPerID(t *testing.T) {
	ids := []string{"status_quo", "ambition", "reversibility", "user_impact", "verification", "scope_boundary", "edge_cases"}
	seen := make(map[InstructionKey]string)
	for _, id := range ids {
		k := KeyDiscoveryQuestion(id)
		if prev, exists := seen[k]; exists {
			t.Fatalf("KeyDiscoveryQuestion(%q) and KeyDiscoveryQuestion(%q) returned same key %q", id, prev, k)
		}
		seen[k] = id
	}
}

func TestKeyDiscoveryQuestion_IsDeterministic(t *testing.T) {
	k1 := KeyDiscoveryQuestion("status_quo")
	k2 := KeyDiscoveryQuestion("status_quo")
	if k1 != k2 {
		t.Fatalf("KeyDiscoveryQuestion(\"status_quo\"): not deterministic: got %q then %q", k1, k2)
	}
}

func TestKeyDiscoveryQuestion_DiffersFromStaticKeys(t *testing.T) {
	staticKeys := []InstructionKey{
		KeyListenFirst,
		KeyModeSelection,
		KeyPremiseChallenge,
		KeySpecTaskGen,
		KeySelfReview,
		KeyRefinePrompt,
	}
	for _, id := range []string{"status_quo", "ambition"} {
		dk := KeyDiscoveryQuestion(id)
		for _, sk := range staticKeys {
			if dk == sk {
				t.Fatalf("KeyDiscoveryQuestion(%q) == static key %q", id, sk)
			}
		}
	}
}

func TestQuestions_ExactCount(t *testing.T) {
	if len(Questions) != 7 {
		t.Fatalf("Questions: expected 7 entries, got %d", len(Questions))
	}
}

func TestQuestions_GoldenOrder(t *testing.T) {
	want := []struct {
		id   string
		text string
	}{
		{"status_quo", "What does the user do today without this feature?"},
		{"ambition", "Describe the 1-star and 10-star versions."},
		{"reversibility", "Does this change involve an irreversible decision?"},
		{"user_impact", "Does this change affect existing users' behavior?"},
		{"verification", "How do you verify this works correctly?"},
		{"scope_boundary", "What should this feature NOT do?"},
		{"edge_cases", "Which boundary conditions, error states, or exceptional inputs could cause this change to misbehave? List cases that need protective tests."},
	}
	for i, w := range want {
		t.Run(w.id, func(t *testing.T) {
			if Questions[i].ID != w.id {
				t.Fatalf("Questions[%d].ID: got %q, want %q", i, Questions[i].ID, w.id)
			}
			if Questions[i].Text != w.text {
				t.Fatalf("Questions[%d].Text: got %q, want %q", i, Questions[i].Text, w.text)
			}
		})
	}
}

func TestQuestions_ConcernsFieldExists(t *testing.T) {
	for i, q := range Questions {
		_ = q.Concerns
		if q.ID == "" {
			t.Fatalf("Questions[%d]: ID must not be empty", i)
		}
	}
}

func TestModeOptions_ExactCount(t *testing.T) {
	if len(ModeOptions) != 5 {
		t.Fatalf("ModeOptions: expected 5 entries, got %d", len(ModeOptions))
	}
}

func TestModeOptions_GoldenOrder(t *testing.T) {
	want := []struct {
		id          string
		label       string
		description string
	}{
		{"full", "Full discovery", "Standard 7 questions with all concern extras. Default for new features."},
		{"validate", "Validate my plan", "I already know what I want — challenge my assumptions, find gaps."},
		{"technical-depth", "Technical depth", "Focus on architecture, data flow, performance, integration points."},
		{"ship-fast", "Ship fast", "Minimum viable scope. What can we defer? What's the MVP?"},
		{"explore", "Explore scope", "Think bigger. 10x version? Adjacent opportunities? What are we missing?"},
	}
	for i, w := range want {
		t.Run(w.id, func(t *testing.T) {
			if ModeOptions[i].ID != w.id {
				t.Fatalf("ModeOptions[%d].ID: got %q, want %q", i, ModeOptions[i].ID, w.id)
			}
			if ModeOptions[i].Label != w.label {
				t.Fatalf("ModeOptions[%d].Label: got %q, want %q", i, ModeOptions[i].Label, w.label)
			}
			if ModeOptions[i].Description != w.description {
				t.Fatalf("ModeOptions[%d].Description: got %q, want %q", i, ModeOptions[i].Description, w.description)
			}
		})
	}
}

func TestPremisePrompts_ExactCount(t *testing.T) {
	if len(PremisePrompts) != 3 {
		t.Fatalf("PremisePrompts: expected 3 entries, got %d", len(PremisePrompts))
	}
}

func TestPremisePrompts_GoldenTexts(t *testing.T) {
	want := []string{
		"Is this the right problem to solve? Could a different framing yield a simpler solution?",
		"What happens if we do nothing? Is this a real pain point or a hypothetical one?",
		"What existing code already partially solves this? Can we build on it instead?",
	}
	for i, w := range want {
		if PremisePrompts[i] != w {
			t.Fatalf("PremisePrompts[%d]: got %q, want %q", i, PremisePrompts[i], w)
		}
	}
}

func TestBuiltInExtras_ExactCount(t *testing.T) {
	if len(BuiltInExtras) != 2 {
		t.Fatalf("BuiltInExtras: expected 2 entries, got %d", len(BuiltInExtras))
	}
}

func TestBuiltInExtras_GoldenTexts(t *testing.T) {
	want := []string{
		"What tests should be written? (unit, integration, e2e — be specific about what behavior to test)",
		"What documentation needs updating? (README, API docs, CHANGELOG, inline comments)",
	}
	for i, w := range want {
		if BuiltInExtras[i] != w {
			t.Fatalf("BuiltInExtras[%d]: got %q, want %q", i, BuiltInExtras[i], w)
		}
	}
}

func TestModeRules_Full(t *testing.T) {
	want := []string{
		"Ask each discovery question as written. Push for specific, concrete answers.",
		"If the answer is vague, ask follow-up questions before accepting.",
	}
	got := ModeRules("full")
	if len(got) != len(want) {
		t.Fatalf("ModeRules(\"full\"): got %d rules, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("ModeRules(\"full\")[%d]: got %q, want %q", i, got[i], w)
		}
	}
}

func TestModeRules_Validate(t *testing.T) {
	want := []string{
		"The user has a plan. Your job is to challenge it, not explore it.",
		"For each question, identify assumptions and ask: 'What would prove this wrong?'",
		"If the description already answers a question, present your understanding and ask to confirm.",
		"When pre-filling answers from a rich description, plan, or prior discussion, DISTINGUISH between what the user EXPLICITLY STATED and what you INFERRED. Format each pre-filled item as: '[STATED] GPU skinning in all 3 renderers — you said this during technical discussion' or '[INFERRED] tangent space is 10-star scope — I assumed this based on complexity'. The user confirms stated items and corrects inferred items.",
		"Present pre-filled answers ONE ITEM AT A TIME for confirmation, not as a completed block. The user's job is to correct your inferences, not rubber-stamp your summary. If you pre-fill 5 items and 2 are wrong, the user must be able to catch them individually.",
	}
	got := ModeRules("validate")
	if len(got) != len(want) {
		t.Fatalf("ModeRules(\"validate\"): got %d rules, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("ModeRules(\"validate\")[%d]: got %q, want %q", i, got[i], w)
		}
	}
}

func TestModeRules_TechnicalDepth(t *testing.T) {
	want := []string{
		"Focus on architecture, data flow, performance, and integration points.",
		"Before each question, scan the codebase for related implementations.",
		"Ask: 'How does this interact with [existing system]?' for each integration point.",
	}
	got := ModeRules("technical-depth")
	if len(got) != len(want) {
		t.Fatalf("ModeRules(\"technical-depth\"): got %d rules, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("ModeRules(\"technical-depth\")[%d]: got %q, want %q", i, got[i], w)
		}
	}
}

func TestModeRules_ShipFast(t *testing.T) {
	want := []string{
		"Focus on minimum viable scope.",
		"For each question, also ask: 'What can we defer to a follow-up?'",
		"Push for the smallest version that delivers value.",
	}
	got := ModeRules("ship-fast")
	if len(got) != len(want) {
		t.Fatalf("ModeRules(\"ship-fast\"): got %d rules, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("ModeRules(\"ship-fast\")[%d]: got %q, want %q", i, got[i], w)
		}
	}
}

func TestModeRules_Explore(t *testing.T) {
	want := []string{
		"Think bigger. What's the 10x version?",
		"For each question, ask about adjacent opportunities.",
		"Suggest possibilities the user might not have considered.",
	}
	got := ModeRules("explore")
	if len(got) != len(want) {
		t.Fatalf("ModeRules(\"explore\"): got %d rules, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("ModeRules(\"explore\")[%d]: got %q, want %q", i, got[i], w)
		}
	}
}

func TestModeRules_UnknownMode_ReturnsEmpty(t *testing.T) {
	got := ModeRules("unknown-or-empty")
	if len(got) != 0 {
		t.Fatalf("ModeRules(\"unknown-or-empty\"): expected empty/nil, got %v", got)
	}
}

func TestModeRules_EmptyString_ReturnsEmpty(t *testing.T) {
	got := ModeRules("")
	if len(got) != 0 {
		t.Fatalf("ModeRules(\"\"): expected empty/nil, got %v", got)
	}
}

func TestInstruction_KeyListenFirst_GoldenText(t *testing.T) {
	want := "The user just created this spec. Before starting discovery, ask them to share whatever context they have — requirements, notes, tasks, or just a brief description. Say: 'Tell me about this — share as much context as you have.' The shared context is the primary reference for this spec and will be passed verbatim to the test-writer, executor, and verifier sub-agents during task execution. Listen first, then proceed."
	got, ok := Instruction(KeyListenFirst)
	if !ok {
		t.Fatalf("Instruction(KeyListenFirst): expected ok=true, got false")
	}
	if got != want {
		t.Fatalf("Instruction(KeyListenFirst): got %q, want %q", got, want)
	}
}

func TestInstruction_KeyListenFirst_InformsUserContextIsMainReference(t *testing.T) {
	got, ok := Instruction(KeyListenFirst)
	if !ok {
		t.Fatalf("Instruction(KeyListenFirst): expected ok=true, got false")
	}
	lower := strings.ToLower(got)
	hasReference := strings.Contains(lower, "referans") || strings.Contains(lower, "reference") ||
		strings.Contains(lower, "ana") || strings.Contains(lower, "main") || strings.Contains(lower, "primary")
	if !hasReference {
		t.Fatalf("Instruction(KeyListenFirst): expected to inform user that shared context is the main reference, got %q", got)
	}
}

func TestInstruction_KeyListenFirst_MentionsSubAgents(t *testing.T) {
	got, ok := Instruction(KeyListenFirst)
	if !ok {
		t.Fatalf("Instruction(KeyListenFirst): expected ok=true, got false")
	}
	lower := strings.ToLower(got)
	hasTestWriter := strings.Contains(lower, "test-writer") || strings.Contains(lower, "testwriter")
	hasExecutor := strings.Contains(lower, "executor")
	hasVerifier := strings.Contains(lower, "verifier")
	if !hasTestWriter || !hasExecutor || !hasVerifier {
		t.Fatalf("Instruction(KeyListenFirst): expected to mention test-writer, executor, and verifier sub-agents; got %q", got)
	}
}

func TestInstruction_KeyListenFirst_MentionsExecutionPropagation(t *testing.T) {
	got, ok := Instruction(KeyListenFirst)
	if !ok {
		t.Fatalf("Instruction(KeyListenFirst): expected ok=true, got false")
	}
	lower := strings.ToLower(got)
	hasPropagation := strings.Contains(lower, "execution") || strings.Contains(lower, "task")
	if !hasPropagation {
		t.Fatalf("Instruction(KeyListenFirst): expected to mention that context is passed during execution, got %q", got)
	}
}

func TestInstruction_KeyModeSelection_GoldenText(t *testing.T) {
	want := "Before starting discovery, select the discovery mode via AskUserQuestion. Use the options provided in interactiveOptions — do NOT present them as prose or a numbered list."
	got, ok := Instruction(KeyModeSelection)
	if !ok {
		t.Fatalf("Instruction(KeyModeSelection): expected ok=true, got false")
	}
	if got != want {
		t.Fatalf("Instruction(KeyModeSelection): got %q, want %q", got, want)
	}
}

func TestInstruction_KeyPremiseChallenge_GoldenText(t *testing.T) {
	want := "Read the spec description. Identify 2-4 premises the spec assumes. Present each premise and ask the user to agree or disagree. Submit as JSON: {\"premises\":[{\"text\":\"...\",\"agreed\":true/false,\"revision\":\"...\"}]}"
	got, ok := Instruction(KeyPremiseChallenge)
	if !ok {
		t.Fatalf("Instruction(KeyPremiseChallenge): expected ok=true, got false")
	}
	if got != want {
		t.Fatalf("Instruction(KeyPremiseChallenge): got %q, want %q", got, want)
	}
}

func TestInstruction_KeyDiscoveryQuestion_StatusQuo_GoldenText(t *testing.T) {
	want := "What does the user do today without this feature?"
	got, ok := Instruction(KeyDiscoveryQuestion("status_quo"))
	if !ok {
		t.Fatalf("Instruction(KeyDiscoveryQuestion(\"status_quo\")): expected ok=true, got false")
	}
	if got != want {
		t.Fatalf("Instruction(KeyDiscoveryQuestion(\"status_quo\")): got %q, want %q", got, want)
	}
}

func TestInstruction_AllDiscoveryQuestionKeys_Resolvable(t *testing.T) {
	ids := []string{"status_quo", "ambition", "reversibility", "user_impact", "verification", "scope_boundary", "edge_cases"}
	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			k := KeyDiscoveryQuestion(id)
			got, ok := Instruction(k)
			if !ok {
				t.Fatalf("Instruction(KeyDiscoveryQuestion(%q)): expected ok=true, got false", id)
			}
			if got == "" {
				t.Fatalf("Instruction(KeyDiscoveryQuestion(%q)): expected non-empty string, got empty", id)
			}
		})
	}
}

func TestInstruction_AllStaticDiscoveryKeys_Resolvable(t *testing.T) {
	cases := []struct {
		name string
		key  InstructionKey
	}{
		{"KeyListenFirst", KeyListenFirst},
		{"KeyModeSelection", KeyModeSelection},
		{"KeyPremiseChallenge", KeyPremiseChallenge},
		{"KeySpecTaskGen", KeySpecTaskGen},
		{"KeySelfReview", KeySelfReview},
		{"KeyRefinePrompt", KeyRefinePrompt},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := Instruction(tc.key)
			if !ok {
				t.Fatalf("Instruction(%s): expected ok=true, got false", tc.name)
			}
			if got == "" {
				t.Fatalf("Instruction(%s): expected non-empty string, got empty", tc.name)
			}
		})
	}
}

func TestInstruction_KeySpecTaskGen_ContainsRequiredMarkers(t *testing.T) {
	got, ok := Instruction(KeySpecTaskGen)
	if !ok {
		t.Fatalf("Instruction(KeySpecTaskGen): expected ok=true, got false")
	}
	if got == "" {
		t.Fatalf("Instruction(KeySpecTaskGen): expected non-empty string, got empty")
	}
	if !strings.Contains(got, `"tasks"`) {
		t.Fatalf("Instruction(KeySpecTaskGen): expected to contain %q", `"tasks"`)
	}
	if !strings.Contains(got, `"ac"`) {
		t.Fatalf("Instruction(KeySpecTaskGen): expected to contain %q", `"ac"`)
	}
	if !strings.Contains(strings.ToLower(got), "edge") {
		t.Fatalf("Instruction(KeySpecTaskGen): expected to contain \"edge\" (case-insensitive)")
	}
}

func TestInstruction_KeySpecTaskGen_ContainsLinkedEdgeCases(t *testing.T) {
	got, ok := Instruction(KeySpecTaskGen)
	if !ok {
		t.Fatalf("Instruction(KeySpecTaskGen): expected ok=true, got false")
	}
	if !strings.Contains(got, "linkedEdgeCases") {
		t.Fatalf("Instruction(KeySpecTaskGen): expected to contain \"linkedEdgeCases\" field name, got %q", got)
	}
}

func TestInstruction_KeySpecTaskGen_RequiresEachECLinkedToAtLeastOneTask(t *testing.T) {
	got, ok := Instruction(KeySpecTaskGen)
	if !ok {
		t.Fatalf("Instruction(KeySpecTaskGen): expected ok=true, got false")
	}
	lower := strings.ToLower(got)
	hasEvery := strings.Contains(lower, "every") || strings.Contains(lower, "each") || strings.Contains(lower, "at least one")
	hasLink := strings.Contains(lower, "link") || strings.Contains(lower, "bind") || strings.Contains(lower, "connect")
	if !hasEvery || !hasLink {
		t.Fatalf("Instruction(KeySpecTaskGen): expected to require each EC to be linked to at least one task, got %q", got)
	}
}

func TestInstruction_KeySelfReview_NonEmpty(t *testing.T) {
	got, ok := Instruction(KeySelfReview)
	if !ok {
		t.Fatalf("Instruction(KeySelfReview): expected ok=true, got false")
	}
	if got == "" {
		t.Fatalf("Instruction(KeySelfReview): expected non-empty string, got empty")
	}
}

func TestInstruction_KeySelfReview_ContainsFiveChecks(t *testing.T) {
	got, ok := Instruction(KeySelfReview)
	if !ok {
		t.Fatalf("Instruction(KeySelfReview): expected ok=true, got false")
	}
	checks := []string{
		"Placeholder scan",
		"Consistency",
		"Scope",
		"Ambiguity",
		"Edge cases",
	}
	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Fatalf("Instruction(KeySelfReview): expected to contain %q (one of 5 review checks), got %q", check, got)
		}
	}
}

func TestInstruction_KeySelfReview_ContainsACNetCheck(t *testing.T) {
	got, ok := Instruction(KeySelfReview)
	if !ok {
		t.Fatalf("Instruction(KeySelfReview): expected ok=true, got false")
	}
	if !strings.Contains(strings.ToLower(got), "ac") && !strings.Contains(strings.ToLower(got), "acceptance") {
		t.Fatalf("Instruction(KeySelfReview): expected to contain AC or acceptance criteria check, got %q", got)
	}
}

func TestInstruction_KeySelfReview_ContainsECLinkedCheck(t *testing.T) {
	got, ok := Instruction(KeySelfReview)
	if !ok {
		t.Fatalf("Instruction(KeySelfReview): expected ok=true, got false")
	}
	if !strings.Contains(strings.ToLower(got), "edge") {
		t.Fatalf("Instruction(KeySelfReview): expected to contain edge case linkage check, got %q", got)
	}
}

func TestInstruction_KeySelfReview_ContainsVerificationMeasurableCheck(t *testing.T) {
	got, ok := Instruction(KeySelfReview)
	if !ok {
		t.Fatalf("Instruction(KeySelfReview): expected ok=true, got false")
	}
	if !strings.Contains(strings.ToLower(got), "verif") && !strings.Contains(strings.ToLower(got), "measur") {
		t.Fatalf("Instruction(KeySelfReview): expected to contain verification or measurable check, got %q", got)
	}
}

func TestInstruction_KeySelfReview_ContainsScopeLeakCheck(t *testing.T) {
	got, ok := Instruction(KeySelfReview)
	if !ok {
		t.Fatalf("Instruction(KeySelfReview): expected ok=true, got false")
	}
	if !strings.Contains(strings.ToLower(got), "scope") {
		t.Fatalf("Instruction(KeySelfReview): expected to contain scope check, got %q", got)
	}
}

func TestInstruction_KeySelfReview_ContainsAtomicCheck(t *testing.T) {
	got, ok := Instruction(KeySelfReview)
	if !ok {
		t.Fatalf("Instruction(KeySelfReview): expected ok=true, got false")
	}
	if !strings.Contains(strings.ToLower(got), "atom") && !strings.Contains(strings.ToLower(got), "single") {
		t.Fatalf("Instruction(KeySelfReview): expected to contain task atomicity check, got %q", got)
	}
}

func TestInstruction_KeyRefinePrompt_ContainsRequiredMarkers(t *testing.T) {
	got, ok := Instruction(KeyRefinePrompt)
	if !ok {
		t.Fatalf("Instruction(KeyRefinePrompt): expected ok=true, got false")
	}
	if got == "" {
		t.Fatalf("Instruction(KeyRefinePrompt): expected non-empty string, got empty")
	}
	if !strings.Contains(strings.ToLower(got), "refine") {
		t.Fatalf("Instruction(KeyRefinePrompt): expected to contain \"refine\" (case-insensitive)")
	}
	if !strings.Contains(strings.ToLower(got), "approve") {
		t.Fatalf("Instruction(KeyRefinePrompt): expected to contain \"approve\" (case-insensitive)")
	}
}

func TestInstruction_UnknownKey_StillReturnsFalse(t *testing.T) {
	val, ok := Instruction(InstructionKey("__discovery_nonexistent__"))
	if ok {
		t.Fatalf("Instruction unknown key: expected ok=false, got true")
	}
	if val != "" {
		t.Fatalf("Instruction unknown key: expected empty string, got %q", val)
	}
}

func TestQuestion_StructFields(t *testing.T) {
	q := Question{ID: "x", Text: "y", Concerns: []string{"z"}}
	if q.ID != "x" {
		t.Fatalf("Question.ID: got %q, want %q", q.ID, "x")
	}
	if q.Text != "y" {
		t.Fatalf("Question.Text: got %q, want %q", q.Text, "y")
	}
	if len(q.Concerns) != 1 || q.Concerns[0] != "z" {
		t.Fatalf("Question.Concerns: got %v, want [z]", q.Concerns)
	}
}

func TestModeOption_StructFields(t *testing.T) {
	m := ModeOption{ID: "a", Label: "b", Description: "c"}
	if m.ID != "a" {
		t.Fatalf("ModeOption.ID: got %q, want %q", m.ID, "a")
	}
	if m.Label != "b" {
		t.Fatalf("ModeOption.Label: got %q, want %q", m.Label, "b")
	}
	if m.Description != "c" {
		t.Fatalf("ModeOption.Description: got %q, want %q", m.Description, "c")
	}
}
