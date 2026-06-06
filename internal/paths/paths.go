package paths

import "path/filepath"

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
