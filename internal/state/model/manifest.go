package model

// ProjectTraits captures detected language/framework/CI signals used to seed
// concern selection and runner defaults during `tddmaster init`.
type ProjectTraits struct {
	Languages  []string `json:"languages"  yaml:"languages"`
	Frameworks []string `json:"frameworks" yaml:"frameworks"`
	CI         []string `json:"ci"         yaml:"ci"`
	TestRunner *string  `json:"testRunner" yaml:"testRunner"`
}

// CodingToolId enumerates supported downstream coding assistants that receive
// synced prompt/configuration artifacts.
type CodingToolId string

const (
	CodingToolClaudeCode CodingToolId = "claude-code"
	CodingToolOpencode   CodingToolId = "opencode"
	CodingToolCodex      CodingToolId = "codex"
)

// UserConfig is the optional per-project user override persisted in manifest.yml.
type UserConfig struct {
	Name  string `json:"name"  yaml:"name"`
	Email string `json:"email" yaml:"email"`
}

// Manifest is the TDD-specific subschema of NosManifest. It lives at
// manifest.yml → tddmaster.tdd and controls verifier retries, refactor round
// caps, and whether TDD enforcement is globally enabled.
type Manifest struct {
	TddMode                bool    `json:"tddMode" yaml:"tddMode"`
	TestRunner             *string `json:"testRunner,omitempty" yaml:"testRunner"`
	MaxVerificationRetries int     `json:"maxVerificationRetries" yaml:"maxVerificationRetries"`
	MaxRefactorRounds      int     `json:"maxRefactorRounds,omitempty" yaml:"maxRefactorRounds,omitempty"`
	SkipVerify             bool    `json:"skipVerify,omitempty" yaml:"skipVerify,omitempty"`
}

// NosManifest is the root `tddmaster:` section persisted in manifest.yml. It
// combines project-level metadata, concern selection, downstream tool list,
// and the TDD subschema.
type NosManifest struct {
	Concerns                   []string       `json:"concerns"                   yaml:"concerns"`
	Tools                      []CodingToolId `json:"tools"                      yaml:"tools"`
	DefaultRunner              string         `json:"defaultRunner,omitempty"    yaml:"defaultRunner,omitempty"`
	Project                    ProjectTraits  `json:"project"                    yaml:"project"`
	MaxIterationsBeforeRestart int            `json:"maxIterationsBeforeRestart" yaml:"maxIterationsBeforeRestart"`
	Tdd                        *Manifest      `json:"tdd,omitempty"              yaml:"tdd,omitempty"`
	VerifyCommand              *string        `json:"verifyCommand"              yaml:"verifyCommand"`
	AllowGit                   bool           `json:"allowGit"                   yaml:"allowGit"`
	Command                    string         `json:"command"                    yaml:"command"`
	User                       *UserConfig    `json:"user,omitempty"             yaml:"user,omitempty"`
}

// IsTDDEnabled returns true when the TDD workflow is enabled in this manifest.
// Returns false when the Tdd field is nil or when TddMode is explicitly false.
func (m NosManifest) IsTDDEnabled() bool {
	return m.Tdd != nil && m.Tdd.TddMode
}

// IsVerifierSkipped returns true when the verifier sub-agent should be skipped.
// Returns false when Tdd is nil or when SkipVerify is not set.
func (m NosManifest) IsVerifierSkipped() bool {
	return m.Tdd != nil && m.Tdd.SkipVerify
}
