package engine

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pragmataW/tddmaster/internal/errs"
	"github.com/pragmataW/tddmaster/internal/manifest"
	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/rules"
	"github.com/pragmataW/tddmaster/internal/spec"
)

type Context struct {
	root         string
	slug         string
	command      string
	defs         []PhaseDef
	state        spec.State
	progress     spec.Progress
	settings     spec.Settings
	maxIteration int
	rules        rules.Set
}

func Build(root, slug string, defs []PhaseDef) (*Context, error) {
	if !spec.Exists(root, slug) {
		return nil, errs.Newf(errs.KeySpecNotFoundInRoot, slug, root)
	}

	state, err := spec.LoadState(root, slug)
	if err != nil {
		return nil, errs.Wrap(errs.KeyLoadState, err)
	}

	progress, err := spec.LoadProgress(root, slug)
	if err != nil {
		return nil, errs.Wrap(errs.KeyLoadProgress, err)
	}

	settings, err := spec.LoadSettings(root, slug)
	if err != nil {
		return nil, errs.Wrap(errs.KeyLoadSettings, err)
	}

	maxIter := manifest.Defaults().MaxIterationBeforeStart
	command := manifest.Defaults().Command
	if data, readErr := os.ReadFile(paths.Manifest(root)); readErr == nil {
		var m manifest.Manifest
		if jsonErr := json.Unmarshal(data, &m); jsonErr == nil {
			manifest.Normalize(&m)
			maxIter = m.MaxIterationBeforeStart
			command = m.Command
		}
	}

	ruleSet, err := rules.Load(root)
	if err != nil {
		return nil, errs.Wrap(errs.KeyLoadRules, err)
	}

	return &Context{
		root:         root,
		slug:         slug,
		command:      command,
		defs:         defs,
		state:        state,
		progress:     progress,
		settings:     settings,
		maxIteration: maxIter,
		rules:        ruleSet,
	}, nil
}

func (c *Context) Rules() rules.Set {
	return c.rules
}

func (c *Context) Phase() PhaseID {
	return PhaseID(c.state.Phase)
}

func (c *Context) Slug() string {
	return c.slug
}

func (c *Context) Root() string {
	return c.root
}

func (c *Context) State() spec.State {
	return c.state
}

func (c *Context) WriteSpecMd(content string) error {
	return spec.SaveSpecMd(c.root, c.slug, content)
}

func (c *Context) activePhaseDef() *PhaseDef {
	current := c.Phase()
	if current == PhaseComplete {
		return nil
	}
	for i := range c.defs {
		if c.defs[i].ID == current {
			return &c.defs[i]
		}
	}
	return nil
}

// RewindPhase moves the spec back to an earlier phase. A driver needs it when
// its own answer says the previous phase was not finished after all; the phase
// machine otherwise only ever moves forward.
func (c *Context) RewindPhase(target PhaseID) error {
	c.state.Phase = string(target)
	if err := spec.SaveState(c.root, c.slug, c.state); err != nil {
		return errs.Wrap(errs.KeySaveState, err)
	}
	return nil
}

func (c *Context) advancePhase() error {
	next := NextPhase(c.defs, PhaseID(c.state.Phase))
	c.state.Phase = string(next)
	if err := spec.SaveState(c.root, c.slug, c.state); err != nil {
		return errs.Wrap(errs.KeySaveState, err)
	}
	return nil
}

// terminalMessage gives the caller something to show when a spec ends; a bare
// terminal action leaves the orchestrator with nothing to report to the user.
func (c *Context) terminalMessage() string {
	tasks := c.Progress().Tasks
	done := 0
	for _, t := range tasks {
		if t.Done {
			done++
		}
	}
	return fmt.Sprintf("Spec %q is complete: %d/%d tasks done. Artifacts are under %s.",
		c.slug, done, len(tasks), paths.SpecDir(c.root, c.slug))
}

// answerPlaceholder describes what the caller has to put after --answer= for a
// given input format, so the emitted submitCmd is copy-pasteable.
func answerPlaceholder(f InputFormat) string {
	switch f {
	case FormatJSON:
		return "<json>"
	case FormatFlag:
		return "<flag>"
	default:
		return "<text>"
	}
}

// withSubmitCmd fills expectedInput.submitCmd on every action that expects an
// answer. Drivers used to leave it empty, which forced the caller to rebuild the
// command from the slug by hand even though the contract advertises the field.
func (c *Context) withSubmitCmd(a Action) Action {
	switch a.Action {
	case ActionAsk, ActionInstruct:
		if a.ExpectedInput.SubmitCmd == "" {
			a.ExpectedInput.SubmitCmd = fmt.Sprintf("%s next %s --answer='%s'",
				c.command, c.slug, answerPlaceholder(a.ExpectedInput.Format))
		}
	case ActionNotify:
		if a.ExpectedInput.SubmitCmd == "" {
			a.ExpectedInput.SubmitCmd = fmt.Sprintf("%s next %s", c.command, c.slug)
		}
	}
	for i := range a.Tasks {
		if a.Tasks[i].ExpectedInput.SubmitCmd != "" {
			continue
		}
		a.Tasks[i].ExpectedInput.SubmitCmd = fmt.Sprintf("%s next %s --answer='%s'",
			c.command, c.slug, answerPlaceholder(a.Tasks[i].ExpectedInput.Format))
	}
	return a
}

func (c *Context) Next() (Action, error) {
	ph := c.activePhaseDef()
	if ph == nil {
		return Action{Action: ActionTerminal, Instruction: c.terminalMessage()}, nil
	}
	action, phaseDone := ph.Driver.Next(c, ph)
	if phaseDone {
		if err := c.advancePhase(); err != nil {
			return Action{}, err
		}
		if action.Action == "" {
			return c.Next()
		}
	}
	return c.withSubmitCmd(action), nil
}

func (c *Context) Progress() spec.Progress {
	return c.progress
}

func (c *Context) SaveProgress(p spec.Progress) error {
	if err := spec.SaveProgress(c.root, c.slug, p); err != nil {
		return err
	}
	c.progress = p
	return nil
}

func (c *Context) Settings() spec.Settings {
	return c.settings
}

func (c *Context) SaveSettings(s spec.Settings) error {
	if err := spec.SaveSettings(c.root, c.slug, s); err != nil {
		return err
	}
	c.settings = s
	return nil
}

func (c *Context) MaxIteration() int {
	return c.maxIteration
}

func (c *Context) LoadTraceability() (spec.Traceability, error) {
	return spec.LoadTraceability(c.root, c.slug)
}

func (c *Context) SaveTraceability(t spec.Traceability) error {
	return spec.SaveTraceability(c.root, c.slug, t)
}

func (c *Context) SaveAnalysis(a spec.Analysis) error {
	return spec.SaveAnalysis(c.root, c.slug, a)
}

func (c *Context) AnswerValue(key string) string {
	entries, ok := c.state.Answers[key]
	if !ok || len(entries) == 0 {
		return ""
	}
	return entries[0].Value
}

func (c *Context) HasAnswer(key string) bool {
	entries, ok := c.state.Answers[key]
	if !ok || len(entries) == 0 {
		return false
	}
	return entries[0].Value != ""
}

func (c *Context) SetAnswer(key, value string) error {
	if c.state.Answers == nil {
		c.state.Answers = make(map[string][]spec.Answer)
	}
	c.state.Answers[key] = []spec.Answer{{Key: key, Value: value}}
	return spec.SaveState(c.root, c.slug, c.state)
}

func (c *Context) Submit(answer []byte) (Action, error) {
	ph := c.activePhaseDef()
	if ph == nil {
		return Action{Action: ActionTerminal, Instruction: c.terminalMessage()}, nil
	}

	action, phaseDone, err := ph.Driver.Submit(c, ph, answer)
	if err != nil {
		return Action{}, err
	}

	if phaseDone {
		if err := c.advancePhase(); err != nil {
			return Action{}, err
		}
	}

	if action.Action == "" {
		return c.Next()
	}

	return c.withSubmitCmd(action), nil
}
