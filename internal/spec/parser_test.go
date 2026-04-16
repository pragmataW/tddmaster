
package spec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSpecContent_EmptyContent(t *testing.T) {
	parsed := ParseSpecContent("my-spec", "")
	assert.Equal(t, "my-spec", parsed.Name)
	assert.Empty(t, parsed.Tasks)
	assert.Empty(t, parsed.OutOfScope)
	assert.Empty(t, parsed.Verification)
}

func TestParseSpecContent_Tasks(t *testing.T) {
	content := `# Spec: test

## Tasks

- [ ] task-1: First task
- [x] task-2: Second task done
- [ ] task-3: Third task
`
	parsed := ParseSpecContent("test", content)

	assert.Len(t, parsed.Tasks, 3)
	assert.Equal(t, "task-1", parsed.Tasks[0].ID)
	assert.Equal(t, "First task", parsed.Tasks[0].Title)
	assert.Equal(t, "task-2", parsed.Tasks[1].ID)
	assert.Equal(t, "Second task done", parsed.Tasks[1].Title)
	assert.Equal(t, "task-3", parsed.Tasks[2].ID)
}

func TestParseSpecContent_TasksWithFiles(t *testing.T) {
	content := `## Tasks

- [ ] task-1: Implement feature
  Files: ` + "`src/main.go`" + `, ` + "`src/util.go`" + `
- [ ] task-2: Write tests
`
	parsed := ParseSpecContent("test", content)

	assert.Len(t, parsed.Tasks, 2)
	assert.Equal(t, []string{"src/main.go", "src/util.go"}, parsed.Tasks[0].Files)
	assert.Nil(t, parsed.Tasks[1].Files)
}

func TestParseSpecContent_OutOfScope(t *testing.T) {
	content := `## Out of Scope

- No authentication
- No mobile support
`
	parsed := ParseSpecContent("test", content)

	assert.Len(t, parsed.OutOfScope, 2)
	assert.Equal(t, "No authentication", parsed.OutOfScope[0])
	assert.Equal(t, "No mobile support", parsed.OutOfScope[1])
}

func TestParseSpecContent_Verification(t *testing.T) {
	content := `## Verification

- All tests pass
- E2E flow works
`
	parsed := ParseSpecContent("test", content)

	assert.Len(t, parsed.Verification, 2)
	assert.Equal(t, "All tests pass", parsed.Verification[0])
	assert.Equal(t, "E2E flow works", parsed.Verification[1])
}

func TestParseSpecContent_MultipleSection(t *testing.T) {
	content := `# Spec: full-spec

## Status: draft

## Tasks

- [ ] task-1: Build feature
- [ ] task-2: Write tests

## Out of Scope

- No i18n

## Verification

- Run all tests
`
	parsed := ParseSpecContent("full-spec", content)

	assert.Equal(t, "full-spec", parsed.Name)
	assert.Len(t, parsed.Tasks, 2)
	assert.Len(t, parsed.OutOfScope, 1)
	assert.Len(t, parsed.Verification, 1)
}

func TestFindNextTask_ReturnsFirstNotCompleted(t *testing.T) {
	tasks := []ParsedTask{
		{ID: "task-1", Title: "First"},
		{ID: "task-2", Title: "Second"},
		{ID: "task-3", Title: "Third"},
	}

	next := FindNextTask(tasks, []string{"task-1"})
	assert.NotNil(t, next)
	assert.Equal(t, "task-2", next.ID)
}

func TestFindNextTask_AllCompleted(t *testing.T) {
	tasks := []ParsedTask{
		{ID: "task-1", Title: "First"},
		{ID: "task-2", Title: "Second"},
	}

	next := FindNextTask(tasks, []string{"task-1", "task-2"})
	assert.Nil(t, next)
}

func TestFindNextTask_EmptyTasks(t *testing.T) {
	next := FindNextTask([]ParsedTask{}, []string{})
	assert.Nil(t, next)
}

func TestFindNextTask_NoCompleted(t *testing.T) {
	tasks := []ParsedTask{
		{ID: "task-1", Title: "First"},
		{ID: "task-2", Title: "Second"},
	}

	next := FindNextTask(tasks, []string{})
	assert.NotNil(t, next)
	assert.Equal(t, "task-1", next.ID)
}
