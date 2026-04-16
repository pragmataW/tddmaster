
// Dashboard events — append-only JSONL log at `.tddmaster/.events/events.jsonl`.

package dashboard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/pragmataW/tddmaster/internal/state"
)

// =============================================================================
// Types
// =============================================================================

// EventType represents the type of a dashboard event.
type EventType string

const (
	EventTypePhaseChange        EventType = "phase-change"
	EventTypeMention            EventType = "mention"
	EventTypeMentionReply       EventType = "mention-reply"
	EventTypeSignoff            EventType = "signoff"
	EventTypeNote               EventType = "note"
	EventTypeTaskCompleted      EventType = "task-completed"
	EventTypeSpecCreated        EventType = "spec-created"
	EventTypeAnswerAdded        EventType = "answer-added"
	EventTypeDelegationCreated  EventType = "delegation-created"
	EventTypeDelegationAnswered EventType = "delegation-answered"
	EventTypeApproveBlocked     EventType = "approve-blocked"
)

// DashboardEvent is an event in the JSONL log.
type DashboardEvent struct {
	Ts   string    `json:"ts"`
	Type EventType `json:"type"`
	Spec string    `json:"spec"`
	User string    `json:"user"`
	// Extra holds any additional fields.
	Extra map[string]interface{} `json:"-"`
}

// MarshalJSON serializes DashboardEvent, merging Extra into the output.
func (e DashboardEvent) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"ts":   e.Ts,
		"type": e.Type,
		"spec": e.Spec,
		"user": e.User,
	}
	for k, v := range e.Extra {
		m[k] = v
	}
	return json.Marshal(m)
}

// UnmarshalJSON deserializes DashboardEvent, collecting unknown fields into Extra.
func (e *DashboardEvent) UnmarshalJSON(data []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if v, ok := raw["ts"].(string); ok {
		e.Ts = v
	}
	if v, ok := raw["type"].(string); ok {
		e.Type = EventType(v)
	}
	if v, ok := raw["spec"].(string); ok {
		e.Spec = v
	}
	if v, ok := raw["user"].(string); ok {
		e.User = v
	}

	e.Extra = make(map[string]interface{})
	for k, v := range raw {
		switch k {
		case "ts", "type", "spec", "user":
			// already handled
		default:
			e.Extra[k] = v
		}
	}
	return nil
}

// GetExtra returns an extra field value as string, or empty string if not found.
func (e DashboardEvent) GetExtra(key string) string {
	if e.Extra == nil {
		return ""
	}
	if v, ok := e.Extra[key].(string); ok {
		return v
	}
	return ""
}

// =============================================================================
// Paths
// =============================================================================

const eventsDir = state.TddmasterDir + "/.events"
const eventsFile = eventsDir + "/events.jsonl"

// EventsDir returns the events directory (relative to root).
var EventsDir = eventsDir

// EventsFile returns the events file path (relative to root).
var EventsFile = eventsFile

// =============================================================================
// Write
// =============================================================================

// AppendEvent appends a single event to the JSONL log. Creates file/dir if needed.
func AppendEvent(root string, event DashboardEvent) error {
	dir := filepath.Join(root, eventsDir)
	file := filepath.Join(root, eventsFile)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	line := string(data) + "\n"

	var existing []byte
	existing, _ = os.ReadFile(file)
	return os.WriteFile(file, append(existing, []byte(line)...), 0o644)
}

// =============================================================================
// Read
// =============================================================================

// ReadEventsOpts holds options for ReadEvents.
type ReadEventsOpts struct {
	Limit int    // 0 means no limit
	Since string // ISO timestamp; omit if empty
}

// ReadEvents reads events from the JSONL log. Newest first by default.
func ReadEvents(root string, opts *ReadEventsOpts) ([]DashboardEvent, error) {
	file := filepath.Join(root, eventsFile)

	content, err := os.ReadFile(file)
	if err != nil {
		return []DashboardEvent{}, nil
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	var events []DashboardEvent
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var ev DashboardEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		events = append(events, ev)
	}

	// Filter by since
	if opts != nil && opts.Since != "" {
		var filtered []DashboardEvent
		for _, ev := range events {
			if ev.Ts > opts.Since {
				filtered = append(filtered, ev)
			}
		}
		events = filtered
	}

	// Newest first
	for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
		events[i], events[j] = events[j], events[i]
	}

	// Limit
	if opts != nil && opts.Limit > 0 && len(events) > opts.Limit {
		events = events[:opts.Limit]
	}

	if events == nil {
		return []DashboardEvent{}, nil
	}
	return events, nil
}
