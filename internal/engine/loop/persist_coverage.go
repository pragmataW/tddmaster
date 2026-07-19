package loop

import "github.com/pragmataW/tddmaster/internal/engine"

func persistCoverage(c *engine.Context, taskID string, report StageReport) error {
	tr, err := c.LoadTraceability()
	if err != nil {
		return err
	}
	if tr.Coverage == nil {
		tr.Coverage = map[string]map[string]float64{}
	}
	if tr.Coverage[taskID] == nil {
		tr.Coverage[taskID] = map[string]float64{}
	}
	for _, entry := range report.FileCoverage {
		tr.Coverage[taskID][entry.File] = entry.Coverage
	}
	return c.SaveTraceability(tr)
}
