package service

import (
	"regexp"
	"strings"

	"github.com/pragmataW/tddmaster/internal/state"
)

var (
	tenStarRe         = regexp.MustCompile(`(?is)10[- ]?star[:\s]+(.+?)(?:\n|$)`)
	fiveStarRe        = regexp.MustCompile(`(?is)5[- ]?star[:\s]+(.+?)(?:\n|$)`)
	oneStarPrefixRe   = regexp.MustCompile(`(?i)1[- ]?star[:\s]+[^.]*\.\s*`)
	leadingArticleRe  = regexp.MustCompile(`(?i)^(the|a|an|with|plus|also)\s+`)
	goalPrefixRe      = regexp.MustCompile(`(?i)^(the\s+)?(target|goal|objective)[:\s]+`)
	trailingPuncRe    = regexp.MustCompile(`[.\x{2026}]+$`)
	bulletPrefixRe    = regexp.MustCompile(`^\s*[-\x{2022}*]\s*`)
	shouldPrefixRe    = regexp.MustCompile(`(?i)^should\s+(we|i)\s+`)
)

// isTestTask reports whether a task string is a test-related task.
func isTestTask(task string) bool {
	return strings.Contains(strings.ToLower(task), "test")
}

// deriveTasks derives tasks from discovery answers and decisions. When tddMode
// is true, test-related tasks are moved to the beginning of the list to enforce
// test-first ordering.
func deriveTasks(answers []state.DiscoveryAnswer, decisions []state.Decision, tddMode bool) []string {
	var tasks []string

	if ambition := findAnswer(answers, "ambition"); ambition != nil {
		tasks = appendIfNonEmpty(tasks, deriveAmbitionTask(ambition.Answer))
	}

	if verification := findAnswer(answers, "verification"); verification != nil {
		for _, line := range strings.Split(verification.Answer, "\n") {
			item := strings.TrimSpace(bulletPrefixRe.ReplaceAllString(line, ""))
			if item != "" {
				tasks = append(tasks, item)
			}
		}
	}

	for _, d := range decisions {
		lower := strings.ToLower(d.Choice)
		if !strings.Contains(lower, "accepted") && !strings.Contains(lower, "add to scope") {
			continue
		}
		taskText := strings.TrimRight(shouldPrefixRe.ReplaceAllString(d.Question, ""), "?")
		taskText = strings.TrimSpace(taskText)
		if taskText != "" {
			tasks = append(tasks, strings.ToUpper(taskText[:1])+taskText[1:])
		}
	}

	if len(tasks) == 0 {
		tasks = append(tasks, "_Tasks need to be defined before execution. Add tasks manually or run discovery with more detail._")
	}

	tasks = append(tasks, "Write or update tests for all new and changed behavior")
	tasks = append(tasks, "Update documentation for all public-facing changes (README, API docs, CHANGELOG)")

	if tddMode {
		var testTasks, otherTasks []string
		for _, task := range tasks {
			if isTestTask(task) {
				testTasks = append(testTasks, task)
			} else {
				otherTasks = append(otherTasks, task)
			}
		}
		tasks = append(testTasks, otherTasks...)
	}

	return tasks
}

// deriveAmbitionTask extracts the implementation goal from an ambition answer.
// Returns empty string if no meaningful goal could be extracted.
func deriveAmbitionTask(text string) string {
	var goalText string
	if m := tenStarRe.FindStringSubmatch(text); m != nil {
		goalText = strings.TrimSpace(m[1])
	} else if m := fiveStarRe.FindStringSubmatch(text); m != nil {
		goalText = strings.TrimSpace(m[1])
	} else {
		goalText = strings.TrimSpace(oneStarPrefixRe.ReplaceAllString(text, ""))
	}

	cleaned := goalPrefixRe.ReplaceAllString(leadingArticleRe.ReplaceAllString(goalText, ""), "")
	cleaned = strings.TrimSpace(cleaned)

	if cleaned != "" {
		cleaned = strings.ToUpper(cleaned[:1]) + cleaned[1:]
	}
	cleaned = strings.TrimSpace(trailingPuncRe.ReplaceAllString(cleaned, ""))

	if len(cleaned) > 140 {
		cleaned = cleaned[:137] + "..."
	}
	if len(cleaned) <= 3 {
		return ""
	}
	return cleaned
}

func findAnswer(answers []state.DiscoveryAnswer, questionID string) *state.DiscoveryAnswer {
	for i := range answers {
		if answers[i].QuestionID == questionID {
			return &answers[i]
		}
	}
	return nil
}

func appendIfNonEmpty(tasks []string, task string) []string {
	if task == "" {
		return tasks
	}
	return append(tasks, task)
}
