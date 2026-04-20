// Package model holds the marker-file signal tables and canonical
// identifiers used by the detect service. These values are pure data —
// no I/O, no logic — so they can be read safely from multiple
// goroutines and from tests.
package model

import "github.com/pragmataW/tddmaster/internal/state"

// Marker file names.
const (
	FilePackageJSON = "package.json"
	FileDenoJSON    = "deno.json"
	FileGoMod       = "go.mod"
	FileCargoToml   = "Cargo.toml"
	FilePyProject   = "pyproject.toml"
	FileSetupPy     = "setup.py"
)

// npm/package.json field names.
const (
	PkgFieldDeps    = "dependencies"
	PkgFieldDevDeps = "devDependencies"
)

// Canonical language identifiers emitted into ProjectTraits.Languages.
const (
	LangTypeScript = "typescript"
	LangGo         = "go"
	LangRust       = "rust"
	LangPython     = "python"
)

// Canonical CI-provider identifiers emitted into ProjectTraits.CI.
const (
	CIGithubActions = "github-actions"
	CIGitlab        = "gitlab-ci"
	CIJenkins       = "jenkins"
	CICircleCI      = "circleci"
)

// TestRunnerDeno is the canonical test-runner name emitted when a
// deno.json marker is present.
const TestRunnerDeno = "deno"

// LanguageSignal maps a marker file path (relative to repo root) to a
// language identifier. Entries are evaluated in order and duplicates
// are de-duplicated by language name.
type LanguageSignal struct {
	File     string
	Language string
}

var LanguageSignals = []LanguageSignal{
	{File: FilePackageJSON, Language: LangTypeScript},
	{File: FileDenoJSON, Language: LangTypeScript},
	{File: FileGoMod, Language: LangGo},
	{File: FileCargoToml, Language: LangRust},
	{File: FilePyProject, Language: LangPython},
	{File: FileSetupPy, Language: LangPython},
}

// FrameworkSignal maps an npm dependency key to its canonical framework
// identifier emitted into ProjectTraits.Frameworks.
type FrameworkSignal struct {
	Dep  string
	Name string
}

var FrameworkSignals = []FrameworkSignal{
	{Dep: "react", Name: "react"},
	{Dep: "vue", Name: "vue"},
	{Dep: "svelte", Name: "svelte"},
	{Dep: "next", Name: "nextjs"},
	{Dep: "express", Name: "express"},
	{Dep: "hono", Name: "hono"},
}

// CISignal maps a marker path to a CI-provider identifier.
type CISignal struct {
	Path string
	Name string
}

var CISignals = []CISignal{
	{Path: ".github/workflows", Name: CIGithubActions},
	{Path: ".gitlab-ci.yml", Name: CIGitlab},
	{Path: "Jenkinsfile", Name: CIJenkins},
	{Path: ".circleci", Name: CICircleCI},
}

// TestRunnerSignals lists npm devDependency keys checked for JS/TS test
// runners, in priority order — the first match wins.
var TestRunnerSignals = []string{
	"vitest",
	"jest",
	"playwright",
}

// ToolSignal maps a coding-tool identifier to a list of marker paths
// (any match enables the tool).
type ToolSignal struct {
	ID    state.CodingToolId
	Paths []string
}

var ToolSignals = []ToolSignal{
	{ID: state.CodingToolClaudeCode, Paths: []string{"CLAUDE.md", ".claude"}},
	{ID: state.CodingToolCodex, Paths: []string{".codex", ".codex/config.toml"}},
	{ID: state.CodingToolOpencode, Paths: []string{".opencode", "opencode.json"}},
}
