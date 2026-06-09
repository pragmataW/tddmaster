package loop

import "github.com/pragmataW/tddmaster/internal/engine"

func persistCoverage(c *engine.Context, report StageReport) error {
	tr, err := c.LoadTraceability()
	if err != nil {
		return err
	}
	if tr.Coverage == nil {
		tr.Coverage = map[string]float64{}
	}
	for _, entry := range report.FileCoverage {
		tr.Coverage[entry.File] = entry.Coverage
	}
	return c.SaveTraceability(tr)
}
