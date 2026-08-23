package adapter

import "github.com/pragmataW/tddmaster/internal/manifest"

type SyncContext struct {
	Root          string
	Manifest      *manifest.Manifest
	CommandPrefix string
}

type ToolAdapter interface {
	ID() manifest.ToolID
	Sync(SyncContext) error
	// Files lists the paths Sync writes, relative to the project root, so the
	// init summary can report what actually landed on disk.
	Files() []string
}
