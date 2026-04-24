package discovery

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/state"
)

// versionTermPattern extracts runtime/library version mentions (e.g. "Node.js 20")
// that should trigger a pre-discovery web-search instruction.
var versionTermPattern = regexp.MustCompile(`(?i)\b(Node\.?js|Deno|Bun|Go|Rust|Python|Ruby|Java|Kotlin|Swift|PHP|React|Vue|Angular|Svelte|Next\.?js|Nuxt|Remix|Astro|SolidJS|Qwik|TypeScript|Webpack|Vite|esbuild|Rollup|Terraform|Docker|Kubernetes|PostgreSQL|MySQL|Redis|MongoDB|SQLite|Prisma|Drizzle|gRPC|GraphQL|tRPC)\s+v?(\d+(?:\.\d+)?(?:\.\d+)?\+?)\b`)

func extractVersionTerms(description string) []string {
	if description == "" {
		return nil
	}
	matches := versionTermPattern.FindAllStringSubmatch(description, -1)
	var terms []string
	for _, m := range matches {
		if len(m) >= 3 {
			terms = append(terms, m[1]+" "+m[2])
		}
	}
	return terms
}

func buildPreDiscoveryResearch(description string) *model.PreDiscoveryResearch {
	terms := extractVersionTerms(description)
	if len(terms) == 0 {
		return nil
	}
	return &model.PreDiscoveryResearch{
		Required:       true,
		Instruction:    model.PreDiscoveryResearchInstruction,
		ExtractedTerms: terms,
	}
}

func generateFollowUpHints(answer string) []string {
	var hints []string
	lower := strings.ToLower(answer)

	techPatterns := []string{
		"websocket", "graphql", "grpc", "redis", "postgres", "mongodb",
		"kafka", "rabbitmq", "docker", "kubernetes", "lambda", "s3",
	}
	for _, tech := range techPatterns {
		if strings.Contains(lower, tech) {
			hints = append(hints, fmt.Sprintf("Answer mentions %s — consider: error handling, versioning, fallback strategy", tech))
		}
	}

	if strings.Contains(lower, "should work") || strings.Contains(lower, "standard approach") ||
		strings.Contains(lower, "probably") || strings.Contains(lower, "i think") ||
		strings.Contains(lower, "not sure") {
		hints = append(hints, "Answer is vague — ask for specifics")
	}

	if strings.Contains(lower, "and also") || strings.Contains(lower, "we might") ||
		strings.Contains(lower, "could also") || strings.Contains(lower, "maybe we should") {
		hints = append(hints, "Scope expansion signal — clarify if in scope or deferred")
	}

	if strings.Contains(lower, "tricky") || strings.Contains(lower, "complicated") ||
		strings.Contains(lower, "risky") || strings.Contains(lower, "not sure about") {
		hints = append(hints, "Risk signal — dig deeper into what makes it risky")
	}

	if strings.Contains(lower, "depends on") || strings.Contains(lower, "after") ||
		strings.Contains(lower, "blocked by") || strings.Contains(lower, "waiting for") {
		hints = append(hints, "Dependency detected — clarify what happens if dependency isn't ready")
	}

	if strings.Contains(lower, "real-time") || strings.Contains(lower, "scalab") ||
		strings.Contains(lower, "performance") || strings.Contains(lower, "latency") ||
		strings.Contains(lower, "concurrent") {
		hints = append(hints, "Performance/scale mention — ask about limits, degradation, monitoring")
	}

	return hints
}

func getModeRules(mode state.DiscoveryMode) []string {
	switch mode {
	case state.DiscoveryModeFull:
		return model.ModeRulesFull
	case state.DiscoveryModeValidate:
		return model.ModeRulesValidate
	case state.DiscoveryModeTechnicalDepth:
		return model.ModeRulesTechnicalDepth
	case state.DiscoveryModeShipFast:
		return model.ModeRulesShipFast
	case state.DiscoveryModeExplore:
		return model.ModeRulesExplore
	}
	return nil
}

func buildDiscoveryQuestion(question model.QuestionWithExtras, prefills []state.DiscoveryPrefillQuestion) model.DiscoveryQuestion {
	extras := make([]string, len(question.Extras))
	for i, e := range question.Extras {
		extras[i] = e.Text
	}
	out := model.DiscoveryQuestion{
		ID:       question.ID,
		Text:     question.Text,
		Concerns: question.Concerns,
		Extras:   extras,
	}
	if items := state.GetPrefillsForQuestion(prefills, question.ID); len(items) > 0 {
		out.Prefills = items
	}
	return out
}

func selectCurrentDiscoveryQuestion(
	questions []model.QuestionWithExtras,
	answers []state.DiscoveryAnswer,
	currentIdx int,
) (*model.QuestionWithExtras, int) {
	answered := make(map[string]bool, len(answers))
	for _, a := range answers {
		answered[a.QuestionID] = true
	}

	if currentIdx >= 0 && currentIdx < len(questions) {
		candidate := questions[currentIdx]
		if !answered[candidate.ID] {
			return &candidate, currentIdx
		}
	}

	for i := range questions {
		if !answered[questions[i].ID] {
			return &questions[i], i
		}
	}
	return nil, len(questions)
}
