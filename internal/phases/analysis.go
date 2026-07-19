package phases

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/spec"
)

type analysisDriver struct{}

func AnalysisDriver() engine.Driver {
	return &analysisDriver{}
}

const analysisMaxAttempts = 5

const (
	answerKeyComplete = "analysis_complete"
	answerKeyAudited  = "analysis_audited"
	answerKeyFindings = "analysis_findings"
	answerKeyAttempts = "analysis_attempts"
)

const (
	optReturnToRefinement = "return-to-refinement"
	optAcceptAnyway       = "accept-anyway"
	optEdit               = "edit"
)

const auditorAgent = "tddmaster-auditor"

type analysisGateAnswer struct {
	Action  string          `json:"action"`
	Payload json.RawMessage `json:"payload"`
}

func findingKey(f spec.Finding) string {
	return strings.Join([]string{f.Category, f.TaskID, f.AcID, f.Detail}, "\x00")
}

func mergeFindings(auditor []spec.Finding, lint []spec.Finding) []spec.Finding {
	merged := make([]spec.Finding, 0, len(auditor)+len(lint))
	seen := make(map[string]bool, len(auditor)+len(lint))
	add := func(findings []spec.Finding) {
		for _, f := range findings {
			k := findingKey(f)
			if seen[k] {
				continue
			}
			seen[k] = true
			merged = append(merged, f)
		}
	}
	add(lint)
	add(auditor)
	return merged
}

// anyActionable reports whether the analysis contains at least one finding that
// requires user confirmation. Policy: every finding except pure-info severity is
// actionable, so a single non-info finding pauses the phase for the user.
func anyActionable(findings []spec.Finding) bool {
	for _, f := range findings {
		if !f.IsInfo() {
			return true
		}
	}
	return false
}

func buildAuditorInstruction(c *engine.Context, tasks []spec.Task, lint []spec.Finding) string {
	var parts []string
	parts = append(parts, "Perform a cross-artifact analysis of the task list below. Return JSON {\"verdict\":\"clean|issues|block\",\"findings\":[{severity,category,taskId,acId,detail,suggestion,source}]}.")
	parts = append(parts, "Severity must be one of: block, warn, info. STRICT POLICY: any finding with severity other than info pauses the phase for an explicit user decision. Use info ONLY for purely advisory notes that need no action; if a finding implies any change to the tasks or criteria, use warn or block.")
	parts = append(parts, "")
	parts = append(parts, "Tasks:")
	for _, t := range tasks {
		parts = append(parts, fmt.Sprintf("- %s: %s", t.ID, t.Title))
		for _, cr := range t.Criteria {
			parts = append(parts, fmt.Sprintf("  - %s: %s", cr.ID, strings.TrimSpace(cr.Then)))
		}
		if len(t.EdgeCases) > 0 {
			parts = append(parts, "  edge cases:")
			for _, ec := range t.EdgeCases {
				parts = append(parts, "    - "+ec)
			}
		}
		if t.Exec != nil && t.Exec.Plan != nil && len(t.Exec.Plan.TouchedFiles) > 0 {
			parts = append(parts, "  approved touched files:")
			for _, f := range t.Exec.Plan.TouchedFiles {
				parts = append(parts, "    - "+f)
			}
		}
	}

	if ec := c.AnswerValue("edge_cases"); ec != "" {
		parts = append(parts, "")
		parts = append(parts, "Discovery edge cases: "+ec)
	}
	if sb := c.AnswerValue("scope_boundary"); sb != "" {
		parts = append(parts, "Scope boundary: "+sb)
	}

	parts = append(parts, "")
	parts = append(parts, "Precomputed structural findings (linter):")
	if len(lint) == 0 {
		parts = append(parts, "- none")
	}
	for _, f := range lint {
		parts = append(parts, fmt.Sprintf("- [%s] %s %s: %s", f.Severity, f.Category, f.TaskID, f.Detail))
	}

	return strings.Join(parts, "\n")
}

func (d *analysisDriver) Next(c *engine.Context, ph *engine.PhaseDef) (engine.Action, bool) {
	if c.HasAnswer(answerKeyComplete) {
		return engine.Action{}, true
	}

	tasks := c.Progress().Tasks
	lint := spec.BuildLint(tasks)

	if c.HasAnswer(answerKeyAudited) {
		var findings []spec.Finding
		if raw := c.AnswerValue(answerKeyFindings); raw != "" {
			if err := json.Unmarshal([]byte(raw), &findings); err != nil {
				return engine.Action{Action: engine.ActionError, Instruction: fmt.Sprintf("stored analysis findings are corrupt: %v", err)}, false
			}
		}
		if !anyActionable(findings) {
			return engine.Action{}, true
		}

		var detail []string
		detail = append(detail, "The analysis flagged findings that need your decision (every finding except info severity). Choose how to proceed:")
		for _, f := range findings {
			if !f.IsInfo() {
				detail = append(detail, fmt.Sprintf("- [%s] %s %s: %s", f.Severity, f.Category, f.TaskID, f.Detail))
			}
		}
		return engine.Action{
			Action:      engine.ActionAsk,
			Instruction: strings.Join(detail, "\n"),
			InteractiveOptions: []engine.InteractiveOption{
				{Label: optReturnToRefinement, Description: "Return to refinement to address the blocking findings."},
				{Label: optAcceptAnyway, Description: "Accept the analysis despite the blocking findings and continue."},
				{Label: optEdit, Description: "Edit the tasks inline, then re-run the audit."},
			},
			CommandMap: map[string]string{
				optReturnToRefinement: fmt.Sprintf("tddmaster next %s --answer='{\"action\":\"return-to-refinement\",\"payload\":{...}}'", c.Slug()),
				optAcceptAnyway:       fmt.Sprintf("tddmaster next %s --answer='accept-anyway'", c.Slug()),
				optEdit:               fmt.Sprintf("tddmaster next %s --answer='{\"action\":\"edit\",\"payload\":{...}}'", c.Slug()),
			},
		}, false
	}

	return engine.Action{
		Action:        engine.ActionInstruct,
		DelegateAgent: auditorAgent,
		Instruction:   buildAuditorInstruction(c, tasks, lint),
		ExpectedInput: engine.ExpectedInput{
			Format:  engine.FormatJSON,
			Example: `{"verdict":"clean","findings":[]}`,
		},
	}, false
}

func (d *analysisDriver) applyEdit(c *engine.Context, payload []byte) error {
	var rp spec.RefinePayload
	if err := json.Unmarshal(payload, &rp); err != nil {
		return fmt.Errorf("invalid refine payload: %w", err)
	}
	pr := c.Progress()
	seq := pr.TaskSeq
	newTasks, newSeq, err := spec.ApplyRefinement(pr.Tasks, rp, c.Settings().TDDEnabled, seq)
	if err != nil {
		return err
	}
	pr.Tasks = newTasks
	pr.TaskSeq = newSeq
	return c.SaveProgress(pr)
}

func (d *analysisDriver) clearAuditState(c *engine.Context) error {
	if err := c.SetAnswer(answerKeyAudited, ""); err != nil {
		return err
	}
	return c.SetAnswer(answerKeyFindings, "")
}

func (d *analysisDriver) Submit(c *engine.Context, ph *engine.PhaseDef, answer []byte) (engine.Action, bool, error) {
	if c.HasAnswer(answerKeyComplete) {
		return engine.Action{}, true, nil
	}

	if c.HasAnswer(answerKeyAudited) {
		trimmed := strings.TrimSpace(string(answer))
		if trimmed == optAcceptAnyway {
			if err := c.SetAnswer(answerKeyComplete, "1"); err != nil {
				return engine.Action{}, false, err
			}
			return engine.Action{}, true, nil
		}

		var gate analysisGateAnswer
		if err := json.Unmarshal(answer, &gate); err != nil {
			return engine.Action{Action: engine.ActionError, Instruction: fmt.Sprintf("malformed gate answer: %v", err)}, false, nil
		}

		if gate.Action == optAcceptAnyway {
			if err := c.SetAnswer(answerKeyComplete, "1"); err != nil {
				return engine.Action{}, false, err
			}
			return engine.Action{}, true, nil
		}

		if len(gate.Payload) > 0 {
			if err := d.applyEdit(c, gate.Payload); err != nil {
				return engine.Action{Action: engine.ActionError, Instruction: err.Error()}, false, nil
			}
		}

		attempts := 0
		if c.HasAnswer(answerKeyAttempts) {
			attempts, _ = strconv.Atoi(c.AnswerValue(answerKeyAttempts))
		}
		attempts++
		if err := c.SetAnswer(answerKeyAttempts, strconv.Itoa(attempts)); err != nil {
			return engine.Action{}, false, err
		}
		if attempts >= analysisMaxAttempts {
			if err := c.SetAnswer(answerKeyComplete, "1"); err != nil {
				return engine.Action{}, false, err
			}
			return engine.Action{}, true, nil
		}
		if err := d.clearAuditState(c); err != nil {
			return engine.Action{}, false, err
		}
		return engine.Action{}, false, nil
	}

	var auditor spec.Analysis
	if err := json.Unmarshal(answer, &auditor); err != nil {
		return engine.Action{Action: engine.ActionError, Instruction: fmt.Sprintf("malformed auditor JSON: %v", err)}, false, nil
	}

	tasks := c.Progress().Tasks
	lint := spec.BuildLint(tasks)
	merged := mergeFindings(auditor.Findings, lint)

	if err := c.SaveAnalysis(spec.Analysis{Verdict: auditor.Verdict, Findings: merged}); err != nil {
		return engine.Action{}, false, err
	}

	mergedJSON, err := json.Marshal(merged)
	if err != nil {
		return engine.Action{}, false, err
	}
	if err := c.SetAnswer(answerKeyFindings, string(mergedJSON)); err != nil {
		return engine.Action{}, false, err
	}
	if err := c.SetAnswer(answerKeyAudited, "1"); err != nil {
		return engine.Action{}, false, err
	}

	if !anyActionable(merged) {
		if err := c.SetAnswer(answerKeyComplete, "1"); err != nil {
			return engine.Action{}, false, err
		}
		return engine.Action{}, true, nil
	}

	return engine.Action{}, false, nil
}
