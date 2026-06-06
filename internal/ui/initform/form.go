package initform

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/pragmataW/tddmaster/internal/manifest"
)

type FormResult struct {
	Tools        []manifest.ToolID
	MaxIteration int
	Confirmed    bool
}

func Run(existing manifest.Manifest) (FormResult, error) {
	isRerun := len(existing.SelectedTools) > 0

	var bannerBody string
	if isRerun {
		bannerBody = "existing configuration found — updating it"
	} else {
		bannerBody = "TDD-driven spec orchestration for AI-assisted development."
	}

	var selectedTools []manifest.ToolID
	selectedTools = append(selectedTools, existing.SelectedTools...)

	maxIterStr := strconv.Itoa(existing.MaxIterationBeforeStart)
	if existing.MaxIterationBeforeStart <= 0 {
		maxIterStr = "15"
	}

	var confirmed bool

	var toolOpts []huh.Option[manifest.ToolID]
	for _, e := range manifest.Catalog {
		toolOpts = append(toolOpts, huh.NewOption(e.Label, e.ID))
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("tddmaster").
				Description(bannerBody),
		),
		huh.NewGroup(
			huh.NewMultiSelect[manifest.ToolID]().
				Title("Which AI tools should files be generated for?").
				Description("CLAUDE.md and sub-agent definitions are written for each selected tool.").
				Options(toolOpts...).
				Value(&selectedTools).
				Validate(func(v []manifest.ToolID) error {
					if len(v) == 0 {
						return fmt.Errorf("at least one tool must be selected")
					}
					return nil
				}),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Maximum verification iterations before execution stops?").
				Description("Enter a positive integer (default: 15).").
				Value(&maxIterStr).
				Validate(func(s string) error {
					n, err := strconv.Atoi(s)
					if err != nil {
						return fmt.Errorf("enter a valid integer")
					}
					if n <= 0 {
						return fmt.Errorf("value must be greater than zero")
					}
					return nil
				}),
		),
		huh.NewGroup(
			huh.NewConfirm().
				Title("Proceed with this configuration?").
				Value(&confirmed),
		),
	).WithTheme(brandTheme())

	if err := form.Run(); err != nil {
		return FormResult{}, fmt.Errorf("form: %w", err)
	}

	maxIter, _ := strconv.Atoi(maxIterStr)

	return FormResult{
		Tools:        selectedTools,
		MaxIteration: maxIter,
		Confirmed:    confirmed,
	}, nil
}
