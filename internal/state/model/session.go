package model

// Session represents an active tddmaster work session, used by `tddmaster
// session ls` and for stale-session garbage collection.
type Session struct {
	ID           string  `json:"id"`
	Spec         *string `json:"spec"`
	Mode         string  `json:"mode"`
	Phase        *string `json:"phase"`
	PID          int     `json:"pid"`
	StartedAt    string  `json:"startedAt"`
	LastActiveAt string  `json:"lastActiveAt"`
	Tool         string  `json:"tool"`
	ProjectRoot  *string `json:"projectRoot,omitempty"`
}

// SpecStateEntry holds a spec name paired with its loaded state, used by
// ListSpecStates callers.
type SpecStateEntry struct {
	Name  string
	State StateFile
}
