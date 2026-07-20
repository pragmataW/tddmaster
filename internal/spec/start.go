package spec

import (
	"os"
	"regexp"
	"time"

	"github.com/pragmataW/tddmaster/internal/errs"
	"github.com/pragmataW/tddmaster/internal/paths"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

func ValidSlug(slug string) bool {
	return slugPattern.MatchString(slug)
}

type Result struct {
	Slug          string   `json:"slug"`
	FilesWritten  []string `json:"filesWritten"`
	AlreadyExists bool     `json:"alreadyExists"`
}

func Start(root, slug string, now time.Time) (Result, error) {
	if !slugPattern.MatchString(slug) {
		return Result{}, errs.Newf(errs.KeyInvalidSlugMustMatch, slug, slugPattern.String())
	}

	if _, err := os.Stat(paths.Manifest(root)); err != nil {
		return Result{}, errs.New(errs.KeyManifestNotFound)
	}

	if Exists(root, slug) {
		return Result{Slug: slug, AlreadyExists: true}, nil
	}

	if _, err := os.Stat(paths.ArchiveSpecDir(root, slug)); err == nil {
		return Result{}, errs.Newf(errs.KeySpecInArchive, slug, slug)
	}

	dir := paths.SpecDir(root, slug)
	_, statErr := os.Stat(dir)
	dirExisted := statErr == nil
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Result{}, err
	}

	if err := writeInitialFiles(root, slug, now); err != nil {
		cleanupStart(root, slug, dirExisted)
		return Result{}, err
	}

	return Result{
		Slug: slug,
		FilesWritten: []string{
			paths.SpecState(root, slug),
			paths.SpecSettings(root, slug),
			paths.SpecProgress(root, slug),
			paths.SpecTraceability(root, slug),
			paths.SpecAnalysis(root, slug),
		},
		AlreadyExists: false,
	}, nil
}

func writeInitialFiles(root, slug string, now time.Time) error {
	state := State{
		Version:   1,
		Slug:      slug,
		Phase:     PhaseInitial,
		Answers:   map[string][]Answer{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := SaveState(root, slug, state); err != nil {
		return err
	}

	if err := SaveSettings(root, slug, DefaultSettings()); err != nil {
		return err
	}

	progress := Progress{
		Spec:      slug,
		Status:    StatusDraft,
		Tasks:     []Task{},
		UpdatedAt: now,
	}
	if err := SaveProgress(root, slug, progress); err != nil {
		return err
	}

	if err := SaveTraceability(root, slug, Traceability{}); err != nil {
		return err
	}

	return SaveAnalysis(root, slug, Analysis{Verdict: "", Findings: []Finding{}})
}

func cleanupStart(root, slug string, dirExisted bool) {
	os.Remove(paths.SpecAnalysis(root, slug))
	os.Remove(paths.SpecTraceability(root, slug))
	os.Remove(paths.SpecProgress(root, slug))
	os.Remove(paths.SpecSettings(root, slug))
	os.Remove(paths.SpecState(root, slug))
	if !dirExisted {
		os.Remove(paths.SpecDir(root, slug))
	}
}
