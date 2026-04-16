
package pack_test

import (
	"testing"

	"github.com/pragmataW/tddmaster/internal/pack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// ValidatePackManifest
// =============================================================================

func TestValidatePackManifest_ValidMinimal(t *testing.T) {
	data := map[string]interface{}{
		"name":        "my-pack",
		"version":     "1.0.0",
		"description": "A test pack",
	}

	manifest, err := pack.ValidatePackManifest(data)
	require.NoError(t, err)
	assert.Equal(t, "my-pack", manifest.Name)
	assert.Equal(t, "1.0.0", manifest.Version)
	assert.Equal(t, "A test pack", manifest.Description)
}

func TestValidatePackManifest_ValidFull(t *testing.T) {
	author := "Jane Doe"
	data := map[string]interface{}{
		"name":        "full-pack",
		"version":     "2.0.0",
		"description": "A full pack",
		"author":      author,
		"tags":        []interface{}{"go", "test"},
		"requires":    []interface{}{"dep-pack"},
		"rules":       []interface{}{"rule1.md"},
		"concerns":    []interface{}{"concern1"},
		"folderRules": map[string]interface{}{"src/": "rule2.md"},
	}

	manifest, err := pack.ValidatePackManifest(data)
	require.NoError(t, err)
	assert.Equal(t, "full-pack", manifest.Name)
	assert.Equal(t, "2.0.0", manifest.Version)
	assert.Equal(t, "A full pack", manifest.Description)
	require.NotNil(t, manifest.Author)
	assert.Equal(t, author, *manifest.Author)
	assert.Equal(t, []string{"go", "test"}, manifest.Tags)
	assert.Equal(t, []string{"dep-pack"}, manifest.Requires)
	assert.Equal(t, []string{"rule1.md"}, manifest.Rules)
	assert.Equal(t, []string{"concern1"}, manifest.Concerns)
	assert.Equal(t, map[string]string{"src/": "rule2.md"}, manifest.FolderRules)
}

func TestValidatePackManifest_MissingName(t *testing.T) {
	data := map[string]interface{}{
		"version":     "1.0.0",
		"description": "A test pack",
	}

	_, err := pack.ValidatePackManifest(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "'name'")
}

func TestValidatePackManifest_EmptyName(t *testing.T) {
	data := map[string]interface{}{
		"name":        "",
		"version":     "1.0.0",
		"description": "A test pack",
	}

	_, err := pack.ValidatePackManifest(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "'name'")
}

func TestValidatePackManifest_MissingVersion(t *testing.T) {
	data := map[string]interface{}{
		"name":        "my-pack",
		"description": "A test pack",
	}

	_, err := pack.ValidatePackManifest(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "'version'")
}

func TestValidatePackManifest_EmptyVersion(t *testing.T) {
	data := map[string]interface{}{
		"name":        "my-pack",
		"version":     "",
		"description": "A test pack",
	}

	_, err := pack.ValidatePackManifest(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "'version'")
}

func TestValidatePackManifest_MissingDescription(t *testing.T) {
	data := map[string]interface{}{
		"name":    "my-pack",
		"version": "1.0.0",
	}

	_, err := pack.ValidatePackManifest(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "'description'")
}

func TestValidatePackManifest_EmptyDescription(t *testing.T) {
	data := map[string]interface{}{
		"name":        "my-pack",
		"version":     "1.0.0",
		"description": "",
	}

	_, err := pack.ValidatePackManifest(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "'description'")
}

func TestValidatePackManifest_NotAnObject(t *testing.T) {
	_, err := pack.ValidatePackManifest("not an object")
	require.Error(t, err)
}

// =============================================================================
// CreateEmptyPacksFile
// =============================================================================

func TestCreateEmptyPacksFile(t *testing.T) {
	pf := pack.CreateEmptyPacksFile()
	assert.NotNil(t, pf.Installed)
	assert.Empty(t, pf.Installed)
}
