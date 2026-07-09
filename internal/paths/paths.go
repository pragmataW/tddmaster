package paths

import "path/filepath"

const (
	DirSpecs         = "specs"
	DirRules         = "rules"
	FileState        = "state.json"
	FileSettings     = "settings.json"
	FileProgress     = "progress.json"
	FileSpec         = "spec.md"
	FileTraceability = "traceability.json"
	FileAnalysis     = "analysis.json"
)

func Tddmaster(root string) string {
	return filepath.Join(root, ".tddmaster")
}

func Manifest(root string) string {
	return filepath.Join(Tddmaster(root), "manifest.json")
}

func ClaudeAgents(root string) string {
	return filepath.Join(root, ".claude", "agents")
}

func ClaudeMd(root string) string {
	return filepath.Join(root, "CLAUDE.md")
}

func CursorAgents(root string) string {
	return filepath.Join(root, ".cursor", "agents")
}

func CodexAgents(root string) string {
	return filepath.Join(root, ".codex", "agents")
}

func OpenCodeAgents(root string) string {
	return filepath.Join(root, ".opencode", "agents")
}

func AgentsMd(root string) string {
	return filepath.Join(root, "AGENTS.md")
}

func Specs(root string) string {
	return filepath.Join(Tddmaster(root), DirSpecs)
}

func SpecDir(root, slug string) string {
	return filepath.Join(Specs(root), slug)
}

func SpecState(root, slug string) string {
	return filepath.Join(SpecDir(root, slug), FileState)
}

func SpecSettings(root, slug string) string {
	return filepath.Join(SpecDir(root, slug), FileSettings)
}

func SpecProgress(root, slug string) string {
	return filepath.Join(SpecDir(root, slug), FileProgress)
}

func SpecMd(root, slug string) string {
	return filepath.Join(SpecDir(root, slug), FileSpec)
}

func SpecTraceability(root, slug string) string {
	return filepath.Join(SpecDir(root, slug), FileTraceability)
}

func SpecAnalysis(root, slug string) string {
	return filepath.Join(SpecDir(root, slug), FileAnalysis)
}

func Rules(root string) string {
	return filepath.Join(Tddmaster(root), DirRules)
}

func RulesAgentDir(root, agent string) string {
	return filepath.Join(Rules(root), agent)
}
