package errs

import "fmt"

var templateMap = map[ErrorKey]string{
	KeyResolveRoot:          "resolve root",
	KeyInvalidSlug:          "invalid slug %q",
	KeySpecDoesNotExist:     "spec %q does not exist",
	KeyNoTTYForce:           "no TTY detected: pass --force to skip confirmation",
	KeyNoTTYNonInteractive:  "no TTY detected: pass --non-interactive to skip prompts",
	KeyCancelSpec:           "cancel spec",
	KeyListSpecs:            "list specs",
	KeyStartSpec:            "start spec",
	KeyArchiveSpec:          "archive spec",
	KeyRestoreSpec:          "restore spec",
	KeyRestoreConflict:      "restore conflict: spec %q is already active",
	KeySpecDirNotFound:      "spec directory not found for slug %q: make sure the slug is correct and exists in .tddmaster/specs/",
	KeyListenPort:           "failed to listen on an available port",
	KeyWebServer:            "web server failed",
	KeySpecNotFoundRunStart: "spec %q not found: run tddmaster start %s first",
	KeyRefineWrongPhase:     "refine only valid in refinement phase, current phase: %s",
	KeyAnswerRequired:       "--answer is required",
	KeyInvalidJSONInAnswer:  "invalid JSON in --answer",
	KeyInvalidJSONInAnswerQ: "invalid JSON in --answer: %q",
	KeyUnmarshalAnswer:      "unmarshal answer",
	KeySaveProgress:         "save progress",
	KeySaveSpecMDRollback:   "save spec md: %v (progress rollback also failed: %v)",
	KeySaveSpecMDRolledBack: "save spec md (progress rolled back)",
	KeyPruneTraceability:    "prune traceability",
	KeyMarshalOutput:        "marshal output",
	KeyGetCwd:               "get cwd",
	KeyRuleNonInteractive:   "non-interactive mode requires both --scope and --name",
	KeyContentExclusive:     "--content and --content-file are mutually exclusive",
	KeyReadContentFile:      "read content-file %q",
	KeyBuildContext:         "build context",
	KeyEngine:               "engine",
	KeyMarshalAction:        "marshal action",
	KeyToolRequiredInit:     "at least one tool is required: use --tools=claude-code",
	KeyScaffold:             "scaffold",

	KeyLoadState:    "load state",
	KeyLoadProgress: "load progress",
	KeyLoadSettings: "load settings",
	KeyLoadRules:    "load rules",
	KeySaveState:    "save state",

	KeyForm:                 "form",
	KeyToolMustSelect:       "at least one tool must be selected",
	KeyEnterValidInteger:    "enter a valid integer",
	KeyValueGreaterThanZero: "value must be greater than zero",

	KeyUnknownTarget:  "unknown target %q",
	KeyEmptySlug:      "name %q produces an empty slug",
	KeyPathEscape:     "path escape detected: %q",
	KeyMkdir:          "mkdir %q",
	KeyWriteFile:      "write %q",
	KeyRuleFileExists: "rule file already exists: %q",
	KeyCreateFile:     "create %q",
	KeyCloseFile:      "close %q",
	KeyUI:             "ui",

	KeyReadFile:             "read %s",
	KeyParseFile:            "parse %s",
	KeyInvalidSlugMustMatch: "invalid slug %q: must match %s",
	KeyManifestNotFound:     "manifest not found: run 'tddmaster init' first",
	KeySpecInArchive:        "spec %q exists in the archive: run 'tddmaster restore %s' first or pick another slug",
	KeyDupTaskIDRemove:      "duplicate task id in remove: %s",
	KeyUnknownTaskID:        "unknown task id: %s",
	KeyAddRequiresTitle:     "add op requires a non-empty title",
	KeyCannotRemoveDeps:     "cannot remove %s: %s depend on it; update their dependsOn in the same payload",

	KeyUnknownResetTarget:           "unknown reset target phase %q",
	KeyUnknownTargetPhase:           "unknown target phase %q: valid phases are %v",
	KeyRollbackRuleLearningDisabled: "cannot roll back to phase %q: rule learning is disabled for this spec",
	KeyRollbackUnrecognizedPhase:    "cannot roll back: current phase %q is not a recognized phase (valid phases are %v)",
	KeyRollbackNotEarlier:           "cannot roll back to %q: not earlier than current phase %q",
	KeySpecNotActive:                "spec %q is not active",
	KeySpecAlreadyArchived:          "spec %q is already archived",
	KeyArchivedSpecNotFound:         "archived spec %q not found",
	KeyActiveSpecExists:             "an active spec %q already exists",

	KeyAdapterCreateDir:        "create %s agents dir",
	KeyAdapterCreateDirDefault: "create agents dir",
	KeyAdapterRenderDoc:        "render agents doc",
	KeyAdapterWriteAgent:       "write %s agent %s",
	KeyAdapterWriteAgentFile:   "write agent file %s",
	KeyAdapterRenderBody:       "render %s body",
	KeyAdapterWriteDoc:         "write %s",
	KeyRenderClaudeMD:          "render claude_md",

	KeyCreateDashboardDir: "failed to create dashboard directory",
	KeyWriteDashboardHTML: "failed to write dashboard html",

	KeyUnknownTemplate: "unknown template %q",
	KeyParseTemplate:   "parse template %q",
	KeyExecuteTemplate: "execute template %q",

	KeyAtLeastOneTask:        "at least one task required",
	KeyTaskTitleRequired:     "task %d: title is required",
	KeyTaskACRequired:        "task %d: at least one acceptance criterion required",
	KeyInvalidJSONAnswer:     "invalid JSON answer",
	KeySelfReviewNeedApprove: "self-review requires approve",
	KeyExpectedApprove:       "expected \"approve\", got %q",
	KeyInvalidRefinePayload:  "invalid refine payload",
	KeyRefinementExpects:     "refinement expects 'approve' or 'done', got %q",
	KeyParseSettings:         "parse settings",
	KeyUnrecognizedApproval:  "unrecognized approval answer: %q",
	KeyInvalidProposalJSON:   "invalid proposal JSON",
	KeyProposalNeedsRule:     "proposal must contain at least one rule",

	KeyMarshalManifest:    "failed to marshal manifest",
	KeyWriteManifest:      "failed to write manifest",
	KeyToolRequired:       "at least one tool is required",
	KeyCreateTddmasterDir: "failed to create .tddmaster dir",
	KeyAdapter:            "adapter %s",

	KeySpecNotFoundInRoot: "spec %q not found in %q: run start first",

	KeyRefactorBypass:              "refactor bypass: cannot complete refactor phase with pending notes before applying refactor",
	KeyGateAnswerInvalid:           "gate answer must carry either a plan (accept) or planFeedback (revise/reject)",
	KeyTraceabilityEmpty:           "RED phase: traceability is required but report.Traceability is empty",
	KeyTraceabilityMissingTestPath: "RED phase: traceability entry missing TestFilePath",
	KeyTraceabilityMissingFunc:     "RED phase: traceability entry missing FunctionName",
	KeyTraceabilityMissingACEC:     "RED phase: traceability entry must have at least one AC or EC",
	KeyReportMissingTaskID:         "report missing taskId; ready tasks: %s",
	KeyTaskAlreadyDone:             "task %q is already done; ready tasks: %s",
	KeyTaskNotReady:                "task %q is not ready (waiting on dependencies); ready tasks: %s",
	KeyUnknownTaskIDReady:          "unknown taskId %q; ready tasks: %s",
	KeyNoApplicableStage:           "no applicable stage for task %s (exec: %+v)",
}

type registryError struct {
	key     ErrorKey
	msg     string
	wrapped error
}

func (e *registryError) Error() string {
	if e.wrapped != nil {
		if e.msg == "" {
			return e.wrapped.Error()
		}
		return e.msg + ": " + e.wrapped.Error()
	}
	return e.msg
}

func (e *registryError) Unwrap() error { return e.wrapped }

func (e *registryError) Is(target error) bool {
	s, ok := target.(*sentinelError)
	return ok && s.key == e.key
}

type sentinelError struct{ key ErrorKey }

func (s *sentinelError) Error() string { return string(s.key) }

func Sentinel(key ErrorKey) error { return &sentinelError{key: key} }

func template(key ErrorKey) string {
	tpl, ok := templateMap[key]
	if !ok {
		panic(fmt.Sprintf("errs: no template registered for key %q", key))
	}
	return tpl
}

func New(key ErrorKey) error {
	return &registryError{key: key, msg: template(key)}
}

func Newf(key ErrorKey, args ...any) error {
	return &registryError{key: key, msg: fmt.Sprintf(template(key), args...)}
}

func Msgf(key ErrorKey, args ...any) string {
	return fmt.Sprintf(template(key), args...)
}

func Wrap(key ErrorKey, err error, args ...any) error {
	msg := template(key)
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	return &registryError{key: key, msg: msg, wrapped: err}
}
