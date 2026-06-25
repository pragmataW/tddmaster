package spec

type Finding struct {
	Severity   string `json:"severity"`
	Category   string `json:"category"`
	TaskID     string `json:"taskId,omitempty"`
	AcID       string `json:"acId,omitempty"`
	Detail     string `json:"detail"`
	Suggestion string `json:"suggestion,omitempty"`
	Source     string `json:"source"`
}

func (f Finding) IsBlock() bool { return f.Severity == "block" }

// IsInfo reports whether the finding is purely advisory. Only info-severity
// findings let the cross-artifact phase pass through without user confirmation.
func (f Finding) IsInfo() bool { return f.Severity == "info" }

type Analysis struct {
	Verdict  string    `json:"verdict"`
	Findings []Finding `json:"findings"`
}
