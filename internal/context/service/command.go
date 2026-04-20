// Package service hosts the business logic that turns a StateFile into a
// NextOutput. It reads `internal/context/model` types and does not export
// additional data shapes itself.
package service

// DefaultCommandPrefix is the canonical CLI binary name.
const DefaultCommandPrefix = "tddmaster"

// Renderer is the command-string helper. Previously this was a package-level
// global (`commandPrefix`) in compiler.go; carrying it as a receiver avoids the
// global mutable state and lets multiple CLI binaries co-exist in one process.
type Renderer struct {
	Prefix string
}

// NewRenderer constructs a Renderer. Empty prefix falls back to DefaultCommandPrefix.
func NewRenderer(prefix string) Renderer {
	if prefix == "" {
		prefix = DefaultCommandPrefix
	}
	return Renderer{Prefix: prefix}
}

// C builds a full command: prefix + subcommand.
func (r Renderer) C(sub string) string {
	return r.Prefix + " " + sub
}

// CS builds a spec-scoped command: prefix + "spec <name> <sub>". When specName
// is nil, falls back to the unscoped command.
func (r Renderer) CS(sub string, specName *string) string {
	if specName == nil {
		return r.C(sub)
	}
	return r.C("spec " + *specName + " " + sub)
}
