package model

import (
	"bytes"
	"encoding/json"
)

type ConcernExtra struct {
	QuestionID string `json:"questionId"`
	Text       string `json:"text"`
}

type ReviewDimensionScope string

const (
	ReviewDimensionScopeAll  ReviewDimensionScope = "all"
	ReviewDimensionScopeUI   ReviewDimensionScope = "ui"
	ReviewDimensionScopeAPI  ReviewDimensionScope = "api"
	ReviewDimensionScopeData ReviewDimensionScope = "data"
)

type ReviewDimension struct {
	ID               string               `json:"id"`
	Label            string               `json:"label"`
	Prompt           string               `json:"prompt"`
	EvidenceRequired bool                 `json:"evidenceRequired"`
	Scope            ReviewDimensionScope `json:"scope"`
}

// ConcernReminderScope enumerates the surfaces a concern reminder applies to.
// A reminder with an empty scope list is a general (tier1) reminder; a reminder
// with any scope entry is tier2 and filtered per target file or classification.
type ConcernReminderScope string

const (
	ConcernReminderScopeUI        ConcernReminderScope = "ui"
	ConcernReminderScopeAPI       ConcernReminderScope = "api"
	ConcernReminderScopeMigration ConcernReminderScope = "migration"
)

// ConcernReminder is one reminder attached to a concern definition.
type ConcernReminder struct {
	Text  string                 `json:"text"`
	Scope []ConcernReminderScope `json:"scope,omitempty"`
}

func (r ConcernReminder) HasScope() bool {
	return len(r.Scope) > 0
}

func (r ConcernReminder) AppliesToScope(target ConcernReminderScope) bool {
	for _, s := range r.Scope {
		if s == target {
			return true
		}
	}
	return false
}

type ConcernDefinition struct {
	ID                 string            `json:"id"`
	Name               string            `json:"name"`
	Description        string            `json:"description"`
	Extras             []ConcernExtra    `json:"extras"`
	SpecSections       []string          `json:"specSections"`
	Reminders          []ConcernReminder `json:"reminders"`
	AcceptanceCriteria []string          `json:"acceptanceCriteria"`
	ReviewDimensions   []ReviewDimension `json:"reviewDimensions,omitempty"`
	Registries         []string          `json:"registries,omitempty"`
	DreamStatePrompt   *string           `json:"dreamStatePrompt,omitempty"`
}

// UnmarshalJSON accepts both the new `[]ConcernReminder` object form and the
// legacy `[]string` form for the `reminders` field. A bare string is promoted
// to ConcernReminder{Text: s} with no scope (tier1).
func (c *ConcernDefinition) UnmarshalJSON(data []byte) error {
	type concernAlias ConcernDefinition
	aux := &struct {
		Reminders json.RawMessage `json:"reminders"`
		*concernAlias
	}{
		concernAlias: (*concernAlias)(c),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if len(aux.Reminders) == 0 || string(aux.Reminders) == "null" {
		c.Reminders = nil
		return nil
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(aux.Reminders, &raw); err != nil {
		return err
	}
	c.Reminders = make([]ConcernReminder, 0, len(raw))
	for _, elem := range raw {
		trimmed := bytes.TrimSpace(elem)
		if len(trimmed) == 0 {
			continue
		}
		if trimmed[0] == '{' {
			var rem ConcernReminder
			if err := json.Unmarshal(elem, &rem); err != nil {
				return err
			}
			c.Reminders = append(c.Reminders, rem)
			continue
		}
		var s string
		if err := json.Unmarshal(elem, &s); err != nil {
			return err
		}
		c.Reminders = append(c.Reminders, ConcernReminder{Text: s})
	}
	return nil
}
