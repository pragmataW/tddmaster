// Package state is the entry point for tddmaster's state subsystem. It splits
// into two sub-trees:
//
//   - internal/state/model     — pure data shapes (StateFile, Phase,
//                                DiscoveryState, ExecutionState, NosManifest,
//                                ConcernDefinition, Session). No I/O.
//   - internal/state/service/* — business logic grouped by concern:
//     paths, atomic, persistence, manifest, concerns, sessions, projectroot,
//     specflag, identity, machine, discovery, execution, tdd.
//
// This root package re-exports the type aliases and function wrappers that
// cmd/* and other internal packages (context, spec, sync) consume. Call-site
// imports stay on "internal/state" and the public API surface is preserved.
package state

import (
	"os"

	"github.com/pragmataW/tddmaster/internal/state/model"
	"github.com/pragmataW/tddmaster/internal/state/service/atomic"
	"github.com/pragmataW/tddmaster/internal/state/service/concerns"
	"github.com/pragmataW/tddmaster/internal/state/service/discovery"
	"github.com/pragmataW/tddmaster/internal/state/service/execution"
	"github.com/pragmataW/tddmaster/internal/state/service/identity"
	"github.com/pragmataW/tddmaster/internal/state/service/machine"
	"github.com/pragmataW/tddmaster/internal/state/service/manifest"
	"github.com/pragmataW/tddmaster/internal/state/service/paths"
	"github.com/pragmataW/tddmaster/internal/state/service/persistence"
	"github.com/pragmataW/tddmaster/internal/state/service/projectroot"
	"github.com/pragmataW/tddmaster/internal/state/service/sessions"
	"github.com/pragmataW/tddmaster/internal/state/service/specflag"
	"github.com/pragmataW/tddmaster/internal/state/service/tdd"
)

// -----------------------------------------------------------------------------
// Type aliases — data shapes
// -----------------------------------------------------------------------------

type (
	Phase            = model.Phase
	CompletionReason = model.CompletionReason
	DiscoveryMode    = model.DiscoveryMode

	DiscoveryAnswer           = model.DiscoveryAnswer
	AttributedDiscoveryAnswer = model.AttributedDiscoveryAnswer
	ConfidenceFinding         = model.ConfidenceFinding
	Premise                   = model.Premise
	SelectedApproach          = model.SelectedApproach
	FollowUp                  = model.FollowUp
	Delegation                = model.Delegation
	DiscoveryPrefillItem      = model.DiscoveryPrefillItem
	DiscoveryPrefillQuestion  = model.DiscoveryPrefillQuestion
	DiscoveryState            = model.DiscoveryState
	UserInfo                  = model.UserInfo

	RefactorNote       = model.RefactorNote
	VerificationResult = model.VerificationResult
	StatusReport       = model.StatusReport
	DebtItem           = model.DebtItem
	DebtState          = model.DebtState
	SpecTask           = model.SpecTask
	SpecClassification = model.SpecClassification
	ExecutionState     = model.ExecutionState
	SpecState          = model.SpecState
	Decision           = model.Decision
	RevisitEntry       = model.RevisitEntry
	PhaseTransition    = model.PhaseTransition
	CustomAC           = model.CustomAC
	SpecNote           = model.SpecNote

	StateFile         = model.StateFile
	AnswerFingerprint = model.AnswerFingerprint

	ProjectTraits = model.ProjectTraits
	CodingToolId  = model.CodingToolId
	UserConfig    = model.UserConfig
	Manifest      = model.Manifest
	NosManifest   = model.NosManifest

	ConcernExtra         = model.ConcernExtra
	ReviewDimensionScope = model.ReviewDimensionScope
	ReviewDimension      = model.ReviewDimension
	ConcernReminderScope = model.ConcernReminderScope
	ConcernReminder      = model.ConcernReminder
	ConcernDefinition    = model.ConcernDefinition

	Session        = model.Session
	SpecStateEntry = model.SpecStateEntry
	User           = model.User

	Paths                    = paths.Paths
	RequireSpecFlagResult    = specflag.RequireSpecFlagResult
	ResolveProjectRootResult = projectroot.ResolveProjectRootResult
)

// -----------------------------------------------------------------------------
// Enum re-exports
// -----------------------------------------------------------------------------

const (
	PhaseUninitialized       = model.PhaseUninitialized
	PhaseIdle                = model.PhaseIdle
	PhaseDiscovery           = model.PhaseDiscovery
	PhaseDiscoveryRefinement = model.PhaseDiscoveryRefinement
	PhaseSpecProposal        = model.PhaseSpecProposal
	PhaseSpecApproved        = model.PhaseSpecApproved
	PhaseExecuting           = model.PhaseExecuting
	PhaseBlocked             = model.PhaseBlocked
	PhaseCompleted           = model.PhaseCompleted

	CompletionReasonDone      = model.CompletionReasonDone
	CompletionReasonCancelled = model.CompletionReasonCancelled
	CompletionReasonWontfix   = model.CompletionReasonWontfix

	DiscoveryModeFull           = model.DiscoveryModeFull
	DiscoveryModeValidate       = model.DiscoveryModeValidate
	DiscoveryModeTechnicalDepth = model.DiscoveryModeTechnicalDepth
	DiscoveryModeShipFast       = model.DiscoveryModeShipFast
	DiscoveryModeExplore        = model.DiscoveryModeExplore

	TDDCycleRed      = model.TDDCycleRed
	TDDCycleGreen    = model.TDDCycleGreen
	TDDCycleRefactor = model.TDDCycleRefactor

	CodingToolClaudeCode = model.CodingToolClaudeCode
	CodingToolOpencode   = model.CodingToolOpencode
	CodingToolCodex      = model.CodingToolCodex

	ReviewDimensionScopeAll  = model.ReviewDimensionScopeAll
	ReviewDimensionScopeUI   = model.ReviewDimensionScopeUI
	ReviewDimensionScopeAPI  = model.ReviewDimensionScopeAPI
	ReviewDimensionScopeData = model.ReviewDimensionScopeData

	ConcernReminderScopeUI        = model.ConcernReminderScopeUI
	ConcernReminderScopeAPI       = model.ConcernReminderScopeAPI
	ConcernReminderScopeMigration = model.ConcernReminderScopeMigration

	TddmasterDir = paths.TddmasterDir
)

// ValidTransitions re-exports the phase transition table.
var ValidTransitions = model.ValidTransitions

// ErrTaskNotCompleted is returned by UncompleteTask when the given task ID is
// not present in CompletedTasks.
var ErrTaskNotCompleted = execution.ErrTaskNotCompleted

// -----------------------------------------------------------------------------
// Helper constructors
// -----------------------------------------------------------------------------

func NormalizeAnswer(a DiscoveryAnswer) AttributedDiscoveryAnswer {
	return discovery.NormalizeAnswer(a)
}

func NormalizeAttributedAnswer(a AttributedDiscoveryAnswer) AttributedDiscoveryAnswer {
	return discovery.NormalizeAttributedAnswer(a)
}

func GetAnswersForQuestion(answers []DiscoveryAnswer, questionID string) []AttributedDiscoveryAnswer {
	return discovery.GetAnswersForQuestion(answers, questionID)
}

func GetCombinedAnswer(answers []DiscoveryAnswer, questionID string) string {
	return discovery.GetCombinedAnswer(answers, questionID)
}

func GetPrefillsForQuestion(prefills []DiscoveryPrefillQuestion, questionID string) []DiscoveryPrefillItem {
	return discovery.GetPrefillsForQuestion(prefills, questionID)
}

// -----------------------------------------------------------------------------
// Persistence
// -----------------------------------------------------------------------------

func CreateInitialState() StateFile { return persistence.CreateInitialState() }

func ReadState(root string) (StateFile, error) { return persistence.ReadState(root) }

func WriteState(root string, s StateFile) error { return persistence.WriteState(root, s) }

func ResolveState(root string, specName *string) (StateFile, error) {
	return persistence.ResolveState(root, specName)
}

func ReadActiveSpec(root string) (*string, error) { return persistence.ReadActiveSpec(root) }

func ReadSpecState(root, specName string) (StateFile, error) {
	return persistence.ReadSpecState(root, specName)
}

func WriteSpecState(root, specName string, s StateFile) error {
	return persistence.WriteSpecState(root, specName, s)
}

func ListSpecStates(root string) ([]SpecStateEntry, error) { return persistence.ListSpecStates(root) }

func WriteStateAndSpec(root string, s StateFile) error { return persistence.WriteStateAndSpec(root, s) }

func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	return atomic.WriteFileAtomic(path, data, perm)
}

// -----------------------------------------------------------------------------
// Manifest
// -----------------------------------------------------------------------------

func ReadManifest(root string) (*NosManifest, error) { return manifest.ReadManifest(root) }

func WriteManifest(root string, config NosManifest) error {
	return manifest.WriteManifest(root, config)
}

func IsInitialized(root string) (bool, error) { return manifest.IsInitialized(root) }

func CreateInitialManifest(
	concerns []string,
	tools []CodingToolId,
	project ProjectTraits,
) NosManifest {
	return manifest.CreateInitialManifest(concerns, tools, project)
}

func ParseManifest(data []byte) (Manifest, error) { return manifest.ParseManifest(data) }
func MarshalManifest(m Manifest) ([]byte, error)  { return manifest.MarshalManifest(m) }
func LoadManifest(root string) (Manifest, error)  { return manifest.LoadManifest(root) }

// -----------------------------------------------------------------------------
// Concerns
// -----------------------------------------------------------------------------

func ReadConcern(root, concernID string) (*ConcernDefinition, error) {
	return concerns.ReadConcern(root, concernID)
}

func WriteConcern(root string, concern ConcernDefinition) error {
	return concerns.WriteConcern(root, concern)
}

func ListConcerns(root string) ([]ConcernDefinition, error) { return concerns.ListConcerns(root) }

// -----------------------------------------------------------------------------
// Sessions
// -----------------------------------------------------------------------------

func CreateSession(root string, session Session) error {
	return sessions.CreateSession(root, session)
}

func ReadSession(root, sessionID string) (*Session, error) {
	return sessions.ReadSession(root, sessionID)
}

func ListSessions(root string) ([]Session, error) { return sessions.ListSessions(root) }

func DeleteSession(root, sessionID string) (bool, error) {
	return sessions.DeleteSession(root, sessionID)
}

func UpdateSessionPhase(root, sessionID, phase string) error {
	return sessions.UpdateSessionPhase(root, sessionID, phase)
}

func GcStaleSessions(root string) ([]string, error) { return sessions.GcStaleSessions(root) }

func IsSessionStale(session Session) bool { return sessions.IsSessionStale(session) }

func GenerateSessionId() (string, error) { return sessions.GenerateSessionId() }

// -----------------------------------------------------------------------------
// Project root discovery
// -----------------------------------------------------------------------------

func FindProjectRoot(startDir string) (string, error) { return projectroot.FindProjectRoot(startDir) }

func ResolveProjectRoot() (ResolveProjectRootResult, error) { return projectroot.ResolveProjectRoot() }

func ScaffoldDir(root string) error { return projectroot.ScaffoldDir(root) }

// -----------------------------------------------------------------------------
// Spec flag parsing
// -----------------------------------------------------------------------------

func ParseSpecFlag(args []string) *string { return specflag.ParseSpecFlag(args) }

func RequireSpecFlag(args []string) RequireSpecFlagResult { return specflag.RequireSpecFlag(args) }

func UsesOldSpecFlag(args []string) bool { return specflag.UsesOldSpecFlag(args) }

// -----------------------------------------------------------------------------
// State machine
// -----------------------------------------------------------------------------

func CanTransition(from, to Phase) bool { return machine.CanTransition(from, to) }

func AssertTransition(from, to Phase) error { return machine.AssertTransition(from, to) }

func Transition(state StateFile, to Phase) (StateFile, error) {
	return machine.Transition(state, to)
}

func StartSpec(state StateFile, specName, branch string, description *string) (StateFile, error) {
	return machine.StartSpec(state, specName, branch, description)
}

func CompleteDiscovery(state StateFile) (StateFile, error) {
	return machine.CompleteDiscovery(state)
}

func ApproveDiscoveryReview(state StateFile) (StateFile, error) {
	return machine.ApproveDiscoveryReview(state)
}

func ApproveSpec(state StateFile) (StateFile, error) { return machine.ApproveSpec(state) }

func StartExecution(state StateFile) (StateFile, error) { return machine.StartExecution(state) }

func BlockExecution(state StateFile, reason string) (StateFile, error) {
	return machine.BlockExecution(state, reason)
}

func CompleteSpec(state StateFile, reason CompletionReason, note *string) (StateFile, error) {
	return machine.CompleteSpec(state, reason, note)
}

func ReopenSpec(state StateFile) (StateFile, error) { return machine.ReopenSpec(state) }

func ResumeCompletedSpec(state StateFile) (StateFile, error) {
	return machine.ResumeCompletedSpec(state)
}

func RevisitSpec(state StateFile, reason string) (StateFile, error) {
	return machine.RevisitSpec(state, reason)
}

func ResetToIdle(state StateFile) (StateFile, error) { return machine.ResetToIdle(state) }

// -----------------------------------------------------------------------------
// Discovery operations
// -----------------------------------------------------------------------------

func SetDiscoveryMode(state StateFile, mode DiscoveryMode) (StateFile, error) {
	return discovery.SetDiscoveryMode(state, mode)
}

func CompletePremises(state StateFile, premises []Premise) (StateFile, error) {
	return discovery.CompletePremises(state, premises)
}

func SelectApproach(state StateFile, approach SelectedApproach) (StateFile, error) {
	return discovery.SelectApproach(state, approach)
}

func SkipAlternatives(state StateFile) (StateFile, error) { return discovery.SkipAlternatives(state) }

func AddDiscoveryAnswer(state StateFile, questionID, answer string) (StateFile, error) {
	return discovery.AddDiscoveryAnswer(state, questionID, answer)
}

func AdvanceDiscoveryQuestion(state StateFile) (StateFile, error) {
	return discovery.AdvanceDiscoveryQuestion(state)
}

func ApproveDiscoveryAnswers(state StateFile) (StateFile, error) {
	return discovery.ApproveDiscoveryAnswers(state)
}

func SetUserContext(state StateFile, context string) StateFile {
	return discovery.SetUserContext(state, context)
}

func SetDiscoveryPrefills(state StateFile, prefills []DiscoveryPrefillQuestion) StateFile {
	return discovery.SetDiscoveryPrefills(state, prefills)
}

func MarkUserContextProcessed(state StateFile) StateFile {
	return discovery.MarkUserContextProcessed(state)
}

func SetContributors(state StateFile, contributors []string) StateFile {
	return discovery.SetContributors(state, contributors)
}

// -----------------------------------------------------------------------------
// Follow-ups and delegations
// -----------------------------------------------------------------------------

func AddFollowUp(state StateFile, parentQuestionID, question, createdBy string) StateFile {
	return discovery.AddFollowUp(state, parentQuestionID, question, createdBy)
}

func AnswerFollowUp(state StateFile, followUpID, answer string) StateFile {
	return discovery.AnswerFollowUp(state, followUpID, answer)
}

func SkipFollowUp(state StateFile, followUpID string) StateFile {
	return discovery.SkipFollowUp(state, followUpID)
}

func GetPendingFollowUps(state StateFile) []FollowUp { return discovery.GetPendingFollowUps(state) }

func GetFollowUpsForQuestion(state StateFile, parentQuestionID string) []FollowUp {
	return discovery.GetFollowUpsForQuestion(state, parentQuestionID)
}

func AddDelegation(state StateFile, questionID, delegatedTo, delegatedBy string) StateFile {
	return discovery.AddDelegation(state, questionID, delegatedTo, delegatedBy)
}

func AnswerDelegation(state StateFile, questionID, answer, answeredBy string) StateFile {
	return discovery.AnswerDelegation(state, questionID, answer, answeredBy)
}

func GetPendingDelegations(state StateFile) []Delegation {
	return discovery.GetPendingDelegations(state)
}

// -----------------------------------------------------------------------------
// Execution operations
// -----------------------------------------------------------------------------

func AdvanceExecution(state StateFile, progress string) (StateFile, error) {
	return execution.AdvanceExecution(state, progress)
}

func AddDecision(state StateFile, decision Decision) StateFile {
	return execution.AddDecision(state, decision)
}

func AddCustomAC(state StateFile, text string, user *UserInfo) StateFile {
	return execution.AddCustomAC(state, text, user)
}

func AddSpecNote(state StateFile, text string, user *UserInfo) StateFile {
	return execution.AddSpecNote(state, text, user)
}

func ClampConfidence(value float64) int { return execution.ClampConfidence(value) }

func AddConfidenceFinding(state StateFile, finding string, confidence float64, basis string) (StateFile, error) {
	return execution.AddConfidenceFinding(state, finding, confidence, basis)
}

func GetLowConfidenceFindings(state StateFile, threshold int) []ConfidenceFinding {
	return execution.GetLowConfidenceFindings(state, threshold)
}

func GetAverageConfidence(state StateFile) *float64 { return execution.GetAverageConfidence(state) }

func RecordTransition(state StateFile, from, to Phase, user *UserInfo, reason *string) StateFile {
	return execution.RecordTransition(state, from, to, user, reason)
}

func UncompleteTask(state StateFile, taskID string) (StateFile, error) {
	return execution.UncompleteTask(state, taskID)
}

// -----------------------------------------------------------------------------
// TDD
// -----------------------------------------------------------------------------

func CurrentTaskID(st StateFile) string { return tdd.CurrentTaskID(st) }

func IsTaskTDDEnabled(st StateFile, taskID string, cfg *NosManifest) bool {
	return tdd.IsTaskTDDEnabled(st, taskID, cfg)
}

func ShouldRunTDDForCurrentTask(st StateFile, cfg *NosManifest) bool {
	return tdd.ShouldRunTDDForCurrentTask(st, cfg)
}

func AnyTaskUsesTDD(st StateFile, cfg *NosManifest) bool {
	return tdd.AnyTaskUsesTDD(st, cfg)
}

func RecordTDDVerification(
	st StateFile,
	maxRetries int,
	passed bool,
	output string,
	failedACs []string,
	uncoveredEdgeCases []string,
) (StateFile, error) {
	return tdd.RecordTDDVerification(st, maxRetries, passed, output, failedACs, uncoveredEdgeCases)
}

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
	return tdd.RecordTDDVerificationFull(
		st, maxRetries, maxRefactorRounds, passed, output, failedACs, uncoveredEdgeCases, refactorNotes,
	)
}

func StartTDDCycleForTask(st *StateFile) { tdd.StartTDDCycleForTask(st) }

func MarkRefactorApplied(st *StateFile) { tdd.MarkRefactorApplied(st) }

// -----------------------------------------------------------------------------
// Identity
// -----------------------------------------------------------------------------

func GetConfigDir() string    { return identity.GetConfigDir() }
func GetUserFilePath() string { return identity.GetUserFilePath() }

func GetCurrentUser(args ...string) (*User, error) { return identity.GetCurrentUser(args...) }
func SetCurrentUser(user User) error               { return identity.SetCurrentUser(user) }
func ClearCurrentUser() (bool, error)              { return identity.ClearCurrentUser() }
func DetectGitUser() (*User, error)                { return identity.DetectGitUser() }
func FormatUser(user User) string                  { return identity.FormatUser(user) }
func ShortUser(user User) string                   { return identity.ShortUser(user) }
func UnknownUser() User                            { return identity.UnknownUser() }
func ResolveUser(args ...string) (User, error)     { return identity.ResolveUser(args...) }
