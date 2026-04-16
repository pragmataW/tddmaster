
// User identity — resolves current user from ~/.config/eser/tddmaster/user.json
// or git config. Per-machine identity, not per-project.

package state

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// User represents a user identity.
type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// =============================================================================
// Config directory
// =============================================================================

// GetConfigDir returns the tddmaster config directory (~/.config/eser/tddmaster or XDG_CONFIG_HOME/eser/tddmaster).
func GetConfigDir() string {
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg != "" {
		return filepath.Join(xdg, "eser", "tddmaster")
	}
	home := os.Getenv("HOME")
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}
	if home == "" {
		home = "~"
	}
	return filepath.Join(home, ".config", "eser", "tddmaster")
}

// GetUserFilePath returns the path to the user identity file.
func GetUserFilePath() string {
	return filepath.Join(GetConfigDir(), "user.json")
}

// =============================================================================
// Read / Write
// =============================================================================

// GetCurrentUser returns the configured user from ~/.config/eser/tddmaster/user.json,
// or nil if not set.
func GetCurrentUser(_ ...string) (*User, error) {
	filePath := GetUserFilePath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil
	}

	var raw struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, nil
	}
	if raw.Name == "" {
		return nil, nil
	}
	return &User{Name: raw.Name, Email: raw.Email}, nil
}

// SetCurrentUser writes user identity to ~/.config/eser/tddmaster/user.json.
func SetCurrentUser(user User) error {
	dir := GetConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(map[string]string{
		"name":  user.Name,
		"email": user.Email,
	}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return WriteFileAtomic(GetUserFilePath(), data, 0o644)
}

// ClearCurrentUser removes the user identity file. Returns false if not found.
func ClearCurrentUser() (bool, error) {
	if err := os.Remove(GetUserFilePath()); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// DetectGitUser detects user from git config. Returns nil if git is not available.
func DetectGitUser() (*User, error) {
	nameBytes, err := exec.Command("git", "config", "user.name").Output()
	if err != nil {
		return nil, nil
	}
	name := strings.TrimSpace(string(nameBytes))
	if name == "" {
		return nil, nil
	}

	emailBytes, err := exec.Command("git", "config", "user.email").Output()
	email := ""
	if err == nil {
		email = strings.TrimSpace(string(emailBytes))
	}

	return &User{Name: name, Email: email}, nil
}

// =============================================================================
// Formatting
// =============================================================================

// FormatUser formats user as "Name <email>" or just "Name" if no email.
func FormatUser(user User) string {
	if user.Email != "" {
		return user.Name + " <" + user.Email + ">"
	}
	return user.Name
}

// ShortUser returns just the user's name.
func ShortUser(user User) string {
	return user.Name
}

// UnknownUser returns a fallback user when no identity is configured.
func UnknownUser() User {
	return User{Name: "Unknown User", Email: ""}
}

// =============================================================================
// Resolution chain
// =============================================================================

// ResolveUser resolves user: config file → git config → Unknown User.
// Never returns an error.
func ResolveUser(_ ...string) (User, error) {
	// 1. Config file
	configured, err := GetCurrentUser()
	if err == nil && configured != nil {
		return *configured, nil
	}

	// 2. Git config
	gitUser, err := DetectGitUser()
	if err == nil && gitUser != nil {
		return *gitUser, nil
	}

	// 3. Fallback
	return UnknownUser(), nil
}
