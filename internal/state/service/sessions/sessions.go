// Package sessions manages active tddmaster work session JSON files under
// .tddmaster/.sessions/.
package sessions

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pragmataW/tddmaster/internal/state/model"
	"github.com/pragmataW/tddmaster/internal/state/service/atomic"
	"github.com/pragmataW/tddmaster/internal/state/service/paths"
)

const staleThresholdMs = 2 * 60 * 60 * 1000 // 2 hours

// CreateSession writes a session file.
func CreateSession(root string, session model.Session) error {
	dir := filepath.Join(root, paths.SessionsDir)
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return atomic.WriteFileAtomic(filepath.Join(dir, session.ID+".json"), data, 0o644)
}

// ReadSession reads a session file by ID.
func ReadSession(root, sessionID string) (*model.Session, error) {
	filePath := filepath.Join(root, paths.SessionsDir, sessionID+".json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil
	}

	var s model.Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, nil
	}
	return &s, nil
}

// ListSessions lists all sessions.
func ListSessions(root string) ([]model.Session, error) {
	dir := filepath.Join(root, paths.SessionsDir)
	var sessions []model.Session

	entries, err := os.ReadDir(dir)
	if err != nil {
		return sessions, nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		var s model.Session
		if err := json.Unmarshal(data, &s); err != nil {
			continue
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

// DeleteSession removes a session file. Returns false if not found.
func DeleteSession(root, sessionID string) (bool, error) {
	filePath := filepath.Join(root, paths.SessionsDir, sessionID+".json")
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// UpdateSessionPhase updates the phase and lastActiveAt of a session.
func UpdateSessionPhase(root, sessionID, phase string) error {
	session, err := ReadSession(root, sessionID)
	if err != nil || session == nil {
		return err
	}

	updated := *session
	updated.Phase = &phase
	updated.LastActiveAt = time.Now().UTC().Format(time.RFC3339)

	data, err := json.MarshalIndent(updated, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return atomic.WriteFileAtomic(filepath.Join(root, paths.SessionsDir, sessionID+".json"), data, 0o644)
}

// GcStaleSessions removes stale sessions and returns their IDs.
func GcStaleSessions(root string) ([]string, error) {
	sessions, err := ListSessions(root)
	if err != nil {
		return nil, err
	}
	var removed []string
	for _, s := range sessions {
		if IsSessionStale(s) {
			if _, err := DeleteSession(root, s.ID); err == nil {
				removed = append(removed, s.ID)
			}
		}
	}
	return removed, nil
}

// IsSessionStale returns true if the session has been inactive for more than 2 hours.
func IsSessionStale(session model.Session) bool {
	t, err := time.Parse(time.RFC3339, session.LastActiveAt)
	if err != nil {
		return true
	}
	elapsed := time.Since(t).Milliseconds()
	return elapsed > staleThresholdMs
}

// GenerateSessionId generates an 8-char random hex session ID.
func GenerateSessionId() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%08x", b), nil
}
