
// Dashboard state — builds a unified view of all specs, events, and user state.

package dashboard

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/pragmataW/tddmaster/internal/spec"
	"github.com/pragmataW/tddmaster/internal/state"
)

// =============================================================================
// Types
// =============================================================================

// User represents a user identity.
type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Question represents a pending question on a spec.
type Question struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	User string `json:"user"`
	Ts   string `json:"ts"`
}

// SpecTask represents a task in a spec summary.
type SpecTask struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Done        bool     `json:"done"`
	Files       []string `json:"files,omitempty"`
}

// SpecSummary holds a summary view of a spec.
type SpecSummary struct {
	Name                string                `json:"name"`
	Slug                string                `json:"slug"`
	Phase               state.Phase           `json:"phase"`
	Description         string                `json:"description"`
	Tasks               []SpecTask            `json:"tasks"`
	Contributors        []string              `json:"contributors"`
	Delegations         []state.Delegation    `json:"delegations"`
	PendingQuestions    []Question            `json:"pendingQuestions"`
	PendingSignoffs     []string              `json:"pendingSignoffs"`
	Roadmap             string                `json:"roadmap"`
	CreatedAt           string                `json:"createdAt"`
	UpdatedAt           string                `json:"updatedAt"`
	AvgConfidence       *float64              `json:"avgConfidence"`
	LowConfidenceItems  int                   `json:"lowConfidenceItems"`
}

// Mention represents a pending mention.
type Mention struct {
	ID       string `json:"id"`
	Spec     string `json:"spec"`
	From     string `json:"from"`
	To       string `json:"to"`
	Question string `json:"question"`
	Status   string `json:"status"` // "pending" | "replied"
	Reply    string `json:"reply,omitempty"`
	Ts       string `json:"ts"`
}

// SignoffEntry represents a pending signoff.
type SignoffEntry struct {
	Spec   string `json:"spec"`
	Role   string `json:"role"`
	Status string `json:"status"` // "pending" | "signed"
	User   string `json:"user,omitempty"`
	Ts     string `json:"ts,omitempty"`
}

// RoleMap maps role names to lists of users.
type RoleMap map[string][]string

// DashboardState holds the full dashboard view.
type DashboardState struct {
	Specs           []SpecSummary    `json:"specs"`
	ActiveSpec      *SpecSummary     `json:"activeSpec"`
	PendingMentions []Mention        `json:"pendingMentions"`
	PendingSignoffs []SignoffEntry   `json:"pendingSignoffs"`
	RecentEvents    []DashboardEvent `json:"recentEvents"`
	CurrentUser     *User            `json:"currentUser"`
	Roles           RoleMap          `json:"roles"`
}

// =============================================================================
// Roadmap builder
// =============================================================================

var roadmapSteps = []string{
	"IDLE",
	"DISCOVERY",
	"REVIEW",
	"DRAFT",
	"APPROVED",
	"EXECUTING",
	"DONE",
	"IDLE",
}

var phaseToRoadmap = map[string]string{
	"DISCOVERY_REFINEMENT": "REVIEW",
	"SPEC_PROPOSAL":        "DRAFT",
	"SPEC_APPROVED":        "APPROVED",
	"COMPLETED":            "DONE",
	"BLOCKED":              "EXECUTING",
}

func buildRoadmap(phase state.Phase) string {
	mapped, ok := phaseToRoadmap[string(phase)]
	if !ok {
		mapped = string(phase)
	}

	var parts []string
	for _, p := range roadmapSteps {
		if p == mapped {
			parts = append(parts, "[ "+p+" ]")
		} else {
			parts = append(parts, p)
		}
	}

	result := ""
	for i, p := range parts {
		if i > 0 {
			result += " → "
		}
		result += p
	}
	return result
}

// =============================================================================
// State Builder
// =============================================================================

// GetSpecSummary builds a SpecSummary from persisted state.
func GetSpecSummary(root, specName string) (SpecSummary, error) {
	s, err := state.ResolveState(root, &specName)
	if err != nil {
		s, _ = state.ReadState(root)
	}

	// Parse spec.md for tasks
	parsed, _ := spec.ParseSpec(root, specName)

	completedSet := make(map[string]bool)
	for _, t := range s.Execution.CompletedTasks {
		completedSet[t] = true
	}

	var tasks []SpecTask
	if parsed != nil {
		for _, t := range parsed.Tasks {
			st := SpecTask{
				ID:          t.ID,
				Description: t.Title,
				Done:        completedSet[t.ID],
			}
			if len(t.Files) > 0 {
				st.Files = t.Files
			}
			tasks = append(tasks, st)
		}
	}
	if tasks == nil {
		tasks = []SpecTask{}
	}

	// Extract contributors from discovery answers
	seen := make(map[string]bool)
	var contributors []string
	for _, a := range s.Discovery.Answers {
		// DiscoveryAnswer doesn't carry user directly; skip "Unknown User"
		_ = a
	}
	// Also check attributed answers via TransitionHistory users
	for _, t := range s.TransitionHistory {
		if t.User != "" && t.User != "Unknown User" && !seen[t.User] {
			seen[t.User] = true
			contributors = append(contributors, t.User)
		}
	}
	if contributors == nil {
		contributors = []string{}
	}

	// Extract pending questions from notes
	var pendingQuestions []Question
	for _, n := range s.SpecNotes {
		if len(n.Text) > 11 && n.Text[:11] == "[QUESTION] " {
			pendingQuestions = append(pendingQuestions, Question{
				ID:   n.ID,
				Text: n.Text[11:],
				User: n.User,
				Ts:   n.Timestamp,
			})
		}
	}
	if pendingQuestions == nil {
		pendingQuestions = []Question{}
	}

	// Determine timestamps
	createdAt := ""
	if len(s.TransitionHistory) > 0 {
		createdAt = s.TransitionHistory[0].Timestamp
	}
	if createdAt == "" {
		// Use a zero-value placeholder; TS used new Date().toISOString()
		createdAt = "1970-01-01T00:00:00Z"
	}
	updatedAt := createdAt
	if s.LastCalledAt != nil && *s.LastCalledAt != "" {
		updatedAt = *s.LastCalledAt
	}

	// Confidence scoring
	findings := s.Execution.ConfidenceFindings
	var avgConfidence *float64
	lowConfidenceItems := 0
	if len(findings) > 0 {
		sum := 0.0
		for _, f := range findings {
			sum += float64(f.Confidence)
			if f.Confidence < 5 {
				lowConfidenceItems++
			}
		}
		avg := float64(int(sum/float64(len(findings))*10)) / 10
		avgConfidence = &avg
	}

	description := ""
	if s.SpecDescription != nil {
		description = *s.SpecDescription
	}

	delegations := s.Discovery.Delegations
	if delegations == nil {
		delegations = []state.Delegation{}
	}

	return SpecSummary{
		Name:               specName,
		Slug:               specName,
		Phase:              s.Phase,
		Description:        description,
		Tasks:              tasks,
		Contributors:       contributors,
		Delegations:        delegations,
		PendingQuestions:   pendingQuestions,
		PendingSignoffs:    []string{},
		Roadmap:            buildRoadmap(s.Phase),
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
		AvgConfidence:      avgConfidence,
		LowConfidenceItems: lowConfidenceItems,
	}, nil
}

// GetState builds the full dashboard state.
func GetState(root string) (DashboardState, error) {
	// Load all specs
	specStates, _ := state.ListSpecStates(root)

	specNames := make(map[string]bool)
	for _, s := range specStates {
		specNames[s.Name] = true
	}

	// Also check spec directories without state files
	specsDir := filepath.Join(root, state.TddmasterDir, "specs")
	if dirEntries, err := os.ReadDir(specsDir); err == nil {
		for _, entry := range dirEntries {
			if entry.IsDir() && !specNames[entry.Name()] {
				specNames[entry.Name()] = true
			}
		}
	}

	var specs []SpecSummary
	for name := range specNames {
		summary, err := GetSpecSummary(root, name)
		if err != nil {
			continue
		}
		specs = append(specs, summary)
	}

	// Sort: active specs first, then by updatedAt
	sort.Slice(specs, func(i, j int) bool {
		aActive := specs[i].Phase != state.PhaseCompleted && specs[i].Phase != state.PhaseIdle
		bActive := specs[j].Phase != state.PhaseCompleted && specs[j].Phase != state.PhaseIdle
		if aActive && !bActive {
			return true
		}
		if !aActive && bActive {
			return false
		}
		return specs[i].UpdatedAt > specs[j].UpdatedAt
	})
	if specs == nil {
		specs = []SpecSummary{}
	}

	// Active spec = first non-completed, non-idle spec
	var activeSpec *SpecSummary
	for i := range specs {
		if specs[i].Phase != state.PhaseCompleted && specs[i].Phase != state.PhaseIdle {
			activeSpec = &specs[i]
			break
		}
	}

	// Recent events
	recentEvents, _ := ReadEvents(root, &ReadEventsOpts{Limit: 50})
	if recentEvents == nil {
		recentEvents = []DashboardEvent{}
	}

	// Pending mentions from events
	replyIDs := make(map[string]bool)
	for _, ev := range recentEvents {
		if ev.Type == EventTypeMentionReply {
			if id := ev.GetExtra("mentionId"); id != "" {
				replyIDs[id] = true
			}
		}
	}

	var pendingMentions []Mention
	for _, ev := range recentEvents {
		if ev.Type != EventTypeMention {
			continue
		}
		id := ev.GetExtra("id")
		if replyIDs[id] {
			continue
		}
		pendingMentions = append(pendingMentions, Mention{
			ID:       id,
			Spec:     ev.Spec,
			From:     ev.GetExtra("from"),
			To:       ev.GetExtra("to"),
			Question: ev.GetExtra("question"),
			Status:   "pending",
			Ts:       ev.Ts,
		})
	}
	if pendingMentions == nil {
		pendingMentions = []Mention{}
	}

	// Pending signoffs from events
	var pendingSignoffs []SignoffEntry
	for _, ev := range recentEvents {
		if ev.Type != EventTypeSignoff {
			continue
		}
		if ev.GetExtra("status") != "pending" {
			continue
		}
		pendingSignoffs = append(pendingSignoffs, SignoffEntry{
			Spec:   ev.Spec,
			Role:   ev.GetExtra("role"),
			Status: "pending",
			Ts:     ev.Ts,
		})
	}
	if pendingSignoffs == nil {
		pendingSignoffs = []SignoffEntry{}
	}

	// Current user
	u, _ := state.ResolveUser(root)
	var currentUser *User
	if u.Name != "" {
		currentUser = &User{Name: u.Name, Email: u.Email}
	}

	// Roles from manifest (NosManifest doesn't have a Roles field; reserved for future use)
	var roles RoleMap

	return DashboardState{
		Specs:           specs,
		ActiveSpec:      activeSpec,
		PendingMentions: pendingMentions,
		PendingSignoffs: pendingSignoffs,
		RecentEvents:    recentEvents,
		CurrentUser:     currentUser,
		Roles:           roles,
	}, nil
}
