package loop

import (
	"strconv"
	"strings"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/errs"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func normalizeTraceID(prefix, raw string) (string, bool) {
	id := strings.ToLower(strings.TrimSpace(raw))
	if !strings.HasPrefix(id, prefix) {
		return "", false
	}
	n, err := strconv.Atoi(strings.TrimPrefix(id, prefix))
	if err != nil || n < 1 {
		return "", false
	}
	return prefix + strconv.Itoa(n), true
}

func knownCriterionIDs(task spec.Task) []string {
	known := make([]string, 0, len(task.Criteria))
	for _, c := range task.Criteria {
		known = append(known, strings.ToLower(strings.TrimSpace(c.ID)))
	}
	return known
}

func knownEdgeCaseIDs(task spec.Task) []string {
	known := make([]string, 0, len(task.EdgeCases))
	for i := range task.EdgeCases {
		known = append(known, edgeCaseID(i))
	}
	return known
}

func idListLabel(known []string) string {
	if len(known) == 0 {
		return "none"
	}
	return strings.Join(known, ", ")
}

func normalizeTraceIDs(fn, prefix string, raw, known []string) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	valid := make(map[string]bool, len(known))
	for _, id := range known {
		valid[id] = true
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		id, ok := normalizeTraceID(prefix, item)
		if !ok {
			return nil, errs.Newf(errs.KeyTraceabilityMalformedID, fn, item, prefix, prefix)
		}
		if !valid[id] {
			return nil, errs.Newf(errs.KeyTraceabilityUnknownID, fn, id, idListLabel(known))
		}
		out = append(out, id)
	}
	return out, nil
}

// coverageRoundActive reports whether this RED round exists only to close a
// coverage gap. In that round a new test may legitimately cover no criterion —
// it protects code an earlier task or stage already implemented — so the
// at-least-one-id rule is relaxed.
func coverageRoundActive(settings spec.Settings, state *spec.ExecState) bool {
	if state == nil || !coverageEnforced(settings) {
		return false
	}
	return state.CoverageUnreported || len(lowCoverageStateFiles(*state, settings)) > 0
}

func validateAndPersistTraceability(c *engine.Context, task spec.Task, report StageReport) error {
	if len(report.Traceability) == 0 {
		return errs.New(errs.KeyTraceabilityEmpty)
	}

	allowUnmapped := coverageRoundActive(c.Settings(), task.Exec)
	criterionIDs := knownCriterionIDs(task)
	edgeCaseIDs := knownEdgeCaseIDs(task)

	normalized := make([]TraceReportEntry, 0, len(report.Traceability))
	for _, entry := range report.Traceability {
		if entry.TestFilePath == "" {
			return errs.New(errs.KeyTraceabilityMissingTestPath)
		}
		if entry.FunctionName == "" {
			return errs.New(errs.KeyTraceabilityMissingFunc)
		}

		ac, err := normalizeTraceIDs(entry.FunctionName, spec.CriterionIDPrefix, entry.AC, criterionIDs)
		if err != nil {
			return err
		}
		ec, err := normalizeTraceIDs(entry.FunctionName, spec.EdgeCaseIDPrefix, entry.EC, edgeCaseIDs)
		if err != nil {
			return err
		}
		if len(ac) == 0 && len(ec) == 0 && !allowUnmapped {
			return errs.Newf(errs.KeyTraceabilityMissingACEC, entry.FunctionName)
		}

		entry.AC = ac
		entry.EC = ec
		normalized = append(normalized, entry)
	}

	tr, err := c.LoadTraceability()
	if err != nil {
		return err
	}
	if tr.Entries == nil {
		tr.Entries = map[string][]spec.TraceEntry{}
	}

	for _, entry := range normalized {
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
