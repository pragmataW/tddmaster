package spec

type Severity string

const (
	SeverityBlock Severity = "block"
	SeverityWarn  Severity = "warn"
	SeverityInfo  Severity = "info"
)

type FindingSource string

const (
	SourceLinter  FindingSource = "linter"
	SourceAuditor FindingSource = "auditor"
)

type Finding struct {
	Severity   Severity      `json:"severity"`
	Category   string        `json:"category"`
	TaskID     string        `json:"taskId,omitempty"`
	AcID       string        `json:"acId,omitempty"`
	Detail     string        `json:"detail"`
	Suggestion string        `json:"suggestion,omitempty"`
	Source     FindingSource `json:"source"`
}

func (f Finding) IsBlock() bool { return f.Severity == SeverityBlock }

// IsInfo reports whether the finding is purely advisory. Only info-severity
// findings let the cross-artifact phase pass through without user confirmation.
func (f Finding) IsInfo() bool { return f.Severity == SeverityInfo }

type Analysis struct {
	Verdict  string    `json:"verdict"`
	Findings []Finding `json:"findings"`
}
