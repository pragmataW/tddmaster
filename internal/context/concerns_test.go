
package context_test

import (
	"testing"

	ctx "github.com/pragmataW/tddmaster/internal/context"
	"github.com/pragmataW/tddmaster/internal/defaults"
	"github.com/pragmataW/tddmaster/internal/state"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Helpers
// =============================================================================

func allConcernsFixture() []state.ConcernDefinition {
	return defaults.DefaultConcerns()
}

func findConcern(concerns []state.ConcernDefinition, id string) state.ConcernDefinition {
	for _, c := range concerns {
		if c.ID == id {
			return c
		}
	}
	panic("concern not found: " + id)
}

// =============================================================================
// GetConcernExtras
// =============================================================================

func TestGetConcernExtras_EmptyForNoConcerns(t *testing.T) {
	result := ctx.GetConcernExtras([]state.ConcernDefinition{}, "status_quo")
	assert.Len(t, result, 0)
}

func TestGetConcernExtras_ReturnsMatchingExtras(t *testing.T) {
	allConcerns := allConcernsFixture()
	openSource := findConcern(allConcerns, "open-source")

	result := ctx.GetConcernExtras([]state.ConcernDefinition{openSource}, "status_quo")

	assert.Len(t, result, 1)
	assert.Equal(t, "Is this workaround common in the community?", result[0].Text)
}

func TestGetConcernExtras_IgnoresNonMatchingQuestionIds(t *testing.T) {
	allConcerns := allConcernsFixture()
	openSource := findConcern(allConcerns, "open-source")

	result := ctx.GetConcernExtras([]state.ConcernDefinition{openSource}, "reversibility")
	assert.Len(t, result, 0)
}

// =============================================================================
// GetReminders
// =============================================================================

func TestGetReminders_PrefixesWithConcernId(t *testing.T) {
	allConcerns := allConcernsFixture()
	openSource := findConcern(allConcerns, "open-source")

	result := ctx.GetReminders([]state.ConcernDefinition{openSource}, nil)

	for _, r := range result {
		assert.Contains(t, r, "open-source: ")
	}
}

func TestGetReminders_FlattensAcrossMultipleConcerns(t *testing.T) {
	allConcerns := allConcernsFixture()
	openSource := findConcern(allConcerns, "open-source")
	moveFast := findConcern(allConcerns, "move-fast")

	result := ctx.GetReminders([]state.ConcernDefinition{openSource, moveFast}, nil)

	osCount := 0
	mfCount := 0
	for _, r := range result {
		if len(r) >= 12 && r[:12] == "open-source:" {
			osCount++
		}
		if len(r) >= 10 && r[:10] == "move-fast:" {
			mfCount++
		}
	}

	assert.Equal(t, len(openSource.Reminders), osCount)
	assert.Equal(t, len(moveFast.Reminders), mfCount)
	assert.Equal(t, osCount+mfCount, len(result))
}

// =============================================================================
// DetectTensions
// =============================================================================

func TestDetectTensions_EmptyWhenNoConflicts(t *testing.T) {
	allConcerns := allConcernsFixture()
	openSource := findConcern(allConcerns, "open-source")
	longLived := findConcern(allConcerns, "long-lived")

	result := ctx.DetectTensions([]state.ConcernDefinition{openSource, longLived})
	assert.Len(t, result, 0)
}

func TestDetectTensions_DetectsMoveVsCompliance(t *testing.T) {
	allConcerns := allConcernsFixture()
	moveFast := findConcern(allConcerns, "move-fast")
	compliance := findConcern(allConcerns, "compliance")

	result := ctx.DetectTensions([]state.ConcernDefinition{moveFast, compliance})

	assert.Len(t, result, 1)
	assert.Contains(t, result[0].Between, "move-fast")
	assert.Contains(t, result[0].Between, "compliance")
}

func TestDetectTensions_DetectsMoveVsLongLived(t *testing.T) {
	allConcerns := allConcernsFixture()
	moveFast := findConcern(allConcerns, "move-fast")
	longLived := findConcern(allConcerns, "long-lived")

	result := ctx.DetectTensions([]state.ConcernDefinition{moveFast, longLived})

	assert.Len(t, result, 1)
	assert.Contains(t, result[0].Between, "long-lived")
}

func TestDetectTensions_DetectsBeautifulVsMovefast(t *testing.T) {
	allConcerns := allConcernsFixture()
	beautiful := findConcern(allConcerns, "beautiful-product")
	moveFast := findConcern(allConcerns, "move-fast")

	result := ctx.DetectTensions([]state.ConcernDefinition{beautiful, moveFast})

	assert.Len(t, result, 1)
	assert.Contains(t, result[0].Between, "beautiful-product")
}

func TestDetectTensions_MultipleTensionsFromMultipleConcerns(t *testing.T) {
	allConcerns := allConcernsFixture()
	moveFast := findConcern(allConcerns, "move-fast")
	compliance := findConcern(allConcerns, "compliance")
	longLived := findConcern(allConcerns, "long-lived")
	beautiful := findConcern(allConcerns, "beautiful-product")

	result := ctx.DetectTensions([]state.ConcernDefinition{moveFast, compliance, longLived, beautiful})

	// move-fast conflicts with: compliance, long-lived, beautiful-product
	assert.Equal(t, 3, len(result))
}

func TestDetectTensions_WellEngineeredVsMovefast(t *testing.T) {
	allConcerns := allConcernsFixture()
	wellEngineered := findConcern(allConcerns, "well-engineered")
	moveFast := findConcern(allConcerns, "move-fast")

	result := ctx.DetectTensions([]state.ConcernDefinition{wellEngineered, moveFast})

	assert.Len(t, result, 1)
	assert.Contains(t, result[0].Between, "well-engineered")
	assert.Contains(t, result[0].Between, "move-fast")
}

// =============================================================================
// GetReviewDimensions
// =============================================================================

func TestGetReviewDimensions_CollectsFromAllConcerns(t *testing.T) {
	allConcerns := allConcernsFixture()
	longLived := findConcern(allConcerns, "long-lived")
	beautiful := findConcern(allConcerns, "beautiful-product")

	result := ctx.GetReviewDimensions([]state.ConcernDefinition{longLived, beautiful}, nil)

	assert.GreaterOrEqual(t, len(result), 10)

	hasLongLived := false
	hasBeautiful := false
	for _, d := range result {
		if d.ConcernID == "long-lived" {
			hasLongLived = true
		}
		if d.ConcernID == "beautiful-product" {
			hasBeautiful = true
		}
	}
	assert.True(t, hasLongLived)
	assert.True(t, hasBeautiful)
}

func TestGetReviewDimensions_MoveFastHasOneDimension(t *testing.T) {
	allConcerns := allConcernsFixture()
	moveFast := findConcern(allConcerns, "move-fast")

	result := ctx.GetReviewDimensions([]state.ConcernDefinition{moveFast}, nil)
	assert.Equal(t, 1, len(result))
}

func TestGetReviewDimensions_FiltersUIWhenNotInvolved(t *testing.T) {
	allConcerns := allConcernsFixture()
	beautiful := findConcern(allConcerns, "beautiful-product")

	classification := &state.SpecClassification{
		InvolvesWebUI:        false,
		InvolvesCLI:          false,
		InvolvesPublicAPI:    false,
		InvolvesMigration:    false,
		InvolvesDataHandling: false,
	}

	result := ctx.GetReviewDimensions([]state.ConcernDefinition{beautiful}, classification)
	// beautiful-product dimensions are all scope: "ui" — should be filtered out
	assert.Len(t, result, 0)
}

func TestGetReviewDimensions_IncludesAllWhenNilClassification(t *testing.T) {
	allConcerns := allConcernsFixture()
	beautiful := findConcern(allConcerns, "beautiful-product")

	result := ctx.GetReviewDimensions([]state.ConcernDefinition{beautiful}, nil)
	assert.GreaterOrEqual(t, len(result), 5)
}

// =============================================================================
// GetRegistryDimensionIds
// =============================================================================

func TestGetRegistryDimensionIds_CollectsFromConcerns(t *testing.T) {
	allConcerns := allConcernsFixture()
	longLived := findConcern(allConcerns, "long-lived")

	result := ctx.GetRegistryDimensionIds([]state.ConcernDefinition{longLived})

	assert.Contains(t, result, "error-rescue")
	assert.Contains(t, result, "failure-modes")
}

func TestGetRegistryDimensionIds_Deduplicates(t *testing.T) {
	allConcerns := allConcernsFixture()
	longLived := findConcern(allConcerns, "long-lived")

	// Create a mock concern with overlapping registry
	mockConcern := state.ConcernDefinition{
		ID:         "test-dup",
		Registries: []string{"error-rescue"},
	}

	result := ctx.GetRegistryDimensionIds([]state.ConcernDefinition{longLived, mockConcern})

	count := 0
	for _, id := range result {
		if id == "error-rescue" {
			count++
		}
	}
	assert.Equal(t, 1, count)
}

func TestGetRegistryDimensionIds_EmptyForConcernsWithoutRegistries(t *testing.T) {
	allConcerns := allConcernsFixture()
	moveFast := findConcern(allConcerns, "move-fast")
	beautiful := findConcern(allConcerns, "beautiful-product")

	result := ctx.GetRegistryDimensionIds([]state.ConcernDefinition{moveFast, beautiful})
	assert.Len(t, result, 0)
}

// =============================================================================
// GetDreamStatePrompts
// =============================================================================

func TestGetDreamStatePrompts_CollectsPrompts(t *testing.T) {
	allConcerns := allConcernsFixture()
	longLived := findConcern(allConcerns, "long-lived")

	result := ctx.GetDreamStatePrompts([]state.ConcernDefinition{longLived})

	assert.Len(t, result, 1)
	assert.Contains(t, result[0], "CURRENT STATE")
}

func TestGetDreamStatePrompts_EmptyForConcernsWithoutPrompts(t *testing.T) {
	allConcerns := allConcernsFixture()
	moveFast := findConcern(allConcerns, "move-fast")
	compliance := findConcern(allConcerns, "compliance")

	result := ctx.GetDreamStatePrompts([]state.ConcernDefinition{moveFast, compliance})
	assert.Len(t, result, 0)
}
