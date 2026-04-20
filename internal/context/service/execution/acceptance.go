package execution

import (
	"fmt"
	"strings"

	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/spec"
	"github.com/pragmataW/tddmaster/internal/state"
)

// isACRelevant reports whether a concern-generated AC is relevant for the
// active spec classification. Documentation-oriented ACs are filtered by the
// same classification rules as other ACs; the mandatory "Documentation
// updated" criterion is appended unconditionally by buildAcceptanceCriteria.
func isACRelevant(acText string, classification *state.SpecClassification) bool {
	if classification == nil {
		return false
	}

	lower := strings.ToLower(acText)

	if strings.Contains(lower, "mobile") || strings.Contains(lower, "layout") ||
		strings.Contains(lower, "interaction design") {
		return classification.InvolvesWebUI
	}
	if strings.Contains(lower, "ui state") || strings.Contains(lower, "skeleton ui") {
		return classification.InvolvesWebUI || classification.InvolvesCLI
	}
	if strings.Contains(lower, "api doc") || strings.Contains(lower, "public api") {
		return classification.InvolvesPublicAPI
	}
	if strings.Contains(lower, "migration") || strings.Contains(lower, "backward compat") ||
		strings.Contains(lower, "deprecat") {
		return classification.InvolvesMigration
	}
	if strings.Contains(lower, "audit trail") || strings.Contains(lower, "access control") ||
		strings.Contains(lower, "data handling") || strings.Contains(lower, "data retention") {
		return classification.InvolvesDataHandling
	}
	return true
}

func buildAcceptanceCriteria(
	activeConcerns []state.ConcernDefinition,
	verifyFailed bool,
	verifyOutput string,
	debt *state.DebtState,
	classification *state.SpecClassification,
	parsedSpec *spec.ParsedSpec,
	naItems []string,
) []model.AcceptanceCriterion {
	var criteria []model.AcceptanceCriterion
	naSet := make(map[string]bool, len(naItems))
	for _, id := range naItems {
		naSet[id] = true
	}

	acCounter := 0
	nextID := func() string {
		acCounter++
		return fmt.Sprintf("ac-%d", acCounter)
	}

	if debt != nil {
		for _, item := range debt.Items {
			if naSet[item.ID] {
				continue
			}
			criteria = append(criteria, model.AcceptanceCriterion{
				ID:   item.ID,
				Text: fmt.Sprintf("[DEBT from iteration %d] %s", item.Since, item.Text),
			})
		}
	}

	if verifyFailed {
		truncated := verifyOutput
		if len(truncated) > model.VerificationOutputTruncateShort {
			truncated = truncated[:model.VerificationOutputTruncateShort]
		}
		criteria = append(criteria, model.AcceptanceCriterion{
			ID:   nextID(),
			Text: "[FAILED] Tests — fix this first: " + truncated,
		})
	}

	if parsedSpec != nil {
		for _, item := range parsedSpec.Verification {
			id := nextID()
			if naSet[id] {
				continue
			}
			criteria = append(criteria, model.AcceptanceCriterion{ID: id, Text: item})
		}
	}

	for _, concern := range activeConcerns {
		for _, ac := range concern.AcceptanceCriteria {
			if !isACRelevant(ac, classification) {
				continue
			}
			id := nextID()
			if naSet[id] {
				continue
			}
			criteria = append(criteria, model.AcceptanceCriterion{
				ID:   id,
				Text: fmt.Sprintf("(%s) %s", concern.ID, ac),
			})
		}
	}

	if parsedSpec != nil {
		for _, t := range parsedSpec.Tasks {
			if len(t.Files) > 0 {
				criteria = append(criteria, model.AcceptanceCriterion{
					ID:   "scope-check",
					Text: fmt.Sprintf("Scope check: only files listed in task (%s) should be modified. Report any out-of-scope changes with justification.", strings.Join(t.Files, ", ")),
				})
				break
			}
		}
	}

	criteria = append(criteria,
		model.AcceptanceCriterion{ID: "mandatory-tests", Text: "Tests written and passing for all new and changed behavior"},
		model.AcceptanceCriterion{ID: "mandatory-docs", Text: "Documentation updated for all public-facing changes"},
	)

	return criteria
}
