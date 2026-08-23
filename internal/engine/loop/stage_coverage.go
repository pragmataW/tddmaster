package loop

import (
	"sort"
	"strings"

	"github.com/pragmataW/tddmaster/internal/spec"
)

type FileCoverageEntry struct {
	File     string  `json:"file"`
	Coverage float64 `json:"coverage"`
}

// isTestFile reports whether a path is a test file rather than production code.
// Coverage is measured on production files only; a test file has no coverage of
// its own to report.
func isTestFile(path string) bool {
	base := strings.ToLower(path)
	if i := strings.LastIndexAny(base, "/\\"); i >= 0 {
		base = base[i+1:]
	}
	ext := ""
	if i := strings.LastIndex(base, "."); i > 0 {
		ext = base[i:]
		base = base[:i]
	}
	switch {
	case strings.HasSuffix(base, "_test"), strings.HasSuffix(base, ".test"), strings.HasSuffix(base, ".spec"):
		return true
	case strings.HasPrefix(base, "test_") && ext == ".py":
		return true
	}
	return false
}

func measurableFiles(files []string) []string {
	out := make([]string, 0, len(files))
	for _, f := range files {
		if f == "" || isTestFile(f) {
			continue
		}
		out = append(out, f)
	}
	return out
}

// lowCoverageStateFiles lists the files whose last recorded coverage is below
// the configured threshold, sorted for stable prompt output.
func lowCoverageStateFiles(st spec.ExecState, s spec.Settings) []string {
	if !coverageEnforced(s) || len(st.LastCoverage) == 0 {
		return nil
	}
	threshold := float64(s.MinTestCoverage)
	var low []string
	for file, pct := range st.LastCoverage {
		if pct < threshold {
			low = append(low, file)
		}
	}
	sort.Strings(low)
	return low
}

func coverageEnforced(s spec.Settings) bool {
	return s.MinTestCoverage > 0
}

func lowCoverageFiles(r StageReport, s spec.Settings) []string {
	result := []string{}
	for _, e := range r.FileCoverage {
		if e.Coverage < float64(s.MinTestCoverage) {
			result = append(result, e.File)
		}
	}
	return result
}

func coverageMet(r StageReport, s spec.Settings) bool {
	if !coverageEnforced(s) {
		return true
	}
	return len(r.FileCoverage) > 0 && len(lowCoverageFiles(r, s)) == 0
}

func coverageMap(r StageReport) map[string]float64 {
	m := map[string]float64{}
	for _, e := range r.FileCoverage {
		m[e.File] = e.Coverage
	}
	return m
}
