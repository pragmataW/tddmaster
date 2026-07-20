package loop

import (
	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/errs"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func validateAndPersistTraceability(c *engine.Context, task spec.Task, report StageReport) error {
	if len(report.Traceability) == 0 {
		return errs.New(errs.KeyTraceabilityEmpty)
	}

	for _, entry := range report.Traceability {
		if entry.TestFilePath == "" {
			return errs.New(errs.KeyTraceabilityMissingTestPath)
		}
		if entry.FunctionName == "" {
			return errs.New(errs.KeyTraceabilityMissingFunc)
		}
		if len(entry.AC) == 0 && len(entry.EC) == 0 {
			return errs.New(errs.KeyTraceabilityMissingACEC)
		}
	}

	tr, err := c.LoadTraceability()
	if err != nil {
		return err
	}
	if tr.Entries == nil {
		tr.Entries = map[string][]spec.TraceEntry{}
	}

	for _, entry := range report.Traceability {
		taskID := entry.TaskID
		if taskID == "" {
			taskID = task.ID
		}
		newEntry := spec.TraceEntry{
			FunctionName: entry.FunctionName,
			TaskID:       taskID,
			CriterionIDs: entry.AC,
			EC:           entry.EC,
		}

		existing := tr.Entries[entry.TestFilePath]
		replaced := false
		for i, e := range existing {
			if e.FunctionName == entry.FunctionName {
				existing[i] = newEntry
				replaced = true
				break
			}
		}
		if !replaced {
			existing = append(existing, newEntry)
		}
		tr.Entries[entry.TestFilePath] = existing
	}

	return c.SaveTraceability(tr)
}
