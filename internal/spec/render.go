package spec

import (
	"encoding/json"
	"log"
	"regexp"
	"sort"
	"strings"
)

const (
	decisionRevisedLabel  = "revize"
	decisionRejectedLabel = "reddedildi"
)

var specialKeys = map[string]bool{
	"premises":       true,
	"scope_boundary": true,
	"edge_cases":     true,
	"verification":   true,
}

var hiddenKeys = map[string]bool{
	"mode":            true,
	"listen_context":  true,
	"self_review":     true,
	"synthesis":       true,
	"tasks_generated": true,
}

var edgeCaseSplit = regexp.MustCompile(`\(\d+\)\s*`)

func joinAnswers(answers []Answer) string {
	parts := make([]string, len(answers))
	for i, a := range answers {
		parts[i] = a.Value
	}
	return strings.Join(parts, "\n")
}

func sectionValue(answers map[string][]Answer, key string) string {
	vals, ok := answers[key]
	if !ok {
		return ""
	}
	return joinAnswers(vals)
}

func renderDecisions(val string) []string {
	var parsed struct {
		Premises []struct {
			Text     string `json:"text"`
			Agreed   bool   `json:"agreed"`
			Revision string `json:"revision"`
		} `json:"premises"`
	}
	if err := json.Unmarshal([]byte(val), &parsed); err != nil {
		log.Printf("tddmaster: failed to parse premises for spec.md decisions: %v", err)
		return nil
	}
	if len(parsed.Premises) == 0 {
		return nil
	}
	items := make([]string, 0, len(parsed.Premises))
	for _, p := range parsed.Premises {
		item := strings.TrimSpace(p.Text)
		if item == "" {
			continue
		}
		if !p.Agreed {
			if rev := strings.TrimSpace(p.Revision); rev != "" {
				item += " (" + decisionRevisedLabel + ": " + rev + ")"
			} else {
				item += " (" + decisionRejectedLabel + ")"
			}
		}
		items = append(items, item)
	}
	return items
}

func ParseEdgeCases(val string) []string {
	val = strings.TrimSpace(val)
	if val == "" {
		return nil
	}
	var raw []string
	if edgeCaseSplit.MatchString(val) {
		raw = edgeCaseSplit.Split(val, -1)
	} else {
		raw = strings.Split(val, "\n")
	}
	items := make([]string, 0, len(raw))
	for _, p := range raw {
		p = strings.TrimSpace(p)
		p = strings.TrimRight(p, ".")
		p = strings.TrimSpace(p)
		if p == "" || strings.HasSuffix(p, ":") {
			continue
		}
		items = append(items, p)
	}
	return items
}

func writeBullets(b *strings.Builder, header string, items []string, raw string) {
	b.WriteString("## ")
	b.WriteString(header)
	b.WriteString("\n")
	switch {
	case len(items) > 0:
		for _, it := range items {
			b.WriteString("- ")
			b.WriteString(it)
			b.WriteString("\n")
		}
	case strings.TrimSpace(raw) != "":
		b.WriteString(raw)
		b.WriteString("\n")
	default:
		b.WriteString("_None_\n")
	}
	b.WriteString("\n")
}

func RenderSpecMd(slug string, st State, pr Progress) string {
	var b strings.Builder

	b.WriteString("# Spec: ")
	b.WriteString(slug)
	b.WriteString("\n\n")

	b.WriteString("## Status\n")
	b.WriteString(pr.Status)
	b.WriteString("\n\n")

	var nonSpecialKeys []string
	for k := range st.Answers {
		if specialKeys[k] || hiddenKeys[k] {
			continue
		}
		nonSpecialKeys = append(nonSpecialKeys, k)
	}
	sort.Strings(nonSpecialKeys)

	b.WriteString("## Discovery Answers\n")
	if len(nonSpecialKeys) == 0 {
		b.WriteString("_None_\n")
	} else {
		for _, k := range nonSpecialKeys {
			b.WriteString("### ")
			b.WriteString(k)
			b.WriteString("\n")
			b.WriteString(joinAnswers(st.Answers[k]))
			b.WriteString("\n\n")
		}
	}

	decRaw := sectionValue(st.Answers, "premises")
	writeBullets(&b, "Decisions", renderDecisions(decRaw), decRaw)

	scopeRaw := sectionValue(st.Answers, "scope_boundary")
	writeBullets(&b, "Out of Scope", nil, scopeRaw)

	edgeRaw := sectionValue(st.Answers, "edge_cases")
	writeBullets(&b, "Edge Cases", ParseEdgeCases(edgeRaw), "")

	b.WriteString("## Tasks\n")
	if len(pr.Tasks) == 0 {
		b.WriteString("_None_\n")
	} else {
		for _, task := range pr.Tasks {
			if task.Done {
				b.WriteString("- [x] ")
			} else {
				b.WriteString("- [ ] ")
			}
			b.WriteString(task.ID)
			b.WriteString(": ")
			b.WriteString(task.Title)
			if task.TDDEnabled {
				b.WriteString(" (TDD)")
			}
			if task.Important {
				b.WriteString(" (important)")
			}
			b.WriteString("\n")
			for _, ac := range task.AC {
				b.WriteString("  - ")
				b.WriteString(ac)
				b.WriteString("\n")
			}
		}
	}
	b.WriteString("\n")

	b.WriteString("## Verification\n")
	val := sectionValue(st.Answers, "verification")
	if val == "" {
		b.WriteString("_None_\n")
	} else {
		b.WriteString(val)
		b.WriteString("\n")
	}

	return b.String()
}
