
package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatUser(t *testing.T) {
	t.Run("formats user with email", func(t *testing.T) {
		user := User{Name: "Alice", Email: "alice@example.com"}
		result := FormatUser(user)
		assert.Equal(t, "Alice <alice@example.com>", result)
	})

	t.Run("formats user without email", func(t *testing.T) {
		user := User{Name: "Alice", Email: ""}
		result := FormatUser(user)
		assert.Equal(t, "Alice", result)
	})
}

func TestShortUser(t *testing.T) {
	user := User{Name: "Alice", Email: "alice@example.com"}
	result := ShortUser(user)
	assert.Equal(t, "Alice", result)
}

func TestUnknownUser(t *testing.T) {
	result := UnknownUser()
	assert.Equal(t, "Unknown User", result.Name)
	assert.Equal(t, "", result.Email)
}

func TestGetConfigDir(t *testing.T) {
	t.Run("uses XDG_CONFIG_HOME when set", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/custom/config")
		result := GetConfigDir()
		assert.Equal(t, filepath.Join("/custom/config", "eser", "tddmaster"), result)
	})

	t.Run("uses HOME when XDG not set", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("HOME", "/home/alice")
		result := GetConfigDir()
		assert.Equal(t, filepath.Join("/home/alice", ".config", "eser", "tddmaster"), result)
	})
}

func TestGetCurrentUser(t *testing.T) {
	t.Run("returns nil when file missing", func(t *testing.T) {
		// Override config dir to temp dir
		dir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", dir)

		user, err := GetCurrentUser()
		require.NoError(t, err)
		assert.Nil(t, user)
	})
}

func TestSetCurrentUser(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	user := User{Name: "Bob", Email: "bob@example.com"}
	err := SetCurrentUser(user)
	require.NoError(t, err)

	// Verify file was written
	filePath := GetUserFilePath()
	_, err = os.Stat(filePath)
	require.NoError(t, err)

	// Read it back
	loaded, err := GetCurrentUser()
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, "Bob", loaded.Name)
	assert.Equal(t, "bob@example.com", loaded.Email)
}

func TestClearCurrentUser(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	t.Run("returns false when file does not exist", func(t *testing.T) {
		ok, err := ClearCurrentUser()
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("returns true after clearing", func(t *testing.T) {
		err := SetCurrentUser(User{Name: "Alice", Email: ""})
		require.NoError(t, err)

		ok, err := ClearCurrentUser()
		require.NoError(t, err)
		assert.True(t, ok)

		// File should be gone
		_, err = os.Stat(GetUserFilePath())
		assert.True(t, os.IsNotExist(err))
	})
}

func TestResolveUser(t *testing.T) {
	t.Run("returns Unknown User when no config and no git", func(t *testing.T) {
		// Use a temp dir with no user config
		dir := t.TempDir()
		subdir := filepath.Join(dir, "tddmaster-config")
		t.Setenv("XDG_CONFIG_HOME", subdir)

		// We can't easily disable git, but if git has no user, it will also return unknown
		// The function should always return a non-nil user
		user, err := ResolveUser()
		require.NoError(t, err)
		// Just verify we get something (could be Unknown User or git user)
		assert.NotEmpty(t, user.Name)
	})

	t.Run("returns configured user from file", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", dir)

		err := SetCurrentUser(User{Name: "TestUser", Email: "test@example.com"})
		require.NoError(t, err)

		user, err := ResolveUser()
		require.NoError(t, err)
		assert.Equal(t, "TestUser", user.Name)
		assert.Equal(t, "test@example.com", user.Email)
	})
}
