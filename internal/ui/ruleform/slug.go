package ruleform

import (
	"regexp"
	"slices"
	"strings"

	"github.com/pragmataW/tddmaster/internal/rules"
)

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

func Targets() []string {
	return append([]string{"global"}, rules.KnownAgents...)
}

func isKnownAgent(s string) bool {
	return slices.Contains(rules.KnownAgents, s)
}

func Slugify(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '/' || r == '\\'
	})

	hasSeparator := strings.ContainsAny(name, "/\\")

	if hasSeparator {
		var filtered []string
		for _, p := range parts {
			if p != "." && p != ".." {
				filtered = append(filtered, p)
			}
		}
		name = strings.Join(filtered, "-")
	}

	lower := strings.ToLower(name)
	slug := nonAlphaNum.ReplaceAllString(lower, "-")
	slug = strings.Trim(slug, "-")
	return slug
}

func EnsureMd(slug string) string {
	if slug == "" {
		return ""
	}
	if strings.HasSuffix(slug, ".md") {
		return slug
	}
	return slug + ".md"
}
