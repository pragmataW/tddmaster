package model

// User represents a tddmaster user identity. Persisted per-machine at
// ~/.config/eser/tddmaster/user.json; resolved fresh on each CLI invocation.
type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}
