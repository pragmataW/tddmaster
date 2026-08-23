package projectcmds

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

type Command struct {
	Label string
	Value string
}

const (
	LabelBuild     = "build"
	LabelTypeCheck = "type-check"
	LabelTest      = "test"
	LabelLint      = "lint"
	LabelCoverage  = "coverage"
	goCoverageCmd  = `(coverage_file="$(mktemp "${TMPDIR:-/tmp}/tddmaster-coverage.XXXXXX")" && trap 'rm -f "$coverage_file"' 0 && go test -coverprofile="$coverage_file" ./... && go tool cover -func="$coverage_file")`
)

var labelOrder = map[string]int{
	LabelBuild:     0,
	LabelTypeCheck: 1,
	LabelTest:      2,
	LabelLint:      3,
	LabelCoverage:  4,
}

var makeTargetRe = regexp.MustCompile(`(?m)^([A-Za-z0-9_.-]+):`)

var makeTargetLabels = map[string]string{
	"build":    LabelBuild,
	"check":    LabelTypeCheck,
	"test":     LabelTest,
	"lint":     LabelLint,
	"cover":    LabelCoverage,
	"coverage": LabelCoverage,
}

var scriptLabels = map[string]string{
	"build":         LabelBuild,
	"typecheck":     LabelTypeCheck,
	"type-check":    LabelTypeCheck,
	"test":          LabelTest,
	"lint":          LabelLint,
	"coverage":      LabelCoverage,
	"test:coverage": LabelCoverage,
}

type collector struct {
	seen map[string]bool
	out  []Command
}

func (c *collector) add(label, value string) {
	if label == "" || value == "" || c.seen[label] {
		return
	}
	c.seen[label] = true
	c.out = append(c.out, Command{Label: label, Value: value})
}

func exists(root, name string) bool {
	_, err := os.Stat(filepath.Join(root, name))
	return err == nil
}

func Detect(root string) []Command {
	if root == "" {
		return nil
	}
	c := &collector{seen: map[string]bool{}}

	detectMake(root, c)
	detectGo(root, c)
	detectNode(root, c)
	detectRust(root, c)
	detectPython(root, c)
	detectJVM(root, c)

	sort.SliceStable(c.out, func(i, j int) bool {
		return labelOrder[c.out[i].Label] < labelOrder[c.out[j].Label]
	})
	return c.out
}

func detectMake(root string, c *collector) {
	data, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		return
	}
	for _, m := range makeTargetRe.FindAllStringSubmatch(string(data), -1) {
		if label, ok := makeTargetLabels[m[1]]; ok {
			c.add(label, "make "+m[1])
		}
	}
}

func detectGo(root string, c *collector) {
	if !exists(root, "go.mod") {
		return
	}
	c.add(LabelBuild, "go build ./...")
	c.add(LabelTest, "go test ./...")
	c.add(LabelCoverage, goCoverageCmd)
}

func packageManager(root string) string {
	switch {
	case exists(root, "pnpm-lock.yaml"):
		return "pnpm"
	case exists(root, "yarn.lock"):
		return "yarn"
	case exists(root, "bun.lockb"):
		return "bun"
	default:
		return "npm"
	}
}

func detectNode(root string, c *collector) {
	data, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return
	}
	pm := packageManager(root)
	names := make([]string, 0, len(pkg.Scripts))
	for name := range pkg.Scripts {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if label, ok := scriptLabels[name]; ok {
			c.add(label, pm+" run "+name)
		}
	}
}

func detectRust(root string, c *collector) {
	if !exists(root, "Cargo.toml") {
		return
	}
	c.add(LabelTypeCheck, "cargo check")
	c.add(LabelTest, "cargo test")
}

func detectPython(root string, c *collector) {
	if !exists(root, "pyproject.toml") && !exists(root, "setup.cfg") {
		return
	}
	c.add(LabelTest, "pytest")
}

func detectJVM(root string, c *collector) {
	switch {
	case exists(root, "pom.xml"):
		c.add(LabelTest, "mvn -q test")
	case exists(root, "build.gradle"), exists(root, "build.gradle.kts"):
		c.add(LabelTest, "gradle test")
	}
}
