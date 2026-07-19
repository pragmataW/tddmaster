package spec

import (
	"encoding/json"
	"testing"
)

func TestFinding_IsBlock(t *testing.T) {
	cases := []struct {
		name     string
		severity Severity
		want     bool
	}{
		{"BlockSeverity_True", SeverityBlock, true},
		{"WarnSeverity_False", SeverityWarn, false},
		{"InfoSeverity_False", SeverityInfo, false},
		{"EmptySeverity_False", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := Finding{Severity: tc.severity}
			got := f.IsBlock()
			if got != tc.want {
				t.Errorf("Finding{Severity:%q}.IsBlock() = %v, want %v", tc.severity, got, tc.want)
			}
		})
	}
}

func TestAnalysis_JSONRoundTrip(t *testing.T) {
	original := Analysis{
		Verdict: "issues",
		Findings: []Finding{
			{
				Severity:   "block",
				Category:   "untestable",
				TaskID:     "task-1",
				AcID:       "ac-1",
				Detail:     "Then clause is empty",
				Suggestion: "Provide a concrete observable outcome",
				Source:     "criterion_lint",
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	raw := string(data)
	if !containsKey(raw, `"verdict"`) {
		t.Errorf("marshaled JSON missing key \"verdict\": %s", raw)
	}
	if !containsKey(raw, `"findings"`) {
		t.Errorf("marshaled JSON missing key \"findings\": %s", raw)
	}

	var got Analysis
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}

	if got.Verdict != original.Verdict {
		t.Errorf("Verdict: got %q, want %q", got.Verdict, original.Verdict)
	}
	if len(got.Findings) != 1 {
		t.Fatalf("Findings length: got %d, want 1", len(got.Findings))
	}
	f := got.Findings[0]
	if f.Severity != "block" {
		t.Errorf("Findings[0].Severity: got %q, want %q", f.Severity, "block")
	}
	if f.Category != "untestable" {
		t.Errorf("Findings[0].Category: got %q, want %q", f.Category, "untestable")
	}
	if f.TaskID != "task-1" {
		t.Errorf("Findings[0].TaskID: got %q, want %q", f.TaskID, "task-1")
	}
	if f.AcID != "ac-1" {
		t.Errorf("Findings[0].AcID: got %q, want %q", f.AcID, "ac-1")
	}
	if f.Detail != "Then clause is empty" {
		t.Errorf("Findings[0].Detail: got %q, want %q", f.Detail, "Then clause is empty")
	}
	if f.Source != "criterion_lint" {
		t.Errorf("Findings[0].Source: got %q, want %q", f.Source, "criterion_lint")
	}
}

func TestAnalysis_EmptyFindings_JSON(t *testing.T) {
	original := Analysis{
		Verdict:  "clean",
		Findings: nil,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	var got Analysis
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}

	if got.Verdict != "clean" {
		t.Errorf("Verdict: got %q, want %q", got.Verdict, "clean")
	}

	_ = got.Findings
}

func containsKey(s, key string) bool {
	return len(s) > 0 && len(key) > 0 && func() bool {
		for i := 0; i <= len(s)-len(key); i++ {
			if s[i:i+len(key)] == key {
				return true
			}
		}
		return false
	}()
}
