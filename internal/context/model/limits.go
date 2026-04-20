package model

// Numeric thresholds centralised as named constants. These were scattered as
// magic numbers across compiler.go, prefill.go and split_detector.go before the
// model extraction. User opted to keep them as code-level constants rather than
// promoting them to NosManifest config.
const (
	// RichUserContextThreshold — listen-first payload above this rune length
	// is treated as rich context and fed into the prefill pipeline.
	RichUserContextThreshold = 200

	// RichDescriptionThreshold — spec description above this byte length is
	// embedded in RichDescriptionOutput for agent-side extraction.
	RichDescriptionThreshold = 500

	// MaxPlanSize — plan document bytes beyond this are silently skipped from
	// PlanContext embedding.
	MaxPlanSize = 50 * 1024

	// StaleSessionMS — elapsed milliseconds since last `next` call beyond which
	// the protocol guide is re-emitted. Note: historical name was the typo
	// `staleSesssionMS` (three s). See docs/bugs.md Context Package Findings.
	StaleSessionMS = 5 * 60 * 1000

	// VerificationOutputTruncateShort — cap on verification output embedded in a
	// single AC text.
	VerificationOutputTruncateShort = 200

	// VerificationOutputTruncateFull — cap on verification output embedded in
	// the top-level `verificationOutput` field.
	VerificationOutputTruncateFull = 2000

	// DefaultMaxIter — fallback MaxIterationsBeforeRestart when config omits it.
	DefaultMaxIter = 15

	// DebtUrgentThreshold — consecutive iterations of carry-over debt beyond
	// which the debt note is phrased urgently.
	DebtUrgentThreshold = 3

	// IdleOptionsCap — maximum InteractiveOption count rendered on IDLE.
	IdleOptionsCap = 4

	// ContinuableSpecsCap — maximum non-completed specs surfaced on IDLE.
	ContinuableSpecsCap = 2

	// Slug and task estimate bounds for split_detector / slugify.
	SlugMaxWords    = 4
	SlugMaxLength   = 50
	TaskEstimateMin = 2
	TaskEstimateMax = 5

	// SplitMinTotalTasks — split proposals below this estimated work are dropped.
	SplitMinTotalTasks = 3

	// Prefill segment boundaries.
	PrefillSegmentMinRunes = 12
	PrefillBasisMaxRunes   = 120
)
