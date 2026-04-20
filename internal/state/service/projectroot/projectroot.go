// Package projectroot discovers the tddmaster project root directory and
// scaffolds its on-disk structure.
package projectroot

import (
	"os"
	"path/filepath"

	"github.com/pragmataW/tddmaster/internal/state/service/paths"
)

const maxProjectRootDepth = 100

// ResolveProjectRootResult holds the result of ResolveProjectRoot.
type ResolveProjectRootResult struct {
	Root  string
	Found bool
}

// FindProjectRoot walks up the directory tree to find the nearest directory
// containing .tddmaster/. Bounded depth guards against pathological symlink
// loops and deeply nested trees.
func FindProjectRoot(startDir string) (string, error) {
	dir := startDir
	for depth := 0; depth < maxProjectRootDepth; depth++ {
		if _, err := os.Stat(filepath.Join(dir, paths.TddmasterDir)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
	return "", nil
}

// ResolveProjectRoot resolves the tddmaster project root with priority:
//  1. TDDMASTER_PROJECT_ROOT env var
//  2. Walk up from cwd to find .tddmaster/
//  3. Fall back to cwd
func ResolveProjectRoot() (ResolveProjectRootResult, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return ResolveProjectRootResult{}, err
	}

	envRoot := os.Getenv("TDDMASTER_PROJECT_ROOT")
	if envRoot != "" {
		if _, err := os.Stat(filepath.Join(envRoot, paths.TddmasterDir)); err == nil {
			return ResolveProjectRootResult{Root: envRoot, Found: true}, nil
		}
	}

	found, err := FindProjectRoot(cwd)
	if err != nil {
		return ResolveProjectRootResult{}, err
	}
	if found != "" {
		return ResolveProjectRootResult{Root: found, Found: true}, nil
	}

	if envRoot != "" {
		return ResolveProjectRootResult{Root: envRoot, Found: false}, nil
	}

	return ResolveProjectRootResult{Root: cwd, Found: false}, nil
}

// ScaffoldDir creates the full .tddmaster directory structure.
func ScaffoldDir(root string) error {
	dirs := []string{
		paths.TddmasterDir,
		paths.StateDir,
		paths.SpecStatesDir,
		paths.ConcernsDir,
		paths.RulesDir,
		paths.SpecsDir,
		paths.WorkflowsDir,
		paths.EventsDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			return err
		}
	}

	gitignorePath := filepath.Join(root, paths.Paths{}.TddmasterGitignore())
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		content := "# tddmaster toolchain runtime state — not tracked by git\n.state/\n.sessions/\n.events/\n"
		if err := os.WriteFile(gitignorePath, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}
