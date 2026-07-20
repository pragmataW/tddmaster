package ruleform

import (
	"os"
	"path/filepath"

	"github.com/pragmataW/tddmaster/internal/errs"
	"github.com/pragmataW/tddmaster/internal/paths"
)

func TargetDir(root, target string) (string, error) {
	if target == "global" {
		return paths.Rules(root), nil
	}
	if isKnownAgent(target) {
		return paths.RulesAgentDir(root, target), nil
	}
	return "", errs.Newf(errs.KeyUnknownTarget, target)
}

func resolveFullPath(root, target, rawName string) (dir, full string, err error) {
	slug := Slugify(rawName)
	if slug == "" {
		return "", "", errs.Newf(errs.KeyEmptySlug, rawName)
	}

	name := EnsureMd(slug)

	dir, err = TargetDir(root, target)
	if err != nil {
		return "", "", err
	}

	full = filepath.Join(dir, name)

	if filepath.Dir(full) != dir {
		return "", "", errs.Newf(errs.KeyPathEscape, full)
	}

	return dir, full, nil
}

func WriteRule(root, target, rawName, content string) (string, error) {
	dir, full, err := resolveFullPath(root, target, rawName)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", errs.Wrap(errs.KeyMkdir, err, dir)
	}

	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		return "", errs.Wrap(errs.KeyWriteFile, err, full)
	}

	return full, nil
}

func WriteRuleNoOverwrite(root, target, rawName, content string) (string, error) {
	dir, full, err := resolveFullPath(root, target, rawName)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", errs.Wrap(errs.KeyMkdir, err, dir)
	}

	f, err := os.OpenFile(full, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return "", errs.Newf(errs.KeyRuleFileExists, full)
		}
		return "", errs.Wrap(errs.KeyCreateFile, err, full)
	}
	if _, err := f.Write([]byte(content)); err != nil {
		f.Close()
		return "", errs.Wrap(errs.KeyWriteFile, err, full)
	}
	if err := f.Close(); err != nil {
		return "", errs.Wrap(errs.KeyCloseFile, err, full)
	}

	return full, nil
}
