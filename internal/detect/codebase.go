
// Package detect provides project trait and coding tool detection.
package detect

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/pragmataW/tddmaster/internal/state"
)

// =============================================================================
// Detection Helpers
// =============================================================================

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func readJSONField(path string, field string) map[string]interface{} {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil
	}

	val, ok := parsed[field]
	if !ok || val == nil {
		return nil
	}

	m, ok := val.(map[string]interface{})
	if !ok {
		return nil
	}

	return m
}

// =============================================================================
// Language Detection
// =============================================================================

func detectLanguages(root string) []string {
	var languages []string

	if pathExists(filepath.Join(root, "package.json")) || pathExists(filepath.Join(root, "deno.json")) {
		languages = append(languages, "typescript")
	}
	if pathExists(filepath.Join(root, "go.mod")) {
		languages = append(languages, "go")
	}
	if pathExists(filepath.Join(root, "Cargo.toml")) {
		languages = append(languages, "rust")
	}
	if pathExists(filepath.Join(root, "pyproject.toml")) || pathExists(filepath.Join(root, "setup.py")) {
		languages = append(languages, "python")
	}

	return languages
}

// =============================================================================
// Framework Detection
// =============================================================================

func detectFrameworks(root string) []string {
	var frameworks []string
	deps := readJSONField(filepath.Join(root, "package.json"), "dependencies")

	if deps != nil {
		if _, ok := deps["react"]; ok {
			frameworks = append(frameworks, "react")
		}
		if _, ok := deps["vue"]; ok {
			frameworks = append(frameworks, "vue")
		}
		if _, ok := deps["svelte"]; ok {
			frameworks = append(frameworks, "svelte")
		}
		if _, ok := deps["next"]; ok {
			frameworks = append(frameworks, "nextjs")
		}
		if _, ok := deps["express"]; ok {
			frameworks = append(frameworks, "express")
		}
		if _, ok := deps["hono"]; ok {
			frameworks = append(frameworks, "hono")
		}
	}

	return frameworks
}

// =============================================================================
// CI Detection
// =============================================================================

func detectCI(root string) []string {
	var ci []string

	if pathExists(filepath.Join(root, ".github", "workflows")) {
		ci = append(ci, "github-actions")
	}
	if pathExists(filepath.Join(root, ".gitlab-ci.yml")) {
		ci = append(ci, "gitlab-ci")
	}
	if pathExists(filepath.Join(root, "Jenkinsfile")) {
		ci = append(ci, "jenkins")
	}
	if pathExists(filepath.Join(root, ".circleci")) {
		ci = append(ci, "circleci")
	}

	return ci
}

// =============================================================================
// Test Runner Detection
// =============================================================================

func detectTestRunner(root string) *string {
	if pathExists(filepath.Join(root, "deno.json")) {
		s := "deno"
		return &s
	}

	deps := readJSONField(filepath.Join(root, "package.json"), "devDependencies")

	if deps != nil {
		if _, ok := deps["vitest"]; ok {
			s := "vitest"
			return &s
		}
		if _, ok := deps["jest"]; ok {
			s := "jest"
			return &s
		}
		if _, ok := deps["playwright"]; ok {
			s := "playwright"
			return &s
		}
	}

	return nil
}

// =============================================================================
// Full Detection
// =============================================================================

// DetectProject detects project traits — language, framework, CI, test runner.
func DetectProject(root string) state.ProjectTraits {
	return state.ProjectTraits{
		Languages:  detectLanguages(root),
		Frameworks: detectFrameworks(root),
		CI:         detectCI(root),
		TestRunner: detectTestRunner(root),
	}
}
