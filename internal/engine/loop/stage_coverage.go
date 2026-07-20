package loop

import "github.com/pragmataW/tddmaster/internal/spec"

type FileCoverageEntry struct {
	File     string  `json:"file"`
	Coverage float64 `json:"coverage"`
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
