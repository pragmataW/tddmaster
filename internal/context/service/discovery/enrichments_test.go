package discovery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildPlanContext_EmptyPathReturnsNil(t *testing.T) {
	assert.Nil(t, buildPlanContext(""))
}

func TestBuildPlanContext_MissingFileReturnsNil(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "does-not-exist.md")
	assert.Nil(t, buildPlanContext(missing))
}

func TestBuildPlanContext_SmallPlanIsEmbedded(t *testing.T) {
	dir := t.TempDir()
	planPath := filepath.Join(dir, "plan.md")
	body := "short plan body"
	require.NoError(t, os.WriteFile(planPath, []byte(body), 0o644))

	got := buildPlanContext(planPath)
	require.NotNil(t, got)
	assert.True(t, got.Provided, "plan below MaxPlanSize must be embedded")
	assert.Equal(t, body, got.Content)
	assert.Equal(t, model.PlanContextInstruction, got.Instruction)
}

func TestBuildPlanContext_OversizedPlanReturnsVisibleSignal(t *testing.T) {
	dir := t.TempDir()
	planPath := filepath.Join(dir, "plan.md")
	oversized := strings.Repeat("x", model.MaxPlanSize+1)
	require.NoError(t, os.WriteFile(planPath, []byte(oversized), 0o644))

	got := buildPlanContext(planPath)
	require.NotNil(t, got, "oversized plan must still produce a PlanContext with a visible signal")
	assert.False(t, got.Provided, "oversized plan must not embed content")
	assert.Empty(t, got.Content, "oversized plan must not leak content into the context")
	assert.Equal(t, model.PlanContextOversizedInstruction, got.Instruction)
	assert.NotEmpty(t, got.Instruction, "oversized plan must carry an instruction so the agent knows to read the file")
}
