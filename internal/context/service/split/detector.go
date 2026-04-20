// Package split analyses discovery answers for independent work areas and
// proposes spec splits when the scope straddles unrelated domains.
package split

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/state"
)

// separatorPatterns indicate independent work areas mentioned in the same answer.
var separatorPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\badditionally\b`),
	regexp.MustCompile(`(?i)\bseparately\b`),
	regexp.MustCompile(`(?i)\band also\b`),
	regexp.MustCompile(`(?i)\banother issue\b`),
	regexp.MustCompile(`(?i)\bsecond problem\b`),
	regexp.MustCompile(`(?i)\bon the other hand\b`),
	regexp.MustCompile(`(?i)\bplus\b`),
}

// alsoSentencePattern matches "Also" at sentence boundary.
var alsoSentencePattern = regexp.MustCompile(`(?im)(?:^|[.!?]\s+)also[,\s]`)

// andVerbPattern matches AND joining unrelated verb phrases.
var andVerbPattern = regexp.MustCompile(`(?i)\b(fix|add|restore|update|remove|refactor|rewrite|implement|create|migrate|convert)\s+\S+(?:\s+\S+){0,4}\s+AND\s+(fix|add|restore|update|remove|refactor|rewrite|implement|create|migrate|convert)\s+`)

// couplingPatterns detect prose indicators that two areas must land together.
var couplingPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\buse\s+(?:the\s+)?(?:new\s+)?(\w+)\s+(?:type|interface|class|module|function)\b`),
	regexp.MustCompile(`(?i)\bafter\s+(?:adding|creating|implementing)\b`),
	regexp.MustCompile(`(?i)\bdepends\s+on\b`),
	regexp.MustCompile(`(?i)\bprerequisite\b`),
	regexp.MustCompile(`(?i)\brequires\s+(?:the\s+)?(?:above|previous|first)\b`),
	regexp.MustCompile(`(?i)\bthen\s+use\b`),
}

// numberedListPatternCheck matches any of the three numbered-list shapes.
var numberedListPatternCheck = regexp.MustCompile(`(?im)(?:^|\n|\.\s+)\s*(?:\(\d+\)|\d+\.\s|(?:first|second|third|fourth|fifth)[:;,])`)

// ordinalRe matches "First:", "Second:", etc. at line/sentence boundaries.
var ordinalRe = regexp.MustCompile(`(?im)(?:^|\n|[.!?]\s+)\s*(?:first|second|third|fourth|fifth)[:;,]\s*`)

var actionVerbsRe = regexp.MustCompile(`(?i)\b(fix|add|restore|update|remove|refactor|rewrite|implement|create|migrate|convert|replace|extract|move|rename|split|merge|test|verify|validate|configure|setup|install|deploy)\b`)

var pascalCaseRe = regexp.MustCompile(`\b[A-Z][a-zA-Z0-9]+\b`)
var filePathRe = regexp.MustCompile(`\b[\w-]+\.\w{1,4}\b`)
var nonAlphanumRe = regexp.MustCompile(`[^a-z0-9\s-]`)
var multiHyphenRe = regexp.MustCompile(`--+`)

var stopWords = map[string]bool{
	"the": true, "a": true, "an": true, "is": true, "are": true,
	"was": true, "were": true, "be": true, "been": true, "being": true,
	"have": true, "has": true, "had": true, "do": true, "does": true,
	"did": true, "will": true, "would": true, "shall": true, "should": true,
	"may": true, "might": true, "must": true, "can": true, "could": true,
	"to": true, "of": true, "in": true, "for": true, "on": true,
	"with": true, "at": true, "by": true, "from": true, "that": true,
	"this": true, "it": true, "its": true, "and": true, "or": true,
	"but": true, "not": true, "no": true, "so": true, "if": true,
	"then": true, "too": true, "very": true, "just": true,
}

var noSplit = model.SplitProposal{
	Detected:  false,
	Reason:    "",
	Proposals: []model.SplitProposalItem{},
}

// rawArea is an intermediate detected area before being slug/estimate-tagged.
type rawArea struct {
	text            string
	sourceQuestions []string
}

// Analyze analyzes discovery answers for independent work areas.
func Analyze(answers []state.DiscoveryAnswer, discoveryMode string) model.SplitProposal {
	if discoveryMode == "ship-fast" {
		return noSplit
	}

	answerMap := make(map[string]string)
	for _, a := range answers {
		answerMap[a.QuestionID] = a.Answer
	}

	statusQuo := answerMap["status_quo"]
	ambition := answerMap["ambition"]

	areas := detectAreas(statusQuo, ambition)
	if len(areas) < 2 {
		return noSplit
	}

	totalTasks := 0
	for _, a := range areas {
		totalTasks += a.EstimatedTasks
	}
	if totalTasks <= model.SplitMinTotalTasks {
		return noSplit
	}

	if areTightlyCoupled(areas) {
		return noSplit
	}

	return model.SplitProposal{
		Detected:  true,
		Reason:    "Discovery answers cover " + strconv.Itoa(len(areas)) + " independent areas that could be separate specs.",
		Proposals: areas,
	}
}

func detectAreas(statusQuo, ambition string) []model.SplitProposalItem {
	if areas := detectNumberedLists(statusQuo, ambition); len(areas) >= 2 {
		return areasToItems(areas)
	}
	if areas := detectBySeparators(statusQuo, ambition); len(areas) >= 2 {
		return areasToItems(areas)
	}
	if areas := detectByAndPattern(statusQuo, ambition); len(areas) >= 2 {
		return areasToItems(areas)
	}
	return nil
}

func areasToItems(areas []rawArea) []model.SplitProposalItem {
	out := make([]model.SplitProposalItem, len(areas))
	for i, a := range areas {
		out[i] = toProposalItem(a)
	}
	return out
}

func detectNumberedLists(statusQuo, ambition string) []rawArea {
	sources := [][2]string{
		{statusQuo, "status_quo"},
		{ambition, "ambition"},
	}
	for _, s := range sources {
		text, questionID := s[0], s[1]
		if !numberedListPatternCheck.MatchString(text) {
			continue
		}
		items := splitNumberedList(text)
		if len(items) >= 2 {
			areas := make([]rawArea, len(items))
			for i, item := range items {
				areas[i] = rawArea{text: strings.TrimSpace(item), sourceQuestions: []string{questionID}}
			}
			return areas
		}
	}
	return nil
}

func splitNumberedList(text string) []string {
	parenRe := regexp.MustCompile(`\(\d+\)\s*`)
	parenParts := parenRe.Split(text, -1)
	var parenItems []string
	for _, p := range parenParts[1:] {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			parenItems = append(parenItems, trimmed)
		}
	}
	if len(parenItems) >= 2 {
		return parenItems
	}

	dotRe := regexp.MustCompile(`(?:^|\n|[.!?]\s+)\s*\d+\.\s+`)
	locs := dotRe.FindAllStringIndex(text, -1)
	if len(locs) >= 2 {
		var dotItems []string
		for i, loc := range locs {
			var end int
			if i+1 < len(locs) {
				end = locs[i+1][0]
			} else {
				end = len(text)
			}
			matchEnd := loc[1]
			if matchEnd <= end {
				part := strings.TrimSpace(text[matchEnd:end])
				if part != "" {
					dotItems = append(dotItems, part)
				}
			}
		}
		if len(dotItems) >= 2 {
			return dotItems
		}
	}

	ordinalLocs := ordinalRe.FindAllStringIndex(text, -1)
	if len(ordinalLocs) >= 2 {
		var ordinalItems []string
		for i, loc := range ordinalLocs {
			var end int
			if i+1 < len(ordinalLocs) {
				end = ordinalLocs[i+1][0]
			} else {
				end = len(text)
			}
			matchEnd := loc[1]
			if matchEnd <= end {
				part := strings.TrimSpace(text[matchEnd:end])
				if part != "" {
					ordinalItems = append(ordinalItems, part)
				}
			}
		}
		if len(ordinalItems) >= 2 {
			return ordinalItems
		}
	}
	return nil
}

func detectBySeparators(statusQuo, ambition string) []rawArea {
	sources := [][2]string{
		{statusQuo, "status_quo"},
		{ambition, "ambition"},
	}
	for _, s := range sources {
		text, questionID := s[0], s[1]

		if alsoSentencePattern.MatchString(text) {
			parts := splitOnSeparator(text, alsoSentencePattern)
			if len(parts) >= 2 {
				areas := make([]rawArea, len(parts))
				for i, p := range parts {
					areas[i] = rawArea{text: strings.TrimSpace(p), sourceQuestions: []string{questionID}}
				}
				return areas
			}
		}

		for _, pattern := range separatorPatterns {
			if pattern.MatchString(text) {
				raw := pattern.Split(text, -1)
				var parts []string
				for _, p := range raw {
					trimmed := strings.TrimSpace(p)
					if trimmed != "" {
						parts = append(parts, trimmed)
					}
				}
				if len(parts) >= 2 {
					areas := make([]rawArea, len(parts))
					for i, p := range parts {
						areas[i] = rawArea{text: p, sourceQuestions: []string{questionID}}
					}
					return areas
				}
			}
		}
	}
	return nil
}

func splitOnSeparator(text string, pattern *regexp.Regexp) []string {
	loc := pattern.FindStringIndex(text)
	if loc == nil {
		return nil
	}
	match := pattern.FindString(text)
	before := strings.TrimSpace(text[:loc[0]])
	after := strings.TrimSpace(text[loc[0]+len(match):])
	if before != "" && after != "" {
		return []string{before, after}
	}
	return nil
}

func detectByAndPattern(statusQuo, ambition string) []rawArea {
	sources := [][2]string{
		{statusQuo, "status_quo"},
		{ambition, "ambition"},
	}
	for _, s := range sources {
		text, questionID := s[0], s[1]
		loc := andVerbPattern.FindStringIndex(text)
		if loc == nil {
			continue
		}
		upper := strings.ToUpper(text)
		andIdx := strings.Index(upper[loc[0]:], " AND ")
		if andIdx < 0 {
			continue
		}
		andIdx += loc[0]
		before := strings.TrimSpace(text[:andIdx])
		after := strings.TrimSpace(text[andIdx+5:])
		if before != "" && after != "" {
			return []rawArea{
				{text: before, sourceQuestions: []string{questionID}},
				{text: after, sourceQuestions: []string{questionID}},
			}
		}
	}
	return nil
}

func toProposalItem(area rawArea) model.SplitProposalItem {
	return model.SplitProposalItem{
		Name:            slugify(area.text),
		Description:     area.text,
		EstimatedTasks:  estimateTasks(area.text),
		RelevantAnswers: append([]string(nil), area.sourceQuestions...),
	}
}

func slugify(text string) string {
	lower := strings.ToLower(text)
	cleaned := nonAlphanumRe.ReplaceAllString(lower, "")
	rawWords := strings.Fields(cleaned)

	var words []string
	for _, w := range rawWords {
		if w != "" && !stopWords[w] {
			words = append(words, w)
		}
	}
	if len(words) > model.SlugMaxWords {
		words = words[:model.SlugMaxWords]
	}

	slug := strings.Join(words, "-")
	slug = multiHyphenRe.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > model.SlugMaxLength {
		slug = slug[:model.SlugMaxLength]
	}
	if slug == "" {
		return "area"
	}
	return slug
}

func estimateTasks(text string) int {
	matches := actionVerbsRe.FindAllString(text, -1)
	result := len(matches) + 1
	if result < model.TaskEstimateMin {
		result = model.TaskEstimateMin
	}
	if result > model.TaskEstimateMax {
		result = model.TaskEstimateMax
	}
	return result
}

func areTightlyCoupled(areas []model.SplitProposalItem) bool {
	for i := 0; i < len(areas); i++ {
		for j := i + 1; j < len(areas); j++ {
			a := areas[i]
			b := areas[j]
			combined := a.Description + " " + b.Description
			for _, pattern := range couplingPatterns {
				if pattern.MatchString(combined) {
					return true
				}
			}
			aNouns := extractKeyNouns(a.Description)
			bNouns := extractKeyNouns(b.Description)
			for n := range aNouns {
				if bNouns[n] {
					if len(n) > 0 && ((n[0] >= 'A' && n[0] <= 'Z') || strings.Contains(n, ".")) {
						return true
					}
				}
			}
		}
	}
	return false
}

func extractKeyNouns(text string) map[string]bool {
	nouns := make(map[string]bool)
	for _, m := range pascalCaseRe.FindAllString(text, -1) {
		nouns[m] = true
	}
	for _, m := range filePathRe.FindAllString(text, -1) {
		nouns[m] = true
	}
	return nouns
}
