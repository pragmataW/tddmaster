
package context

import (
	"regexp"
	"strings"

	"github.com/pragmataW/tddmaster/internal/state"
)

// =============================================================================
// Types
// =============================================================================

// SplitProposalItem is a single proposed sub-spec in a split.
type SplitProposalItem struct {
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	EstimatedTasks  int      `json:"estimatedTasks"`
	RelevantAnswers []string `json:"relevantAnswers"`
}

// SplitProposal is the result of analyzing discovery answers for potential spec splits.
type SplitProposal struct {
	Detected  bool                `json:"detected"`
	Reason    string              `json:"reason"`
	Proposals []SplitProposalItem `json:"proposals"`
}

// =============================================================================
// Constants
// =============================================================================

var noSplit = SplitProposal{
	Detected:  false,
	Reason:    "",
	Proposals: []SplitProposalItem{},
}

// Separator words/phrases that indicate independent work areas.
var separatorPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\badditionally\b`),
	regexp.MustCompile(`(?i)\bseparately\b`),
	regexp.MustCompile(`(?i)\band also\b`),
	regexp.MustCompile(`(?i)\banother issue\b`),
	regexp.MustCompile(`(?i)\bsecond problem\b`),
	regexp.MustCompile(`(?i)\bon the other hand\b`),
	regexp.MustCompile(`(?i)\bplus\b`),
}

// "Also" at sentence boundary.
var alsoSentencePattern = regexp.MustCompile(`(?im)(?:^|[.!?]\s+)also[,\s]`)

// AND joining unrelated verb phrases.
var andVerbPattern = regexp.MustCompile(`(?i)\b(fix|add|restore|update|remove|refactor|rewrite|implement|create|migrate|convert)\s+\S+(?:\s+\S+){0,4}\s+AND\s+(fix|add|restore|update|remove|refactor|rewrite|implement|create|migrate|convert)\s+`)

// Coupling indicators.
var couplingPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\buse\s+(?:the\s+)?(?:new\s+)?(\w+)\s+(?:type|interface|class|module|function)\b`),
	regexp.MustCompile(`(?i)\bafter\s+(?:adding|creating|implementing)\b`),
	regexp.MustCompile(`(?i)\bdepends\s+on\b`),
	regexp.MustCompile(`(?i)\bprerequisite\b`),
	regexp.MustCompile(`(?i)\brequires\s+(?:the\s+)?(?:above|previous|first)\b`),
	regexp.MustCompile(`(?i)\bthen\s+use\b`),
}

// =============================================================================
// Public API
// =============================================================================

// AnalyzeForSplit analyzes discovery answers for independent work areas.
func AnalyzeForSplit(answers []state.DiscoveryAnswer, discoveryMode string) SplitProposal {
	if discoveryMode == "ship-fast" {
		return noSplit
	}

	answerMap := make(map[string]string)
	for _, a := range answers {
		answerMap[a.QuestionID] = a.Answer
	}

	statusQuo := answerMap["status_quo"]
	ambition := answerMap["ambition"]
	// scopeBoundary and verification unused in detection but kept for API parity
	_ = answerMap["scope_boundary"]
	_ = answerMap["verification"]

	areas := detectAreas(statusQuo, ambition)

	if len(areas) < 2 {
		return noSplit
	}

	totalTasks := 0
	for _, a := range areas {
		totalTasks += a.EstimatedTasks
	}
	if totalTasks <= 3 {
		return noSplit
	}

	if areTightlyCoupled(areas) {
		return noSplit
	}

	return SplitProposal{
		Detected: true,
		Reason:   "Discovery answers cover " + itoa(len(areas)) + " independent areas that could be separate specs.",
		Proposals: areas,
	}
}

// =============================================================================
// Area Detection
// =============================================================================

type rawArea struct {
	text            string
	sourceQuestions []string
}

func detectAreas(statusQuo, ambition string) []SplitProposalItem {
	// Try numbered list detection first
	numberedAreas := detectNumberedLists(statusQuo, ambition)
	if len(numberedAreas) >= 2 {
		result := make([]SplitProposalItem, len(numberedAreas))
		for i, a := range numberedAreas {
			result[i] = toProposalItem(a)
		}
		return result
	}

	// Try separator word detection
	separatorAreas := detectBySeparators(statusQuo, ambition)
	if len(separatorAreas) >= 2 {
		result := make([]SplitProposalItem, len(separatorAreas))
		for i, a := range separatorAreas {
			result[i] = toProposalItem(a)
		}
		return result
	}

	// Try AND pattern detection
	andAreas := detectByAndPattern(statusQuo, ambition)
	if len(andAreas) >= 2 {
		result := make([]SplitProposalItem, len(andAreas))
		for i, a := range andAreas {
			result[i] = toProposalItem(a)
		}
		return result
	}

	return nil
}

// numberedListPatternCheck matches numbered list patterns for quick detection.
var numberedListPatternCheck = regexp.MustCompile(`(?im)(?:^|\n|\.\s+)\s*(?:\(\d+\)|\d+\.\s|(?:first|second|third|fourth|fifth)[:;,])`)

func detectNumberedLists(statusQuo, ambition string) []rawArea {
	sources := [][2]string{
		{statusQuo, "status_quo"},
		{ambition, "ambition"},
	}

	for _, s := range sources {
		text := s[0]
		questionID := s[1]

		if !numberedListPatternCheck.MatchString(text) {
			continue
		}

		items := splitNumberedList(text)
		if len(items) >= 2 {
			areas := make([]rawArea, len(items))
			for i, item := range items {
				areas[i] = rawArea{
					text:            strings.TrimSpace(item),
					sourceQuestions: []string{questionID},
				}
			}
			return areas
		}
	}

	return nil
}

// ordinalRe matches "First:", "Second:", etc. at line/sentence boundaries.
var ordinalRe = regexp.MustCompile(`(?im)(?:^|\n|[.!?]\s+)\s*(?:first|second|third|fourth|fifth)[:;,]\s*`)

func splitNumberedList(text string) []string {
	// Try parenthesized numbers: (1) ... (2) ...
	parenRe := regexp.MustCompile(`\(\d+\)\s*`)
	parenParts := parenRe.Split(text, -1)
	var parenItems []string
	for _, p := range parenParts[1:] { // skip preamble
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			parenItems = append(parenItems, trimmed)
		}
	}
	if len(parenItems) >= 2 {
		return parenItems
	}

	// Try "N. " pattern at line start or sentence boundary (". N. ").
	// We include the sentence terminator in the split so N. after period also works.
	dotRe := regexp.MustCompile(`(?:^|\n|[.!?]\s+)\s*\d+\.\s+`)
	// Use FindAllStringIndex to locate all boundaries
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
			// Extract text from after the match to the next match (or end)
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

	// Try "First: ... Second: ..." pattern — can appear after sentence boundary
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
		text := s[0]
		questionID := s[1]

		// Check for "also" at sentence boundary
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

		// Check other separator patterns
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
		text := s[0]
		questionID := s[1]

		loc := andVerbPattern.FindStringIndex(text)
		if loc != nil {
			// Split on the AND
			upper := strings.ToUpper(text)
			andIdx := strings.Index(upper[loc[0]:], " AND ")
			if andIdx >= 0 {
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
		}
	}

	return nil
}

// =============================================================================
// Proposal Item Construction
// =============================================================================

func toProposalItem(area rawArea) SplitProposalItem {
	return SplitProposalItem{
		Name:            slugify(area.text),
		Description:     area.text,
		EstimatedTasks:  estimateTasks(area.text),
		RelevantAnswers: append([]string(nil), area.sourceQuestions...),
	}
}

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

var nonAlphanumRe = regexp.MustCompile(`[^a-z0-9\s-]`)
var multiHyphenRe = regexp.MustCompile(`--+`)

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

	if len(words) > 4 {
		words = words[:4]
	}

	slug := strings.Join(words, "-")
	slug = multiHyphenRe.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 50 {
		slug = slug[:50]
	}

	if slug == "" {
		return "area"
	}
	return slug
}

var actionVerbsRe = regexp.MustCompile(`(?i)\b(fix|add|restore|update|remove|refactor|rewrite|implement|create|migrate|convert|replace|extract|move|rename|split|merge|test|verify|validate|configure|setup|install|deploy)\b`)

func estimateTasks(text string) int {
	matches := actionVerbsRe.FindAllString(text, -1)
	verbCount := len(matches)
	result := verbCount + 1
	if result < 2 {
		result = 2
	}
	if result > 5 {
		result = 5
	}
	return result
}

// =============================================================================
// Coupling Detection
// =============================================================================

var pascalCaseRe = regexp.MustCompile(`\b[A-Z][a-zA-Z0-9]+\b`)
var filePathRe = regexp.MustCompile(`\b[\w-]+\.\w{1,4}\b`)

func areTightlyCoupled(areas []SplitProposalItem) bool {
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

			aNounsSet := extractKeyNouns(a.Description)
			bNounsSet := extractKeyNouns(b.Description)

			for n := range aNounsSet {
				if bNounsSet[n] {
					// If shared noun starts with uppercase or contains a dot
					if len(n) > 0 && (n[0] >= 'A' && n[0] <= 'Z') || strings.Contains(n, ".") {
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

	// PascalCase identifiers
	for _, m := range pascalCaseRe.FindAllString(text, -1) {
		nouns[m] = true
	}

	// File-like paths
	for _, m := range filePathRe.FindAllString(text, -1) {
		nouns[m] = true
	}

	return nouns
}

// itoa converts int to string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	buf := make([]byte, 20)
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
