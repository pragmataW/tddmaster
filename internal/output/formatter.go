
// Package output provides CLI output formatting and command prefix utilities.
//
// Port of tddmaster/output/formatter.ts.
package output

import (
	"encoding/json"
	"fmt"
	"strings"
)

// =============================================================================
// Types
// =============================================================================

// OutputFormat represents the output serialization format.
type OutputFormat string

const (
	OutputFormatJSON     OutputFormat = "json"
	OutputFormatMarkdown OutputFormat = "markdown"
	OutputFormatText     OutputFormat = "text"
)

// =============================================================================
// Arg Parsing
// =============================================================================

// ParseOutputFormat extracts the -o / --output flag from args.
// Returns OutputFormatJSON if not specified.
func ParseOutputFormat(args []string) OutputFormat {
	if args == nil {
		return OutputFormatJSON
	}

	for i, arg := range args {
		if arg == "-o" || strings.HasPrefix(arg, "--output") {
			if strings.Contains(arg, "=") {
				parts := strings.SplitN(arg, "=", 2)
				if len(parts) == 2 {
					return normalizeFormat(parts[1])
				}
				return OutputFormatJSON
			}

			// Next arg is the format value
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				return normalizeFormat(args[i+1])
			}
		}
	}

	return OutputFormatJSON
}

// StripOutputFlag removes -o / --output flags and their values from args.
func StripOutputFlag(args []string) []string {
	if args == nil {
		return []string{}
	}

	result := make([]string, 0, len(args))
	skipNext := false

	for _, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}

		if strings.HasPrefix(arg, "--output=") {
			continue
		}
		if arg == "-o" || arg == "--output" {
			skipNext = true
			continue
		}

		result = append(result, arg)
	}

	return result
}

func normalizeFormat(raw string) OutputFormat {
	lower := strings.ToLower(raw)
	if lower == "md" || lower == "markdown" {
		return OutputFormatMarkdown
	}
	if lower == "text" || lower == "txt" || lower == "plain" {
		return OutputFormatText
	}
	return OutputFormatJSON
}

// =============================================================================
// JSON formatter (default — for agents and pipes)
// =============================================================================

func formatJSON(data any) string {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}

// =============================================================================
// Safe property accessor for map[string]any
// =============================================================================

func getString(obj map[string]any, key string) (string, bool) {
	v, ok := obj[key]
	if !ok || v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

func getBool(obj map[string]any, key string) (bool, bool) {
	v, ok := obj[key]
	if !ok || v == nil {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

func getMap(obj map[string]any, key string) (map[string]any, bool) {
	v, ok := obj[key]
	if !ok || v == nil {
		return nil, false
	}
	m, ok := v.(map[string]any)
	return m, ok
}

func getSlice(obj map[string]any, key string) ([]any, bool) {
	v, ok := obj[key]
	if !ok || v == nil {
		return nil, false
	}
	s, ok := v.([]any)
	return s, ok
}

func getFloat(obj map[string]any, key string) (float64, bool) {
	v, ok := obj[key]
	if !ok || v == nil {
		return 0, false
	}
	f, ok := v.(float64)
	return f, ok
}

// =============================================================================
// Markdown formatter (human-readable)
// =============================================================================

func formatMarkdown(data any) string {
	obj, ok := toMap(data)
	if !ok {
		return formatJSON(data)
	}

	var lines []string

	if phase, ok := getString(obj, "phase"); ok {
		lines = append(lines, fmt.Sprintf("# tddmaster — %s", phase))
		lines = append(lines, "")
	}

	if instruction, ok := getString(obj, "instruction"); ok {
		lines = append(lines, "## Instruction")
		lines = append(lines, "")
		lines = append(lines, instruction)
		lines = append(lines, "")
	}

	// Discovery questions (batched array)
	if questions, ok := getSlice(obj, "questions"); ok && len(questions) > 0 {
		for _, qv := range questions {
			qMap, ok := qv.(map[string]any)
			if !ok {
				continue
			}
			qID, _ := getString(qMap, "id")
			qText, _ := getString(qMap, "text")
			lines = append(lines, fmt.Sprintf("## Question: %s", qID))
			lines = append(lines, "")
			lines = append(lines, fmt.Sprintf("> %s", qText))
			if extras, ok := getSlice(qMap, "extras"); ok && len(extras) > 0 {
				lines = append(lines, "")
				lines = append(lines, "Also consider:")
				for _, ev := range extras {
					if e, ok := ev.(string); ok {
						lines = append(lines, fmt.Sprintf("- %s", e))
					}
				}
			}
			lines = append(lines, "")
		}
	}

	if statusReport, ok := getMap(obj, "statusReport"); ok {
		lines = append(lines, "## Acceptance Criteria")
		lines = append(lines, "")
		if criteria, ok := getSlice(statusReport, "criteria"); ok {
			for _, cv := range criteria {
				if c, ok := cv.(string); ok {
					lines = append(lines, fmt.Sprintf("- [ ] %s", c))
				}
			}
		}
		lines = append(lines, "")
	}

	if debt, ok := getMap(obj, "previousIterationDebt"); ok {
		fromIter, _ := getFloat(debt, "fromIteration")
		note, _ := getString(debt, "note")
		lines = append(lines, fmt.Sprintf("## Debt (from iteration %d)", int(fromIter)))
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("> %s", note))
		lines = append(lines, "")
		if items, ok := getSlice(debt, "items"); ok {
			for _, iv := range items {
				if item, ok := iv.(string); ok {
					lines = append(lines, fmt.Sprintf("- %s", item))
				}
			}
		}
		lines = append(lines, "")
	}

	if failed, ok := getBool(obj, "verificationFailed"); ok && failed {
		lines = append(lines, "## Verification FAILED")
		lines = append(lines, "")
		lines = append(lines, "```")
		out, _ := getString(obj, "verificationOutput")
		lines = append(lines, out)
		lines = append(lines, "```")
		lines = append(lines, "")
	}

	if behavioral, ok := getMap(obj, "behavioral"); ok {
		lines = append(lines, "## Behavioral")
		lines = append(lines, "")
		tone, _ := getString(behavioral, "tone")
		lines = append(lines, fmt.Sprintf("**Tone:** %s", tone))
		lines = append(lines, "")
		if rules, ok := getSlice(behavioral, "rules"); ok {
			for _, rv := range rules {
				if r, ok := rv.(string); ok {
					lines = append(lines, fmt.Sprintf("- %s", r))
				}
			}
		}
		if urgency, ok := getString(behavioral, "urgency"); ok {
			lines = append(lines, "")
			lines = append(lines, fmt.Sprintf("**Urgency:** %s", urgency))
		}
		lines = append(lines, "")
	}

	if meta, ok := getMap(obj, "meta"); ok {
		resumeHint, _ := getString(meta, "resumeHint")
		lines = append(lines, "---")
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("*%s*", resumeHint))
		lines = append(lines, "")
	}

	if transition, ok := getMap(obj, "transition"); ok {
		lines = append(lines, "## Next Steps")
		lines = append(lines, "")
		for key, value := range transition {
			if key != "iteration" {
				lines = append(lines, fmt.Sprintf("- **%s:** `%v`", key, value))
			}
		}
		lines = append(lines, "")
	}

	if summary, ok := getMap(obj, "summary"); ok {
		spec, _ := getString(summary, "spec")
		iterations, _ := getFloat(summary, "iterations")
		decisionsCount, _ := getFloat(summary, "decisionsCount")
		lines = append(lines, "## Summary")
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("- Spec: %s", spec))
		lines = append(lines, fmt.Sprintf("- Iterations: %d", int(iterations)))
		lines = append(lines, fmt.Sprintf("- Decisions: %d", int(decisionsCount)))
		lines = append(lines, "")
	}

	if clearCtx, ok := getMap(obj, "clearContext"); ok {
		reason, _ := getString(clearCtx, "reason")
		lines = append(lines, "---")
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("**Action required:** %s", reason))
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// =============================================================================
// Text formatter (plain, no formatting)
// =============================================================================

func formatText(data any) string {
	obj, ok := toMap(data)
	if !ok {
		return fmt.Sprintf("%v", data)
	}

	var lines []string

	if phase, ok := getString(obj, "phase"); ok {
		lines = append(lines, fmt.Sprintf("[%s]", phase))
	}

	if instruction, ok := getString(obj, "instruction"); ok {
		lines = append(lines, instruction)
	}

	if questions, ok := getSlice(obj, "questions"); ok && len(questions) > 0 {
		for _, qv := range questions {
			qMap, ok := qv.(map[string]any)
			if !ok {
				continue
			}
			qID, _ := getString(qMap, "id")
			qText, _ := getString(qMap, "text")
			lines = append(lines, "")
			lines = append(lines, fmt.Sprintf("Question [%s]: %s", qID, qText))
			if extras, ok := getSlice(qMap, "extras"); ok {
				for _, ev := range extras {
					if e, ok := ev.(string); ok {
						lines = append(lines, fmt.Sprintf("  - %s", e))
					}
				}
			}
		}
	}

	if statusReport, ok := getMap(obj, "statusReport"); ok {
		lines = append(lines, "")
		lines = append(lines, "Criteria:")
		if criteria, ok := getSlice(statusReport, "criteria"); ok {
			for _, cv := range criteria {
				if c, ok := cv.(string); ok {
					lines = append(lines, fmt.Sprintf("  - %s", c))
				}
			}
		}
	}

	if debt, ok := getMap(obj, "previousIterationDebt"); ok {
		note, _ := getString(debt, "note")
		lines = append(lines, "")
		lines = append(lines, note)
		if items, ok := getSlice(debt, "items"); ok {
			for _, iv := range items {
				if item, ok := iv.(string); ok {
					lines = append(lines, fmt.Sprintf("  - %s", item))
				}
			}
		}
	}

	if failed, ok := getBool(obj, "verificationFailed"); ok && failed {
		out, _ := getString(obj, "verificationOutput")
		if len(out) > 200 {
			out = out[:200]
		}
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Verification failed: %s", out))
	}

	if meta, ok := getMap(obj, "meta"); ok {
		resumeHint, _ := getString(meta, "resumeHint")
		lines = append(lines, "")
		lines = append(lines, resumeHint)
	}

	if summary, ok := getMap(obj, "summary"); ok {
		spec, _ := getString(summary, "spec")
		iterations, _ := getFloat(summary, "iterations")
		decisionsCount, _ := getFloat(summary, "decisionsCount")
		lines = append(lines, fmt.Sprintf("Spec: %s, Iterations: %d, Decisions: %d",
			spec, int(iterations), int(decisionsCount)))
	}

	if clearCtx, ok := getMap(obj, "clearContext"); ok {
		reason, _ := getString(clearCtx, "reason")
		lines = append(lines, "")
		lines = append(lines, reason)
	}

	return strings.Join(lines, "\n")
}

// toMap tries to convert data to map[string]any.
// If data is already map[string]any, return it directly.
// If data is a struct or other type, marshal/unmarshal via JSON.
func toMap(data any) (map[string]any, bool) {
	if m, ok := data.(map[string]any); ok {
		return m, true
	}
	// Marshal and unmarshal to normalize struct types
	b, err := json.Marshal(data)
	if err != nil {
		return nil, false
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, false
	}
	return m, true
}

// =============================================================================
// Format dispatch
// =============================================================================

// Format serializes data to the given output format string.
func Format(data any, fmt OutputFormat) string {
	switch fmt {
	case OutputFormatMarkdown:
		return formatMarkdown(data)
	case OutputFormatText:
		return formatText(data)
	default:
		return formatJSON(data)
	}
}

// WriteFormatted writes formatted output to the given writer.
func WriteFormatted(w interface{ WriteString(string) (int, error) }, data any, fmt OutputFormat) error {
	text := Format(data, fmt)
	_, err := w.WriteString(text + "\n")
	return err
}
