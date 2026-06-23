package ruleform

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pragmataW/tddmaster/internal/paths"
)

func TargetDir(root, target string) (string, error) {
	if target == "global" {
		return paths.Rules(root), nil
	}
	if isKnownAgent(target) {
		return paths.RulesAgentDir(root, target), nil
	}
	return "", fmt.Errorf("unknown target %q", target)
}

func WriteRule(root, target, rawName, content string) (string, error) {
	slug := Slugify(rawName)
	if slug == "" {
		return "", fmt.Errorf("name %q produces an empty slug", rawName)
	}

	name := EnsureMd(slug)

	dir, err := TargetDir(root, target)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %q: %w", dir, err)
	}

	full := filepath.Join(dir, name)

	if filepath.Dir(full) != dir {
		return "", fmt.Errorf("path escape detected: %q", full)
	}

	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write %q: %w", full, err)
	}

	return full, nil
}
