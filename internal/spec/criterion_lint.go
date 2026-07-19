package spec

import "strings"

func LintCriteria(t Task) []Finding {
	findings := []Finding{}
	for _, c := range t.Criteria {
		then := strings.TrimSpace(c.Then)
		if then == "" && strings.TrimSpace(c.Raw) == "" {
			findings = append(findings, Finding{
				Severity:   SeverityBlock,
				Category:   "untestable",
				TaskID:     t.ID,
				AcID:       c.ID,
				Detail:     "Then is empty and cannot be verified",
				Suggestion: "Describe a concrete, observable outcome in Then",
				Source:     SourceLinter,
			})
		}
		if strings.TrimSpace(c.When) == "" && strings.TrimSpace(c.Raw) == "" {
			findings = append(findings, Finding{
				Severity:   SeverityWarn,
				Category:   "weak-criterion",
				TaskID:     t.ID,
				AcID:       c.ID,
				Detail:     "When and Raw are both empty, leaving the trigger unspecified",
				Suggestion: "State the action or input that triggers this criterion",
				Source:     SourceLinter,
			})
		}
	}
	return findings
}
