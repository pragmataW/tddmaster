package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadWriteState(t *testing.T) {
	dir := t.TempDir()

	t.Run("ReadState returns initial state when file missing", func(t *testing.T) {
		s, err := ReadState(dir)
		require.NoError(t, err)
		assert.Equal(t, PhaseIdle, s.Phase)
		assert.Equal(t, "0.1.0", s.Version)
	})

	t.Run("WriteState + ReadState round-trip", func(t *testing.T) {
		original := CreateInitialState()
		original.Phase = PhaseDiscovery
		specName := "test-spec"
		original.Spec = &specName

		err := WriteState(dir, original)
		require.NoError(t, err)

		loaded, err := ReadState(dir)
		require.NoError(t, err)
		assert.Equal(t, PhaseDiscovery, loaded.Phase)
		assert.Equal(t, "test-spec", *loaded.Spec)
	})
}

func TestReadWriteSpecState(t *testing.T) {
	dir := t.TempDir()

	t.Run("ReadSpecState returns initial state when file missing", func(t *testing.T) {
		s, err := ReadSpecState(dir, "non-existent")
		require.NoError(t, err)
		assert.Equal(t, PhaseIdle, s.Phase)
	})

	t.Run("WriteSpecState + ReadSpecState round-trip", func(t *testing.T) {
		original := CreateInitialState()
		original.Phase = PhaseExecuting
		specName := "my-spec"
		original.Spec = &specName

		err := WriteSpecState(dir, "my-spec", original)
		require.NoError(t, err)

		loaded, err := ReadSpecState(dir, "my-spec")
		require.NoError(t, err)
		assert.Equal(t, PhaseExecuting, loaded.Phase)
		assert.Equal(t, "my-spec", *loaded.Spec)
	})
}

func TestListSpecStates(t *testing.T) {
	dir := t.TempDir()

	t.Run("returns empty list when no states exist", func(t *testing.T) {
		results, err := ListSpecStates(dir)
		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("lists all spec states", func(t *testing.T) {
		s1 := CreateInitialState()
		s1.Phase = PhaseDiscovery
		s2 := CreateInitialState()
		s2.Phase = PhaseExecuting

		require.NoError(t, WriteSpecState(dir, "spec-a", s1))
		require.NoError(t, WriteSpecState(dir, "spec-b", s2))

		results, err := ListSpecStates(dir)
		require.NoError(t, err)
		assert.Len(t, results, 2)

		names := make(map[string]bool)
		for _, r := range results {
			names[r.Name] = true
		}
		assert.True(t, names["spec-a"])
		assert.True(t, names["spec-b"])
	})
}

func TestWriteStateAndSpec(t *testing.T) {
	dir := t.TempDir()

	t.Run("writes both main state and spec state", func(t *testing.T) {
		s := CreateInitialState()
		s.Phase = PhaseDiscovery
		specName := "combined-spec"
		s.Spec = &specName

		err := WriteStateAndSpec(dir, s)
		require.NoError(t, err)

		main, err := ReadState(dir)
		require.NoError(t, err)
		assert.Equal(t, PhaseDiscovery, main.Phase)

		spec, err := ReadSpecState(dir, "combined-spec")
		require.NoError(t, err)
		assert.Equal(t, PhaseDiscovery, spec.Phase)
	})

	t.Run("WriteStateAndSpec with nil spec only writes main", func(t *testing.T) {
		s := CreateInitialState()
		s.Spec = nil

		err := WriteStateAndSpec(dir, s)
		require.NoError(t, err)

		main, err := ReadState(dir)
		require.NoError(t, err)
		assert.Nil(t, main.Spec)
	})
}

func TestScaffoldDir(t *testing.T) {
	dir := t.TempDir()

	err := ScaffoldDir(dir)
	require.NoError(t, err)

	// Verify directories exist
	dirsToCheck := []string{
		TddmasterDir,
		stateDir,
		specStatesDir,
		concernsDir,
		rulesDir,
		specsDir,
		workflowsDir,
		eventsDir,
	}

	for _, d := range dirsToCheck {
		info, err := os.Stat(filepath.Join(dir, d))
		require.NoError(t, err, "directory should exist: %s", d)
		assert.True(t, info.IsDir())
	}

	// Verify .gitignore was created
	gitignorePath := filepath.Join(dir, TddmasterDir, ".gitignore")
	_, err = os.Stat(gitignorePath)
	require.NoError(t, err)

	// Running again should not fail (idempotent)
	err = ScaffoldDir(dir)
	require.NoError(t, err)
}

func TestParseSpecFlag(t *testing.T) {
	t.Run("returns nil when no --spec= flag", func(t *testing.T) {
		result := ParseSpecFlag([]string{"--other=value", "positional"})
		assert.Nil(t, result)
	})

	t.Run("returns spec name from --spec= flag", func(t *testing.T) {
		result := ParseSpecFlag([]string{"--spec=my-spec"})
		require.NotNil(t, result)
		assert.Equal(t, "my-spec", *result)
	})

	t.Run("handles nil args", func(t *testing.T) {
		result := ParseSpecFlag(nil)
		assert.Nil(t, result)
	})
}

func TestRequireSpecFlag(t *testing.T) {
	t.Run("returns error when missing", func(t *testing.T) {
		result := RequireSpecFlag([]string{})
		assert.False(t, result.OK)
		assert.Contains(t, result.Error, "spec name is required")
	})

	t.Run("returns spec name when present", func(t *testing.T) {
		result := RequireSpecFlag([]string{"--spec=my-spec"})
		assert.True(t, result.OK)
		assert.Equal(t, "my-spec", result.Spec)
	})
}

func TestUsesOldSpecFlag(t *testing.T) {
	assert.True(t, UsesOldSpecFlag([]string{"--spec=my-spec"}))
	assert.False(t, UsesOldSpecFlag([]string{"my-spec"}))
	assert.False(t, UsesOldSpecFlag(nil))
}

func TestSessions(t *testing.T) {
	dir := t.TempDir()

	t.Run("create and read session", func(t *testing.T) {
		session := Session{
			ID:           "abc12345",
			Mode:         "spec",
			PID:          12345,
			StartedAt:    "2024-01-01T00:00:00Z",
			LastActiveAt: "2024-01-01T01:00:00Z",
			Tool:         "claude-code",
		}

		err := CreateSession(dir, session)
		require.NoError(t, err)

		loaded, err := ReadSession(dir, "abc12345")
		require.NoError(t, err)
		require.NotNil(t, loaded)
		assert.Equal(t, "abc12345", loaded.ID)
		assert.Equal(t, "spec", loaded.Mode)
	})

	t.Run("ReadSession returns nil for missing session", func(t *testing.T) {
		loaded, err := ReadSession(dir, "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, loaded)
	})

	t.Run("ListSessions returns all sessions", func(t *testing.T) {
		dir2 := t.TempDir()
		session1 := Session{
			ID: "sess1", Mode: "free", PID: 1,
			StartedAt: "2024-01-01T00:00:00Z", LastActiveAt: "2024-01-01T00:00:00Z", Tool: "claude-code",
		}
		session2 := Session{
			ID: "sess2", Mode: "spec", PID: 2,
			StartedAt: "2024-01-01T00:00:00Z", LastActiveAt: "2024-01-01T00:00:00Z", Tool: "claude-code",
		}
		require.NoError(t, CreateSession(dir2, session1))
		require.NoError(t, CreateSession(dir2, session2))

		sessions, err := ListSessions(dir2)
		require.NoError(t, err)
		assert.Len(t, sessions, 2)
	})

	t.Run("DeleteSession removes file", func(t *testing.T) {
		dir3 := t.TempDir()
		session := Session{
			ID: "del-me", Mode: "free", PID: 1,
			StartedAt: "2024-01-01T00:00:00Z", LastActiveAt: "2024-01-01T00:00:00Z", Tool: "claude-code",
		}
		require.NoError(t, CreateSession(dir3, session))

		ok, err := DeleteSession(dir3, "del-me")
		require.NoError(t, err)
		assert.True(t, ok)

		// Should return false for non-existent
		ok, err = DeleteSession(dir3, "del-me")
		require.NoError(t, err)
		assert.False(t, ok)
	})
}

func TestIsSessionStale(t *testing.T) {
	t.Run("old session is stale", func(t *testing.T) {
		session := Session{
			ID:           "old",
			Mode:         "free",
			PID:          1,
			StartedAt:    "2020-01-01T00:00:00Z",
			LastActiveAt: "2020-01-01T00:00:00Z",
			Tool:         "cursor",
		}
		assert.True(t, IsSessionStale(session))
	})

	t.Run("recent session is not stale", func(t *testing.T) {
		session := Session{
			ID:           "new",
			Mode:         "free",
			PID:          1,
			StartedAt:    "2020-01-01T00:00:00Z",
			LastActiveAt: "2099-01-01T00:00:00Z", // future
			Tool:         "cursor",
		}
		assert.False(t, IsSessionStale(session))
	})
}

func TestGenerateSessionId(t *testing.T) {
	id1, err := GenerateSessionId()
	require.NoError(t, err)
	assert.Len(t, id1, 8)

	id2, err := GenerateSessionId()
	require.NoError(t, err)
	assert.Len(t, id2, 8)

	// Very unlikely to be equal
	assert.NotEqual(t, id1, id2)
}

func TestReadWriteManifest(t *testing.T) {
	dir := t.TempDir()

	t.Run("ReadManifest returns nil when no file", func(t *testing.T) {
		m, err := ReadManifest(dir)
		require.NoError(t, err)
		assert.Nil(t, m)
	})

	t.Run("WriteManifest + ReadManifest round-trip", func(t *testing.T) {
		manifest := CreateInitialManifest(
			[]string{"security"},
			[]CodingToolId{CodingToolClaudeCode},
			ProjectTraits{
				Languages:  []string{"go"},
				Frameworks: []string{},
				CI:         []string{},
				TestRunner: nil,
			},
		)

		err := WriteManifest(dir, manifest)
		require.NoError(t, err)

		loaded, err := ReadManifest(dir)
		require.NoError(t, err)
		require.NotNil(t, loaded)
		assert.Equal(t, 15, loaded.MaxIterationsBeforeRestart)
		assert.Equal(t, "tddmaster", loaded.Command)
		assert.False(t, loaded.AllowGit)
	})
}

func TestReadWriteConcern(t *testing.T) {
	dir := t.TempDir()

	t.Run("ReadConcern returns nil when not found", func(t *testing.T) {
		c, err := ReadConcern(dir, "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, c)
	})

	t.Run("WriteConcern + ReadConcern round-trip", func(t *testing.T) {
		concern := ConcernDefinition{
			ID:                 "security",
			Name:               "Security",
			Description:        "Security concern",
			Extras:             []ConcernExtra{},
			SpecSections:       []string{"auth"},
			Reminders:          []string{"use HTTPS"},
			AcceptanceCriteria: []string{"all endpoints require auth"},
		}

		err := WriteConcern(dir, concern)
		require.NoError(t, err)

		loaded, err := ReadConcern(dir, "security")
		require.NoError(t, err)
		require.NotNil(t, loaded)
		assert.Equal(t, "security", loaded.ID)
		assert.Equal(t, "Security", loaded.Name)
	})
}

func TestIsInitialized(t *testing.T) {
	dir := t.TempDir()

	t.Run("returns false when no manifest", func(t *testing.T) {
		ok, err := IsInitialized(dir)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("returns true when manifest has tddmaster section", func(t *testing.T) {
		manifest := CreateInitialManifest(
			[]string{},
			[]CodingToolId{},
			ProjectTraits{Languages: []string{}, Frameworks: []string{}, CI: []string{}},
		)
		err := WriteManifest(dir, manifest)
		require.NoError(t, err)

		ok, err := IsInitialized(dir)
		require.NoError(t, err)
		assert.True(t, ok)
	})
}

func TestFindProjectRoot(t *testing.T) {
	dir := t.TempDir()

	t.Run("returns empty string when .tddmaster not found", func(t *testing.T) {
		result, err := FindProjectRoot(dir)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("returns dir when .tddmaster exists", func(t *testing.T) {
		err := os.MkdirAll(filepath.Join(dir, TddmasterDir), 0o755)
		require.NoError(t, err)

		result, err := FindProjectRoot(dir)
		require.NoError(t, err)
		assert.Equal(t, dir, result)
	})
}

func TestResolveState(t *testing.T) {
	dir := t.TempDir()

	t.Run("returns active state when spec is nil", func(t *testing.T) {
		s, err := ResolveState(dir, nil)
		require.NoError(t, err)
		assert.Equal(t, PhaseIdle, s.Phase)
	})

	t.Run("returns error when spec directory not found", func(t *testing.T) {
		specName := "nonexistent-spec"
		_, err := ResolveState(dir, &specName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("returns spec state when spec exists", func(t *testing.T) {
		// Create spec directory
		specDir := filepath.Join(dir, specsDir, "test-spec")
		require.NoError(t, os.MkdirAll(specDir, 0o755))

		// Write spec state
		s := CreateInitialState()
		s.Phase = PhaseDiscovery
		specName := "test-spec"
		s.Spec = &specName
		require.NoError(t, WriteSpecState(dir, "test-spec", s))

		specName2 := "test-spec"
		loaded, err := ResolveState(dir, &specName2)
		require.NoError(t, err)
		assert.Equal(t, PhaseDiscovery, loaded.Phase)
	})
}
