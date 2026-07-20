package errs

type ErrorKey string

const (
	KeyResolveRoot          ErrorKey = "cmd:resolve-root"
	KeyInvalidSlug          ErrorKey = "cmd:invalid-slug"
	KeySpecDoesNotExist     ErrorKey = "cmd:spec-does-not-exist"
	KeyNoTTYForce           ErrorKey = "cmd:no-tty-force"
	KeyNoTTYNonInteractive  ErrorKey = "cmd:no-tty-non-interactive"
	KeyCancelSpec           ErrorKey = "cmd:cancel-spec"
	KeyListSpecs            ErrorKey = "cmd:list-specs"
	KeyStartSpec            ErrorKey = "cmd:start-spec"
	KeyArchiveSpec          ErrorKey = "cmd:archive-spec"
	KeyRestoreSpec          ErrorKey = "cmd:restore-spec"
	KeyRestoreConflict      ErrorKey = "cmd:restore-conflict"
	KeySpecDirNotFound      ErrorKey = "cmd:spec-dir-not-found"
	KeyListenPort           ErrorKey = "cmd:listen-port"
	KeyWebServer            ErrorKey = "cmd:web-server"
	KeySpecNotFoundRunStart ErrorKey = "cmd:spec-not-found-run-start"
	KeyRefineWrongPhase     ErrorKey = "cmd:refine-wrong-phase"
	KeyAnswerRequired       ErrorKey = "cmd:answer-required"
	KeyInvalidJSONInAnswer  ErrorKey = "cmd:invalid-json-in-answer"
	KeyInvalidJSONInAnswerQ ErrorKey = "cmd:invalid-json-in-answer-q"
	KeyUnmarshalAnswer      ErrorKey = "cmd:unmarshal-answer"
	KeySaveProgress         ErrorKey = "cmd:save-progress"
	KeySaveSpecMDRollback   ErrorKey = "cmd:save-spec-md-rollback"
	KeySaveSpecMDRolledBack ErrorKey = "cmd:save-spec-md-rolled-back"
	KeyPruneTraceability    ErrorKey = "cmd:prune-traceability"
	KeyMarshalOutput        ErrorKey = "cmd:marshal-output"
	KeyGetCwd               ErrorKey = "cmd:get-cwd"
	KeyRuleNonInteractive   ErrorKey = "cmd:rule-non-interactive"
	KeyContentExclusive     ErrorKey = "cmd:content-mutually-exclusive"
	KeyReadContentFile      ErrorKey = "cmd:read-content-file"
	KeyBuildContext         ErrorKey = "cmd:build-context"
	KeyEngine               ErrorKey = "cmd:engine"
	KeyMarshalAction        ErrorKey = "cmd:marshal-action"
	KeyToolRequiredInit     ErrorKey = "cmd:tool-required-init"
	KeyScaffold             ErrorKey = "cmd:scaffold"
)

const (
	KeyLoadState    ErrorKey = "state:load-state"
	KeyLoadProgress ErrorKey = "state:load-progress"
	KeyLoadSettings ErrorKey = "state:load-settings"
	KeyLoadRules    ErrorKey = "state:load-rules"
	KeySaveState    ErrorKey = "state:save-state"
)

const (
	KeyForm                 ErrorKey = "form:form"
	KeyToolMustSelect       ErrorKey = "form:tool-must-select"
	KeyEnterValidInteger    ErrorKey = "form:enter-valid-integer"
	KeyValueGreaterThanZero ErrorKey = "form:value-greater-than-zero"
)

const (
	KeyUnknownTarget  ErrorKey = "ruleform:unknown-target"
	KeyEmptySlug      ErrorKey = "ruleform:empty-slug"
	KeyPathEscape     ErrorKey = "ruleform:path-escape"
	KeyMkdir          ErrorKey = "ruleform:mkdir"
	KeyWriteFile      ErrorKey = "ruleform:write-file"
	KeyRuleFileExists ErrorKey = "ruleform:rule-file-exists"
	KeyCreateFile     ErrorKey = "ruleform:create-file"
	KeyCloseFile      ErrorKey = "ruleform:close-file"
	KeyUI             ErrorKey = "ruleform:ui"
)

const (
	KeyReadFile             ErrorKey = "spec:read-file"
	KeyParseFile            ErrorKey = "spec:parse-file"
	KeyInvalidSlugMustMatch ErrorKey = "spec:invalid-slug-must-match"
	KeyManifestNotFound     ErrorKey = "spec:manifest-not-found"
	KeySpecInArchive        ErrorKey = "spec:spec-in-archive"
	KeyDupTaskIDRemove      ErrorKey = "spec:dup-task-id-remove"
	KeyUnknownTaskID        ErrorKey = "spec:unknown-task-id"
	KeyAddRequiresTitle     ErrorKey = "spec:add-requires-title"
	KeyCannotRemoveDeps     ErrorKey = "spec:cannot-remove-dependents"
)

const (
	KeyUnknownResetTarget           ErrorKey = "lifecycle:unknown-reset-target"
	KeyUnknownTargetPhase           ErrorKey = "lifecycle:unknown-target-phase"
	KeyRollbackRuleLearningDisabled ErrorKey = "lifecycle:rollback-rule-learning-disabled"
	KeyRollbackUnrecognizedPhase    ErrorKey = "lifecycle:rollback-unrecognized-phase"
	KeyRollbackNotEarlier           ErrorKey = "lifecycle:rollback-not-earlier"
	KeySpecNotActive                ErrorKey = "lifecycle:spec-not-active"
	KeySpecAlreadyArchived          ErrorKey = "lifecycle:spec-already-archived"
	KeyArchivedSpecNotFound         ErrorKey = "lifecycle:archived-spec-not-found"
	KeyActiveSpecExists             ErrorKey = "lifecycle:active-spec-exists"
)

const (
	KeyAdapterCreateDir        ErrorKey = "adapter:create-dir"
	KeyAdapterCreateDirDefault ErrorKey = "adapter:create-dir-default"
	KeyAdapterRenderDoc        ErrorKey = "adapter:render-doc"
	KeyAdapterWriteAgent       ErrorKey = "adapter:write-agent"
	KeyAdapterWriteAgentFile   ErrorKey = "adapter:write-agent-file"
	KeyAdapterRenderBody       ErrorKey = "adapter:render-body"
	KeyAdapterWriteDoc         ErrorKey = "adapter:write-doc"
	KeyRenderClaudeMD          ErrorKey = "adapter:render-claude-md"
)

const (
	KeyCreateDashboardDir ErrorKey = "visualize:create-dashboard-dir"
	KeyWriteDashboardHTML ErrorKey = "visualize:write-dashboard-html"
)

const (
	KeyUnknownTemplate ErrorKey = "prompts:unknown-template"
	KeyParseTemplate   ErrorKey = "prompts:parse-template"
	KeyExecuteTemplate ErrorKey = "prompts:execute-template"
)

const (
	KeyAtLeastOneTask        ErrorKey = "phases:at-least-one-task"
	KeyTaskTitleRequired     ErrorKey = "phases:task-title-required"
	KeyTaskACRequired        ErrorKey = "phases:task-ac-required"
	KeyInvalidJSONAnswer     ErrorKey = "phases:invalid-json-answer"
	KeySelfReviewNeedApprove ErrorKey = "phases:self-review-need-approve"
	KeyExpectedApprove       ErrorKey = "phases:expected-approve"
	KeyInvalidRefinePayload  ErrorKey = "phases:invalid-refine-payload"
	KeyRefinementExpects     ErrorKey = "phases:refinement-expects"
	KeyParseSettings         ErrorKey = "phases:parse-settings"
	KeyUnrecognizedApproval  ErrorKey = "phases:unrecognized-approval"
	KeyInvalidProposalJSON   ErrorKey = "phases:invalid-proposal-json"
	KeyProposalNeedsRule     ErrorKey = "phases:proposal-needs-rule"
)

const (
	KeyMarshalManifest    ErrorKey = "scaffold:marshal-manifest"
	KeyWriteManifest      ErrorKey = "scaffold:write-manifest"
	KeyToolRequired       ErrorKey = "scaffold:tool-required"
	KeyCreateTddmasterDir ErrorKey = "scaffold:create-tddmaster-dir"
	KeyAdapter            ErrorKey = "scaffold:adapter"
)

const (
	KeySpecNotFoundInRoot ErrorKey = "engine:spec-not-found-in-root"
)

const (
	KeyRefactorBypass              ErrorKey = "loop:refactor-bypass"
	KeyGateAnswerInvalid           ErrorKey = "loop:gate-answer-invalid"
	KeyTraceabilityEmpty           ErrorKey = "loop:traceability-empty"
	KeyTraceabilityMissingTestPath ErrorKey = "loop:traceability-missing-test-path"
	KeyTraceabilityMissingFunc     ErrorKey = "loop:traceability-missing-func"
	KeyTraceabilityMissingACEC     ErrorKey = "loop:traceability-missing-ac-ec"
	KeyReportMissingTaskID         ErrorKey = "loop:report-missing-task-id"
	KeyTaskAlreadyDone             ErrorKey = "loop:task-already-done"
	KeyTaskNotReady                ErrorKey = "loop:task-not-ready"
	KeyUnknownTaskIDReady          ErrorKey = "loop:unknown-task-id-ready"
	KeyNoApplicableStage           ErrorKey = "loop:no-applicable-stage"
)
