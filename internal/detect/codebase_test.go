
package detect

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper: create a temporary directory with the given files (path → content).
func makeTempDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()

	for rel, content := range files {
		full := filepath.Join(dir, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	}

	return dir
}

// =============================================================================
// pathExists
// =============================================================================

func TestPathExists_File(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"foo.txt": "hello"})
	assert.True(t, pathExists(filepath.Join(dir, "foo.txt")))
}

func TestPathExists_Missing(t *testing.T) {
	dir := t.TempDir()
	assert.False(t, pathExists(filepath.Join(dir, "no-such-file")))
}

// =============================================================================
// detectLanguages
// =============================================================================

func TestDetectLanguages_Typescript_PackageJson(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"package.json": "{}"})
	langs := detectLanguages(dir)
	assert.Contains(t, langs, "typescript")
}

func TestDetectLanguages_Typescript_DenoJson(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"deno.json": "{}"})
	langs := detectLanguages(dir)
	assert.Contains(t, langs, "typescript")
}

func TestDetectLanguages_Go(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"go.mod": "module example.com/foo\n\ngo 1.22\n"})
	langs := detectLanguages(dir)
	assert.Contains(t, langs, "go")
}

func TestDetectLanguages_Rust(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"Cargo.toml": "[package]\nname = \"foo\"\n"})
	langs := detectLanguages(dir)
	assert.Contains(t, langs, "rust")
}

func TestDetectLanguages_Python_Pyproject(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"pyproject.toml": "[tool.poetry]\nname = \"foo\"\n"})
	langs := detectLanguages(dir)
	assert.Contains(t, langs, "python")
}

func TestDetectLanguages_Python_SetupPy(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"setup.py": "from setuptools import setup\nsetup()\n"})
	langs := detectLanguages(dir)
	assert.Contains(t, langs, "python")
}

func TestDetectLanguages_Empty(t *testing.T) {
	dir := t.TempDir()
	langs := detectLanguages(dir)
	assert.Empty(t, langs)
}

// =============================================================================
// detectFrameworks
// =============================================================================

func TestDetectFrameworks_React(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"package.json": `{"dependencies":{"react":"^18.0.0"}}`,
	})
	fw := detectFrameworks(dir)
	assert.Contains(t, fw, "react")
}

func TestDetectFrameworks_Vue(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"package.json": `{"dependencies":{"vue":"^3.0.0"}}`,
	})
	fw := detectFrameworks(dir)
	assert.Contains(t, fw, "vue")
}

func TestDetectFrameworks_Next(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"package.json": `{"dependencies":{"next":"^14.0.0"}}`,
	})
	fw := detectFrameworks(dir)
	assert.Contains(t, fw, "nextjs")
}

func TestDetectFrameworks_NoPackageJson(t *testing.T) {
	dir := t.TempDir()
	fw := detectFrameworks(dir)
	assert.Empty(t, fw)
}

// =============================================================================
// detectCI
// =============================================================================

func TestDetectCI_GithubActions(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		".github/workflows/ci.yml": "on: push\n",
	})
	ci := detectCI(dir)
	assert.Contains(t, ci, "github-actions")
}

func TestDetectCI_GitlabCI(t *testing.T) {
	dir := makeTempDir(t, map[string]string{".gitlab-ci.yml": "stages:\n  - build\n"})
	ci := detectCI(dir)
	assert.Contains(t, ci, "gitlab-ci")
}

func TestDetectCI_Jenkins(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"Jenkinsfile": "pipeline {}\n"})
	ci := detectCI(dir)
	assert.Contains(t, ci, "jenkins")
}

func TestDetectCI_CircleCI(t *testing.T) {
	dir := makeTempDir(t, map[string]string{".circleci/config.yml": "version: 2.1\n"})
	ci := detectCI(dir)
	assert.Contains(t, ci, "circleci")
}

func TestDetectCI_Empty(t *testing.T) {
	dir := t.TempDir()
	ci := detectCI(dir)
	assert.Empty(t, ci)
}

// =============================================================================
// detectTestRunner
// =============================================================================

func TestDetectTestRunner_Deno(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"deno.json": "{}"})
	tr := detectTestRunner(dir)
	require.NotNil(t, tr)
	assert.Equal(t, "deno", *tr)
}

func TestDetectTestRunner_Vitest(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"package.json": `{"devDependencies":{"vitest":"^1.0.0"}}`,
	})
	tr := detectTestRunner(dir)
	require.NotNil(t, tr)
	assert.Equal(t, "vitest", *tr)
}

func TestDetectTestRunner_Jest(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"package.json": `{"devDependencies":{"jest":"^29.0.0"}}`,
	})
	tr := detectTestRunner(dir)
	require.NotNil(t, tr)
	assert.Equal(t, "jest", *tr)
}

func TestDetectTestRunner_Playwright(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"package.json": `{"devDependencies":{"playwright":"^1.0.0"}}`,
	})
	tr := detectTestRunner(dir)
	require.NotNil(t, tr)
	assert.Equal(t, "playwright", *tr)
}

func TestDetectTestRunner_None(t *testing.T) {
	dir := t.TempDir()
	tr := detectTestRunner(dir)
	assert.Nil(t, tr)
}

// =============================================================================
// DetectProject (integration)
// =============================================================================

func TestDetectProject_FullStack(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"package.json":             `{"dependencies":{"react":"^18.0.0"},"devDependencies":{"vitest":"^1.0.0"}}`,
		".github/workflows/ci.yml": "on: push\n",
	})
	traits := DetectProject(dir)

	assert.Contains(t, traits.Languages, "typescript")
	assert.Contains(t, traits.Frameworks, "react")
	assert.Contains(t, traits.CI, "github-actions")
	require.NotNil(t, traits.TestRunner)
	assert.Equal(t, "vitest", *traits.TestRunner)
}

func TestDetectProject_Empty(t *testing.T) {
	dir := t.TempDir()
	traits := DetectProject(dir)

	assert.Empty(t, traits.Languages)
	assert.Empty(t, traits.Frameworks)
	assert.Empty(t, traits.CI)
	assert.Nil(t, traits.TestRunner)
}
