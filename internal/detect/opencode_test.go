package detect

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pragmataW/tddmaster/internal/state"
)

// containsTool is a small helper that checks whether a CodingToolId appears
// in the result slice returned by DetectCodingTools.
func containsTool(tools []state.CodingToolId, id state.CodingToolId) bool {
	for _, t := range tools {
		if t == id {
			return true
		}
	}
	return false
}

// TestDetectCodingTools_DetectsOpenCode_FromDotOpencodeDir verifies that
// DetectCodingTools recognises a project that has a .opencode directory.
func TestDetectCodingTools_DetectsOpenCode_FromDotOpencodeDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create the .opencode directory signal.
	if err := os.MkdirAll(filepath.Join(tmpDir, ".opencode"), 0o755); err != nil {
		t.Fatalf("MkdirAll .opencode: %v", err)
	}

	tools := DetectCodingTools(tmpDir)
	if !containsTool(tools, state.CodingToolOpencode) {
		t.Errorf("DetectCodingTools: want CodingToolOpencode in result, got %v", tools)
	}
}

// TestDetectCodingTools_DetectsOpenCode_FromOpencodeJsonFile verifies that
// DetectCodingTools recognises a project that has an opencode.json file.
func TestDetectCodingTools_DetectsOpenCode_FromOpencodeJsonFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create the opencode.json file signal.
	jsonPath := filepath.Join(tmpDir, "opencode.json")
	if err := os.WriteFile(jsonPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("WriteFile opencode.json: %v", err)
	}

	tools := DetectCodingTools(tmpDir)
	if !containsTool(tools, state.CodingToolOpencode) {
		t.Errorf("DetectCodingTools: want CodingToolOpencode in result, got %v", tools)
	}
}

// TestDetectCodingTools_NoOpenCode_WhenAbsent verifies that an empty project
// directory does not produce a false-positive OpenCode detection.
func TestDetectCodingTools_NoOpenCode_WhenAbsent(t *testing.T) {
	tmpDir := t.TempDir() // empty — no signals present

	tools := DetectCodingTools(tmpDir)
	if containsTool(tools, state.CodingToolOpencode) {
		t.Errorf("DetectCodingTools: opencode must NOT appear in result for empty dir, got %v", tools)
	}
}
