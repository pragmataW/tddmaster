// Package specflag parses the --spec=<name> CLI flag used by all tddmaster
// commands that operate on a specific spec.
package specflag

import "strings"

// RequireSpecFlagResult is the result of RequireSpecFlag.
type RequireSpecFlagResult struct {
	OK    bool
	Spec  string
	Error string
}

// ParseSpecFlag parses --spec=<name> from args. Returns nil if not found.
func ParseSpecFlag(args []string) *string {
	for _, arg := range args {
		if strings.HasPrefix(arg, "--spec=") {
			s := arg[len("--spec="):]
			return &s
		}
	}
	return nil
}

// RequireSpecFlag returns the spec name from args, or an error if not found.
func RequireSpecFlag(args []string) RequireSpecFlagResult {
	spec := ParseSpecFlag(args)
	if spec == nil || len(*spec) == 0 {
		return RequireSpecFlagResult{
			OK:    false,
			Error: "Error: spec name is required. Use `tddmaster spec <name> <command>` format.",
		}
	}
	return RequireSpecFlagResult{OK: true, Spec: *spec}
}

// UsesOldSpecFlag returns true if any arg starts with --spec=.
func UsesOldSpecFlag(args []string) bool {
	for _, a := range args {
		if strings.HasPrefix(a, "--spec=") {
			return true
		}
	}
	return false
}
