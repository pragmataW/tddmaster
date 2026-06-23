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

func resolveFullPath(root, target, rawName string) (dir, full string, err error) {
	slug := Slugify(rawName)
	if slug == "" {
		return "", "", fmt.Errorf("name %q produces an empty slug", rawName)
	}

	name := EnsureMd(slug)

	dir, err = TargetDir(root, target)
	if err != nil {
		return "", "", err
	}

	full = filepath.Join(dir, name)

	if filepath.Dir(full) != dir {
		return "", "", fmt.Errorf("path escape detected: %q", full)
	}

	return dir, full, nil
}

func WriteRule(root, target, rawName, content string) (string, error) {
	dir, full, err := resolveFullPath(root, target, rawName)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %q: %w", dir, err)
	}

	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write %q: %w", full, err)
	}

	return full, nil
}

func WriteRuleNoOverwrite(root, target, rawName, content string) (string, error) {
	dir, full, err := resolveFullPath(root, target, rawName)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %q: %w", dir, err)
	}

	f, err := os.OpenFile(full, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return "", fmt.Errorf("rule file already exists: %q", full)
		}
		return "", fmt.Errorf("create %q: %w", full, err)
	}
	if _, err := f.Write([]byte(content)); err != nil {
		f.Close()
		return "", fmt.Errorf("write %q: %w", full, err)
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("close %q: %w", full, err)
	}

	return full, nil
}
