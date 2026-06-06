package spec

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/pragmataW/tddmaster/internal/paths"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

type Result struct {
	Slug          string   `json:"slug"`
	FilesWritten  []string `json:"filesWritten"`
	AlreadyExists bool     `json:"alreadyExists"`
}

func Start(root, slug string, now time.Time) (Result, error) {
	if !slugPattern.MatchString(slug) {
		return Result{}, fmt.Errorf("invalid slug %q: must match %s", slug, slugPattern.String())
	}

	if _, err := os.Stat(paths.Manifest(root)); err != nil {
		return Result{}, fmt.Errorf("manifest not found: run 'tddmaster init' first")
	}

	if Exists(root, slug) {
		return Result{Slug: slug, AlreadyExists: true}, nil
	}

	if err := os.MkdirAll(paths.SpecDir(root, slug), 0o755); err != nil {
		return Result{}, err
	}

	state := State{
		Version:   1,
		Slug:      slug,
		Phase:     PhaseInitial,
		Answers:   map[string][]Answer{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := SaveState(root, slug, state); err != nil {
		return Result{}, err
	}

	if err := SaveSettings(root, slug, DefaultSettings()); err != nil {
		return Result{}, err
	}

	progress := Progress{
		Spec:      slug,
		Status:    StatusDraft,
		Tasks:     []Task{},
		UpdatedAt: now,
	}
	if err := SaveProgress(root, slug, progress); err != nil {
		return Result{}, err
	}

	return Result{
		Slug: slug,
		FilesWritten: []string{
			paths.SpecState(root, slug),
			paths.SpecSettings(root, slug),
			paths.SpecProgress(root, slug),
		},
		AlreadyExists: false,
	}, nil
}
