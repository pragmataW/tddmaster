package prompts

import (
	"embed"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/pragmataW/tddmaster/internal/errs"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

type RenderData struct {
	Command           string
	ParallelSubagents bool
}

const (
	templateDir = "templates"
)

func Render(name string, data RenderData) (string, error) {
	path := templateDir + "/" + name + ".tmpl"
	content, err := templatesFS.ReadFile(path)
	if err != nil {
		return "", errs.Wrap(errs.KeyUnknownTemplate, err, name)
	}
	src := string(content)
	tmpl, err := template.New(name).Parse(src)
	if err != nil {
		return "", errs.Wrap(errs.KeyParseTemplate, err, name)
	}
	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", errs.Wrap(errs.KeyExecuteTemplate, err, name)
	}
	return sb.String(), nil
}

func TemplateNames() []string {
	entries, err := templatesFS.ReadDir(templateDir)
	if err != nil {
		panic(fmt.Sprintf("prompts: read template dir: %v", err))
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		n := e.Name()
		if trimmed, ok := strings.CutSuffix(n, ".tmpl"); ok {
			names = append(names, trimmed)
		}
	}
	sort.Strings(names)
	return names
}
