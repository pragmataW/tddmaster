
package output_test

import (
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/output"
)

// =============================================================================
// ParseOutputFormat
// =============================================================================

func TestParseOutputFormat_DefaultJSON(t *testing.T) {
	if got := output.ParseOutputFormat(nil); got != output.OutputFormatJSON {
		t.Errorf("ParseOutputFormat(nil) = %q, want %q", got, output.OutputFormatJSON)
	}
	if got := output.ParseOutputFormat([]string{"spec", "list"}); got != output.OutputFormatJSON {
		t.Errorf("ParseOutputFormat no flag = %q, want %q", got, output.OutputFormatJSON)
	}
}

func TestParseOutputFormat_ShortFlag(t *testing.T) {
	cases := []struct {
		args []string
		want output.OutputFormat
	}{
		{[]string{"-o", "json"}, output.OutputFormatJSON},
		{[]string{"-o", "markdown"}, output.OutputFormatMarkdown},
		{[]string{"-o", "md"}, output.OutputFormatMarkdown},
		{[]string{"-o", "text"}, output.OutputFormatText},
		{[]string{"-o", "txt"}, output.OutputFormatText},
		{[]string{"-o", "plain"}, output.OutputFormatText},
	}
	for _, tc := range cases {
		got := output.ParseOutputFormat(tc.args)
		if got != tc.want {
			t.Errorf("ParseOutputFormat(%v) = %q, want %q", tc.args, got, tc.want)
		}
	}
}

func TestParseOutputFormat_LongFlagEquals(t *testing.T) {
	cases := []struct {
		args []string
		want output.OutputFormat
	}{
		{[]string{"--output=json"}, output.OutputFormatJSON},
		{[]string{"--output=markdown"}, output.OutputFormatMarkdown},
		{[]string{"--output=text"}, output.OutputFormatText},
	}
	for _, tc := range cases {
		got := output.ParseOutputFormat(tc.args)
		if got != tc.want {
			t.Errorf("ParseOutputFormat(%v) = %q, want %q", tc.args, got, tc.want)
		}
	}
}

// =============================================================================
// StripOutputFlag
// =============================================================================

func TestStripOutputFlag_RemovesShortFlag(t *testing.T) {
	args := []string{"spec", "-o", "markdown", "list"}
	got := output.StripOutputFlag(args)
	want := []string{"spec", "list"}
	if !equal(got, want) {
		t.Errorf("StripOutputFlag = %v, want %v", got, want)
	}
}

func TestStripOutputFlag_RemovesLongFlagEquals(t *testing.T) {
	args := []string{"spec", "--output=json", "list"}
	got := output.StripOutputFlag(args)
	want := []string{"spec", "list"}
	if !equal(got, want) {
		t.Errorf("StripOutputFlag = %v, want %v", got, want)
	}
}

func TestStripOutputFlag_Nil(t *testing.T) {
	got := output.StripOutputFlag(nil)
	if len(got) != 0 {
		t.Errorf("StripOutputFlag(nil) = %v, want empty", got)
	}
}

// =============================================================================
// Format — JSON
// =============================================================================

func TestFormat_JSON(t *testing.T) {
	data := map[string]any{"phase": "EXECUTING", "iteration": 3}
	got := output.Format(data, output.OutputFormatJSON)
	if !strings.Contains(got, `"phase"`) {
		t.Errorf("JSON output missing 'phase' key: %s", got)
	}
	if !strings.Contains(got, `"EXECUTING"`) {
		t.Errorf("JSON output missing 'EXECUTING' value: %s", got)
	}
}

// =============================================================================
// Format — Markdown
// =============================================================================

func TestFormat_Markdown_Phase(t *testing.T) {
	data := map[string]any{"phase": "EXECUTING"}
	got := output.Format(data, output.OutputFormatMarkdown)
	if !strings.Contains(got, "# tddmaster — EXECUTING") {
		t.Errorf("Markdown output missing phase header: %s", got)
	}
}

func TestFormat_Markdown_Instruction(t *testing.T) {
	data := map[string]any{"instruction": "Do the thing."}
	got := output.Format(data, output.OutputFormatMarkdown)
	if !strings.Contains(got, "## Instruction") {
		t.Errorf("Markdown missing Instruction header: %s", got)
	}
	if !strings.Contains(got, "Do the thing.") {
		t.Errorf("Markdown missing instruction text: %s", got)
	}
}

func TestFormat_Markdown_VerificationFailed(t *testing.T) {
	data := map[string]any{
		"verificationFailed":  true,
		"verificationOutput":  "tests failed",
	}
	got := output.Format(data, output.OutputFormatMarkdown)
	if !strings.Contains(got, "## Verification FAILED") {
		t.Errorf("Markdown missing verification failed header: %s", got)
	}
}

func TestFormat_Markdown_Questions(t *testing.T) {
	data := map[string]any{
		"questions": []any{
			map[string]any{
				"id":   "q1",
				"text": "What is the approach?",
				"extras": []any{"Consider performance", "Consider maintainability"},
			},
		},
	}
	got := output.Format(data, output.OutputFormatMarkdown)
	if !strings.Contains(got, "## Question: q1") {
		t.Errorf("Markdown missing question header: %s", got)
	}
	if !strings.Contains(got, "Also consider:") {
		t.Errorf("Markdown missing extras: %s", got)
	}
}

// =============================================================================
// Format — Text
// =============================================================================

func TestFormat_Text_Phase(t *testing.T) {
	data := map[string]any{"phase": "IDLE"}
	got := output.Format(data, output.OutputFormatText)
	if !strings.Contains(got, "[IDLE]") {
		t.Errorf("Text output missing [IDLE]: %s", got)
	}
}

func TestFormat_Text_VerificationFailed_Truncates(t *testing.T) {
	longOutput := strings.Repeat("x", 300)
	data := map[string]any{
		"verificationFailed": true,
		"verificationOutput": longOutput,
	}
	got := output.Format(data, output.OutputFormatText)
	if !strings.Contains(got, "Verification failed:") {
		t.Errorf("Text missing verification failed: %s", got)
	}
	// Should be truncated to 200 chars of output
	if strings.Contains(got, longOutput) {
		t.Errorf("Text did not truncate verification output")
	}
}

func TestFormat_Text_Summary(t *testing.T) {
	data := map[string]any{
		"summary": map[string]any{
			"spec":           "my-spec",
			"iterations":     float64(5),
			"decisionsCount": float64(3),
		},
	}
	got := output.Format(data, output.OutputFormatText)
	if !strings.Contains(got, "Spec: my-spec") {
		t.Errorf("Text missing summary spec: %s", got)
	}
	if !strings.Contains(got, "Iterations: 5") {
		t.Errorf("Text missing iterations: %s", got)
	}
}

// =============================================================================
// Helpers
// =============================================================================

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
