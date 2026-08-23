package projectcmds

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func labels(cmds []Command) map[string]string {
	out := make(map[string]string, len(cmds))
	for _, c := range cmds {
		out[c.Label] = c.Value
	}
	return out
}

func TestDetect_EmptyRootReturnsNothing(t *testing.T) {
	if got := Detect(""); got != nil {
		t.Fatalf("Detect(\"\"): got %v, want nil", got)
	}
	if got := Detect(t.TempDir()); got != nil {
		t.Fatalf("Detect(empty dir): got %v, want nil", got)
	}
}

func TestDetect_Go(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "go.mod", "module example.com/x\n")

	got := labels(Detect(dir))
	if got[LabelBuild] != "go build ./..." {
		t.Fatalf("build: got %q", got[LabelBuild])
	}
	if got[LabelTest] != "go test ./..." {
		t.Fatalf("test: got %q", got[LabelTest])
	}
	if got[LabelCoverage] == "" {
		t.Fatal("go project must expose a coverage command")
	}
}

func TestDetect_GoCoverageUsesTemporaryProfileAndCleansIt(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "go.mod", "module example.com/x\n\ngo 1.22\n")
	write(t, dir, "x.go", "package x\n\nfunc X() int { return 1 }\n")
	write(t, dir, "x_test.go", "package x\n\nimport \"testing\"\n\nfunc TestX(t *testing.T) { if X() != 1 { t.Fatal(\"bad\") } }\n")

	coverageCommand := labels(Detect(dir))[LabelCoverage]
	if strings.Contains(coverageCommand, "coverage.out") {
		t.Fatalf("coverage command must not leave a fixed profile in the project: %q", coverageCommand)
	}
	if !strings.Contains(coverageCommand, "mktemp") || !strings.Contains(coverageCommand, "rm -f") {
		t.Fatalf("coverage command must create and clean a temporary profile: %q", coverageCommand)
	}

	cmd := exec.Command("sh", "-c", coverageCommand)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("coverage command failed: %v\n%s", err, output)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read project dir: %v", err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "coverage") {
			t.Fatalf("coverage command left project artifact %q", entry.Name())
		}
	}
}

func TestDetect_MakefileWinsOverLanguageDefault(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "go.mod", "module example.com/x\n")
	write(t, dir, "Makefile", "build:\n\tgo build ./...\ntest:\n\tgo test -race ./...\n")

	got := labels(Detect(dir))
	if got[LabelBuild] != "make build" {
		t.Fatalf("Makefile build target must win: got %q", got[LabelBuild])
	}
	if got[LabelTest] != "make test" {
		t.Fatalf("Makefile test target must win: got %q", got[LabelTest])
	}
}

func TestDetect_NodeUsesLockfilePackageManager(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "package.json", `{"scripts":{"build":"tsc","test":"vitest","lint":"eslint ."}}`)
	write(t, dir, "pnpm-lock.yaml", "lockfileVersion: 9\n")

	got := labels(Detect(dir))
	if got[LabelBuild] != "pnpm run build" {
		t.Fatalf("build: got %q", got[LabelBuild])
	}
	if got[LabelTest] != "pnpm run test" {
		t.Fatalf("test: got %q", got[LabelTest])
	}
	if got[LabelLint] != "pnpm run lint" {
		t.Fatalf("lint: got %q", got[LabelLint])
	}
}

func TestDetect_NodeSkipsScriptsThatDoNotExist(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "package.json", `{"scripts":{"start":"node ."}}`)

	if got := Detect(dir); len(got) != 0 {
		t.Fatalf("no known script present, want no commands, got %v", got)
	}
}

func TestDetect_RustAndPythonAndJVM(t *testing.T) {
	for _, tc := range []struct {
		name  string
		file  string
		body  string
		label string
		want  string
	}{
		{"rust", "Cargo.toml", "[package]\n", LabelTest, "cargo test"},
		{"python", "pyproject.toml", "[project]\n", LabelTest, "pytest"},
		{"maven", "pom.xml", "<project/>", LabelTest, "mvn -q test"},
		{"gradle", "build.gradle", "plugins {}\n", LabelTest, "gradle test"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			write(t, dir, tc.file, tc.body)
			if got := labels(Detect(dir))[tc.label]; got != tc.want {
				t.Fatalf("%s: got %q, want %q", tc.label, got, tc.want)
			}
		})
	}
}

func TestDetect_MalformedPackageJSONIsIgnored(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "package.json", "{not json")

	if got := Detect(dir); len(got) != 0 {
		t.Fatalf("malformed package.json must yield no commands, got %v", got)
	}
}

func TestDetect_OrdersBuildBeforeTest(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "package.json", `{"scripts":{"test":"vitest","build":"tsc"}}`)

	cmds := Detect(dir)
	if len(cmds) < 2 {
		t.Fatalf("expected build and test, got %v", cmds)
	}
	if cmds[0].Label != LabelBuild || cmds[1].Label != LabelTest {
		t.Fatalf("expected build before test, got %v", cmds)
	}
}
