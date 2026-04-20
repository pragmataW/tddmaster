// Package identity resolves the tddmaster user identity from the per-machine
// config file, falling back to git config and finally an "Unknown User"
// placeholder.
package identity

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pragmataW/tddmaster/internal/state/model"
	"github.com/pragmataW/tddmaster/internal/state/service/atomic"
)

// GetConfigDir returns the tddmaster config directory
// (~/.config/eser/tddmaster or XDG_CONFIG_HOME/eser/tddmaster).
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

// GetCurrentUser returns the configured user from
// ~/.config/eser/tddmaster/user.json, or nil if not set.
func GetCurrentUser(_ ...string) (*model.User, error) {
	data, err := os.ReadFile(GetUserFilePath())
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
	return &model.User{Name: raw.Name, Email: raw.Email}, nil
}

// SetCurrentUser writes user identity to ~/.config/eser/tddmaster/user.json.
func SetCurrentUser(user model.User) error {
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
	return atomic.WriteFileAtomic(GetUserFilePath(), data, 0o644)
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
func DetectGitUser() (*model.User, error) {
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

	return &model.User{Name: name, Email: email}, nil
}

// FormatUser formats user as "Name <email>" or just "Name" if no email.
func FormatUser(user model.User) string {
	if user.Email != "" {
		return user.Name + " <" + user.Email + ">"
	}
	return user.Name
}

// ShortUser returns just the user's name.
func ShortUser(user model.User) string {
	return user.Name
}

// UnknownUser returns a fallback user when no identity is configured.
func UnknownUser() model.User {
	return model.User{Name: "Unknown User", Email: ""}
}

// ResolveUser resolves user: config file → git config → Unknown User.
// Never returns an error.
func ResolveUser(_ ...string) (model.User, error) {
	configured, err := GetCurrentUser()
	if err == nil && configured != nil {
		return *configured, nil
	}

	gitUser, err := DetectGitUser()
	if err == nil && gitUser != nil {
		return *gitUser, nil
	}

	return UnknownUser(), nil
}
