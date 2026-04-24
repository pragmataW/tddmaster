package shared

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestReadConventions_StripsTddmasterBlock(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "CLAUDE.md"),
		"Keep me\n\n"+NosStart+"\n\nTDDMASTER BLOCK\n"+NosEnd+"\n\nKeep me too\n")

	got := ReadConventions(root, ConventionSources{ProjectFile: "CLAUDE.md"})

	if strings.Contains(got, "TDDMASTER BLOCK") {
		t.Fatalf("tddmaster block was not stripped: %s", got)
	}
	if !strings.Contains(got, "Keep me") || !strings.Contains(got, "Keep me too") {
		t.Fatalf("surrounding content missing: %s", got)
	}
}

func TestReadConventions_NoDelimiters_FullContent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "CLAUDE.md"), "Plain content without delimiters")

	got := ReadConventions(root, ConventionSources{ProjectFile: "CLAUDE.md"})

	if !strings.Contains(got, "Plain content without delimiters") {
		t.Fatalf("full content missing: %s", got)
	}
}

func TestReadConventions_MissingFiles_EmptyString(t *testing.T) {
	root := t.TempDir()

	got := ReadConventions(root, ConventionSources{
		ProjectFile: "does-not-exist.md",
		HomeFile:    filepath.Join(root, "also-missing.md"),
	})

	if got != "" {
		t.Fatalf("expected empty when both files missing, got: %s", got)
	}
}

func TestReadConventions_ExpandsImports(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "CLAUDE.md"), "Before\n@RULES.md\nAfter\n")
	writeFile(t, filepath.Join(root, "RULES.md"), "IMPORTED CONTENT")

	got := ReadConventions(root, ConventionSources{ProjectFile: "CLAUDE.md"})

	if strings.Contains(got, "@RULES.md") {
		t.Fatalf("import directive was not expanded: %s", got)
	}
	if !strings.Contains(got, "IMPORTED CONTENT") {
		t.Fatalf("imported content missing: %s", got)
	}
}

func TestReadConventions_ImportCycle_Breaks(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "A.md"), "A-start\n@B.md\nA-end\n")
	writeFile(t, filepath.Join(root, "B.md"), "B-start\n@A.md\nB-end\n")

	got := ReadConventions(root, ConventionSources{ProjectFile: "A.md"})

	if !strings.Contains(got, "A-start") {
		t.Fatalf("A content missing: %s", got)
	}
	if !strings.Contains(got, "B-start") {
		t.Fatalf("B content missing: %s", got)
	}
}

func TestReadConventions_ImportDepth_Limited(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "L0.md"), "@L1.md")
	writeFile(t, filepath.Join(root, "L1.md"), "@L2.md")
	writeFile(t, filepath.Join(root, "L2.md"), "@L3.md")
	writeFile(t, filepath.Join(root, "L3.md"), "@L4.md")
	writeFile(t, filepath.Join(root, "L4.md"), "DEEP_CONTENT")

	got := ReadConventions(root, ConventionSources{ProjectFile: "L0.md"})

	if strings.Contains(got, "DEEP_CONTENT") {
		t.Fatalf("depth limit not enforced; reached L4: %s", got)
	}
}

func TestReadConventions_HomeAndProject_JoinedWithSeparator(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), "PROJECT_CONTENT")
	writeFile(t, filepath.Join(home, "AGENTS.md"), "HOME_CONTENT")

	got := ReadConventions(root, ConventionSources{
		ProjectFile: "AGENTS.md",
		HomeFile:    filepath.Join(home, "AGENTS.md"),
	})

	if !strings.Contains(got, "PROJECT_CONTENT") || !strings.Contains(got, "HOME_CONTENT") {
		t.Fatalf("missing one side: %s", got)
	}
	projectIdx := strings.Index(got, "PROJECT_CONTENT")
	sepIdx := strings.Index(got, "---")
	homeIdx := strings.Index(got, "HOME_CONTENT")
	if !(projectIdx < sepIdx && sepIdx < homeIdx) {
		t.Fatalf("order or separator wrong: project=%d sep=%d home=%d — %s", projectIdx, sepIdx, homeIdx, got)
	}
}

func TestConventionsPreamble_EmptyWhenInjectFalse(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "CLAUDE.md"), "Some content")

	got := ConventionsPreamble(root, ConventionSources{ProjectFile: "CLAUDE.md"}, []string{"rule-1"}, false)

	if got != "" {
		t.Fatalf("expected empty when inject=false, got: %s", got)
	}
}

func TestConventionsPreamble_EmptyWhenNoSources(t *testing.T) {
	root := t.TempDir()

	got := ConventionsPreamble(root, ConventionSources{ProjectFile: "nope.md"}, nil, true)

	if got != "" {
		t.Fatalf("expected empty with no sources, got: %s", got)
	}
}

func TestConventionsPreamble_HeaderAndContent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "CLAUDE.md"), "Project body")

	got := ConventionsPreamble(root, ConventionSources{ProjectFile: "CLAUDE.md"}, nil, true)

	if !strings.Contains(got, "## Project Conventions") {
		t.Fatalf("missing header: %s", got)
	}
	if !strings.Contains(got, "Project body") {
		t.Fatalf("missing content: %s", got)
	}
}

func TestConventionsPreamble_RulesOnly(t *testing.T) {
	root := t.TempDir()

	got := ConventionsPreamble(root, ConventionSources{ProjectFile: "missing.md"}, []string{"rule-A", "rule-B"}, true)

	if !strings.Contains(got, "### Active Rules") {
		t.Fatalf("missing active rules header: %s", got)
	}
	if !strings.Contains(got, "- rule-A") || !strings.Contains(got, "- rule-B") {
		t.Fatalf("rule bullets missing: %s", got)
	}
}

func TestConventionsPreamble_ConventionsPlusRules(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "CLAUDE.md"), "Proj body")

	got := ConventionsPreamble(root, ConventionSources{ProjectFile: "CLAUDE.md"}, []string{"rule-X"}, true)

	projIdx := strings.Index(got, "Proj body")
	rulesIdx := strings.Index(got, "### Active Rules")
	ruleIdx := strings.Index(got, "- rule-X")
	if projIdx == -1 || rulesIdx == -1 || ruleIdx == -1 {
		t.Fatalf("missing sections: %s", got)
	}
	if !(projIdx < rulesIdx && rulesIdx < ruleIdx) {
		t.Fatalf("order wrong: proj=%d rules=%d rule=%d", projIdx, rulesIdx, ruleIdx)
	}
}

func TestConventionsPreamble_StripsTddmasterBlockFromHome(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeFile(t, filepath.Join(home, "CLAUDE.md"),
		"Before\n"+NosStart+"\nsecret\n"+NosEnd+"\nAfter")

	got := ConventionsPreamble(root, ConventionSources{HomeFile: filepath.Join(home, "CLAUDE.md")}, nil, true)

	if strings.Contains(got, "secret") {
		t.Fatalf("home tddmaster block not stripped: %s", got)
	}
	if !strings.Contains(got, "Before") || !strings.Contains(got, "After") {
		t.Fatalf("surrounding home content missing: %s", got)
	}
}

