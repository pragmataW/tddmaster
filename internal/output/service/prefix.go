// Package service implements the business logic for building CLI command
// strings. The Prefix type replaces a package-level mutable global; the
// top-level output package keeps a single process-wide instance but the
// behaviour is now testable and concurrency-friendly via a pointer receiver.
package service

import "github.com/pragmataW/tddmaster/internal/output/model"

// Prefix holds the current CLI prefix and builds full command strings.
type Prefix struct {
	value string
}

// NewPrefix constructs a Prefix initialised to model.DefaultPrefix.
func NewPrefix() *Prefix {
	return &Prefix{value: model.DefaultPrefix}
}

// Set overrides the prefix. An empty argument restores the default so
// consumers that read a missing config field do not end up with a leading
// space in every rendered command.
func (p *Prefix) Set(v string) {
	if v == "" {
		p.value = model.DefaultPrefix
		return
	}
	p.value = v
}

// Value returns the current prefix.
func (p *Prefix) Value() string {
	return p.value
}

// Build composes the prefix with a subcommand, joined by a single space.
func (p *Prefix) Build(subcommand string) string {
	return p.value + " " + subcommand
}
