package paths

import "path/filepath"

const (
	DirSpecs      = "specs"
	FileState     = "state.json"
	FileSettings  = "settings.json"
	FileProgress  = "progress.json"
	FileSpec      = "spec.md"
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
