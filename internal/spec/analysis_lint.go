package spec

import "strings"

func BuildLint(tasks []Task) []Finding {
	findings := []Finding{}
	for _, task := range tasks {
		if len(task.Criteria) == 0 {
			findings = append(findings, Finding{
				Severity: "block",
				Category: "task-no-ac",
				TaskID:   task.ID,
				Detail:   "Task has no acceptance criteria",
				Source:   "linter",
			})
		}
		findings = append(findings, LintCriteria(task)...)
		seen := map[string]bool{}
		for _, c := range task.Criteria {
			then := strings.TrimSpace(c.Then)
			if then == "" {
				continue
			}
			key := strings.TrimSpace(c.Given) + "\x00" + strings.TrimSpace(c.When) + "\x00" + then
			if seen[key] {
				findings = append(findings, Finding{
					Severity: "warn",
					Category: "duplicate",
					TaskID:   task.ID,
					AcID:     c.ID,
					Detail:   "Criterion duplicates an earlier criterion with identical Given, When, and Then",
					Source:   "linter",
				})
				continue
			}
			seen[key] = true
		}
	}
	return findings
}
