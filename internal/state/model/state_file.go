package model

import (
	"encoding/json"
	"fmt"
)

// StateFile is the JSON shape persisted at .tddmaster/.state/state.json and
// each .tddmaster/.state/specs/<name>.json. It is the central serialization
// contract for tddmaster — every cmd/* read/write goes through this struct.
type StateFile struct {
	Version            string              `json:"version"`
	Phase              Phase               `json:"phase"`
	Spec               *string             `json:"spec"`
	SpecDescription    *string             `json:"specDescription"`
	Branch             *string             `json:"branch"`
	Discovery          DiscoveryState      `json:"discovery"`
	SpecState          SpecState           `json:"specState"`
	Execution          ExecutionState      `json:"execution"`
	Decisions          []Decision          `json:"decisions"`
	LastCalledAt       *string             `json:"lastCalledAt"`
	Classification     *SpecClassification `json:"classification"`
	CompletionReason   *CompletionReason   `json:"completionReason"`
	CompletedAt        *string             `json:"completedAt"`
	CompletionNote     *string             `json:"completionNote"`
	ReopenedFrom       *string             `json:"reopenedFrom"`
	RevisitHistory     []RevisitEntry      `json:"revisitHistory"`
	TransitionHistory  []PhaseTransition   `json:"transitionHistory,omitempty"`
	CustomACs          []CustomAC          `json:"customACs,omitempty"`
	SpecNotes          []SpecNote          `json:"specNotes,omitempty"`
	OverrideTasks      []SpecTask          `json:"overrideTasks,omitempty"`
	OverrideOutOfScope []string            `json:"overrideOutOfScope,omitempty"`
	TaskTDDSelected    *bool               `json:"taskTDDSelected,omitempty"`
	LastAnswer         *AnswerFingerprint  `json:"lastAnswer,omitempty"`
}

// AnswerFingerprint records the last successfully processed answer so that
// re-submissions (error retries, duplicate deliveries) are idempotent within
// the same phase.
type AnswerFingerprint struct {
	Phase     Phase  `json:"phase"`
	Hash      string `json:"hash"`
	Timestamp string `json:"timestamp"`
}

// UnmarshalJSON handles backward-compatible migration of OverrideTasks from
// []string (old format) to []SpecTask (new format). Old state files with
// string arrays are automatically upgraded on first read.
func (s *StateFile) UnmarshalJSON(data []byte) error {
	type stateFileAlias StateFile
	aux := &struct {
		OverrideTasks json.RawMessage `json:"overrideTasks,omitempty"`
		*stateFileAlias
	}{
		stateFileAlias: (*stateFileAlias)(s),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if len(aux.OverrideTasks) == 0 || string(aux.OverrideTasks) == "null" {
		s.OverrideTasks = nil
		return nil
	}
	var tasks []SpecTask
	if err := json.Unmarshal(aux.OverrideTasks, &tasks); err == nil {
		s.OverrideTasks = tasks
		return nil
	}
	var strs []string
	if err := json.Unmarshal(aux.OverrideTasks, &strs); err != nil {
		return fmt.Errorf("overrideTasks: cannot unmarshal as []SpecTask or []string: %w", err)
	}
	s.OverrideTasks = make([]SpecTask, len(strs))
	for i, title := range strs {
		s.OverrideTasks[i] = SpecTask{
			ID:        fmt.Sprintf("task-%d", i+1),
			Title:     title,
			Completed: false,
		}
	}
	return nil
}
