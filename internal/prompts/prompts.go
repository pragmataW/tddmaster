package prompts

import (
	"embed"
	"fmt"
	"sort"
	"strings"
	"text/template"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

type RenderData struct {
	Command string
}

const (
	templateDir = "templates"
)

func Render(name string, data RenderData) (string, error) {
	path := templateDir + "/" + name + ".tmpl"
	content, err := templatesFS.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("unknown template %q: %w", name, err)
	}
	src := string(content)
	tmpl, err := template.New(name).Parse(src)
	if err != nil {
		return "", fmt.Errorf("parse template %q: %w", name, err)
	}
	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", fmt.Errorf("execute template %q: %w", name, err)
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
