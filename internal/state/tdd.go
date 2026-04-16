package state

// CurrentTaskID returns the ID of the first task in OverrideTasks that is not
// present in Execution.CompletedTasks. Returns "" when all known tasks have
// been completed (or when OverrideTasks is empty).
func CurrentTaskID(st StateFile) string {
	if len(st.OverrideTasks) == 0 {
		return ""
	}
	completed := make(map[string]bool, len(st.Execution.CompletedTasks))
	for _, id := range st.Execution.CompletedTasks {
		completed[id] = true
	}
	for _, t := range st.OverrideTasks {
		if !completed[t.ID] {
			return t.ID
		}
	}
	return ""
}

// IsTaskTDDEnabled resolves the effective TDD setting for a single task.
// Resolution order:
//  1. Task-level override (SpecTask.TDDEnabled) when non-nil.
//  2. Spec-level manifest setting (cfg.IsTDDEnabled()).
// Returns false when the task is not found.
func IsTaskTDDEnabled(st StateFile, taskID string, cfg *NosManifest) bool {
	specLevel := cfg != nil && cfg.IsTDDEnabled()
	for _, t := range st.OverrideTasks {
		if t.ID != taskID {
			continue
		}
		if t.TDDEnabled != nil {
			return *t.TDDEnabled
		}
		return specLevel
	}
	return specLevel
}

// ShouldRunTDDForCurrentTask reports whether the next uncompleted task should
// run through the RED/GREEN/REFACTOR cycle. When OverrideTasks carries no
// known current task, falls back to the spec-level manifest setting so older
// state files and bootstrap flows continue to behave as before.
func ShouldRunTDDForCurrentTask(st StateFile, cfg *NosManifest) bool {
	id := CurrentTaskID(st)
	if id == "" {
		return cfg != nil && cfg.IsTDDEnabled()
	}
	return IsTaskTDDEnabled(st, id, cfg)
}

// AnyTaskUsesTDD returns true when at least one known task would run under
// TDD. Used by callers that want to decide whether to inject TDD-only context
// (e.g. behavioral rules) into a mixed-mode spec.
func AnyTaskUsesTDD(st StateFile, cfg *NosManifest) bool {
	specLevel := cfg != nil && cfg.IsTDDEnabled()
	if len(st.OverrideTasks) == 0 {
		return specLevel
	}
	for _, t := range st.OverrideTasks {
		if t.TDDEnabled != nil {
			if *t.TDDEnabled {
				return true
			}
			continue
		}
		if specLevel {
			return true
		}
	}
	return false
}
