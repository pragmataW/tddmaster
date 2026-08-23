package loop

import (
	"os"
	"path/filepath"
)

// gitWorkingTreeRoot returns the nearest ancestor that owns the git metadata
// for root. A project may deliberately use a subdirectory of a repository as
// its tddmaster root, so callers need both the availability bit and the actual
// checkout root in order to preserve that project-relative location.
func gitWorkingTreeRoot(root string) (string, bool) {
	if root == "" {
		return "", false
	}
	dir, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// worktreesAvailable reports whether the project root is inside a git working
// tree. Without git there is nothing to isolate tasks with, so the engine must
// not hand sub-agents a worktree cwd that will never exist.
func worktreesAvailable(root string) bool {
	_, ok := gitWorkingTreeRoot(root)
	return ok
}
