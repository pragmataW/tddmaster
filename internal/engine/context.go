package engine

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pragmataW/tddmaster/internal/manifest"
	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/rules"
	"github.com/pragmataW/tddmaster/internal/spec"
)

type Context struct {
	root         string
	slug         string
	defs         []PhaseDef
	state        spec.State
	progress     spec.Progress
	settings     spec.Settings
	maxIteration int
	rules        rules.Set
}

func Build(root, slug string, defs []PhaseDef) (*Context, error) {
	if !spec.Exists(root, slug) {
		return nil, fmt.Errorf("spec %q not found in %q: run start first", slug, root)
	}

	state, err := spec.LoadState(root, slug)
	if err != nil {
		return nil, fmt.Errorf("load state: %w", err)
	}

	progress, err := spec.LoadProgress(root, slug)
	if err != nil {
		return nil, fmt.Errorf("load progress: %w", err)
	}

	settings, err := spec.LoadSettings(root, slug)
	if err != nil {
		return nil, fmt.Errorf("load settings: %w", err)
	}

	maxIter := manifest.Defaults().MaxIterationBeforeStart
	if data, readErr := os.ReadFile(paths.Manifest(root)); readErr == nil {
		var m manifest.Manifest
		if jsonErr := json.Unmarshal(data, &m); jsonErr == nil {
			manifest.Normalize(&m)
			maxIter = m.MaxIterationBeforeStart
		}
	}

	ruleSet, err := rules.Load(root)
	if err != nil {
		return nil, fmt.Errorf("load rules: %w", err)
	}

	return &Context{
		root:         root,
		slug:         slug,
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

func (c *Context) advancePhase() error {
	next := NextPhase(c.defs, PhaseID(c.state.Phase))
	c.state.Phase = string(next)
	if err := spec.SaveState(c.root, c.slug, c.state); err != nil {
		return fmt.Errorf("save state: %w", err)
	}
	return nil
}

func (c *Context) Next() (Action, error) {
	ph := c.activePhaseDef()
	if ph == nil {
		return Action{Action: ActionTerminal}, nil
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
	return action, nil
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
		return Action{Action: ActionTerminal}, nil
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

	return action, nil
}
