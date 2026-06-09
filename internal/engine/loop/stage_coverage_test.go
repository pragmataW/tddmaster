package loop

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/spec"
)

func TestFileCoverageEntry_JSONTags(t *testing.T) {
	entry := FileCoverageEntry{File: "internal/foo/bar.go", Coverage: 85}
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	raw := string(data)
	if !strings.Contains(raw, `"file"`) {
		t.Error("expected json tag 'file' in marshaled output")
	}
	if !strings.Contains(raw, `"coverage"`) {
		t.Error("expected json tag 'coverage' in marshaled output")
	}
	var got FileCoverageEntry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.File != "internal/foo/bar.go" {
		t.Errorf("File: got %q, want %q", got.File, "internal/foo/bar.go")
	}
	if got.Coverage != 85 {
		t.Errorf("Coverage: got %v, want 85", got.Coverage)
	}
}

func TestStageReport_FileCoverage_JSONTag_OmitEmpty(t *testing.T) {
	r := StageReport{Passed: true}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(data), `"fileCoverage"`) {
		t.Error("expected 'fileCoverage' to be omitted when nil")
	}
}

func TestStageReport_FileCoverage_JSONTag_Present(t *testing.T) {
	r := StageReport{
		FileCoverage: []FileCoverageEntry{{File: "a.go", Coverage: 90}},
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(data), `"fileCoverage"`) {
		t.Error("expected 'fileCoverage' key in marshaled output when non-nil")
	}
}

func TestCoverageEnforced_ZeroReturnsFalse(t *testing.T) {
	s := spec.Settings{MinTestCoverage: 0}
	if coverageEnforced(s) {
		t.Error("coverageEnforced: got true, want false for MinTestCoverage=0")
	}
}

func TestCoverageEnforced_PositiveReturnsTrue(t *testing.T) {
	for _, v := range []int{1, 50, 80, 100} {
		s := spec.Settings{MinTestCoverage: v}
		if !coverageEnforced(s) {
			t.Errorf("coverageEnforced: got false, want true for MinTestCoverage=%d", v)
		}
	}
}

func TestCoverageMet_GateDisabled_AlwaysTrue(t *testing.T) {
	s := spec.Settings{MinTestCoverage: 0}

	withLow := StageReport{
		FileCoverage: []FileCoverageEntry{{File: "a.go", Coverage: 10}},
	}
	if !coverageMet(withLow, s) {
		t.Error("coverageMet: got false, want true when gate disabled and low coverage")
	}

	empty := StageReport{}
	if !coverageMet(empty, s) {
		t.Error("coverageMet: got false, want true when gate disabled and empty FileCoverage")
	}
}

func TestCoverageMet_AllFilesAboveThreshold_ReturnsTrue(t *testing.T) {
	s := spec.Settings{MinTestCoverage: 80}
	r := StageReport{
		FileCoverage: []FileCoverageEntry{
			{File: "a.go", Coverage: 90},
			{File: "b.go", Coverage: 85},
		},
	}
	if !coverageMet(r, s) {
		t.Error("coverageMet: got false, want true when all files >= threshold")
	}
}

func TestCoverageMet_OneFileBelowThreshold_ReturnsFalse(t *testing.T) {
	s := spec.Settings{MinTestCoverage: 80}
	r := StageReport{
		FileCoverage: []FileCoverageEntry{
			{File: "a.go", Coverage: 90},
			{File: "b.go", Coverage: 70},
		},
	}
	if coverageMet(r, s) {
		t.Error("coverageMet: got true, want false when one file below threshold")
	}
}

func TestCoverageMet_EmptyFileCoverage_GateEnabled_ReturnsFalse(t *testing.T) {
	s := spec.Settings{MinTestCoverage: 80}
	r := StageReport{}
	if coverageMet(r, s) {
		t.Error("coverageMet: got true, want false when FileCoverage empty and gate enabled (EC-1)")
	}
}

func TestCoverageMet_Boundary_79FailsThreshold80(t *testing.T) {
	s := spec.Settings{MinTestCoverage: 80}
	r := StageReport{
		FileCoverage: []FileCoverageEntry{{File: "a.go", Coverage: 79}},
	}
	if coverageMet(r, s) {
		t.Error("coverageMet: got true, want false for Coverage=79 with threshold=80 (EC-3)")
	}
}

func TestCoverageMet_Boundary_80MeetsThreshold80(t *testing.T) {
	s := spec.Settings{MinTestCoverage: 80}
	r := StageReport{
		FileCoverage: []FileCoverageEntry{{File: "a.go", Coverage: 80}},
	}
	if !coverageMet(r, s) {
		t.Error("coverageMet: got false, want true for Coverage=80 with threshold=80 (EC-3)")
	}
}

func TestCoverageMet_Boundary_81MeetsThreshold80(t *testing.T) {
	s := spec.Settings{MinTestCoverage: 80}
	r := StageReport{
		FileCoverage: []FileCoverageEntry{{File: "a.go", Coverage: 81}},
	}
	if !coverageMet(r, s) {
		t.Error("coverageMet: got false, want true for Coverage=81 with threshold=80 (EC-3)")
	}
}

func TestLowCoverageFiles_ReturnsBelowThresholdNames(t *testing.T) {
	s := spec.Settings{MinTestCoverage: 80}
	r := StageReport{
		FileCoverage: []FileCoverageEntry{
			{File: "low.go", Coverage: 60},
			{File: "ok.go", Coverage: 90},
			{File: "edge.go", Coverage: 79},
		},
	}
	got := lowCoverageFiles(r, s)
	if len(got) != 2 {
		t.Fatalf("lowCoverageFiles: got %d files, want 2", len(got))
	}
	found := map[string]bool{}
	for _, f := range got {
		found[f] = true
	}
	if !found["low.go"] {
		t.Error("lowCoverageFiles: expected 'low.go' in result")
	}
	if !found["edge.go"] {
		t.Error("lowCoverageFiles: expected 'edge.go' in result")
	}
	if found["ok.go"] {
		t.Error("lowCoverageFiles: did not expect 'ok.go' in result")
	}
}

func TestLowCoverageFiles_AllMeetThreshold_ReturnsEmpty(t *testing.T) {
	s := spec.Settings{MinTestCoverage: 80}
	r := StageReport{
		FileCoverage: []FileCoverageEntry{
			{File: "a.go", Coverage: 80},
			{File: "b.go", Coverage: 100},
		},
	}
	got := lowCoverageFiles(r, s)
	if len(got) != 0 {
		t.Errorf("lowCoverageFiles: got %v, want empty slice", got)
	}
}

func TestLowCoverageFiles_EmptyFileCoverage_ReturnsEmpty(t *testing.T) {
	s := spec.Settings{MinTestCoverage: 80}
	r := StageReport{}
	got := lowCoverageFiles(r, s)
	if len(got) != 0 {
		t.Errorf("lowCoverageFiles: got %v, want empty slice for nil FileCoverage", got)
	}
}

func TestCoverageMap_BuildsCorrectMap(t *testing.T) {
	r := StageReport{
		FileCoverage: []FileCoverageEntry{
			{File: "a.go", Coverage: 90},
			{File: "b.go", Coverage: 75},
		},
	}
	got := coverageMap(r)
	if len(got) != 2 {
		t.Fatalf("coverageMap: got %d entries, want 2", len(got))
	}
	if got["a.go"] != 90 {
		t.Errorf("coverageMap: a.go = %v, want 90", got["a.go"])
	}
	if got["b.go"] != 75 {
		t.Errorf("coverageMap: b.go = %v, want 75", got["b.go"])
	}
}

func TestCoverageMap_EmptyFileCoverage_ReturnsEmptyMap(t *testing.T) {
	r := StageReport{}
	got := coverageMap(r)
	if len(got) != 0 {
		t.Errorf("coverageMap: got %v, want empty map for nil FileCoverage", got)
	}
}
