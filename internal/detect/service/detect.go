// Package service implements project-trait and coding-tool detection
// by probing marker files in a repository root. Hardcoded signal
// tables live in the sibling model package.
package service

import (
	"path/filepath"

	"github.com/pragmataW/tddmaster/internal/detect/model"
	"github.com/pragmataW/tddmaster/internal/state"
)

// DetectProject detects project traits by scanning marker files in
// root. The returned ProjectTraits is always fully populated (slices
// may be nil-or-empty; TestRunner may be nil if none detected).
func DetectProject(root string) state.ProjectTraits {
	return state.ProjectTraits{
		Languages:  detectLanguages(root),
		Frameworks: detectFrameworks(root),
		CI:         detectCI(root),
		TestRunner: detectTestRunner(root),
	}
}

// DetectCodingTools detects available coding tools by checking for
// their marker files in root. Tools are returned in the order defined
// by model.ToolSignals.
func DetectCodingTools(root string) []state.CodingToolId {
	var detected []state.CodingToolId
	for _, signal := range model.ToolSignals {
		for _, p := range signal.Paths {
			if pathExists(filepath.Join(root, p)) {
				detected = append(detected, signal.ID)
				break
			}
		}
	}
	return detected
}

func detectLanguages(root string) []string {
	seen := make(map[string]bool, len(model.LanguageSignals))
	var langs []string
	for _, sig := range model.LanguageSignals {
		if seen[sig.Language] {
			continue
		}
		if pathExists(filepath.Join(root, sig.File)) {
			langs = append(langs, sig.Language)
			seen[sig.Language] = true
		}
	}
	return langs
}

func detectFrameworks(root string) []string {
	deps := readJSONField(filepath.Join(root, model.FilePackageJSON), model.PkgFieldDeps)
	if deps == nil {
		return nil
	}
	var frameworks []string
	for _, sig := range model.FrameworkSignals {
		if _, ok := deps[sig.Dep]; ok {
			frameworks = append(frameworks, sig.Name)
		}
	}
	return frameworks
}

func detectCI(root string) []string {
	var ci []string
	for _, sig := range model.CISignals {
		if pathExists(filepath.Join(root, sig.Path)) {
			ci = append(ci, sig.Name)
		}
	}
	return ci
}

func detectTestRunner(root string) *string {
	if pathExists(filepath.Join(root, model.FileDenoJSON)) {
		s := model.TestRunnerDeno
		return &s
	}
	devDeps := readJSONField(filepath.Join(root, model.FilePackageJSON), model.PkgFieldDevDeps)
	if devDeps == nil {
		return nil
	}
	for _, name := range model.TestRunnerSignals {
		if _, ok := devDeps[name]; ok {
			s := name
			return &s
		}
	}
	return nil
}
