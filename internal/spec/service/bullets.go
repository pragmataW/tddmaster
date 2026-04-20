package service

import (
	"regexp"
	"strings"
)

// sentenceSplitRe matches sentence-ending periods followed by space+uppercase.
// It does NOT match periods inside filenames, extensions, version numbers, or
// URLs. Semicolons are NOT split points — they are legitimate punctuation in
// long technical answers and splitting on them fragmented content mid-sentence.
var sentenceSplitRe = regexp.MustCompile(`\.(?:\s+[A-Z])`)

// numberedMarkerRe matches "(N) " style numbered list markers. The match is on
// the space+open-paren so that the marker stays with the item that follows it.
var numberedMarkerRe = regexp.MustCompile(`\s+\(\d+\)\s+`)

// splitPoint describes a single split location in the text with the end offset
// of the current item and the start offset of the following item.
type splitPoint struct {
	itemEnd   int
	nextStart int
}

// toBulletList splits text into list items by line breaks or sentence
// boundaries. Does NOT split on dots inside filenames, extensions,
// abbreviations, version numbers, or URLs, nor on semicolons within a
// sentence. Also splits on "(N)" style numbered markers.
func toBulletList(text string) []string {
	rawLines := strings.Split(text, "\n")
	var lines []string
	for _, l := range rawLines {
		t := strings.TrimSpace(l)
		if t != "" {
			lines = append(lines, t)
		}
	}

	if len(lines) > 1 {
		return lines
	}

	var splits []splitPoint

	for _, loc := range sentenceSplitRe.FindAllStringIndex(text, -1) {
		splits = append(splits, splitPoint{itemEnd: loc[0] + 1, nextStart: loc[0] + 2})
	}
	for _, loc := range numberedMarkerRe.FindAllStringIndex(text, -1) {
		splits = append(splits, splitPoint{itemEnd: loc[0], nextStart: loc[0] + 1})
	}

	if len(splits) == 0 {
		trimmed := strings.TrimSpace(text)
		if len(trimmed) > 5 {
			return []string{trimmed}
		}
		return []string{}
	}

	for i := 1; i < len(splits); i++ {
		for j := i; j > 0 && splits[j].nextStart < splits[j-1].nextStart; j-- {
			splits[j], splits[j-1] = splits[j-1], splits[j]
		}
	}

	var parts []string
	prev := 0
	for _, sp := range splits {
		if sp.itemEnd <= prev {
			continue
		}
		parts = append(parts, strings.TrimSpace(text[prev:sp.itemEnd]))
		prev = sp.nextStart
	}
	if prev < len(text) {
		parts = append(parts, strings.TrimSpace(text[prev:]))
	}

	var result []string
	for _, p := range parts {
		if len(p) > 5 {
			result = append(result, p)
		}
	}
	return result
}
