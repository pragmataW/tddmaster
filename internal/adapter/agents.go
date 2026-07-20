package adapter

import (
	"os"
	"strings"

	"github.com/pragmataW/tddmaster/internal/errs"
	"github.com/pragmataW/tddmaster/internal/prompts"
)

const (
	markerStart = "<!-- tddmasterStart -->"
	markerEnd   = "<!-- tddmasterEnd -->"
)

type AgentSpec struct {
	File         string
	Name         string
	Description  string
	Tools        string
	Model        string
	BodyTemplate string
}

var AgentSpecs = []AgentSpec{
	{
		File:         "tddmaster-executor",
		Name:         "tddmaster-executor",
		Description:  "Executes a single tddmaster task.",
		Tools:        "Read, Edit, MultiEdit, Write, Bash, Grep, Glob, LS",
		Model:        "sonnet",
		BodyTemplate: "executor",
	},
	{
		File:         "tddmaster-verifier",
		Name:         "tddmaster-verifier",
		Description:  "Independently verifies completed task work. Read-only. Never sees the executor's context.",
		Tools:        "Read, Bash, Grep, Glob, LS",
		Model:        "opus",
		BodyTemplate: "verifier",
	},
	{
		File:         "tddmaster-planner",
		Name:         "tddmaster-planner",
		Description:  "Produces a structured implementation plan for an important tddmaster task. Read-only — does not edit code.",
		Tools:        "Read, Grep, Glob, LS",
		Model:        "opus",
		BodyTemplate: "planner",
	},
	{
		File:         "tddmaster-test-writer",
		Name:         "tddmaster-test-writer",
		Description:  "Writes tests FIRST following TDD principles.",
		Tools:        "Read, Edit, MultiEdit, Write, Bash, Grep, Glob, LS",
		Model:        "sonnet",
		BodyTemplate: "test-writer",
	},
	{
		File:         "tddmaster-rule-synthesizer",
		Name:         "tddmaster-rule-synthesizer",
		Description:  "Synthesizes reusable tddmaster rules from accumulated refactor notes and failed AC reasons.",
		Tools:        "Bash, Read",
		Model:        "sonnet",
		BodyTemplate: "rule-synthesizer",
	},
	{
		File:         "tddmaster-auditor",
		Name:         "tddmaster-auditor",
		Description:  "Audits the spec bundle for semantic quality issues across tasks and criteria. Read-only — does not edit code.",
		Tools:        "Read, Grep, Glob, LS",
		Model:        "opus",
		BodyTemplate: "auditor",
	},
}

func renderBody(spec AgentSpec, cmd string) (string, error) {
	body, err := prompts.Render(spec.BodyTemplate, prompts.RenderData{Command: cmd})
	if err != nil {
		return "", errs.Wrap(errs.KeyAdapterRenderBody, err, spec.BodyTemplate)
	}
	return body, nil
}

func injectMarkedDoc(docPath, rendered string) error {
	block := markerStart + "\n" + rendered + "\n" + markerEnd

	existing, readErr := os.ReadFile(docPath)
	if readErr != nil && !os.IsNotExist(readErr) {
		return errs.Wrap(errs.KeyReadFile, readErr, docPath)
	}

	var newContent string
	if readErr != nil {
		newContent = block
	} else {
		content := string(existing)
		startIdx := strings.Index(content, markerStart)
		endIdx := -1
		if startIdx != -1 {
			rel := strings.Index(content[startIdx+len(markerStart):], markerEnd)
			if rel != -1 {
				endIdx = startIdx + len(markerStart) + rel
			}
		}
		if startIdx != -1 && endIdx != -1 {
			newContent = content[:startIdx] + block + content[endIdx+len(markerEnd):]
		} else {
			content = strings.ReplaceAll(content, markerStart, "")
			content = strings.ReplaceAll(content, markerEnd, "")
			if len(content) > 0 && !strings.HasSuffix(content, "\n") {
				newContent = content + "\n" + block
			} else {
				newContent = content + block
			}
		}
	}

	if err := os.WriteFile(docPath, []byte(newContent), 0o644); err != nil {
		return errs.Wrap(errs.KeyAdapterWriteDoc, err, docPath)
	}
	return nil
}
