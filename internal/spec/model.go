package spec

import "time"

const (
	PhaseInitial = "listen-first"
	StatusDraft  = "draft"
)

type State struct {
	Version   int                 `json:"version"`
	Slug      string              `json:"slug"`
	Phase     string              `json:"phase"`
	Answers   map[string][]Answer `json:"answers"`
	CreatedAt time.Time           `json:"createdAt"`
	UpdatedAt time.Time           `json:"updatedAt"`
}

type Answer struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Settings struct {
	TDDEnabled               bool `json:"tddEnabled"`
	SkipVerifierEnabled      bool `json:"skipVerifierEnabled"`
	ImportantTaskGateEnabled bool `json:"importantTaskGateEnabled"`
}

type Progress struct {
	Spec      string    `json:"spec"`
	Status    string    `json:"status"`
	Tasks     []Task    `json:"tasks"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Task struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	AC         []string `json:"ac"`
	Done       bool     `json:"done"`
	TDDEnabled bool     `json:"tddEnabled"`
	Important  bool     `json:"important"`
}

func DefaultSettings() Settings {
	return Settings{TDDEnabled: true, SkipVerifierEnabled: false, ImportantTaskGateEnabled: false}
}
