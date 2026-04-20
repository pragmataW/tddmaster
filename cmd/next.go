package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	ctxpkg "github.com/pragmataW/tddmaster/internal/context"
	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/output"
	"github.com/pragmataW/tddmaster/internal/spec"
	"github.com/pragmataW/tddmaster/internal/state"
)

// answerHash returns a short fingerprint of an answer, used for idempotency
// (deduping retries of the same --answer within a single phase).
func answerHash(a string) string {
	h := sha256.Sum256([]byte(strings.TrimSpace(a)))
	return hex.EncodeToString(h[:8])
}

// taskIDRe validates that a task ID is in the canonical "task-<N>" format.
var taskIDRe = regexp.MustCompile(`^task-\d+$`)

func newNextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "next",
		Short: "Advance to the next state",
		Long:  "Get the next instruction for the current phase, or submit an answer to advance state.",
		RunE:  runNext,
	}
	cmd.Flags().String("spec", "", "Spec name")
	cmd.Flags().String("answer", "", "Answer to submit to advance the state machine")
	return cmd
}

func runNext(cmd *cobra.Command, args []string) error {
	specFlag, _ := cmd.Flags().GetString("spec")
	answerFlag, _ := cmd.Flags().GetString("answer")

	// Also parse from args (legacy --spec=name --answer=text)
	for _, arg := range args {
		if strings.HasPrefix(arg, "--spec=") && specFlag == "" {
			specFlag = arg[len("--spec="):]
		}
		if strings.HasPrefix(arg, "--answer=") && answerFlag == "" {
			answerFlag = arg[len("--answer="):]
		}
	}

	var specPtr *string
	if specFlag != "" {
		specPtr = &specFlag
	}

	return runNextCore(specPtr, answerFlag)
}

// runNextWithArgs is called from spec.go dispatcher.
func runNextWithArgs(args []string) error {
	var specPtr *string
	var answerText string

	for _, arg := range args {
		if strings.HasPrefix(arg, "--spec=") {
			s := arg[len("--spec="):]
			specPtr = &s
		}
		if strings.HasPrefix(arg, "--answer=") {
			answerText = arg[len("--answer="):]
		}
	}

	return runNextCore(specPtr, answerText)
}

func runNextCore(specPtr *string, answerText string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}

	initialized, _ := state.IsInitialized(root)
	if !initialized {
		return writeJSON(map[string]string{"error": "tddmaster not initialized. Run: " + output.Cmd("init")})
	}

	st, err := state.ResolveState(root, specPtr)
	if err != nil {
		return writeJSON(map[string]string{"error": err.Error()})
	}

	// If no --spec and in active phase, require spec
	if specPtr == nil && st.Phase != state.PhaseIdle && st.Phase != state.PhaseCompleted {
		return writeJSON(map[string]string{
			"error": "Error: --spec=<name> is required. Use `tddmaster spec list` to see available specs.",
		})
	}

	config, _ := state.ReadManifest(root)
	if config == nil {
		return writeJSON(map[string]string{"error": "No config found"})
	}

	// Set command prefix from manifest
	if config.Command != "" {
		output.SetCommandPrefix(config.Command)
		ctxpkg.SetCommandPrefix(config.Command)
	}

	// Load TDD manifest for tddMode setting.
	tddManifest, _ := state.LoadManifest(root)
	tddModeActive := tddManifest.TddMode

	// State integrity check
	if st.Spec != nil && st.Phase != state.PhaseIdle && st.Phase != state.PhaseCompleted {
		specDir := fmt.Sprintf("%s/%s/specs/%s", root, state.TddmasterDir, *st.Spec)
		if _, err := os.Stat(specDir); err != nil {
			return writeJSON(map[string]interface{}{
				"error":      true,
				"message":    fmt.Sprintf("Active spec '%s' directory not found.", *st.Spec),
				"suggestion": fmt.Sprintf("Run `%s` to return to idle.", output.Cmd("reset")),
			})
		}
	}

	// Load active concerns
	allConcerns, _ := state.ListConcerns(root)
	activeConcerns := filterConcerns(allConcerns, config.Concerns)

	if answerText != "" {
		// Idempotency: if the same answer text was just processed in the same
		// phase, skip re-applying it. Prevents duplicate task additions and
		// similar replay damage when a caller retries after an error or the
		// shell delivers the same command twice. "save" bypasses this gate
		// because it is explicitly a no-op idempotent checkpoint already.
		answerTrimmedLower := strings.TrimSpace(strings.ToLower(answerText))
		hash := answerHash(answerText)
		if answerTrimmedLower != "save" &&
			st.LastAnswer != nil &&
			st.LastAnswer.Phase == st.Phase &&
			st.LastAnswer.Hash == hash {
			// Return the current compiled context with an idempotent flag.
			tier1, hints, tier2Count, _ := loadRulesAndHints(root, st, config)
			if tddModeActive {
				tier1 = ctxpkg.InjectTDDRules(tier1)
			}
			var parsedSpec *spec.ParsedSpec
			if st.Spec != nil {
				parsedSpec, _ = spec.ParseSpec(root, *st.Spec)
			}
			u, _ := state.ResolveUser(root)
			compiled := ctxpkg.Compile(model.CompileInput{
				State:            st,
				ActiveConcerns:   activeConcerns,
				Rules:            tier1,
				Config:           config,
				ParsedSpec:       parsedSpec,
				InteractionHints: hints,
				CurrentUser:      &model.CurrentUser{Name: u.Name, Email: u.Email},
				Tier2Count:       tier2Count,
			})
			result := compiledToMap(compiled)
			result["idempotent"] = true
			return writeJSON(result)
		}

		// Handle the answer
		user, _ := state.ResolveUser(root)
		userInfo := &state.UserInfo{Name: user.Name, Email: user.Email}

		newState, regenerateSpec, err := handleAnswer(root, st, config, activeConcerns, answerText, userInfo)
		if err != nil {
			return writeJSON(map[string]string{"error": err.Error()})
		}

		now := time.Now().UTC().Format(time.RFC3339)
		newState.LastCalledAt = &now

		// Record fingerprint of the answer we are about to commit so that a
		// retry of the same text in the same phase becomes a no-op. Phase is
		// the PRE-handler phase so a replay within the same phase is caught
		// even when the handler advanced the phase.
		if answerTrimmedLower != "save" {
			newState.LastAnswer = &state.AnswerFingerprint{
				Phase:     st.Phase,
				Hash:      hash,
				Timestamp: now,
			}
		}

		if err := state.WriteStateAndSpec(root, newState); err != nil {
			return err
		}

		// POST-COMMIT: state is durable on disk. Now regenerate the derivative
		// spec.md / progress.json if the phase handler signaled a structural change.
		// If this fails, state is still consistent — user can re-run to retry.
		if regenerateSpec && newState.Spec != nil {
			if _, genErr := spec.GenerateSpec(root, &newState, activeConcerns); genErr != nil {
				printErr(fmt.Sprintf("warning: state saved but spec.md regeneration failed: %v", genErr))
				printErr("suggestion: re-run the same command to retry spec generation")
			}
		}

		// Update session phase if env var set
		sessionID := os.Getenv("TDDMASTER_SESSION")
		if sessionID != "" {
			_ = state.UpdateSessionPhase(root, sessionID, string(newState.Phase))
		}

		// Compile output
		tier1, hints, tier2Count, _ := loadRulesAndHints(root, newState, config)
		if tddModeActive {
			tier1 = ctxpkg.InjectTDDRules(tier1)
		}
		var parsedSpec *spec.ParsedSpec
		if newState.Spec != nil {
			parsedSpec, _ = spec.ParseSpec(root, *newState.Spec)
		}

		compiled := ctxpkg.Compile(model.CompileInput{
			State:            newState,
			ActiveConcerns:   activeConcerns,
			Rules:            tier1,
			Config:           config,
			ParsedSpec:       parsedSpec,
			InteractionHints: hints,
			CurrentUser:      &model.CurrentUser{Name: user.Name, Email: user.Email},
			Tier2Count:       tier2Count,
		})

		// Inject "saved" flag when "save" was the answer and phase didn't change
		isSaveAnswer := strings.TrimSpace(strings.ToLower(answerText)) == "save"
		phaseSame := newState.Phase == st.Phase
		if isSaveAnswer && phaseSame &&
			(newState.Phase == state.PhaseSpecProposal || newState.Phase == state.PhaseSpecApproved) {
			savedInstruction := "Spec draft saved."
			result := compiledToMap(compiled)
			result["instruction"] = savedInstruction
			result["saved"] = true
			return writeJSON(result)
		}

		return writeJSON(compiled)
	}

	// No answer — update timestamp and output current instruction
	now := time.Now().UTC().Format(time.RFC3339)
	st.LastCalledAt = &now
	if err := state.WriteStateAndSpec(root, st); err != nil {
		return err
	}

	// Update session phase
	sessionID := os.Getenv("TDDMASTER_SESSION")
	if sessionID != "" {
		_ = state.UpdateSessionPhase(root, sessionID, string(st.Phase))
	}

	// Build idle context if needed
	var idleCtx *model.IdleContext
	if st.Phase == state.PhaseIdle {
		specStates, _ := state.ListSpecStates(root)
		scoped, _ := loadRulesCount(root)
		scopedPtr := &scoped
		existingSpecs := make([]model.SpecSummary, 0, len(specStates))
		for _, ss := range specStates {
			es := model.SpecSummary{
				Name:      ss.Name,
				Phase:     string(ss.State.Phase),
				Iteration: ss.State.Execution.Iteration,
			}
			switch ss.State.Phase {
			case state.PhaseExecuting:
				d := fmt.Sprintf("%d tasks done, iteration %d",
					len(ss.State.Execution.CompletedTasks), ss.State.Execution.Iteration)
				es.Detail = &d
			case state.PhaseSpecProposal:
				d := "awaiting approval"
				es.Detail = &d
			case state.PhaseCompleted:
				d := "completed"
				es.Detail = &d
			}
			existingSpecs = append(existingSpecs, es)
		}
		idleCtx = &model.IdleContext{
			ExistingSpecs: existingSpecs,
			RulesCount:    scopedPtr,
		}
	}

	tier1, hints, tier2Count, _ := loadRulesAndHints(root, st, config)
	if tddModeActive {
		tier1 = ctxpkg.InjectTDDRules(tier1)
	}
	var parsedSpec *spec.ParsedSpec
	if st.Spec != nil {
		parsedSpec, _ = spec.ParseSpec(root, *st.Spec)
	}

	user, _ := state.ResolveUser(root)
	compiled := ctxpkg.Compile(model.CompileInput{
		State:            st,
		ActiveConcerns:   activeConcerns,
		Rules:            tier1,
		Config:           config,
		ParsedSpec:       parsedSpec,
		IdleContext:      idleCtx,
		InteractionHints: hints,
		CurrentUser:      &model.CurrentUser{Name: user.Name, Email: user.Email},
		Tier2Count:       tier2Count,
	})

	// Pre-execution TDD mode prompt: when tddMode is active and the spec is
	// approved (pre-execution gate), present a TDD-mode confirmation to the user.
	if tddModeActive && st.Phase == state.PhaseSpecApproved {
		result := compiledToMap(compiled)
		result["tddMode"] = map[string]interface{}{
			"active":      true,
			"instruction": "TDD mode is enabled in this project. Execution will follow red-green-refactor: write failing tests first, then implement. Ask the user: 'TDD mode is active — test tasks will run before implementation tasks. Confirm to proceed.'",
		}
		return writeJSON(result)
	}

	return writeJSON(compiled)
}

// loadRulesCount returns the count of tier1 rules for idle context.
func loadRulesCount(root string) (int, error) {
	rules, err := state.ListSpecStates(root)
	if err != nil {
		return 0, err
	}
	return len(rules), nil
}

// handleAnswer processes an answer for the current phase and returns new
// state plus a regenerateSpec flag. When regenerateSpec is true, the caller
// must run spec.GenerateSpec AFTER committing state to disk — this is the
// ordering that keeps state.json (primary truth) and spec.md (derivative
// view) consistent.
func handleAnswer(
	root string,
	st state.StateFile,
	config *state.NosManifest,
	activeConcerns []state.ConcernDefinition,
	answer string,
	user *state.UserInfo,
) (state.StateFile, bool, error) {
	switch st.Phase {
	case state.PhaseDiscovery:
		newState, err := handleDiscoveryAnswer(root, st, activeConcerns, answer, user)
		if err != nil {
			return st, false, err
		}
		// Regenerate when discovery transitions into spec proposal (spec.md is first written then).
		regen := newState.Spec != nil && newState.Phase == state.PhaseSpecProposal
		return newState, regen, nil

	case state.PhaseDiscoveryRefinement:
		newState, err := handleDiscoveryRefinementAnswer(st, answer, user)
		if err != nil {
			return st, false, err
		}
		regen := newState.Spec != nil && newState.Phase == state.PhaseSpecProposal
		return newState, regen, nil

	case state.PhaseSpecProposal:
		newState, err := handleSpecProposalAnswer(root, st, config, activeConcerns, answer)
		if err != nil {
			return st, false, err
		}
		// Every classification / out-of-scope / refinement change must be reflected
		// in spec.md. "save" is a no-op path; skip regeneration.
		regen := newState.Spec != nil && strings.TrimSpace(strings.ToLower(answer)) != "save"
		return newState, regen, nil

	case state.PhaseSpecApproved:
		newState, err := handleSpecApprovedAnswer(root, st, config, answer)
		return newState, false, err

	case state.PhaseExecuting:
		newState, err := handleExecutingAnswer(root, st, config, answer)
		if err != nil {
			return st, false, err
		}
		// Regenerate when task completion shifted (state → spec.md + progress.json sync).
		regen := newState.Spec != nil && len(newState.Execution.CompletedTasks) > len(st.Execution.CompletedTasks)
		return newState, regen, nil

	case state.PhaseBlocked:
		newState, err := handleBlockedAnswer(st, answer)
		return newState, false, err

	default:
		return st, false, nil
	}
}

func handleDiscoveryAnswer(
	root string,
	st state.StateFile,
	activeConcerns []state.ConcernDefinition,
	answer string,
	user *state.UserInfo,
) (state.StateFile, error) {
	st = processPendingUserContextPrefills(st)

	hasUserContext := st.Discovery.UserContext != nil && len(*st.Discovery.UserContext) > 0
	hasDesc := st.SpecDescription != nil && len(*st.SpecDescription) > 0
	discoveryMode := st.Discovery.Mode

	if !hasUserContext && discoveryMode == nil && hasDesc {
		newState := state.SetUserContext(st, answer)
		newState = storeUserContextPrefills(newState, answer)
		return newState, nil
	}

	// Mode selection
	if discoveryMode == nil && hasDesc {
		validModes := map[string]state.DiscoveryMode{
			"full": state.DiscoveryModeFull, "validate": state.DiscoveryModeValidate,
			"technical-depth": state.DiscoveryModeTechnicalDepth,
			"ship-fast":       state.DiscoveryModeShipFast, "explore": state.DiscoveryModeExplore,
		}
		mode, ok := validModes[strings.TrimSpace(strings.ToLower(answer))]
		if !ok {
			mode = state.DiscoveryModeFull
		}
		return state.SetDiscoveryMode(st, mode)
	}

	// Premise challenge
	premisesCompleted := st.Discovery.PremisesCompleted != nil && *st.Discovery.PremisesCompleted
	if discoveryMode != nil && !premisesCompleted {
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(answer), &parsed); err == nil {
			if premisesRaw, ok := parsed["premises"]; ok {
				premisesSlice, ok := premisesRaw.([]interface{})
				if !ok || len(premisesSlice) == 0 {
					return st, fmt.Errorf("premise challenge requires at least one premise")
				}
				var premises []state.Premise
				userName := "Unknown User"
				if user != nil {
					userName = user.Name
				}
				for _, p := range premisesSlice {
					pm, ok := p.(map[string]interface{})
					if !ok {
						continue
					}
					text, _ := pm["text"].(string)
					agreed, _ := pm["agreed"].(bool)
					var revision *string
					if r, ok := pm["revision"].(string); ok && r != "" {
						revision = &r
					}
					premises = append(premises, state.Premise{
						Text:      text,
						Agreed:    agreed,
						Revision:  revision,
						User:      userName,
						Timestamp: time.Now().UTC().Format(time.RFC3339),
					})
				}
				if len(premises) == 0 {
					return st, fmt.Errorf("premise challenge requires at least one premise. Empty array rejected")
				}
				return state.CompletePremises(st, premises)
			}
		}
		return st, fmt.Errorf("premise challenge requires valid JSON with premises array")
	}

	allQuestions := ctxpkg.GetQuestionsWithExtras(activeConcerns)

	// Try batch JSON answers.
	// Keep batch JSON as a backward-compatible fallback, but only when the payload
	// actually targets canonical discovery question IDs.
	var answersRaw map[string]interface{}
	if err := json.Unmarshal([]byte(answer), &answersRaw); err == nil && answersRaw != nil {
		questionIDs := make(map[string]bool, len(allQuestions))
		for _, question := range allQuestions {
			questionIDs[question.ID] = true
		}

		answersMap := make(map[string]string, len(answersRaw))
		recognized := 0
		for k, v := range answersRaw {
			if !questionIDs[k] {
				continue
			}
			recognized++
			switch val := v.(type) {
			case string:
				answersMap[k] = val
			case []interface{}:
				parts := make([]string, 0, len(val))
				for _, item := range val {
					if s, ok := item.(string); ok {
						parts = append(parts, s)
					}
				}
				answersMap[k] = strings.Join(parts, "\n")
			default:
				answersMap[k] = fmt.Sprintf("%v", v)
			}
		}

		if recognized > 0 {
			newState := st
			persisted := 0
			for qID, qAnswer := range answersMap {
				if strings.TrimSpace(qAnswer) == "" {
					continue
				}
				var ansErr error
				newState, ansErr = state.AddDiscoveryAnswer(newState, qID, qAnswer)
				if ansErr != nil {
					continue
				}
				persisted++
			}

			if persisted == 0 {
				return st, fmt.Errorf("batch discovery answer contained no valid responses")
			}

			batchSubmitted := true
			newState.Discovery.BatchSubmitted = &batchSubmitted
			newState.Discovery.CurrentQuestion = nextUnansweredDiscoveryQuestionIndex(allQuestions, newState.Discovery.Answers)

			if isDiscoveryComplete(newState.Discovery.Answers, allQuestions) {
				var complErr error
				newState, complErr = state.CompleteDiscovery(newState)
				if complErr != nil {
					return newState, nil
				}
			}
			return newState, nil
		}
	}

	// Single answer mode
	currentIdx := nextUnansweredDiscoveryQuestionIndex(allQuestions, st.Discovery.Answers)
	if currentIdx >= len(allQuestions) {
		return st, nil
	}
	currentQ := allQuestions[currentIdx]

	newState, err := state.AddDiscoveryAnswer(st, currentQ.ID, answer)
	if err != nil {
		return st, err
	}
	newState.Discovery.CurrentQuestion = nextUnansweredDiscoveryQuestionIndex(allQuestions, newState.Discovery.Answers)

	// Check if discovery is complete
	if isDiscoveryComplete(newState.Discovery.Answers, allQuestions) {
		newState, _ = state.CompleteDiscovery(newState)
	}

	return newState, nil
}

func processPendingUserContextPrefills(st state.StateFile) state.StateFile {
	if st.Discovery.UserContext == nil {
		return st
	}
	if st.Discovery.UserContextProcessed != nil && *st.Discovery.UserContextProcessed {
		return st
	}
	return storeUserContextPrefills(st, *st.Discovery.UserContext)
}

func storeUserContextPrefills(st state.StateFile, context string) state.StateFile {
	prefills := ctxpkg.ExtractUserContextPrefills(context)
	st = state.SetDiscoveryPrefills(st, prefills)
	return state.MarkUserContextProcessed(st)
}

func nextUnansweredDiscoveryQuestionIndex(
	questions []model.QuestionWithExtras,
	answers []state.DiscoveryAnswer,
) int {
	answeredIDs := make(map[string]bool, len(answers))
	for _, answer := range answers {
		answeredIDs[answer.QuestionID] = true
	}

	for i, question := range questions {
		if !answeredIDs[question.ID] {
			return i
		}
	}

	return len(questions)
}

// isDiscoveryComplete returns true if all required questions have been answered.
func isDiscoveryComplete(answers []state.DiscoveryAnswer, questions []model.QuestionWithExtras) bool {
	answered := make(map[string]bool)
	for _, a := range answers {
		answered[a.QuestionID] = true
	}
	for _, q := range questions {
		if !answered[q.ID] {
			return false
		}
	}
	return len(questions) > 0
}

func handleDiscoveryRefinementAnswer(st state.StateFile, answer string, user *state.UserInfo) (state.StateFile, error) {
	trimmed := strings.TrimSpace(strings.ToLower(answer))

	switch trimmed {
	case "approve":
		approved := st.Discovery.Approved
		altPresented := st.Discovery.AlternativesPresented != nil && *st.Discovery.AlternativesPresented

		if approved && !altPresented {
			// Skip alternatives
			newState, _ := state.SkipAlternatives(st)
			return state.ApproveDiscoveryReview(newState)
		}

		if st.Discovery.Mode != nil {
			return state.ApproveDiscoveryAnswers(st)
		}
		// Backward compat: no mode → direct to SPEC_PROPOSAL
		return state.ApproveDiscoveryReview(st)

	case "keep":
		now := time.Now().UTC().Format(time.RFC3339)
		newState := state.AddDecision(st, state.Decision{
			ID:        fmt.Sprintf("decision-split-keep-%d", time.Now().UnixMilli()),
			Question:  "Split spec into separate areas?",
			Choice:    "Chose to keep as single spec despite multiple areas detected",
			Promoted:  false,
			Timestamp: now,
		})
		return state.ApproveDiscoveryReview(newState)
	}

	// Alternatives selection (after approved, before SPEC_PROPOSAL)
	altPresented := st.Discovery.AlternativesPresented != nil && *st.Discovery.AlternativesPresented
	if st.Discovery.Approved && !altPresented {
		if trimmed == "skip" || trimmed == "none" {
			updatedState, _ := state.SkipAlternatives(st)
			return state.ApproveDiscoveryReview(updatedState)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(answer), &parsed); err == nil {
			if _, hasApproach := parsed["approach"]; hasApproach {
				userName := "Unknown User"
				if user != nil {
					userName = user.Name
				}
				approach := state.SelectedApproach{
					ID:        fmt.Sprintf("%v", parsed["approach"]),
					Name:      fmt.Sprintf("%v", orDefault(parsed["name"], parsed["approach"])),
					Summary:   fmt.Sprintf("%v", orDefault(parsed["summary"], "")),
					Effort:    fmt.Sprintf("%v", orDefault(parsed["effort"], "")),
					Risk:      fmt.Sprintf("%v", orDefault(parsed["risk"], "")),
					User:      userName,
					Timestamp: time.Now().UTC().Format(time.RFC3339),
				}
				updatedState, _ := state.SelectApproach(st, approach)
				return state.ApproveDiscoveryReview(updatedState)
			}
		}
		updatedState, _ := state.SkipAlternatives(st)
		return state.ApproveDiscoveryReview(updatedState)
	}

	// Revision: JSON with { revise: { questionId: "corrected answer" } }
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(answer), &parsed); err == nil {
		if revise, ok := parsed["revise"].(map[string]interface{}); ok {
			newState := st
			for qID, qAnswer := range revise {
				if qa, ok := qAnswer.(string); ok && qa != "" {
					newState, _ = state.AddDiscoveryAnswer(newState, qID, qa)
				}
			}
			return newState, nil
		}
	}

	return st, nil
}

func handleSpecProposalAnswer(
	root string,
	st state.StateFile,
	config *state.NosManifest,
	activeConcerns []state.ConcernDefinition,
	answer string,
) (state.StateFile, error) {
	if strings.TrimSpace(strings.ToLower(answer)) == "save" {
		return st, nil
	}

	if st.Classification == nil {
		classification := &state.SpecClassification{}

		trimmed := strings.TrimSpace(strings.ToLower(answer))
		if trimmed == "none" || trimmed == "skip" {
			// All flags false
		} else {
			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(answer), &parsed); err == nil {
				classification.InvolvesWebUI = getBoolField(parsed, "involvesWebUI") || getBoolField(parsed, "involvesUI")
				classification.InvolvesCLI = getBoolField(parsed, "involvesCLI") || getBoolField(parsed, "involvesUI")
				classification.InvolvesPublicAPI = getBoolField(parsed, "involvesPublicAPI")
				classification.InvolvesMigration = getBoolField(parsed, "involvesMigration")
				classification.InvolvesDataHandling = getBoolField(parsed, "involvesDataHandling")
			}
		}

		newState := st
		newState.Classification = classification
		// spec.md regeneration happens in runNextCore AFTER state is committed
		// (see handleAnswer's regenerateSpec flag). Writing spec.md here before
		// the state write led to desync when the state write later failed.
		return newState, nil
	}

	// Already classified — handle structured refinement verbs or out-of-scope
	// override.
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(answer), &parsed); err != nil {
		return st, fmt.Errorf("refinement answer is not valid JSON: %w", err)
	}

	payload := refinementPayload(parsed)
	if payload == nil {
		return st, fmt.Errorf("refinement payload contains no recognized fields (add/remove/update/notes or outOfScope required)")
	}

	// Handle out-of-scope override separately (simple replacement).
	if scope := readStringArray(payload, "outOfScope"); len(scope) > 0 {
		newState := st
		newState.OverrideOutOfScope = scope
		// spec.md regeneration deferred to runNextCore (post state commit).
		return newState, nil
	}

	verbs, err := parseRefinementVerbs(payload)
	if err != nil {
		return st, fmt.Errorf("refinement: %w", err)
	}

	// Guard: notes must not carry structured task content (task-N: …).
	// Self-review instructions tell the agent never to put a task list in
	// notes, but if a payload sneaks through any upstream routing path we
	// recover here — otherwise a pipe-separated task blob gets persisted
	// to SpecNotes and leaks into spec.md as an unreadable single bullet.
	if verbs.Notes != "" && taskHeaderSearchRe.MatchString(verbs.Notes) {
		if titles := parseTextualTaskList(verbs.Notes); len(titles) > 0 {
			// Recover: treat as full-replace tasks override.
			verbs.Add = titles
			verbs.Remove = []string{legacyReplaceAllSentinel}
			verbs.Update = nil
			verbs.Notes = ""
		} else {
			return st, fmt.Errorf("refinement: notes cannot contain task markers (task-N:); use add/update verbs or send a pipe-separated task list as {\"tasks\":[...]}")
		}
	}

	newState := st

	// Persist free-form notes to SpecNotes.
	if verbs.Notes != "" {
		user, _ := state.ResolveUser(root)
		userInfo := &state.UserInfo{Name: user.Name, Email: user.Email}
		newState = state.AddSpecNote(newState, verbs.Notes, userInfo)
	}

	// Apply task verbs when any structural change is requested.
	if len(verbs.Add) > 0 || len(verbs.Remove) > 0 || len(verbs.Update) > 0 {
		newTasks, newCompleted, err := applyTaskRefinement(
			newState.OverrideTasks,
			verbs,
			newState.Execution.CompletedTasks,
		)
		if err != nil {
			return st, fmt.Errorf("applying task refinement: %w", err)
		}
		newState.OverrideTasks = newTasks
		newState.Execution.CompletedTasks = newCompleted
	}

	// spec.md regeneration deferred to runNextCore (post state commit).
	return newState, nil
}

// textualTaskRe matches a single "task-<id>: title" chunk where <id> is either
// numeric (task-1) or a slug (task-setup, task-entity-tests).
var textualTaskRe = regexp.MustCompile(`^task-[\w-]+:\s*(.+)$`)

// taskHeaderSearchRe locates every "task-<id>:" header inside an arbitrary string.
var taskHeaderSearchRe = regexp.MustCompile(`\btask-[\w-]+:\s*`)

// removeTrailingRe strips trailing "REMOVE ..." instructions that users append
// after the last task content (e.g. "REMOVE task-1 (duplicate)").
var removeTrailingRe = regexp.MustCompile(`(?i)\.\s+REMOVE\s+.*`)

// listMarkerRe matches a leading list marker on a chunk: "1. ", "1) ", "- ", "* ".
var listMarkerRe = regexp.MustCompile(`^(?:\d+[.)]\s+|[-*]\s+)`)

// directivePrefixRe matches a leading directive like "REPLACE all tasks with:"
// or "Update tasks:" — the prefix is stripped before parsing the rest.
var directivePrefixRe = regexp.MustCompile(`(?i)^\s*(?:replace|update|set|new|change)\s+(?:all\s+)?tasks?\s*(?:with|to)?\s*[:.]\s*`)

// refactorHintRe matches verifier narrative output that describes a refactor
// opportunity. Used by applyVerifierReport to catch submit-loss cases where
// the orchestrator drops the structured refactorNotes array but the free-form
// output clearly calls for a refactor.
var refactorHintRe = regexp.MustCompile(
	`(?i)\b(refactor\w*|extract\w*|dedupe\w*|duplicat\w*|cleanup\w*|renam\w*|simplif\w*|improv\w*|DRY)\b`,
)

// parseTextualTaskList interprets s as a structured task list. It handles:
//
//  1. Explicit-separator: "task-1: A | task-2: B" or "task-1: A\ntask-2: B"
//     — when "|" or "\n" is present, every non-blank chunk must match task-<id>:.
//     Leading list markers (1., 1), -, *) are stripped per chunk.
//
//  2. Compressed: "task-1: A. task-2: B. task-3: C" — when no pipe/newline
//     separators are used, two or more embedded task-<id>: headers trigger
//     boundary-based extraction.
//
// A leading directive prefix ("REPLACE all tasks with:", "Update tasks:", etc.)
// is stripped before either path runs. Returns nil for free-form prose so
// callers fall back to notes handling.
func parseTextualTaskList(s string) []string {
	// Strip a leading directive prefix if present.
	s = directivePrefixRe.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	// --- Path 1: pipe / newline separated chunks ---
	if strings.Contains(s, "|") || strings.Contains(s, "\n") {
		var chunks []string
		for _, piece := range strings.Split(s, "|") {
			for _, line := range strings.Split(piece, "\n") {
				t := strings.TrimSpace(line)
				if t == "" {
					continue
				}
				// Strip leading list marker ("1. ", "- ", etc.) so the chunk
				// can match `^task-<id>:`.
				t = listMarkerRe.ReplaceAllString(t, "")
				chunks = append(chunks, t)
			}
		}
		if len(chunks) == 0 {
			return nil
		}
		titles := make([]string, 0, len(chunks))
		for _, c := range chunks {
			m := textualTaskRe.FindStringSubmatch(c)
			if m == nil {
				return nil // mixed content → notes
			}
			title := strings.TrimSpace(m[1])
			if title == "" {
				return nil
			}
			// Trim trailing period that users often add to numbered list items.
			title = strings.TrimRight(title, " .")
			title = strings.TrimSpace(title)
			if title == "" {
				return nil
			}
			titles = append(titles, title)
		}
		return titles
	}

	// --- Path 2: compressed "task-1: A task-2: B" without explicit separators ---
	locs := taskHeaderSearchRe.FindAllStringIndex(s, -1)
	if len(locs) < 2 {
		return nil // zero or one header — ambiguous, treat as prose
	}
	// Reject if there is leading prose before the first task header.
	if strings.TrimSpace(s[:locs[0][0]]) != "" {
		return nil
	}
	titles := make([]string, 0, len(locs))
	for i, loc := range locs {
		var content string
		if i+1 < len(locs) {
			content = s[loc[1]:locs[i+1][0]]
		} else {
			content = s[loc[1]:]
		}
		content = strings.TrimSpace(content)
		// Strip trailing "REMOVE ..." clauses that users append after the task list.
		content = removeTrailingRe.ReplaceAllString(content, "")
		content = strings.TrimRight(content, " .")
		content = strings.TrimSpace(content)
		if content == "" {
			return nil // malformed: empty task title
		}
		titles = append(titles, content)
	}
	return titles
}

// routeNotesAsTasks examines a notes string; if it parses as a structured
// `task-N: title` list, returns a tasks-override payload. Returns nil for
// free-form prose so callers keep the notes path.
func routeNotesAsTasks(s string) map[string]interface{} {
	text := strings.TrimSpace(s)
	if text == "" {
		return nil
	}
	titles := parseTextualTaskList(text)
	if len(titles) == 0 {
		return nil
	}
	ifaces := make([]interface{}, len(titles))
	for i, t := range titles {
		ifaces[i] = t
	}
	return map[string]interface{}{"tasks": ifaces}
}

// refinementPayload returns the object carrying the refinement content whether
// it came as `{"refinement": {...}}`, `{"refinement": "text"}`, or top-level
// fields (`{"add": [...], "remove": [...], "update": {...}, "notes": "..."}`).
// Structured "task-N: title | task-N: title" strings are routed to a full-replace
// tasks payload — whether they arrive under `refinement`, `refinement.notes`, or
// top-level `notes`. All other free-form strings fall through to notes.
func refinementPayload(parsed map[string]interface{}) map[string]interface{} {
	if obj, ok := parsed["refinement"].(map[string]interface{}); ok {
		// refinement.notes carrying a task list → redirect to tasks override.
		if notes, ok := obj["notes"].(string); ok {
			if converted := routeNotesAsTasks(notes); converted != nil {
				return converted
			}
		}
		return obj
	}
	if s, ok := parsed["refinement"].(string); ok && strings.TrimSpace(s) != "" {
		text := strings.TrimSpace(s)
		if converted := routeNotesAsTasks(text); converted != nil {
			return converted
		}
		// Free-form prose — persist as notes.
		return map[string]interface{}{"notes": text}
	}
	// Top-level `notes` carrying a task list must become a tasks override, not a
	// blob note. Free-form notes keep their existing path.
	if notes, ok := parsed["notes"].(string); ok {
		if converted := routeNotesAsTasks(notes); converted != nil {
			return converted
		}
	}
	// Top-level structured refinement without the "refinement" wrapper.
	for _, key := range []string{"add", "remove", "update", "tasks", "outOfScope", "notes"} {
		if _, ok := parsed[key]; ok {
			return parsed
		}
	}
	return nil
}

// refinementVerbs holds the structured operations from a spec refinement payload.
type refinementVerbs struct {
	Add    []string          // task titles to add (new IDs auto-assigned)
	Remove []string          // task IDs to remove (e.g. "task-3")
	Update map[string]string // taskID → new title
	Notes  string            // free-form refinement notes
}

// legacyReplaceAllSentinel is the internal marker used to signal a full-replace
// via the legacy `{tasks: [...]}` format.
const legacyReplaceAllSentinel = "__LEGACY_REPLACE_ALL__"

// parseRefinementVerbs extracts structured refinement verbs from a parsed JSON
// payload. Accepts both the new verb-based format {add, remove, update, notes}
// and the legacy {tasks:[...]} full-replace format.
// Returns an error when the payload contains no actionable content.
func parseRefinementVerbs(payload map[string]interface{}) (refinementVerbs, error) {
	var v refinementVerbs

	// New format: {add: [...], remove: [...], update: {...}, notes: "..."}
	if add := readStringArray(payload, "add"); len(add) > 0 {
		v.Add = add
	}
	if remove := readStringArray(payload, "remove"); len(remove) > 0 {
		v.Remove = remove
	}
	if updateRaw, ok := payload["update"].(map[string]interface{}); ok {
		v.Update = make(map[string]string, len(updateRaw))
		for k, val := range updateRaw {
			if s, ok := val.(string); ok {
				v.Update[k] = s
			}
		}
	}
	if notes, ok := payload["notes"].(string); ok {
		v.Notes = strings.TrimSpace(notes)
	}
	if v.Notes == "" {
		if t, ok := payload["text"].(string); ok {
			v.Notes = strings.TrimSpace(t)
		}
	}

	// Legacy format: {tasks: [...]} — treat as full replace with a sentinel.
	if legacyTasks := readStringArray(payload, "tasks"); len(legacyTasks) > 0 {
		v.Add = legacyTasks
		v.Remove = []string{legacyReplaceAllSentinel}
		return v, nil
	}

	// Require at least one actionable verb.
	if len(v.Add) == 0 && len(v.Remove) == 0 && len(v.Update) == 0 && v.Notes == "" {
		return v, fmt.Errorf("refinement payload contains no actionable content (add/remove/update/notes required)")
	}

	return v, nil
}

// applyTaskRefinement applies structured refinement verbs to the current task
// list. Operations are applied in order: remove → update → add (idempotent).
// Entries in completed that reference removed tasks are also cleaned up.
func applyTaskRefinement(current []state.SpecTask, verbs refinementVerbs, completed []string) ([]state.SpecTask, []string, error) {
	// Handle legacy full-replace sentinel: clear all, renumber from 1.
	if len(verbs.Remove) == 1 && verbs.Remove[0] == legacyReplaceAllSentinel {
		result := make([]state.SpecTask, len(verbs.Add))
		for i, title := range verbs.Add {
			result[i] = state.SpecTask{
				ID:        fmt.Sprintf("task-%d", i+1),
				Title:     title,
				Completed: false,
			}
		}
		return result, nil, nil // completed list cleared on full replace
	}

	tasks := make([]state.SpecTask, len(current))
	copy(tasks, current)

	// Step 1: remove
	if len(verbs.Remove) > 0 {
		removeSet := make(map[string]bool, len(verbs.Remove))
		for _, id := range verbs.Remove {
			removeSet[id] = true
		}
		filtered := tasks[:0]
		for _, t := range tasks {
			if !removeSet[t.ID] {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered

		var newCompleted []string
		for _, id := range completed {
			if !removeSet[id] {
				newCompleted = append(newCompleted, id)
			}
		}
		completed = newCompleted
	}

	// Step 2: update
	for id, newTitle := range verbs.Update {
		found := false
		for i := range tasks {
			if tasks[i].ID == id {
				tasks[i].Title = newTitle
				found = true
				break
			}
		}
		if !found {
			return nil, nil, fmt.Errorf("update: task ID %q not found in current task list", id)
		}
	}

	// Step 3: add — find max numeric ID and increment.
	maxID := 0
	for _, t := range tasks {
		if taskIDRe.MatchString(t.ID) {
			var n int
			fmt.Sscanf(t.ID, "task-%d", &n) //nolint:errcheck
			if n > maxID {
				maxID = n
			}
		}
	}
	for _, title := range verbs.Add {
		title = strings.TrimSpace(title)
		if title == "" {
			continue // skip blank entries
		}
		maxID++
		tasks = append(tasks, state.SpecTask{
			ID:        fmt.Sprintf("task-%d", maxID),
			Title:     title,
			Completed: false,
		})
	}

	// Invariant: remove orphan completed IDs that no longer exist in tasks.
	taskSet := make(map[string]bool, len(tasks))
	for _, t := range tasks {
		taskSet[t.ID] = true
	}
	var cleanCompleted []string
	for _, id := range completed {
		if taskSet[id] {
			cleanCompleted = append(cleanCompleted, id)
		}
	}

	return tasks, cleanCompleted, nil
}

// refinementText extracts the free-form text portion of a refinement payload,
// accepting `text`, `notes`, or a bare string under the `refinement` key.
func refinementText(payload map[string]interface{}) string {
	for _, key := range []string{"text", "notes"} {
		if s, ok := payload[key].(string); ok {
			if trimmed := strings.TrimSpace(s); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func readStringArray(payload map[string]interface{}, key string) []string {
	raw, ok := payload[key].([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, strings.TrimSpace(s))
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func specApprovedTDDSelectionPending(st state.StateFile, config *state.NosManifest) bool {
	return config != nil && config.IsTDDEnabled() &&
		(st.TaskTDDSelected == nil || !*st.TaskTDDSelected)
}

func startExecutionFromApproved(st state.StateFile, config *state.NosManifest) (state.StateFile, error) {
	newState, err := state.StartExecution(st)
	if err != nil {
		return st, err
	}

	if config != nil && config.IsTDDEnabled() && state.ShouldRunTDDForCurrentTask(newState, config) {
		state.StartTDDCycleForTask(&newState)
	}

	return newState, nil
}

func handleSpecApprovedAnswer(root string, st state.StateFile, config *state.NosManifest, answer string) (state.StateFile, error) {
	trimmed := strings.TrimSpace(answer)
	if strings.EqualFold(trimmed, "save") {
		return st, nil
	}

	if specApprovedTDDSelectionPending(st, config) {
		choice, err := parseTDDSelectionAnswer(trimmed)
		if err != nil {
			return st, err
		}
		tasks, err := resolveSelectionTasks(root, st)
		if err != nil {
			return st, err
		}
		newSt := applyTDDSelectionToOverrides(st, tasks, choice)
		tr := true
		newSt.TaskTDDSelected = &tr
		return newSt, nil
	}

	return startExecutionFromApproved(st, config)
}

// tddSelectionChoice captures the user's intent from the TDD selection
// sub-step (shown after a spec is approved when TDD is enabled at spec level).
type tddSelectionChoice struct {
	Mode     string   // "all" | "none" | "custom"
	TDDTasks []string // populated when Mode == "custom"
}

// parseTDDSelectionAnswer accepts one of:
//   - "tdd-all" / "all"                  → Mode = "all"
//   - "tdd-none" / "none"                → Mode = "none"
//   - JSON {"tddTasks":["task-1",...]}    → Mode = "custom"
func parseTDDSelectionAnswer(answer string) (tddSelectionChoice, error) {
	trimmed := strings.TrimSpace(answer)
	low := strings.ToLower(trimmed)
	switch low {
	case "tdd-all", "all":
		return tddSelectionChoice{Mode: "all"}, nil
	case "tdd-none", "none":
		return tddSelectionChoice{Mode: "none"}, nil
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &obj); err != nil {
		return tddSelectionChoice{}, fmt.Errorf("invalid TDD selection answer: expected \"tdd-all\", \"tdd-none\", or JSON {\"tddTasks\":[...]}")
	}
	raw, ok := obj["tddTasks"].([]interface{})
	if !ok {
		return tddSelectionChoice{}, fmt.Errorf("invalid TDD selection answer: JSON must carry a tddTasks array")
	}
	ids := make([]string, 0, len(raw))
	for _, v := range raw {
		s, ok := v.(string)
		if !ok {
			continue
		}
		if !taskIDRe.MatchString(s) {
			continue
		}
		ids = append(ids, s)
	}
	return tddSelectionChoice{Mode: "custom", TDDTasks: ids}, nil
}

// resolveSelectionTasks returns the authoritative task list to write TDD flags
// onto. Prefers StateFile.OverrideTasks; falls back to parsing the spec.md on
// disk.
func resolveSelectionTasks(root string, st state.StateFile) ([]state.SpecTask, error) {
	if len(st.OverrideTasks) > 0 {
		out := make([]state.SpecTask, len(st.OverrideTasks))
		copy(out, st.OverrideTasks)
		return out, nil
	}
	if st.Spec == nil {
		return nil, nil
	}
	parsed, err := spec.ParseSpec(root, *st.Spec)
	if err != nil {
		return nil, err
	}
	if parsed == nil {
		return nil, nil
	}
	out := make([]state.SpecTask, 0, len(parsed.Tasks))
	for _, t := range parsed.Tasks {
		out = append(out, state.SpecTask{ID: t.ID, Title: t.Title, Covers: t.Covers})
	}
	return out, nil
}

// applyTDDSelectionToOverrides writes SpecTask.TDDEnabled on every row of
// StateFile.OverrideTasks according to the user's choice. Tasks discovered
// via resolveSelectionTasks but not yet present in OverrideTasks are appended
// so their flags stick.
func applyTDDSelectionToOverrides(st state.StateFile, tasks []state.SpecTask, choice tddSelectionChoice) state.StateFile {
	existing := make(map[string]int, len(st.OverrideTasks))
	for i, ot := range st.OverrideTasks {
		existing[ot.ID] = i
	}
	merged := append([]state.SpecTask(nil), st.OverrideTasks...)
	for _, t := range tasks {
		if _, ok := existing[t.ID]; ok {
			continue
		}
		merged = append(merged, t)
		existing[t.ID] = len(merged) - 1
	}

	tddSet := make(map[string]bool, len(choice.TDDTasks))
	for _, id := range choice.TDDTasks {
		tddSet[id] = true
	}

	for i := range merged {
		var enabled bool
		switch choice.Mode {
		case "all":
			enabled = true
		case "none":
			enabled = false
		case "custom":
			enabled = tddSet[merged[i].ID]
		}
		b := enabled
		merged[i].TDDEnabled = &b
	}
	st.OverrideTasks = merged
	return st
}

func handleExecutingAnswer(root string, st state.StateFile, config *state.NosManifest, answer string) (state.StateFile, error) {
	// Parse the answer once; if it's already a structured status report we skip
	// the "claim-then-report" two-step handshake and process it directly.
	structured, parseOK := parseStructuredReport(answer)

	if !st.Execution.AwaitingStatusReport && !parseOK {
		newState := st
		newState.Execution.LastProgress = &answer

		// Vague claim — ask for a formal status report next.
		newState.Execution.AwaitingStatusReport = true
		return newState, nil
	}

	if !parseOK {
		return st, fmt.Errorf("expected status report JSON with completed/remaining arrays")
	}

	// skipVerify routing: when the manifest requests verifier-skip, apply the
	// guard before the normal verifier-payload check. The guard handles all
	// cases (error on misrouted verifier reports, advance on executor reports).
	if config != nil && config.IsVerifierSkipped() {
		return verifierPayloadGuard(st, config, structured)
	}

	if verifierPayload, ok := extractVerifierPayload(structured); ok {
		return applyVerifierReport(st, config, verifierPayload)
	}

	return applyExecutorReport(st, config, structured)
}

// verifierPayloadGuard is the entry point for all answer routing when
// skipVerify=true is set in the manifest.
//
// Rules:
//   - GREEN phase → verifier is still required; route normally (verifier
//     report → applyVerifierReport, executor report → error).
//   - Non-GREEN phase + verifier shape in report → error (guard rejects).
//   - Non-GREEN phase + pure executor report → advanceWithoutVerification.
func verifierPayloadGuard(st state.StateFile, config *state.NosManifest, report map[string]interface{}) (state.StateFile, error) {
	phase := st.Execution.TDDCycle

	// GREEN phase always requires a real verifier even when skipVerify=true.
	if config.IsTDDEnabled() && phase == state.TDDCycleGreen {
		if verifierPayload, ok := extractVerifierPayload(report); ok {
			return applyVerifierReport(st, config, verifierPayload)
		}
		return st, fmt.Errorf(
			"verifier report required in GREEN phase even when skipVerify=true; "+
				"current phase: %s", phase,
		)
	}

	// Non-GREEN (or TDD=off): verifier is skipped. Reject verifier reports.
	if _, ok := extractVerifierPayload(report); ok {
		phaseStr := phase
		if phaseStr == "" {
			phaseStr = "(none)"
		}
		return st, fmt.Errorf(
			"verifier report submitted but skipVerify=true; phase: %s", phaseStr,
		)
	}

	// Pure executor report → advance without calling applyVerifierReport.
	return advanceWithoutVerification(st, config, report)
}

// advanceWithoutVerification advances the state when skipVerify=true and the
// report is a pure executor report (no verifier payload). It deliberately does
// NOT call applyVerifierReport so that LastVerification is never set on the
// skip-verify path (AC-7).
//
// TDD=off: complete tasks, increment iteration.
// TDD=on + RED: transition to GREEN (tests accepted by executor).
// TDD=on + REFACTOR: complete task, reseed next task's RED cycle.
func advanceWithoutVerification(st state.StateFile, config *state.NosManifest, report map[string]interface{}) (state.StateFile, error) {
	newState := st
	newState.Execution.AwaitingStatusReport = false

	phase := st.Execution.TDDCycle

	if config.IsTDDEnabled() && phase == state.TDDCycleRed {
		// RED → GREEN: tests have been written; accept without running verifier.
		newState.Execution.TDDCycle = state.TDDCycleGreen
		newState.Execution.Iteration++
		return newState, nil
	}

	if config.IsTDDEnabled() && phase == state.TDDCycleRefactor {
		// REFACTOR → task-complete + reseed next-task RED.
		newState.Execution.Iteration++
		if completed, ok := report["completed"].([]interface{}); ok {
			for _, c := range completed {
				s, ok := c.(string)
				if !ok {
					continue
				}
				newState.Execution.CompletedTasks = append(newState.Execution.CompletedTasks, s)
				for i := range newState.OverrideTasks {
					if newState.OverrideTasks[i].ID == s {
						newState.OverrideTasks[i].Completed = true
						break
					}
				}
			}
		}
		if getBoolField(report, "refactorApplied") {
			state.MarkRefactorApplied(&newState)
		}
		clearTDDRefactorState(&newState)
		reseedTDDCycleIfNeeded(&newState, config)
		return newState, nil
	}

	// TDD=off (or any other phase): standard task completion + iteration bump.
	newState.Execution.Iteration++
	if completed, ok := report["completed"].([]interface{}); ok {
		for _, c := range completed {
			s, ok := c.(string)
			if !ok {
				continue
			}
			newState.Execution.CompletedTasks = append(newState.Execution.CompletedTasks, s)
			for i := range newState.OverrideTasks {
				if newState.OverrideTasks[i].ID == s {
					newState.OverrideTasks[i].Completed = true
					break
				}
			}
		}
	}
	return newState, nil
}

// parseStructuredReport returns the decoded top-level object when the answer is
// a JSON object that looks like a status report (executor or verifier). A bare
// string, array, or random JSON returns ok=false so callers can fall back to
// the vague-claim handshake.
func parseStructuredReport(answer string) (map[string]interface{}, bool) {
	var report map[string]interface{}
	if err := json.Unmarshal([]byte(answer), &report); err != nil {
		return nil, false
	}
	if hasStatusReportShape(report) {
		return report, true
	}
	return nil, false
}

// hasStatusReportShape returns true when report carries any of the known
// top-level fields an executor/verifier emits.
func hasStatusReportShape(report map[string]interface{}) bool {
	for _, key := range []string{
		"completed", "remaining", "blocked", "filesModified",
		"passed", "failedACs", "refactorNotes", "refactorApplied",
		"tddVerification", "verification",
	} {
		if _, ok := report[key]; ok {
			return true
		}
	}
	return false
}

// extractVerifierPayload unwraps the verifier report whether it arrived flat
// (`{"passed": true, ...}`) or wrapped under `tddVerification` / `verification`.
// The nested object MUST itself look like a verifier report (have `passed`);
// otherwise we fall through to executor-report handling.
func extractVerifierPayload(report map[string]interface{}) (map[string]interface{}, bool) {
	if _, ok := report["passed"]; ok {
		return report, true
	}
	for _, key := range []string{"tddVerification", "verification"} {
		nested, ok := report[key].(map[string]interface{})
		if !ok {
			continue
		}
		if _, hasPassed := nested["passed"]; hasPassed {
			return nested, true
		}
	}
	return nil, false
}

// applyExecutorReport processes a report from the executor (or test-writer)
// sub-agent: it appends completed task IDs, flips RefactorApplied when the
// executor signals it, and advances the iteration counter.
func applyExecutorReport(st state.StateFile, config *state.NosManifest, report map[string]interface{}) (state.StateFile, error) {
	// Refactor-bypass guard: while the cycle is parked in REFACTOR with pending
	// notes from the GREEN scan, the executor must either apply the notes
	// (reporting refactorApplied:true) or defer to the verifier. Allowing a
	// bare `completed` submit here would silently skip the refactor round and
	// re-seed the next task's RED, which is exactly the regression we saw.
	if st.Execution.TDDCycle == state.TDDCycleRefactor &&
		!st.Execution.RefactorApplied &&
		!getBoolField(report, "refactorApplied") {
		completedLen := 0
		if c, ok := report["completed"].([]interface{}); ok {
			completedLen = len(c)
		}
		if completedLen > 0 && hasPendingRefactorNotes(st) {
			return st, fmt.Errorf(
				"cannot complete task while in REFACTOR phase with pending notes; " +
					"apply the refactor notes first and report `refactorApplied: true`, " +
					"or submit a verifier report (not an executor report) to advance the cycle",
			)
		}
	}

	newState := st
	newState.Execution.AwaitingStatusReport = false
	newState.Execution.Iteration++

	taskCompleted := false
	if completed, ok := report["completed"].([]interface{}); ok {
		for _, c := range completed {
			s, ok := c.(string)
			if !ok {
				continue
			}
			newState.Execution.CompletedTasks = append(newState.Execution.CompletedTasks, s)
			// Also mark the corresponding OverrideTasks entry as completed so that
			// subsequent spec.md / progress.json regenerations preserve the "done"
			// status. OverrideTasks[].Completed is the authoritative flag used by
			// RenderSpec ([x] vs [ ]) and GenerateSpec (progress.json status).
			for i := range newState.OverrideTasks {
				if newState.OverrideTasks[i].ID == s {
					newState.OverrideTasks[i].Completed = true
					break
				}
			}
			taskCompleted = true
		}
	}

	if getBoolField(report, "refactorApplied") {
		state.MarkRefactorApplied(&newState)
	}

	// When a task finishes, re-seed the TDD cycle for the next task based on
	// its per-task TDD flag. For TDD tasks this starts a fresh RED; for
	// non-TDD tasks we leave TDDCycle empty so the compiler emits the plain
	// executor→verifier flow.
	if taskCompleted {
		reseedTDDCycleIfNeeded(&newState, config)
	}

	return newState, nil
}

// clearTDDRefactorState resets all three TDD/refactor tracking fields on st.
func clearTDDRefactorState(st *state.StateFile) {
	st.Execution.TDDCycle = ""
	st.Execution.RefactorRounds = 0
	st.Execution.RefactorApplied = false
}

// reseedTDDCycleIfNeeded sets or clears Execution.TDDCycle based on whether
// the current task (after any CompletedTasks append) should run under TDD.
func reseedTDDCycleIfNeeded(st *state.StateFile, config *state.NosManifest) {
	if state.ShouldRunTDDForCurrentTask(*st, config) {
		if st.Execution.TDDCycle == "" {
			state.StartTDDCycleForTask(st)
		}
		return
	}
	clearTDDRefactorState(st)
}

// applyVerifierReport routes a verifier report through RecordTDDVerificationFull
// so the TDD cycle transitions and refactor-round bookkeeping run.
func applyVerifierReport(st state.StateFile, config *state.NosManifest, report map[string]interface{}) (state.StateFile, error) {
	passed := getBoolField(report, "passed")
	output, _ := report["output"].(string)

	failedACs := readStringSlice(report, "failedACs")
	uncoveredEdgeCases := readStringSlice(report, "uncoveredEdgeCases")
	refactorNotes := readRefactorNotes(report)

	// Submit-loss guard: a GREEN PASS whose narrative hints at a refactor but
	// omits the refactorNotes field entirely almost always means the orchestrator
	// dropped the verifier's structured notes. Reject so the caller can resubmit
	// with explicit notes (or an explicit empty array to confirm "nothing to do").
	if st.Execution.TDDCycle == state.TDDCycleGreen && passed {
		if _, refactorNotesPresent := report["refactorNotes"]; !refactorNotesPresent &&
			refactorHintRe.MatchString(output) {
			return st, fmt.Errorf(
				"verifier output suggests refactor notes but `refactorNotes` field is empty. " +
					"Include them explicitly as `refactorNotes: [{file, suggestion, rationale}]`, " +
					"or return an empty array only if truly no improvements apply",
			)
		}
	}

	maxRetries := 0
	maxRefactorRounds := 0
	if config != nil && config.Tdd != nil {
		maxRetries = config.Tdd.MaxVerificationRetries
		maxRefactorRounds = config.Tdd.MaxRefactorRounds
	}

	newState, err := state.RecordTDDVerificationFull(
		st, maxRetries, maxRefactorRounds, passed, output, failedACs, uncoveredEdgeCases, refactorNotes, config,
	)
	if err != nil {
		return st, err
	}

	newState.Execution.AwaitingStatusReport = false
	newState.Execution.Iteration++

	// When RecordTDDVerificationFull cleared the cycle (task advancing), seed
	// the next task's cycle according to its per-task TDD flag.
	if newState.Execution.TDDCycle == "" {
		reseedTDDCycleIfNeeded(&newState, config)
	}

	return newState, nil
}

func readStringSlice(m map[string]interface{}, key string) []string {
	raw, ok := m[key].([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// hasPendingRefactorNotes returns true when LastVerification holds refactor
// notes that the executor has not yet consumed. Used by the REFACTOR-phase
// guard in applyExecutorReport.
func hasPendingRefactorNotes(st state.StateFile) bool {
	if st.Execution.LastVerification == nil {
		return false
	}
	return len(st.Execution.LastVerification.RefactorNotes) > 0
}

func readRefactorNotes(m map[string]interface{}) []state.RefactorNote {
	raw, ok := m["refactorNotes"].([]interface{})
	if !ok {
		return nil
	}
	out := make([]state.RefactorNote, 0, len(raw))
	for _, v := range raw {
		entry, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		file, _ := entry["file"].(string)
		suggestion, _ := entry["suggestion"].(string)
		rationale, _ := entry["rationale"].(string)
		if file == "" && suggestion == "" && rationale == "" {
			continue
		}
		out = append(out, state.RefactorNote{File: file, Suggestion: suggestion, Rationale: rationale})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func handleBlockedAnswer(st state.StateFile, answer string) (state.StateFile, error) {
	newState, err := state.Transition(st, state.PhaseExecuting)
	if err != nil {
		return st, err
	}
	resolution := fmt.Sprintf("Resolved: %s", answer)
	newState.Execution.LastProgress = &resolution
	return newState, nil
}

// orDefault returns a if non-nil and non-zero string, else b.
func orDefault(a, b interface{}) interface{} {
	if a == nil {
		return b
	}
	s, ok := a.(string)
	if ok && s == "" {
		return b
	}
	return a
}

// getBoolField safely reads a bool from a map.
func getBoolField(m map[string]interface{}, key string) bool {
	v, ok := m[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}
