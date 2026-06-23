package phases

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/pragmataW/tddmaster/internal/engine"
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
			parts = append(parts, fmt.Sprintf("\nRule %d:", i+1))
			parts = append(parts, "scope: "+r.Scope)
			parts = append(parts, "name: "+r.Name)
			parts = append(parts, "rationale: "+r.Rationale)
			parts = append(parts, "content:")
			parts = append(parts, r.Content)
			parts = append(parts, fmt.Sprintf("command: tddmaster rule add --scope %s --name %s --content-file <path-to-temp-file>", r.Scope, r.Name))
		}
		return engine.Action{
			Action:        engine.ActionInstruct,
			DelegateAgent: string(promptregistry.AgentRuleSynthesizer),
			Instruction:   strings.Join(parts, "\n"),
		}, false
	}

	if c.HasAnswer("rule_proposal") {
		proposalJSON := c.AnswerValue("rule_proposal")
		return engine.Action{
			Action:      engine.ActionAsk,
			Instruction: "Review the proposed rules:\n" + proposalJSON,
			InteractiveOptions: []engine.InteractiveOption{
				{Label: "accept", Description: "Accept the proposed rules and apply them."},
				{Label: "revise", Description: "Request revisions to the proposed rules."},
				{Label: "reject", Description: "Reject the proposed rules without applying."},
			},
			CommandMap: map[string]string{
				"accept": fmt.Sprintf("tddmaster next %s --answer='{\"accepted\":true}'", c.Slug()),
				"revise": fmt.Sprintf("tddmaster next %s --answer='{\"planFeedback\":\"<feedback>\"}'", c.Slug()),
				"reject": fmt.Sprintf("tddmaster next %s --answer='{\"accepted\":false}'", c.Slug()),
			},
		}, false
	}

	var parts []string
	parts = append(parts, "Synthesize rules from the following learnings gathered during execution.")
	parts = append(parts, "Refactor note suggestions:")
	for _, s := range lr.suggestions {
		parts = append(parts, "- "+s)
	}
	parts = append(parts, "Failed AC reasons:")
	for _, r := range lr.failedACs {
		parts = append(parts, "- "+r)
	}
	if c.HasAnswer("rule_feedback") {
		parts = append(parts, "priorFeedback: "+c.AnswerValue("rule_feedback"))
		if c.HasAnswer("rule_attempt") {
			count, _ := strconv.Atoi(c.AnswerValue("rule_attempt"))
			parts = append(parts, fmt.Sprintf("attemptCount: %d", count))
		}
	}
	return engine.Action{
		Action:        engine.ActionInstruct,
		DelegateAgent: string(promptregistry.AgentRuleSynthesizer),
		Instruction:   strings.Join(parts, "\n"),
	}, false
}

func (d *ruleLearningDriver) Submit(c *engine.Context, ph *engine.PhaseDef, answer []byte) (engine.Action, bool, error) {
	if c.HasAnswer("rule_applied") {
		return engine.Action{}, true, nil
	}

	if c.HasAnswer("rule_approved") {
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

		return engine.Action{}, false, fmt.Errorf("unrecognized approval answer: %q", trimmed)
	}

	var proposal ruleProposal
	if err := json.Unmarshal(answer, &proposal); err != nil {
		return engine.Action{}, false, fmt.Errorf("invalid proposal JSON: %w", err)
	}
	if len(proposal.Rules) == 0 {
		return engine.Action{}, false, fmt.Errorf("proposal must contain at least one rule")
	}
	if err := c.SetAnswer("rule_proposal", string(answer)); err != nil {
		return engine.Action{}, false, err
	}
	return engine.Action{}, false, nil
}
