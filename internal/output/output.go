// Package output provides the CLI command-string construction helpers used
// across the cmd/ tree. The public functions delegate to a single
// process-wide service.Prefix; callers should invoke SetCommandPrefix once
// at startup (see cmd/next.go) with the prefix persisted in the manifest.
package output

import "github.com/pragmataW/tddmaster/internal/output/service"

var active = service.NewPrefix()

// SetCommandPrefix overrides the CLI prefix (e.g. read from the manifest
// during init). An empty string restores the default.
func SetCommandPrefix(prefix string) {
	active.Set(prefix)
}

// CmdPrefix returns the current CLI prefix.
func CmdPrefix() string {
	return active.Value()
}

// Cmd builds a full command string: "<prefix> <subcommand>".
func Cmd(subcommand string) string {
	return active.Build(subcommand)
}
