// Package model holds the pure data shapes for the spec package:
// parsed spec documents, progress snapshots, and render options.
// No I/O, no logic — types only.
package model

// ParsedTask represents a single task extracted from a spec.md file.
type ParsedTask struct {
	ID     string
	Title  string
	Files  []string
	Covers []string
}

// ParsedSpec represents the structured content of a spec.md file.
type ParsedSpec struct {
	Name         string
	Tasks        []ParsedTask
	OutOfScope   []string
	Verification []string
}
