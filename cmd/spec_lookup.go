package cmd

import (
	"os"

	"github.com/pragmataW/tddmaster/internal/errs"
	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/spec"
)

// requireActiveSpec distinguishes a truly unknown slug from one that exists in
// the archive. Recommending `start` for an archived slug is a dead end because
// start correctly refuses to create a duplicate and tells the user to restore.
func requireActiveSpec(root, slug string) error {
	if spec.Exists(root, slug) {
		return nil
	}
	if _, err := os.Stat(paths.ArchiveSpecDir(root, slug)); err == nil {
		return errs.Newf(errs.KeySpecInArchive, slug, slug)
	}
	return errs.Newf(errs.KeySpecNotFoundRunStart, slug, slug)
}
