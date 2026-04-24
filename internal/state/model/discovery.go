package model

type DiscoveryAnswer struct {
	QuestionID string `json:"questionId"`
	Answer     string `json:"answer"`
}

// AttributedDiscoveryAnswer is an extended discovery answer with attribution.
// Old format (just questionId+answer) still works via NormalizeAnswer.
type AttributedDiscoveryAnswer struct {
	QuestionID string  `json:"questionId"`
	Answer     string  `json:"answer"`
	User       string  `json:"user"`
	Email      string  `json:"email"`
	Timestamp  string  `json:"timestamp"`
	Type       string  `json:"type"`
	Confidence *int    `json:"confidence,omitempty"`
	Basis      *string `json:"basis,omitempty"`
}

// ConfidenceFinding is a confidence-scored finding from agent analysis.
type ConfidenceFinding struct {
	Finding    string `json:"finding"`
	Confidence int    `json:"confidence"`
	Basis      string `json:"basis"`
}

type Premise struct {
	Text      string  `json:"text"`
	Agreed    bool    `json:"agreed"`
	Revision  *string `json:"revision,omitempty"`
	User      string  `json:"user"`
	Timestamp string  `json:"timestamp"`
}

type SelectedApproach struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Summary   string `json:"summary"`
	Effort    string `json:"effort"`
	Risk      string `json:"risk"`
	User      string `json:"user"`
	Timestamp string `json:"timestamp"`
}

type FollowUp struct {
	ID               string  `json:"id"`
	ParentQuestionID string  `json:"parentQuestionId"`
	Question         string  `json:"question"`
	Answer           *string `json:"answer"`
	Status           string  `json:"status"`
	CreatedBy        string  `json:"createdBy"`
	CreatedAt        string  `json:"createdAt"`
	AnsweredAt       *string `json:"answeredAt,omitempty"`
}

type Delegation struct {
	QuestionID  string  `json:"questionId"`
	DelegatedTo string  `json:"delegatedTo"`
	DelegatedBy string  `json:"delegatedBy"`
	Status      string  `json:"status"`
	DelegatedAt string  `json:"delegatedAt"`
	Answer      *string `json:"answer,omitempty"`
	AnsweredBy  *string `json:"answeredBy,omitempty"`
	AnsweredAt  *string `json:"answeredAt,omitempty"`
}

type DiscoveryPrefillItem struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Basis string `json:"basis"`
}

type DiscoveryPrefillQuestion struct {
	QuestionID string                 `json:"questionId"`
	Items      []DiscoveryPrefillItem `json:"items"`
}

type DiscoveryState struct {
	Answers               []DiscoveryAnswer          `json:"answers"`
	Prefills              []DiscoveryPrefillQuestion `json:"prefills,omitempty"`
	Completed             bool                       `json:"completed"`
	CurrentQuestion       int                        `json:"currentQuestion"`
	Audience              string                     `json:"audience"`
	Approved              bool                       `json:"approved"`
	Mode                  *DiscoveryMode             `json:"mode,omitempty"`
	Premises              []Premise                  `json:"premises,omitempty"`
	SelectedApproach      *SelectedApproach          `json:"selectedApproach,omitempty"`
	PremisesCompleted     *bool                      `json:"premisesCompleted,omitempty"`
	AlternativesPresented *bool                      `json:"alternativesPresented,omitempty"`
	Contributors          []string                   `json:"contributors,omitempty"`
	Delegations           []Delegation               `json:"delegations,omitempty"`
	FollowUps             []FollowUp                 `json:"followUps,omitempty"`
	UserContext           *string                    `json:"userContext,omitempty"`
	UserContextProcessed  *bool                      `json:"userContextProcessed,omitempty"`
	BatchSubmitted        *bool                      `json:"batchSubmitted,omitempty"`
}

// UserInfo carries optional attribution data for mutations that record who
// performed an action. Lives in model because both state_file and service
// layers reference it.
type UserInfo struct {
	Name  string
	Email string
}
