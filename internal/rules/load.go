package rules

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"github.com/pragmataW/tddmaster/internal/paths"
)

var KnownAgents = []string{"test-writer", "executor", "verifier", "planner", "auditor"}

type Set struct {
	root   string
	global []string
	agents map[string][]string
}

type Doc struct {
	Path string
	Body string
}

func Load(root string) (Set, error) {
	rulesDir := paths.Rules(root)
	relRulesDir, err := filepath.Rel(root, rulesDir)
	if err != nil {
		return Set{}, err
	}

	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Set{}, nil
		}
		return Set{}, err
	}

	s := Set{
		root:   root,
		agents: make(map[string][]string),
	}

	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			if !slices.Contains(KnownAgents, name) {
				continue
			}
			agentEntries, err := os.ReadDir(filepath.Join(rulesDir, name))
			if err != nil {
				return Set{}, err
			}
			for _, ae := range agentEntries {
				if ae.IsDir() {
					continue
				}
				if filepath.Ext(ae.Name()) != ".md" {
					continue
				}
				rel := filepath.Join(relRulesDir, name, ae.Name())
				s.agents[name] = append(s.agents[name], rel)
			}
			sort.Strings(s.agents[name])
		} else {
			if filepath.Ext(name) != ".md" {
				continue
			}
			rel := filepath.Join(relRulesDir, name)
			s.global = append(s.global, rel)
		}
	}

	sort.Strings(s.global)
	return s, nil
}

func (s Set) Docs(agent string) []Doc {
	paths := s.For(agent)
	if len(paths) == 0 {
		return nil
	}
	docs := make([]Doc, 0, len(paths))
	for _, rel := range paths {
		body, err := os.ReadFile(filepath.Join(s.root, rel))
		if err != nil {
			return nil
		}
		docs = append(docs, Doc{Path: rel, Body: string(body)})
	}
	return docs
}

func (s Set) For(agent string) []string {
	agentRules := s.agents[agent]
	if len(s.global) == 0 && len(agentRules) == 0 {
		return nil
	}
	result := make([]string, 0, len(s.global)+len(agentRules))
	result = append(result, s.global...)
	result = append(result, agentRules...)
	return result
}
