package execution

import (
	"testing"

	"github.com/pragmataW/tddmaster/internal/state"
	"github.com/stretchr/testify/assert"
)

func TestIsACRelevant_NilClassificationRejectsAllConcernACs(t *testing.T) {
	cases := []string{
		"README updated",
		"Documentation updated for public endpoints",
		"Changes reflected in the docs site",
		"Mobile layout verified",
		"API doc published",
		"Migration plan reviewed",
	}
	for _, ac := range cases {
		assert.False(t, isACRelevant(ac, nil), "nil classification must reject %q", ac)
	}
}

func TestIsACRelevant_MobileACRespectsWebUIClassification(t *testing.T) {
	mobileAC := "Mobile layout verified on iOS"
	assert.False(t, isACRelevant(mobileAC, &state.SpecClassification{InvolvesWebUI: false}),
		"mobile/layout ACs must be rejected when InvolvesWebUI is false")
	assert.True(t, isACRelevant(mobileAC, &state.SpecClassification{InvolvesWebUI: true}),
		"mobile/layout ACs must pass when InvolvesWebUI is true")
}

func TestIsACRelevant_ApiDocRespectsPublicAPIClassification(t *testing.T) {
	apiAC := "API doc updated for new endpoint"
	assert.False(t, isACRelevant(apiAC, &state.SpecClassification{InvolvesPublicAPI: false}),
		"api-doc ACs must be rejected when InvolvesPublicAPI is false")
	assert.True(t, isACRelevant(apiAC, &state.SpecClassification{InvolvesPublicAPI: true}),
		"api-doc ACs must pass when InvolvesPublicAPI is true")
}

func TestIsACRelevant_MigrationRespectsMigrationClassification(t *testing.T) {
	migAC := "Backward compat verified for old consumers"
	assert.False(t, isACRelevant(migAC, &state.SpecClassification{InvolvesMigration: false}))
	assert.True(t, isACRelevant(migAC, &state.SpecClassification{InvolvesMigration: true}))
}

func TestIsACRelevant_GenericACDefaultsToTrueForNonNilClassification(t *testing.T) {
	genericAC := "Team communication updated in the release channel"
	assert.True(t, isACRelevant(genericAC, &state.SpecClassification{}),
		"ACs that do not match any classification keyword default to relevant")
}

func TestBuildAcceptanceCriteria_MandatoryDocsAlwaysAppended(t *testing.T) {
	criteria := buildAcceptanceCriteria(nil, false, "", nil, &state.SpecClassification{}, nil, nil)

	var ids []string
	for _, c := range criteria {
		ids = append(ids, c.ID)
	}
	assert.Contains(t, ids, "mandatory-docs", "mandatory-docs AC must always be appended")
	assert.Contains(t, ids, "mandatory-tests", "mandatory-tests AC must always be appended")
}
