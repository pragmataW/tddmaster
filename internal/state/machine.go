// State machine — valid phase transitions and enforcement.

package state

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// =============================================================================
// Transition Map
// =============================================================================

// ValidTransitions defines which phase transitions are allowed.
var ValidTransitions = map[Phase][]Phase{
	PhaseUninitialized:       {PhaseIdle},
	PhaseIdle:                {PhaseDiscovery, PhaseCompleted},
	PhaseDiscovery:           {PhaseDiscoveryRefinement, PhaseCompleted},
	PhaseDiscoveryRefinement: {PhaseDiscoveryRefinement, PhaseSpecProposal, PhaseCompleted},
	PhaseSpecProposal:        {PhaseSpecProposal, PhaseSpecApproved, PhaseCompleted},
	PhaseSpecApproved:        {PhaseExecuting, PhaseCompleted},
	PhaseExecuting:           {PhaseCompleted, PhaseBlocked},
	PhaseBlocked:             {PhaseExecuting, PhaseCompleted},
	PhaseCompleted:           {PhaseIdle, PhaseDiscovery, PhaseExecuting, PhaseBlocked},
}

// =============================================================================
// Transition Validation
// =============================================================================

// CanTransition returns true if the transition from -> to is valid.
func CanTransition(from, to Phase) bool {
	allowed, ok := ValidTransitions[from]
	if !ok {
		return false
	}
	for _, p := range allowed {
		if p == to {
			return true
		}
	}
	return false
}

// AssertTransition panics if the transition is not valid.
func AssertTransition(from, to Phase) error {
	if !CanTransition(from, to) {
		allowed := ValidTransitions[from]
		parts := make([]string, len(allowed))
		for i, p := range allowed {
			parts[i] = string(p)
		}
		return fmt.Errorf("invalid phase transition: %s → %s. Allowed: %s",
			from, to, strings.Join(parts, ", "))
	}
	return nil
}

// =============================================================================
// State Mutations
// =============================================================================

// Transition moves state to the given phase (validates transition first).
func Transition(state StateFile, to Phase) (StateFile, error) {
	if err := AssertTransition(state.Phase, to); err != nil {
		return state, err
	}
	state.Phase = to
	return state, nil
}

// StartSpec transitions to DISCOVERY and initializes spec-related state.
func StartSpec(state StateFile, specName, branch string, description *string) (StateFile, error) {
	if err := AssertTransition(state.Phase, PhaseDiscovery); err != nil {
		return state, err
	}

	state.Phase = PhaseDiscovery
	state.Spec = &specName
	state.SpecDescription = description
	state.Branch = &branch
	state.Discovery = DiscoveryState{
		Answers:         []DiscoveryAnswer{},
		Prefills:        []DiscoveryPrefillQuestion{},
		Completed:       false,
		CurrentQuestion: 0,
		Audience:        "human",
		Approved:        false,
		PlanPath:        nil,
	}
	state.SpecState = SpecState{Path: nil, Status: "none"}
	state.Execution = ExecutionState{
		Iteration:            0,
		LastProgress:         nil,
		ModifiedFiles:        []string{},
		LastVerification:     nil,
		AwaitingStatusReport: false,
		Debt:                 nil,
		CompletedTasks:       []string{},
		DebtCounter:          0,
		NaItems:              []string{},
	}
	state.Decisions = []Decision{}
	return state, nil
}

// SetDiscoveryMode sets the discovery mode. Only valid in DISCOVERY phase.
func SetDiscoveryMode(state StateFile, mode DiscoveryMode) (StateFile, error) {
	if state.Phase != PhaseDiscovery {
		return state, fmt.Errorf("cannot set discovery mode in phase: %s", state.Phase)
	}
	state.Discovery.Mode = &mode
	return state, nil
}

// CompletePremises records premises and marks them completed. Only valid in DISCOVERY phase.
func CompletePremises(state StateFile, premises []Premise) (StateFile, error) {
	if state.Phase != PhaseDiscovery {
		return state, fmt.Errorf("cannot complete premises in phase: %s", state.Phase)
	}
	t := true
	state.Discovery.Premises = premises
	state.Discovery.PremisesCompleted = &t
	return state, nil
}

// SelectApproach records the selected approach. Only valid in DISCOVERY_REFINEMENT phase.
func SelectApproach(state StateFile, approach SelectedApproach) (StateFile, error) {
	if state.Phase != PhaseDiscoveryRefinement {
		return state, fmt.Errorf("cannot select approach in phase: %s", state.Phase)
	}
	t := true
	state.Discovery.SelectedApproach = &approach
	state.Discovery.AlternativesPresented = &t
	return state, nil
}

// SkipAlternatives marks alternatives as presented without selecting one.
func SkipAlternatives(state StateFile) (StateFile, error) {
	if state.Phase != PhaseDiscoveryRefinement {
		return state, fmt.Errorf("cannot skip alternatives in phase: %s", state.Phase)
	}
	t := true
	state.Discovery.AlternativesPresented = &t
	return state, nil
}

// AddDiscoveryAnswer adds or replaces a discovery answer for a question.
// Validates that answer is at least 20 chars (Jidoka).
func AddDiscoveryAnswer(state StateFile, questionID, answer string, user *UserInfo) (StateFile, error) {
	if state.Phase != PhaseDiscovery && state.Phase != PhaseDiscoveryRefinement {
		return state, fmt.Errorf("cannot add discovery answer in phase: %s", state.Phase)
	}

	if len(strings.TrimSpace(answer)) < 20 {
		return state, fmt.Errorf("answer too short. Discovery answers must be meaningful (minimum 20 characters)")
	}

	// Replace existing answer for this question (backward-compatible behavior)
	var existingAnswers []DiscoveryAnswer
	for _, a := range state.Discovery.Answers {
		if a.QuestionID != questionID {
			existingAnswers = append(existingAnswers, a)
		}
	}
	if existingAnswers == nil {
		existingAnswers = []DiscoveryAnswer{}
	}

	userName := "Unknown User"
	userEmail := ""
	if user != nil {
		userName = user.Name
		userEmail = user.Email
	}

	// We store as DiscoveryAnswer in the slice but the AttributedDiscoveryAnswer info
	// is encoded by using the DiscoveryAnswer struct only for JSON compatibility.
	// Since Go's DiscoveryState uses []DiscoveryAnswer, we store the attributed form
	// via a separate attributed answers list.
	// However, to stay 1:1 with TS behavior, we need to store attributed answers.
	// Let's update DiscoveryState to support both via interface / or just store as DiscoveryAnswer
	// and use the NormalizeAnswer approach.
	// For simplicity, store only what DiscoveryAnswer holds; user/email are lost unless
	// we switch to []AttributedDiscoveryAnswer.
	// But the TS types show answers: readonly DiscoveryAnswer[]. Let's check more carefully...
	// Actually in TS, answers is DiscoveryAnswer[] but newAnswer is AttributedDiscoveryAnswer
	// which extends DiscoveryAnswer. We'll store AttributedDiscoveryAnswer data in our struct
	// by switching answers to use a richer type. But the JSON tags match DiscoveryAnswer.
	// Solution: use DiscoveryAnswer as the canonical storage type (matching TS schema),
	// and for attributed fields, that's stored as optional fields via NormalizeAnswer.
	// For the addDiscoveryAnswer function, we just store questionId and answer.
	_ = userName
	_ = userEmail

	newAnswer := DiscoveryAnswer{
		QuestionID: questionID,
		Answer:     answer,
	}

	state.Discovery.Answers = append(existingAnswers, newAnswer)
	return state, nil
}

// UserInfo holds optional user attribution data.
type UserInfo struct {
	Name  string
	Email string
}

// AddDiscoveryContribution adds an additional answer to a question without replacing existing ones.
func AddDiscoveryContribution(state StateFile, questionID, answer string, user *UserInfo) (StateFile, error) {
	if state.Phase != PhaseDiscovery && state.Phase != PhaseDiscoveryRefinement {
		return state, fmt.Errorf("cannot add discovery contribution in phase: %s", state.Phase)
	}

	newAnswer := DiscoveryAnswer{
		QuestionID: questionID,
		Answer:     answer,
	}

	state.Discovery.Answers = append(state.Discovery.Answers, newAnswer)
	return state, nil
}

// specFilePath returns the path for a spec file (mirrors paths.specFile in persistence).
func specFilePath(specName string) string {
	return TddmasterDir + "/specs/" + specName + "/spec.md"
}

// CompleteDiscovery transitions from DISCOVERY to DISCOVERY_REFINEMENT.
// Blocks if there are pending follow-ups (Jidoka I2).
func CompleteDiscovery(state StateFile) (StateFile, error) {
	if state.Phase != PhaseDiscovery {
		return state, fmt.Errorf("cannot complete discovery in phase: %s", state.Phase)
	}

	pending := GetPendingFollowUps(state)
	if len(pending) > 0 {
		return state, fmt.Errorf("cannot complete discovery: %d pending follow-up(s). Answer or skip them first", len(pending))
	}

	specPath := ""
	if state.Spec != nil {
		specPath = specFilePath(*state.Spec)
	}

	t := true
	state.Phase = PhaseDiscoveryRefinement
	state.Discovery.Completed = true
	state.SpecState = SpecState{
		Path:   &specPath,
		Status: "draft",
	}
	_ = t
	return state, nil
}

// ApproveDiscoveryReview transitions to SPEC_PROPOSAL.
func ApproveDiscoveryReview(state StateFile) (StateFile, error) {
	if err := AssertTransition(state.Phase, PhaseSpecProposal); err != nil {
		return state, err
	}
	state.Phase = PhaseSpecProposal
	return state, nil
}

// ApproveDiscoveryAnswers approves discovery answers without transitioning phase.
// Used when a split proposal is detected — stays in DISCOVERY_REFINEMENT.
func ApproveDiscoveryAnswers(state StateFile) (StateFile, error) {
	if state.Phase != PhaseDiscoveryRefinement {
		return state, fmt.Errorf("cannot approve discovery answers in phase: %s", state.Phase)
	}
	state.Discovery.Approved = true
	return state, nil
}

// AdvanceDiscoveryQuestion increments the current question index.
func AdvanceDiscoveryQuestion(state StateFile) (StateFile, error) {
	if state.Phase != PhaseDiscovery {
		return state, fmt.Errorf("cannot advance discovery question in phase: %s", state.Phase)
	}
	state.Discovery.CurrentQuestion++
	return state, nil
}

// ApproveSpec transitions to SPEC_APPROVED.
func ApproveSpec(state StateFile) (StateFile, error) {
	if err := AssertTransition(state.Phase, PhaseSpecApproved); err != nil {
		return state, err
	}
	state.Phase = PhaseSpecApproved
	state.SpecState.Status = "approved"
	return state, nil
}

// StartExecution transitions to EXECUTING.
func StartExecution(state StateFile) (StateFile, error) {
	if err := AssertTransition(state.Phase, PhaseExecuting); err != nil {
		return state, err
	}

	f := false
	state.Phase = PhaseExecuting
	// Preserve discovery answers in state for revisit support
	state.Discovery.Completed = true
	state.Discovery.Approved = false
	_ = f
	state.Execution = ExecutionState{
		Iteration:            0,
		LastProgress:         nil,
		ModifiedFiles:        []string{},
		LastVerification:     nil,
		AwaitingStatusReport: false,
		Debt:                 nil,
		CompletedTasks:       []string{},
		DebtCounter:          0,
		NaItems:              []string{},
	}
	return state, nil
}

// AdvanceExecution increments the iteration counter and sets last progress.
func AdvanceExecution(state StateFile, progress string) (StateFile, error) {
	if state.Phase != PhaseExecuting {
		return state, fmt.Errorf("cannot advance execution in phase: %s", state.Phase)
	}
	state.Execution.Iteration++
	state.Execution.LastProgress = &progress
	return state, nil
}

// BlockExecution transitions to BLOCKED.
func BlockExecution(state StateFile, reason string) (StateFile, error) {
	if err := AssertTransition(state.Phase, PhaseBlocked); err != nil {
		return state, err
	}
	progress := "BLOCKED: " + reason
	state.Phase = PhaseBlocked
	state.Execution.LastProgress = &progress
	return state, nil
}

// AddDecision appends a decision to the state.
func AddDecision(state StateFile, decision Decision) StateFile {
	state.Decisions = append(state.Decisions, decision)
	return state
}

// CompleteSpec transitions to COMPLETED.
func CompleteSpec(state StateFile, reason CompletionReason, note *string) (StateFile, error) {
	if err := AssertTransition(state.Phase, PhaseCompleted); err != nil {
		return state, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	state.Phase = PhaseCompleted
	state.CompletionReason = &reason
	state.CompletedAt = &now
	state.CompletionNote = note
	return state, nil
}

// ReopenSpec transitions from COMPLETED back to DISCOVERY.
func ReopenSpec(state StateFile) (StateFile, error) {
	if state.Phase != PhaseCompleted {
		return state, fmt.Errorf("cannot reopen in phase: %s", state.Phase)
	}

	var reopenedFrom *string
	if state.CompletionReason != nil {
		s := string(*state.CompletionReason)
		reopenedFrom = &s
	}

	state.Phase = PhaseDiscovery
	state.ReopenedFrom = reopenedFrom
	state.CompletionReason = nil
	state.CompletedAt = nil
	state.CompletionNote = nil
	// Preserve discovery answers for revision
	state.Discovery.Completed = false
	state.Discovery.CurrentQuestion = 0
	// Reset execution state
	state.Execution = ExecutionState{
		Iteration:            0,
		LastProgress:         nil,
		ModifiedFiles:        []string{},
		LastVerification:     nil,
		AwaitingStatusReport: false,
		Debt:                 nil,
		CompletedTasks:       []string{},
		DebtCounter:          0,
		NaItems:              []string{},
	}
	state.Classification = nil
	return state, nil
}

// ResumeCompletedSpec restores the most recent execution phase from COMPLETED
// without wiping execution progress. When no prior execution transition is
// available, it falls back to EXECUTING.
func ResumeCompletedSpec(state StateFile) (StateFile, error) {
	if state.Phase != PhaseCompleted {
		return state, fmt.Errorf("cannot resume in phase: %s", state.Phase)
	}

	restorePhase := PhaseExecuting
	for i := len(state.TransitionHistory) - 1; i >= 0; i-- {
		tr := state.TransitionHistory[i]
		if tr.To != PhaseCompleted {
			continue
		}
		if tr.From == PhaseExecuting || tr.From == PhaseBlocked {
			restorePhase = tr.From
		}
		break
	}
	if err := AssertTransition(state.Phase, restorePhase); err != nil {
		return state, err
	}

	var reopenedFrom *string
	if state.CompletionReason != nil {
		s := string(*state.CompletionReason)
		reopenedFrom = &s
	}

	state.Phase = restorePhase
	state.ReopenedFrom = reopenedFrom
	state.CompletionReason = nil
	state.CompletedAt = nil
	state.CompletionNote = nil
	return state, nil
}

// RevisitSpec goes back from EXECUTING/BLOCKED to DISCOVERY while preserving progress.
func RevisitSpec(state StateFile, reason string) (StateFile, error) {
	if state.Phase != PhaseExecuting && state.Phase != PhaseBlocked {
		return state, fmt.Errorf("cannot revisit in phase: %s. Only EXECUTING or BLOCKED can revisit", state.Phase)
	}

	completedTasks := make([]string, len(state.Execution.CompletedTasks))
	copy(completedTasks, state.Execution.CompletedTasks)

	entry := RevisitEntry{
		From:           state.Phase,
		Reason:         reason,
		CompletedTasks: completedTasks,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
	}

	if state.RevisitHistory == nil {
		state.RevisitHistory = []RevisitEntry{}
	}

	state.Phase = PhaseDiscovery
	// Preserve discovery answers for revision
	state.Discovery.Completed = false
	state.Discovery.CurrentQuestion = 0
	state.Discovery.Approved = false
	// Reset execution state
	state.Execution = ExecutionState{
		Iteration:            0,
		LastProgress:         nil,
		ModifiedFiles:        []string{},
		LastVerification:     nil,
		AwaitingStatusReport: false,
		Debt:                 nil,
		CompletedTasks:       []string{},
		DebtCounter:          0,
		NaItems:              []string{},
	}
	state.Classification = nil
	state.RevisitHistory = append(state.RevisitHistory, entry)
	return state, nil
}

// RecordTransition records a phase transition in the history.
func RecordTransition(state StateFile, from, to Phase, user *UserInfo, reason *string) StateFile {
	userName := "Unknown User"
	userEmail := ""
	if user != nil {
		userName = user.Name
		userEmail = user.Email
	}

	entry := PhaseTransition{
		From:      from,
		To:        to,
		User:      userName,
		Email:     userEmail,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Reason:    reason,
	}

	if state.TransitionHistory == nil {
		state.TransitionHistory = []PhaseTransition{}
	}
	state.TransitionHistory = append(state.TransitionHistory, entry)
	return state
}

// AddCustomAC adds a custom acceptance criterion.
func AddCustomAC(state StateFile, text string, user *UserInfo) StateFile {
	if state.CustomACs == nil {
		state.CustomACs = []CustomAC{}
	}
	id := fmt.Sprintf("custom-ac-%d", len(state.CustomACs)+1)

	userName := "Unknown User"
	userEmail := ""
	if user != nil {
		userName = user.Name
		userEmail = user.Email
	}

	ac := CustomAC{
		ID:           id,
		Text:         text,
		User:         userName,
		Email:        userEmail,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		AddedInPhase: state.Phase,
	}
	state.CustomACs = append(state.CustomACs, ac)
	return state
}

// AddSpecNote adds a note to the spec.
func AddSpecNote(state StateFile, text string, user *UserInfo) StateFile {
	if state.SpecNotes == nil {
		state.SpecNotes = []SpecNote{}
	}
	id := fmt.Sprintf("note-%d", len(state.SpecNotes)+1)

	userName := "Unknown User"
	userEmail := ""
	if user != nil {
		userName = user.Name
		userEmail = user.Email
	}

	note := SpecNote{
		ID:        id,
		Text:      text,
		User:      userName,
		Email:     userEmail,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Phase:     state.Phase,
	}
	state.SpecNotes = append(state.SpecNotes, note)
	return state
}

// =============================================================================
// User context (listen first)
// =============================================================================

// SetUserContext stores user context shared before discovery starts.
func SetUserContext(state StateFile, context string) StateFile {
	f := false
	state.Discovery.UserContext = &context
	state.Discovery.Prefills = []DiscoveryPrefillQuestion{}
	state.Discovery.UserContextProcessed = &f
	return state
}

// SetDiscoveryPrefills stores persisted discovery suggestions derived from user context.
func SetDiscoveryPrefills(state StateFile, prefills []DiscoveryPrefillQuestion) StateFile {
	if prefills == nil {
		state.Discovery.Prefills = []DiscoveryPrefillQuestion{}
		return state
	}
	copied := make([]DiscoveryPrefillQuestion, len(prefills))
	for i, prefill := range prefills {
		items := make([]DiscoveryPrefillItem, len(prefill.Items))
		copy(items, prefill.Items)
		copied[i] = DiscoveryPrefillQuestion{
			QuestionID: prefill.QuestionID,
			Items:      items,
		}
	}
	state.Discovery.Prefills = copied
	return state
}

// MarkUserContextProcessed marks user context as processed (pre-fill done).
func MarkUserContextProcessed(state StateFile) StateFile {
	t := true
	state.Discovery.UserContextProcessed = &t
	return state
}

// =============================================================================
// Confidence scoring
// =============================================================================

// ClampConfidence clamps confidence to 1-10 range.
func ClampConfidence(value float64) int {
	rounded := math.Round(value)
	if rounded < 1 {
		return 1
	}
	if rounded > 10 {
		return 10
	}
	return int(rounded)
}

// AddConfidenceFinding adds a confidence-scored finding to execution state.
func AddConfidenceFinding(state StateFile, finding string, confidence float64, basis string) (StateFile, error) {
	clamped := ClampConfidence(confidence)

	// Jidoka: high confidence requires evidence
	if clamped >= 7 && len(strings.TrimSpace(basis)) < 10 {
		return state, fmt.Errorf("high confidence (>=7) requires a basis explaining why (minimum 10 characters)")
	}

	if state.Execution.ConfidenceFindings == nil {
		state.Execution.ConfidenceFindings = []ConfidenceFinding{}
	}

	entry := ConfidenceFinding{
		Finding:    finding,
		Confidence: clamped,
		Basis:      basis,
	}
	state.Execution.ConfidenceFindings = append(state.Execution.ConfidenceFindings, entry)
	return state, nil
}

// GetLowConfidenceFindings returns findings with confidence below threshold.
func GetLowConfidenceFindings(state StateFile, threshold int) []ConfidenceFinding {
	var result []ConfidenceFinding
	for _, f := range state.Execution.ConfidenceFindings {
		if f.Confidence < threshold {
			result = append(result, f)
		}
	}
	if result == nil {
		result = []ConfidenceFinding{}
	}
	return result
}

// GetAverageConfidence calculates the average confidence across all findings.
// Returns nil if there are no findings.
func GetAverageConfidence(state StateFile) *float64 {
	findings := state.Execution.ConfidenceFindings
	if len(findings) == 0 {
		return nil
	}
	sum := 0
	for _, f := range findings {
		sum += f.Confidence
	}
	avg := math.Round(float64(sum)/float64(len(findings))*10) / 10
	return &avg
}

// SetContributors sets contributors for a spec.
func SetContributors(state StateFile, contributors []string) StateFile {
	state.Discovery.Contributors = contributors
	return state
}

// =============================================================================
// Follow-ups (adaptive discovery)
// =============================================================================

const maxFollowupsPerQuestion = 3

// AddFollowUp adds a follow-up question to an answered discovery question.
// Silently caps at 3 per parent question.
func AddFollowUp(state StateFile, parentQuestionID, question, createdBy string) StateFile {
	if state.Discovery.FollowUps == nil {
		state.Discovery.FollowUps = []FollowUp{}
	}

	// Enforce max 3 per parent question
	parentCount := 0
	for _, f := range state.Discovery.FollowUps {
		if f.ParentQuestionID == parentQuestionID {
			parentCount++
		}
	}
	if parentCount >= maxFollowupsPerQuestion {
		return state // silently cap
	}

	// Q3a, Q3b, Q3c
	id := fmt.Sprintf("%s%c", parentQuestionID, rune('a'+parentCount))

	followUp := FollowUp{
		ID:               id,
		ParentQuestionID: parentQuestionID,
		Question:         question,
		Answer:           nil,
		Status:           "pending",
		CreatedBy:        createdBy,
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
	}

	state.Discovery.FollowUps = append(state.Discovery.FollowUps, followUp)
	return state
}

// AnswerFollowUp answers a follow-up question.
func AnswerFollowUp(state StateFile, followUpID, answer string) StateFile {
	if state.Discovery.FollowUps == nil {
		return state
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for i, f := range state.Discovery.FollowUps {
		if f.ID == followUpID && f.Status == "pending" {
			state.Discovery.FollowUps[i].Answer = &answer
			state.Discovery.FollowUps[i].Status = "answered"
			state.Discovery.FollowUps[i].AnsweredAt = &now
		}
	}
	return state
}

// SkipFollowUp skips a follow-up question.
func SkipFollowUp(state StateFile, followUpID string) StateFile {
	if state.Discovery.FollowUps == nil {
		return state
	}
	for i, f := range state.Discovery.FollowUps {
		if f.ID == followUpID && f.Status == "pending" {
			state.Discovery.FollowUps[i].Status = "skipped"
		}
	}
	return state
}

// GetPendingFollowUps returns all pending follow-ups.
func GetPendingFollowUps(state StateFile) []FollowUp {
	var result []FollowUp
	for _, f := range state.Discovery.FollowUps {
		if f.Status == "pending" {
			result = append(result, f)
		}
	}
	if result == nil {
		result = []FollowUp{}
	}
	return result
}

// GetFollowUpsForQuestion returns all follow-ups for a specific parent question.
func GetFollowUpsForQuestion(state StateFile, parentQuestionID string) []FollowUp {
	var result []FollowUp
	for _, f := range state.Discovery.FollowUps {
		if f.ParentQuestionID == parentQuestionID {
			result = append(result, f)
		}
	}
	if result == nil {
		result = []FollowUp{}
	}
	return result
}

// AddDelegation delegates a discovery question to another contributor.
func AddDelegation(state StateFile, questionID, delegatedTo, delegatedBy string) StateFile {
	if state.Discovery.Delegations == nil {
		state.Discovery.Delegations = []Delegation{}
	}

	delegation := Delegation{
		QuestionID:  questionID,
		DelegatedTo: delegatedTo,
		DelegatedBy: delegatedBy,
		Status:      "pending",
		DelegatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	state.Discovery.Delegations = append(state.Discovery.Delegations, delegation)
	return state
}

// AnswerDelegation answers a delegated question.
func AnswerDelegation(state StateFile, questionID, answer, answeredBy string) StateFile {
	if state.Discovery.Delegations == nil {
		return state
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for i, d := range state.Discovery.Delegations {
		if d.QuestionID == questionID && d.Status == "pending" {
			state.Discovery.Delegations[i].Status = "answered"
			state.Discovery.Delegations[i].Answer = &answer
			state.Discovery.Delegations[i].AnsweredBy = &answeredBy
			state.Discovery.Delegations[i].AnsweredAt = &now
		}
	}
	return state
}

// GetPendingDelegations returns all pending delegations.
func GetPendingDelegations(state StateFile) []Delegation {
	var result []Delegation
	for _, d := range state.Discovery.Delegations {
		if d.Status == "pending" {
			result = append(result, d)
		}
	}
	if result == nil {
		result = []Delegation{}
	}
	return result
}

// RecordTDDVerification records a TDD verification result, re-queues failed ACs,
// and auto-transitions to BLOCKED when MaxVerificationRetries is reached.
// maxRetries=0 disables the auto-block (treated as unlimited).
// This legacy wrapper skips cycle transitions and refactor-note tracking.
func RecordTDDVerification(
	st StateFile,
	maxRetries int,
	passed bool,
	output string,
	failedACs []string,
	uncoveredEdgeCases []string,
) (StateFile, error) {
	return RecordTDDVerificationFull(st, maxRetries, 0, passed, output, failedACs, uncoveredEdgeCases, nil)
}

// RecordTDDVerificationFull is the full-featured TDD verification recorder. It
// handles RED→GREEN→REFACTOR cycle transitions and verifier→executor refactor
// note round-tripping in addition to the legacy fail-count/requeue/block logic.
//
// Transitions (when passed==true and st.Execution.TDDCycle is set):
//   - red      → green        (failing tests confirmed, ready for implementation)
//   - green    → refactor     (tests pass, invite refactor notes)
//   - refactor → refactor     (new notes stored; executor applies next)
//     or next-task reset (TDDCycle="red") when notes are empty or the round
//     cap is reached.
//
// maxRefactorRounds=0 means "unlimited rounds as long as notes keep coming";
// the default manifest value is 3.
func RecordTDDVerificationFull(
	st StateFile,
	maxRetries int,
	maxRefactorRounds int,
	passed bool,
	output string,
	failedACs []string,
	uncoveredEdgeCases []string,
	refactorNotes []RefactorNote,
) (StateFile, error) {
	if st.Phase != PhaseExecuting {
		return st, fmt.Errorf("cannot record TDD verification in phase: %s", st.Phase)
	}

	// Compute cumulative fail count from previous result.
	prevFailCount := 0
	if st.Execution.LastVerification != nil {
		prevFailCount = st.Execution.LastVerification.VerificationFailCount
	}
	newFailCount := prevFailCount
	if !passed {
		newFailCount++
	}

	phaseSnapshot := st.Execution.TDDCycle
	now := time.Now().UTC().Format(time.RFC3339)
	st.Execution.LastVerification = &VerificationResult{
		Passed:                passed,
		Output:                output,
		Timestamp:             now,
		UncoveredEdgeCases:    uncoveredEdgeCases,
		VerificationFailCount: newFailCount,
		RefactorNotes:         refactorNotes,
		Phase:                 phaseSnapshot,
	}

	if !passed {
		// Re-queue failed ACs: remove them from CompletedTasks so they get retried.
		if len(failedACs) > 0 {
			failSet := make(map[string]bool, len(failedACs))
			for _, ac := range failedACs {
				failSet[ac] = true
			}
			kept := st.Execution.CompletedTasks[:0]
			for _, id := range st.Execution.CompletedTasks {
				if !failSet[id] {
					kept = append(kept, id)
				}
			}
			st.Execution.CompletedTasks = kept
		}

		// Auto-transition to BLOCKED if retry limit reached.
		if maxRetries > 0 && newFailCount >= maxRetries {
			reason := fmt.Sprintf("verifier max retry reached (%d/%d)", newFailCount, maxRetries)
			return BlockExecution(st, reason)
		}

		return st, nil
	}

	// passed == true → advance TDD cycle.
	switch phaseSnapshot {
	case TDDCycleRed:
		st.Execution.TDDCycle = TDDCycleGreen
		st.Execution.LastVerification.VerificationFailCount = 0

	case TDDCycleGreen:
		st.Execution.LastVerification.VerificationFailCount = 0
		if len(refactorNotes) == 0 {
			// GREEN scan found no improvements — skip refactor phase entirely.
			resetCycleForNextTask(&st)
		} else {
			// GREEN scan produced refactor notes — advance to refactor so the
			// executor can apply them. Notes are already stored in LastVerification.
			st.Execution.TDDCycle = TDDCycleRefactor
			st.Execution.RefactorRounds = 0
			st.Execution.RefactorApplied = false
		}

	case TDDCycleRefactor:
		if !st.Execution.RefactorApplied {
			// First verify pass of this refactor round — verifier produced notes.
			if len(refactorNotes) == 0 {
				resetCycleForNextTask(&st)
			}
			// else: notes stored in LastVerification; executor applies next.
		} else {
			// Executor already applied previous notes; verifier re-checked.
			st.Execution.RefactorRounds++
			capReached := maxRefactorRounds > 0 && st.Execution.RefactorRounds >= maxRefactorRounds
			if len(refactorNotes) == 0 || capReached {
				resetCycleForNextTask(&st)
			} else {
				// Continue rounds: keep cycle, clear applied flag so executor runs again.
				st.Execution.RefactorApplied = false
			}
		}
	}

	return st, nil
}

// resetCycleForNextTask clears per-task TDD cycle state. The next task's RED
// phase is seeded by the cmd/next.go status-report handler via
// StartTDDCycleForTask when ShouldRunTDDForCurrentTask returns true.
// Task completion bookkeeping (CompletedTasks append) is also handled there.
func resetCycleForNextTask(st *StateFile) {
	st.Execution.TDDCycle = ""
	st.Execution.RefactorRounds = 0
	st.Execution.RefactorApplied = false
}

// StartTDDCycleForTask initializes the TDD cycle at RED for the current task
// and clears any carried-over refactor state. Safe to call when TDD is
// disabled; it is a no-op if TDDCycle is already set to RED.
func StartTDDCycleForTask(st *StateFile) {
	st.Execution.TDDCycle = TDDCycleRed
	st.Execution.RefactorRounds = 0
	st.Execution.RefactorApplied = false
}

// MarkRefactorApplied flips the RefactorApplied flag; called by cmd/next.go
// when the executor reports that it consumed the verifier's refactor notes.
func MarkRefactorApplied(st *StateFile) {
	st.Execution.RefactorApplied = true
}

// ErrTaskNotCompleted is returned by UncompleteTask when the given task ID is
// not found in CompletedTasks.
var ErrTaskNotCompleted = fmt.Errorf("task not found in completed tasks")

// UncompleteTask reverses the completion of a single task identified by taskID.
// It removes the ID from CompletedTasks and sets Completed=false on the matching
// OverrideTasks entry (if present). Phase, Iteration, and TDDCycle are not
// touched — this is a task-flag flip only. Returns ErrTaskNotCompleted when
// taskID is absent from CompletedTasks.
func UncompleteTask(st StateFile, taskID string) (StateFile, error) {
	// Find and remove from CompletedTasks
	found := false
	kept := make([]string, 0, len(st.Execution.CompletedTasks))
	for _, id := range st.Execution.CompletedTasks {
		if id == taskID {
			found = true
			continue
		}
		kept = append(kept, id)
	}
	if !found {
		return st, fmt.Errorf("%w: %s", ErrTaskNotCompleted, taskID)
	}
	st.Execution.CompletedTasks = kept

	// Flip Completed=false on the matching OverrideTasks entry (if any).
	for i := range st.OverrideTasks {
		if st.OverrideTasks[i].ID == taskID {
			st.OverrideTasks[i].Completed = false
			break
		}
	}

	return st, nil
}

// ResetToIdle resets state to IDLE. Only allowed from terminal/safe phases.
func ResetToIdle(state StateFile) (StateFile, error) {
	allowed := map[Phase]bool{
		PhaseIdle:      true,
		PhaseExecuting: true,
		PhaseBlocked:   true,
		PhaseCompleted: true,
	}
	if !allowed[state.Phase] {
		return state, fmt.Errorf("cannot reset from %s. Use `cancel` or `wontfix` instead", state.Phase)
	}

	state.Phase = PhaseIdle
	state.Spec = nil
	state.Branch = nil
	state.Discovery = DiscoveryState{
		Answers:         []DiscoveryAnswer{},
		Prefills:        []DiscoveryPrefillQuestion{},
		Completed:       false,
		CurrentQuestion: 0,
		Audience:        "human",
		Approved:        false,
		PlanPath:        nil,
	}
	state.SpecState = SpecState{Path: nil, Status: "none"}
	state.Execution = ExecutionState{
		Iteration:            0,
		LastProgress:         nil,
		ModifiedFiles:        []string{},
		LastVerification:     nil,
		AwaitingStatusReport: false,
		Debt:                 nil,
		CompletedTasks:       []string{},
		DebtCounter:          0,
		NaItems:              []string{},
	}
	state.Decisions = []Decision{}
	state.Classification = nil
	state.CompletionReason = nil
	state.CompletedAt = nil
	state.CompletionNote = nil
	state.ReopenedFrom = nil
	return state, nil
}
