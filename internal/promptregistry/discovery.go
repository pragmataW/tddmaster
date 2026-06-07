package promptregistry

type Question struct {
	ID       string
	Text     string
	Concerns []string
}

var Questions = []Question{
	{
		ID:   "status_quo",
		Text: "What does the user do today without this feature?",
	},
	{
		ID:   "ambition",
		Text: "Describe the 1-star and 10-star versions.",
	},
	{
		ID:   "reversibility",
		Text: "Does this change involve an irreversible decision?",
	},
	{
		ID:   "user_impact",
		Text: "Does this change affect existing users' behavior?",
	},
	{
		ID:   "verification",
		Text: "How do you verify this works correctly?",
	},
	{
		ID:   "scope_boundary",
		Text: "What should this feature NOT do?",
	},
	{
		ID:   "edge_cases",
		Text: "Which boundary conditions, error states, or exceptional inputs could cause this change to misbehave? List cases that need protective tests.",
	},
}

type ModeOption struct {
	ID          string
	Label       string
	Description string
}

var ModeOptions = []ModeOption{
	{"full", "Full discovery", "Standard 7 questions with all concern extras. Default for new features."},
	{"validate", "Validate my plan", "I already know what I want — challenge my assumptions, find gaps."},
	{"technical-depth", "Technical depth", "Focus on architecture, data flow, performance, integration points."},
	{"ship-fast", "Ship fast", "Minimum viable scope. What can we defer? What's the MVP?"},
	{"explore", "Explore scope", "Think bigger. 10x version? Adjacent opportunities? What are we missing?"},
}

var AskWithSuggestionsDirective = "Ask this question using the AskUserQuestion tool — never as plain prose or a \"write your answer\" prompt. Based on the spec description, the listen-first context, and the selected discovery mode, propose 2-4 concrete candidate answers as options (your best inferences) so the user can pick one or refine it. The user can always write their own answer via the free-form option."

var PremisePrompts = []string{
	"Is this the right problem to solve? Could a different framing yield a simpler solution?",
	"What happens if we do nothing? Is this a real pain point or a hypothetical one?",
	"What existing code already partially solves this? Can we build on it instead?",
}

var BuiltInExtras = []string{
	"What tests should be written? (unit, integration, e2e — be specific about what behavior to test)",
	"What documentation needs updating? (README, API docs, CHANGELOG, inline comments)",
}

func ModeRules(mode string) []string {
	switch mode {
	case "full":
		return []string{
			"Ask each discovery question as written. Push for specific, concrete answers.",
			"If the answer is vague, ask follow-up questions before accepting.",
		}
	case "validate":
		return []string{
			"The user has a plan. Your job is to challenge it, not explore it.",
			"For each question, identify assumptions and ask: 'What would prove this wrong?'",
			"If the description already answers a question, present your understanding and ask to confirm.",
			"When pre-filling answers from a rich description, plan, or prior discussion, DISTINGUISH between what the user EXPLICITLY STATED and what you INFERRED. Format each pre-filled item as: '[STATED] GPU skinning in all 3 renderers — you said this during technical discussion' or '[INFERRED] tangent space is 10-star scope — I assumed this based on complexity'. The user confirms stated items and corrects inferred items.",
			"Present pre-filled answers ONE ITEM AT A TIME for confirmation, not as a completed block. The user's job is to correct your inferences, not rubber-stamp your summary. If you pre-fill 5 items and 2 are wrong, the user must be able to catch them individually.",
		}
	case "technical-depth":
		return []string{
			"Focus on architecture, data flow, performance, and integration points.",
			"Before each question, scan the codebase for related implementations.",
			"Ask: 'How does this interact with [existing system]?' for each integration point.",
		}
	case "ship-fast":
		return []string{
			"Focus on minimum viable scope.",
			"For each question, also ask: 'What can we defer to a follow-up?'",
			"Push for the smallest version that delivers value.",
		}
	case "explore":
		return []string{
			"Think bigger. What's the 10x version?",
			"For each question, ask about adjacent opportunities.",
			"Suggest possibilities the user might not have considered.",
		}
	default:
		return nil
	}
}
