
package dashboard

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddLearningAndReadLearnings(t *testing.T) {
	dir := t.TempDir()

	l1 := Learning{
		Ts:       "2024-01-01T10:00:00Z",
		Spec:     "spec-a",
		Type:     LearningTypeMistake,
		Text:     "Don't forget to close connections",
		Severity: "high",
	}
	l2 := Learning{
		Ts:       "2024-01-01T11:00:00Z",
		Spec:     "spec-b",
		Type:     LearningTypeConvention,
		Text:     "Always use snake_case for file names",
		Severity: "medium",
	}

	require.NoError(t, AddLearning(dir, l1))
	require.NoError(t, AddLearning(dir, l2))

	learnings, err := ReadLearnings(dir)
	require.NoError(t, err)
	assert.Len(t, learnings, 2)
	assert.Equal(t, l1.Text, learnings[0].Text)
	assert.Equal(t, l2.Text, learnings[1].Text)
}

func TestReadLearnings_Empty(t *testing.T) {
	dir := t.TempDir()
	learnings, err := ReadLearnings(dir)
	require.NoError(t, err)
	assert.Empty(t, learnings)
}

func TestRemoveLearning(t *testing.T) {
	dir := t.TempDir()

	l1 := Learning{Ts: "2024-01-01T10:00:00Z", Spec: "spec", Type: LearningTypeMistake, Text: "first", Severity: "low"}
	l2 := Learning{Ts: "2024-01-01T11:00:00Z", Spec: "spec", Type: LearningTypeMistake, Text: "second", Severity: "low"}
	l3 := Learning{Ts: "2024-01-01T12:00:00Z", Spec: "spec", Type: LearningTypeMistake, Text: "third", Severity: "low"}

	require.NoError(t, AddLearning(dir, l1))
	require.NoError(t, AddLearning(dir, l2))
	require.NoError(t, AddLearning(dir, l3))

	ok, err := RemoveLearning(dir, 1) // remove "second"
	require.NoError(t, err)
	assert.True(t, ok)

	remaining, err := ReadLearnings(dir)
	require.NoError(t, err)
	assert.Len(t, remaining, 2)
	assert.Equal(t, "first", remaining[0].Text)
	assert.Equal(t, "third", remaining[1].Text)
}

func TestRemoveLearning_OutOfRange(t *testing.T) {
	dir := t.TempDir()
	l := Learning{Ts: "2024-01-01T10:00:00Z", Spec: "spec", Type: LearningTypeMistake, Text: "one", Severity: "low"}
	require.NoError(t, AddLearning(dir, l))

	ok, err := RemoveLearning(dir, 5)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestGetRelevantLearnings(t *testing.T) {
	dir := t.TempDir()

	learnings := []Learning{
		{Ts: "2024-01-01T10:00:00Z", Spec: "spec", Type: LearningTypeConvention, Text: "use dependency injection for testing", Severity: "high"},
		{Ts: "2024-01-01T11:00:00Z", Spec: "spec", Type: LearningTypeMistake, Text: "avoid global state in tests", Severity: "medium"},
		{Ts: "2024-01-01T12:00:00Z", Spec: "spec", Type: LearningTypeSuccess, Text: "table-driven tests improve coverage", Severity: "low"},
		{Ts: "2024-01-01T13:00:00Z", Spec: "spec", Type: LearningTypeDependency, Text: "redis requires docker for tests", Severity: "low"},
		{Ts: "2024-01-01T14:00:00Z", Spec: "spec", Type: LearningTypeConvention, Text: "always validate inputs early", Severity: "medium"},
		{Ts: "2024-01-01T15:00:00Z", Spec: "spec", Type: LearningTypeMistake, Text: "forgot to handle errors properly", Severity: "high"},
	}

	for _, l := range learnings {
		require.NoError(t, AddLearning(dir, l))
	}

	relevant, err := GetRelevantLearnings(dir, "implement testing with dependency injection")
	require.NoError(t, err)
	assert.LessOrEqual(t, len(relevant), 5)
	assert.GreaterOrEqual(t, len(relevant), 1)
}

func TestFormatLearnings(t *testing.T) {
	learnings := []Learning{
		{Ts: "2024-01-01T10:00:00Z", Spec: "spec-a", Type: LearningTypeMistake, Text: "oops", Severity: "high"},
		{Ts: "2024-01-01T11:00:00Z", Spec: "spec-b", Type: LearningTypeConvention, Text: "always do X", Severity: "medium"},
		{Ts: "2024-01-01T12:00:00Z", Spec: "spec-c", Type: LearningTypeSuccess, Text: "worked well", Severity: "low"},
		{Ts: "2024-01-01T13:00:00Z", Spec: "spec-d", Type: LearningTypeDependency, Text: "needs Y", Severity: "low"},
	}

	formatted := FormatLearnings(learnings)
	assert.Len(t, formatted, 4)
	assert.Contains(t, formatted[0], "Past mistake")
	assert.Contains(t, formatted[1], "Convention")
	assert.Contains(t, formatted[2], "Success")
	assert.Contains(t, formatted[3], "Dependency")
	assert.Contains(t, formatted[0], "spec-a")
}
