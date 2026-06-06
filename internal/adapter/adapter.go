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
}
