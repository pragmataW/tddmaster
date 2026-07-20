package scaffold

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pragmataW/tddmaster/internal/adapter"
	"github.com/pragmataW/tddmaster/internal/errs"
	"github.com/pragmataW/tddmaster/internal/manifest"
	"github.com/pragmataW/tddmaster/internal/paths"
)

type Options struct {
	Root           string
	NonInteractive bool
	Manifest       *manifest.Manifest
}

type Result struct {
	FilesWritten []string
	FilesTouched []string
	Adapters     []manifest.ToolID
	Warnings     []string
}

func LoadManifestOrDefaults(root string) manifest.Manifest {
	data, err := os.ReadFile(paths.Manifest(root))
	if err != nil {
		return manifest.Defaults()
	}
	var m manifest.Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return manifest.Defaults()
	}
	return m
}

func writeManifest(root string, m manifest.Manifest) (string, error) {
	p := paths.Manifest(root)
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", errs.Wrap(errs.KeyMarshalManifest, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(p, data, 0o644); err != nil {
		return "", errs.Wrap(errs.KeyWriteManifest, err)
	}
	return p, nil
}

func Scaffold(opts Options) (Result, error) {
	var m manifest.Manifest
	if opts.Manifest != nil {
		cp := *opts.Manifest
		m = cp
	} else {
		m = LoadManifestOrDefaults(opts.Root)
	}

	if len(m.SelectedTools) == 0 {
		return Result{}, errs.New(errs.KeyToolRequired)
	}

	manifest.Normalize(&m)

	if err := os.MkdirAll(paths.Tddmaster(opts.Root), 0o755); err != nil {
		return Result{}, errs.Wrap(errs.KeyCreateTddmasterDir, err)
	}

	manifestPath, err := writeManifest(opts.Root, m)
	if err != nil {
		return Result{}, err
	}

	var result Result
	result.FilesWritten = append(result.FilesWritten, manifestPath)

	for _, id := range m.SelectedTools {
		a, ok := adapter.Get(id)
		if !ok {
			result.Warnings = append(result.Warnings, fmt.Sprintf("tool %s: unknown — no adapter registered", id))
			continue
		}
		if err := a.Sync(adapter.SyncContext{Root: opts.Root, Manifest: &m, CommandPrefix: m.Command}); err != nil {
			return Result{}, errs.Wrap(errs.KeyAdapter, err, id)
		}
		result.Adapters = append(result.Adapters, id)
	}

	return result, nil
}
