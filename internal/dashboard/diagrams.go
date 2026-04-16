
// Diagram registry — tracks diagrams in the project, detects staleness
// when referenced files change during spec execution.

package dashboard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pragmataW/tddmaster/internal/state"
)

// =============================================================================
// Types
// =============================================================================

// DiagramType represents the type of a diagram.
type DiagramType string

const (
	DiagramTypeMermaid  DiagramType = "mermaid"
	DiagramTypeAscii    DiagramType = "ascii"
	DiagramTypeSVG      DiagramType = "svg"
	DiagramTypePlantuml DiagramType = "plantuml"
)

// DiagramEntry is a diagram tracked in the registry.
type DiagramEntry struct {
	File            string      `json:"file"`
	Line            int         `json:"line"`
	Type            DiagramType `json:"type"`
	Hash            string      `json:"hash"`
	ReferencedFiles []string    `json:"referencedFiles"`
	LastVerified    string      `json:"lastVerified"`
}

// StaleDiagram represents a stale diagram detected during staleness check.
type StaleDiagram struct {
	File   string      `json:"file"`
	Line   int         `json:"line"`
	Type   DiagramType `json:"type"`
	Reason string      `json:"reason"`
}

// =============================================================================
// Paths
// =============================================================================

const diagramsFile = state.TddmasterDir + "/diagrams.json"

// =============================================================================
// Hashing
// =============================================================================

// hashContent computes a simple hash for content comparison (mirrors TS implementation).
func hashContent(content string) string {
	hash := int32(0)
	for _, c := range content {
		chr := int32(c)
		hash = ((hash << 5) - hash) + chr
	}
	if hash < 0 {
		hash = -hash
	}
	return strconv.FormatInt(int64(hash), 36)
}

// =============================================================================
// Reference extraction
// =============================================================================

var filePatternRe = regexp.MustCompile(`(?:[\w@.\-]+/)*[\w@.\-]+\.(?:ts|tsx|js|jsx|go|py|rs|md|json|yaml|yml)`)
var modulePatternRe = regexp.MustCompile(`(?:@[\w\-]+/[\w\-]+|[\w\-]+/[\w\-]+)`)

// extractReferences extracts file references from diagram content.
func extractReferences(content string) []string {
	refs := make(map[string]bool)

	for _, m := range filePatternRe.FindAllString(content, -1) {
		refs[m] = true
	}

	for _, m := range modulePatternRe.FindAllString(content, -1) {
		if !strings.Contains(m, ".") {
			refs[m] = true
		}
	}

	result := make([]string, 0, len(refs))
	for k := range refs {
		result = append(result, k)
	}
	return result
}

// =============================================================================
// Scanning
// =============================================================================

var asciiBoxRe = regexp.MustCompile(`[┌─┬┐│└┴┘╔═╗║╚╝╭╮╰╯]`)
var asciiLineRe = regexp.MustCompile(`[┌─┬┐│└┴┘╔═╗║╚╝╭╮╰╯┼├┤▶→←↓↑|+\-]`)

// scanMarkdownFile scans a markdown file for diagram blocks.
func scanMarkdownFile(root, relPath string) ([]DiagramEntry, error) {
	absPath := filepath.Join(root, relPath)
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, nil
	}

	var entries []DiagramEntry
	lines := strings.Split(string(data), "\n")
	now := time.Now().UTC().Format(time.RFC3339)

	inMermaid := false
	mermaidStart := 0
	var mermaidContent strings.Builder

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```mermaid") {
			inMermaid = true
			mermaidStart = i + 1
			mermaidContent.Reset()
			continue
		}
		if inMermaid && trimmed == "```" {
			inMermaid = false
			content := mermaidContent.String()
			entries = append(entries, DiagramEntry{
				File:            relPath,
				Line:            mermaidStart,
				Type:            DiagramTypeMermaid,
				Hash:            hashContent(content),
				ReferencedFiles: extractReferences(content),
				LastVerified:    now,
			})
			continue
		}
		if inMermaid {
			mermaidContent.WriteString(line)
			mermaidContent.WriteByte('\n')
			continue
		}

		// ASCII diagrams (lines with box-drawing characters)
		if asciiBoxRe.MatchString(line) && i+1 < len(lines) && asciiBoxRe.MatchString(lines[i+1]) {
			var asciiContent strings.Builder
			j := i
			for j < len(lines) && asciiLineRe.MatchString(lines[j]) {
				asciiContent.WriteString(lines[j])
				asciiContent.WriteByte('\n')
				j++
			}
			if asciiContent.Len() > 20 {
				content := asciiContent.String()
				entries = append(entries, DiagramEntry{
					File:            relPath,
					Line:            i + 1,
					Type:            DiagramTypeAscii,
					Hash:            hashContent(content),
					ReferencedFiles: extractReferences(content),
					LastVerified:    now,
				})
			}
		}
	}

	return entries, nil
}

// ScanProject scans the project for all diagrams.
func ScanProject(root string) ([]DiagramEntry, error) {
	var entries []DiagramEntry
	var mdFiles []string

	var scanDir func(dir, prefix string) error
	scanDir = func(dir, prefix string) error {
		dirEntries, err := os.ReadDir(dir)
		if err != nil {
			return nil
		}
		for _, entry := range dirEntries {
			name := entry.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" {
				continue
			}
			relName := name
			if prefix != "" {
				relName = prefix + "/" + name
			}
			if !entry.IsDir() && strings.HasSuffix(name, ".md") {
				mdFiles = append(mdFiles, relName)
			}
			if entry.IsDir() && !strings.HasPrefix(name, ".") {
				_ = scanDir(filepath.Join(dir, name), relName)
			}
		}
		return nil
	}

	_ = scanDir(root, "")

	for _, mdFile := range mdFiles {
		diagrams, _ := scanMarkdownFile(root, mdFile)
		entries = append(entries, diagrams...)
	}

	// Check for .puml files in root
	rootEntries, err := os.ReadDir(root)
	if err == nil {
		for _, entry := range rootEntries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".puml") {
				data, err := os.ReadFile(filepath.Join(root, entry.Name()))
				if err == nil {
					content := string(data)
					entries = append(entries, DiagramEntry{
						File:            entry.Name(),
						Line:            1,
						Type:            DiagramTypePlantuml,
						Hash:            hashContent(content),
						ReferencedFiles: extractReferences(content),
						LastVerified:    time.Now().UTC().Format(time.RFC3339),
					})
				}
			}
		}
	}

	if entries == nil {
		return []DiagramEntry{}, nil
	}
	return entries, nil
}

// =============================================================================
// Registry
// =============================================================================

// ReadRegistry reads the diagram registry.
func ReadRegistry(root string) ([]DiagramEntry, error) {
	filePath := filepath.Join(root, diagramsFile)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return []DiagramEntry{}, nil
	}

	var entries []DiagramEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return []DiagramEntry{}, nil
	}
	return entries, nil
}

// WriteRegistry writes the diagram registry.
func WriteRegistry(root string, entries []DiagramEntry) error {
	filePath := filepath.Join(root, diagramsFile)
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return state.WriteFileAtomic(filePath, data, 0o644)
}

// VerifyDiagram marks a diagram as verified (updates lastVerified).
// Returns true if the diagram was found.
func VerifyDiagram(root, file string, line *int) (bool, error) {
	registry, err := ReadRegistry(root)
	if err != nil {
		return false, err
	}

	found := false
	now := time.Now().UTC().Format(time.RFC3339)
	for i, d := range registry {
		if d.File == file && (line == nil || d.Line == *line) {
			registry[i].LastVerified = now
			found = true
		}
	}

	if found {
		if err := WriteRegistry(root, registry); err != nil {
			return false, err
		}
	}
	return found, nil
}

// =============================================================================
// Staleness check
// =============================================================================

// CheckStaleness checks which diagrams are stale based on modified files.
func CheckStaleness(root string, modifiedFiles []string) ([]StaleDiagram, error) {
	registry, err := ReadRegistry(root)
	if err != nil {
		return nil, err
	}
	if len(registry) == 0 {
		return []StaleDiagram{}, nil
	}

	modSet := make(map[string]bool, len(modifiedFiles))
	for _, f := range modifiedFiles {
		modSet[strings.TrimPrefix(f, "./")] = true
	}

	var stale []StaleDiagram
	for _, diagram := range registry {
	outer:
		for _, ref := range diagram.ReferencedFiles {
			for mod := range modSet {
				if strings.Contains(mod, ref) || strings.Contains(ref, mod) {
					stale = append(stale, StaleDiagram{
						File:   diagram.File,
						Line:   diagram.Line,
						Type:   diagram.Type,
						Reason: mod + " was modified but diagram references " + ref,
					})
					break outer
				}
			}
		}
	}

	if stale == nil {
		return []StaleDiagram{}, nil
	}
	return stale, nil
}
