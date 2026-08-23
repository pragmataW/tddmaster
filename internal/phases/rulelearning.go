package phases

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/errs"
	"github.com/pragmataW/tddmaster/internal/promptregistry"
)

type ruleProposal struct {
	Rules []ruleEntry `json:"rules"`
}

type ruleEntry struct {
	Scope     string `json:"scope"`
	Name      string `json:"name"`
	Content   string `json:"content"`
	Rationale string `json:"rationale"`
}

type approvalAnswer struct {
	Accepted     *bool  `json:"accepted,omitempty"`
	PlanFeedback string `json:"planFeedback,omitempty"`
	Feedback     string `json:"feedback,omitempty"`
}

type ruleLearningDriver struct{}

func RuleLearningDriver() engine.Driver {
	return &ruleLearningDriver{}
}

type learning struct {
	suggestions []string
	failedACs   []string
}

func canonicalRuleScope(scope string) string {
	if scope == "all" {
		return "global"
	}
	return scope
}

func gatherLearnings(c *engine.Context) learning {
	var l learning
	for _, task := range c.Progress().Tasks {
		for _, rn := range task.RefactorNotes {
			l.suggestions = append(l.suggestions, rn.Suggestion)
		}
		l.failedACs = append(l.failedACs, task.FailedACReasons...)
	}
	return l
}

// renderProposal turns the stored proposal into something a human can read.
// Dumping the raw JSON blob made the reviewer parse it before they could judge it.
func renderProposal(proposalJSON string) string {
	var proposal ruleProposal
	if err := json.Unmarshal([]byte(proposalJSON), &proposal); err != nil || len(proposal.Rules) == 0 {
		return proposalJSON
	}
	var parts []string
	for i, r := range proposal.Rules {
		parts = append(parts, fmt.Sprintf("\nRule %d — %s (scope: %s)", i+1, r.Name, canonicalRuleScope(r.Scope)))
		if strings.TrimSpace(r.Rationale) != "" {
			parts = append(parts, "  why: "+r.Rationale)
		}
		for _, line := range strings.Split(strings.TrimRight(r.Content, "\n"), "\n") {
			parts = append(parts, "  "+line)
		}
	}
	return strings.Join(parts, "\n")
}

func (d *ruleLearningDriver) Next(c *engine.Context, ph *engine.PhaseDef) (engine.Action, bool) {
	if c.HasAnswer("rule_applied") {
		return engine.Action{}, true
	}

	lr := gatherLearnings(c)
	if len(lr.suggestions) == 0 && len(lr.failedACs) == 0 {
		return engine.Action{}, true
	}

	if c.HasAnswer("rule_approved") {
		proposalJSON := c.AnswerValue("rule_proposal")
		var proposal ruleProposal
		if err := json.Unmarshal([]byte(proposalJSON), &proposal); err != nil {
			return engine.Action{Action: engine.ActionError, Instruction: fmt.Sprintf("stored rule proposal is corrupt: %v", err)}, false
		}
		if len(proposal.Rules) == 0 {
			return engine.Action{Action: engine.ActionError, Instruction: "stored rule proposal contains no rules"}, false
		}

		var parts []string
		parts = append(parts, "Apply each approved rule below. For each rule, write its content VERBATIM (do not invent, paraphrase, or summarize) to a temp file, then run the shown command with --content-file pointing at that file. Never overwrite an existing rule.")
		for i, r := range proposal.Rules {
			scope := canonicalRuleScope(r.Scope)
			parts = append(parts, fmt.Sprintf("\nRule %d:", i+1))
			parts = append(parts, "scope: "+scope)
			parts = append(parts, "name: "+r.Name)
			parts = append(parts, "rationale: "+r.Rationale)
			parts = append(parts, "content:")
			parts = append(parts, r.Content)
			parts = append(parts, fmt.Sprintf("command: tddmaster rule add --scope %s --name %s --content-file <path-to-temp-file>", scope, r.Name))
		}
		parts = append(parts, promptregistry.RuleApplyReportDirective)
		return engine.Action{
			Action:        engine.ActionInstruct,
			DelegateAgent: string(promptregistry.AgentRuleSynthesizer),
			Instruction:   strings.Join(parts, "\n"),
			ExpectedInput: engine.ExpectedInput{
				Format:    engine.FormatFlag,
				SubmitCmd: fmt.Sprintf("tddmaster next %s --answer='applied'", c.Slug()),
				Example:   "applied",
			},
		}, false
	}

	if c.HasAnswer("rule_proposal") {
		return engine.Action{
			Action:      engine.ActionAsk,
			Instruction: promptregistry.RuleReviewHeader + "\n" + renderProposal(c.AnswerValue("rule_proposal")),
			ExpectedInput: engine.ExpectedInput{
				Format:  engine.FormatJSON,
				Example: promptregistry.ExampleRuleApproval,
			},
			InteractiveOptions: []engine.InteractiveOption{
				{Label: "accept", Description: "Accept the proposed rules and apply them."},
				{Label: "revise", Description: "Request revisions to the proposed rules."},
				{Label: "reject", Description: "Reject the proposed rules without applying."},
			},
			CommandMap: map[string]string{
				"accept": fmt.Sprintf("tddmaster next %s --answer='{\"accepted\":true}'", c.Slug()),
				"revise": fmt.Sprintf("tddmaster next %s --answer='{\"feedback\":\"<feedback>\"}'", c.Slug()),
				"reject": fmt.Sprintf("tddmaster next %s --answer='{\"accepted\":false}'", c.Slug()),
			},
		}, false
	}

	var parts []string
	parts = append(parts, "Synthesize rules from the following learnings gathered during execution.")
	if len(lr.suggestions) > 0 {
		parts = append(parts, "Refactor note suggestions:")
		for _, s := range lr.suggestions {
			parts = append(parts, "- "+s)
		}
	}
	if len(lr.failedACs) > 0 {
		parts = append(parts, "Failed AC reasons:")
		for _, r := range lr.failedACs {
			parts = append(parts, "- "+r)
		}
	}
	if c.HasAnswer("rule_feedback") {
		parts = append(parts, "priorFeedback: "+c.AnswerValue("rule_feedback"))
		if c.HasAnswer("rule_attempt") {
			count, _ := strconv.Atoi(c.AnswerValue("rule_attempt"))
			parts = append(parts, fmt.Sprintf("attemptCount: %d", count))
		}
	}
	parts = append(parts, promptregistry.RuleProposalDirective)
	return engine.Action{
		Action:        engine.ActionInstruct,
		DelegateAgent: string(promptregistry.AgentRuleSynthesizer),
		Instruction:   strings.Join(parts, "\n"),
		ExpectedInput: engine.ExpectedInput{
			Format:  engine.FormatJSON,
			Example: promptregistry.ExampleRuleProposal,
		},
	}, false
}

func (d *ruleLearningDriver) Submit(c *engine.Context, ph *engine.PhaseDef, answer []byte) (engine.Action, bool, error) {
	if c.HasAnswer("rule_applied") {
		return engine.Action{}, true, nil
	}

	if c.HasAnswer("rule_approved") {
		trimmed := strings.TrimSpace(string(answer))
		if trimmed != "applied" {
			if trimmed == "" {
				trimmed = "no application result was reported"
			}
			return engine.Action{
				Action: engine.ActionError,
				Instruction: fmt.Sprintf(
					"Rule application failed and rule learning remains pending: %s\nResolve the failure, apply every remaining approved rule, then submit the literal `applied`.",
					trimmed,
				),
			}, false, nil
		}
		if err := c.SetAnswer("rule_applied", "true"); err != nil {
			return engine.Action{}, false, err
		}
		return engine.Action{}, true, nil
	}

	if c.HasAnswer("rule_proposal") {
		trimmed := strings.TrimSpace(string(answer))
		if trimmed == "accept" {
			if err := c.SetAnswer("rule_approved", "true"); err != nil {
				return engine.Action{}, false, err
			}
			return engine.Action{}, false, nil
		}
		if trimmed == "reject" {
			if err := c.SetAnswer("rule_applied", "true"); err != nil {
				return engine.Action{}, false, err
			}
			return engine.Action{}, false, nil
		}

		var aa approvalAnswer
		if err := json.Unmarshal(answer, &aa); err == nil {
			if aa.Accepted != nil && *aa.Accepted {
				if err := c.SetAnswer("rule_approved", "true"); err != nil {
					return engine.Action{}, false, err
				}
				return engine.Action{}, false, nil
			}
			if aa.Accepted != nil && !*aa.Accepted {
				if err := c.SetAnswer("rule_applied", "true"); err != nil {
					return engine.Action{}, false, err
				}
				return engine.Action{}, false, nil
			}
			feedback := aa.PlanFeedback
			if feedback == "" {
				feedback = aa.Feedback
			}
			if feedback != "" {
				if err := c.SetAnswer("rule_proposal", ""); err != nil {
					return engine.Action{}, false, err
				}
				if err := c.SetAnswer("rule_feedback", feedback); err != nil {
					return engine.Action{}, false, err
				}
				count := 0
				if c.HasAnswer("rule_attempt") {
					count, _ = strconv.Atoi(c.AnswerValue("rule_attempt"))
				}
				count++
				if err := c.SetAnswer("rule_attempt", strconv.Itoa(count)); err != nil {
					return engine.Action{}, false, err
				}
				return engine.Action{}, false, nil
			}
		}

		return engine.Action{}, false, errs.Newf(errs.KeyUnrecognizedApproval, trimmed)
	}

	var proposal ruleProposal
	if err := json.Unmarshal(answer, &proposal); err != nil {
		return engine.Action{}, false, errs.Wrap(errs.KeyInvalidProposalJSON, err)
	}
	if len(proposal.Rules) == 0 {
		return engine.Action{}, false, errs.New(errs.KeyProposalNeedsRule)
	}
	for i := range proposal.Rules {
		proposal.Rules[i].Scope = canonicalRuleScope(proposal.Rules[i].Scope)
	}
	normalized, err := json.Marshal(proposal)
	if err != nil {
		return engine.Action{}, false, errs.Wrap(errs.KeyInvalidProposalJSON, err)
	}
	if err := c.SetAnswer("rule_proposal", string(normalized)); err != nil {
		return engine.Action{}, false, err
	}
	return engine.Action{}, false, nil
}
