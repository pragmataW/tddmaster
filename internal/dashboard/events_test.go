
package dashboard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppendEventAndReadEvents(t *testing.T) {
	dir := t.TempDir()

	ev1 := DashboardEvent{
		Ts:   "2024-01-01T10:00:00Z",
		Type: EventTypePhaseChange,
		Spec: "my-spec",
		User: "alice",
		Extra: map[string]interface{}{
			"from": "SPEC_PROPOSAL",
			"to":   "SPEC_APPROVED",
		},
	}
	ev2 := DashboardEvent{
		Ts:   "2024-01-01T11:00:00Z",
		Type: EventTypeNote,
		Spec: "my-spec",
		User: "bob",
		Extra: map[string]interface{}{
			"text": "hello",
		},
	}

	require.NoError(t, AppendEvent(dir, ev1))
	require.NoError(t, AppendEvent(dir, ev2))

	events, err := ReadEvents(dir, nil)
	require.NoError(t, err)
	assert.Len(t, events, 2)

	// Newest first
	assert.Equal(t, "bob", events[0].User)
	assert.Equal(t, "alice", events[1].User)
}

func TestReadEvents_Empty(t *testing.T) {
	dir := t.TempDir()
	events, err := ReadEvents(dir, nil)
	require.NoError(t, err)
	assert.Empty(t, events)
}

func TestReadEvents_Limit(t *testing.T) {
	dir := t.TempDir()

	for i := 0; i < 5; i++ {
		ev := DashboardEvent{
			Ts:   "2024-01-01T10:00:0" + string(rune('0'+i)) + "Z",
			Type: EventTypeNote,
			Spec: "spec",
			User: "user",
		}
		require.NoError(t, AppendEvent(dir, ev))
	}

	events, err := ReadEvents(dir, &ReadEventsOpts{Limit: 3})
	require.NoError(t, err)
	assert.Len(t, events, 3)
}

func TestReadEvents_Since(t *testing.T) {
	dir := t.TempDir()

	evOld := DashboardEvent{
		Ts:   "2024-01-01T08:00:00Z",
		Type: EventTypeNote,
		Spec: "spec",
		User: "user",
	}
	evNew := DashboardEvent{
		Ts:   "2024-01-01T12:00:00Z",
		Type: EventTypeNote,
		Spec: "spec",
		User: "user",
	}
	require.NoError(t, AppendEvent(dir, evOld))
	require.NoError(t, AppendEvent(dir, evNew))

	events, err := ReadEvents(dir, &ReadEventsOpts{Since: "2024-01-01T10:00:00Z"})
	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, "2024-01-01T12:00:00Z", events[0].Ts)
}

func TestDashboardEvent_MarshalUnmarshal(t *testing.T) {
	ev := DashboardEvent{
		Ts:   "2024-01-01T10:00:00Z",
		Type: EventTypeMention,
		Spec: "spec",
		User: "alice",
		Extra: map[string]interface{}{
			"from":     "alice",
			"to":       "bob",
			"question": "hello?",
			"id":       "mention-123",
		},
	}

	data, err := json.Marshal(ev)
	require.NoError(t, err)

	var decoded DashboardEvent
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, ev.Ts, decoded.Ts)
	assert.Equal(t, ev.Type, decoded.Type)
	assert.Equal(t, ev.Spec, decoded.Spec)
	assert.Equal(t, ev.User, decoded.User)
	assert.Equal(t, "mention-123", decoded.GetExtra("id"))
	assert.Equal(t, "hello?", decoded.GetExtra("question"))
}

func TestAppendEvent_CreatesDir(t *testing.T) {
	dir := t.TempDir()
	ev := DashboardEvent{
		Ts:   "2024-01-01T10:00:00Z",
		Type: EventTypeNote,
		Spec: "spec",
		User: "user",
	}
	require.NoError(t, AppendEvent(dir, ev))

	// Check that events file was created
	evFile := filepath.Join(dir, eventsFile)
	_, err := os.Stat(evFile)
	assert.NoError(t, err)
}
