
// Package output provides CLI output formatting and command prefix utilities.
package output

// =============================================================================
// Command Prefix
//
// Port of tddmaster/output/cmd.ts — adapted for Go CLI.
// =============================================================================

const defaultCmd = "tddmaster"

var _prefix string = defaultCmd

// SetCommandPrefix sets the CLI prefix (e.g. read from manifest during init).
func SetCommandPrefix(prefix string) {
	_prefix = prefix
}

// CmdPrefix returns the current CLI prefix.
func CmdPrefix() string {
	return _prefix
}

// Cmd builds a full command string: prefix + subcommand.
func Cmd(subcommand string) string {
	return _prefix + " " + subcommand
}
