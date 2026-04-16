
// State persistence — read/write .tddmaster/.state/state.json and tddmaster
// config inside .tddmaster/manifest.yml (comment-preserving YAML).

package state

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// =============================================================================
// Paths
// =============================================================================

const TddmasterDir = ".tddmaster"
const stateDir = TddmasterDir + "/.state"
const stateFile = stateDir + "/state.json"
const manifestFile = TddmasterDir + "/manifest.yml"
const concernsDir = TddmasterDir + "/concerns"
const rulesDir = TddmasterDir + "/rules"
const specsDir = TddmasterDir + "/specs"
const workflowsDir = TddmasterDir + "/workflows"
const specStatesDir = stateDir + "/specs"
const activeFile = stateDir + "/active.json"
const sessionsDir = TddmasterDir + "/.sessions"
const eventsDir = TddmasterDir + "/.events"

// Paths provides path computation helpers.
type Paths struct{}

var paths = Paths{}

// EserDir returns the tddmaster directory path relative to root.
func (p Paths) EserDir() string { return TddmasterDir }

// StateDir returns the state directory.
func (p Paths) StateDir() string { return stateDir }

// StateFile returns the state file path.
func (p Paths) StateFile() string { return stateFile }

// ManifestFile returns the manifest file path.
func (p Paths) ManifestFile() string { return manifestFile }

// ConcernsDir returns the concerns directory.
func (p Paths) ConcernsDir() string { return concernsDir }

// RulesDir returns the rules directory.
func (p Paths) RulesDir() string { return rulesDir }

// SpecsDir returns the specs directory.
func (p Paths) SpecsDir() string { return specsDir }

// WorkflowsDir returns the workflows directory.
func (p Paths) WorkflowsDir() string { return workflowsDir }

// SpecStatesDir returns the per-spec states directory.
func (p Paths) SpecStatesDir() string { return specStatesDir }

// ActiveFile returns the active spec file path.
func (p Paths) ActiveFile() string { return activeFile }

// SpecDir returns the directory for a named spec.
func (p Paths) SpecDir(specName string) string {
	return specsDir + "/" + specName
}

// SpecFile returns the spec.md path for a named spec.
func (p Paths) SpecFile(specName string) string {
	return specsDir + "/" + specName + "/spec.md"
}

// SpecStateFile returns the per-spec state file path.
func (p Paths) SpecStateFile(specName string) string {
	return specStatesDir + "/" + specName + ".json"
}

// ConcernFile returns the JSON file path for a concern.
func (p Paths) ConcernFile(concernID string) string {
	return concernsDir + "/" + concernID + ".json"
}

// SessionsDir returns the sessions directory.
func (p Paths) SessionsDir() string { return sessionsDir }

// SessionFile returns the session file path.
func (p Paths) SessionFile(sessionID string) string {
	return sessionsDir + "/" + sessionID + ".json"
}

// EventsDir returns the events directory.
func (p Paths) EventsDir() string { return eventsDir }

// TddmasterGitignore returns the .gitignore path.
func (p Paths) TddmasterGitignore() string { return TddmasterDir + "/.gitignore" }

// =============================================================================
// State File
// =============================================================================

// ReadState reads the main state file, returning initial state on any error.
func ReadState(root string) (StateFile, error) {
	filePath := filepath.Join(root, stateFile)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return CreateInitialState(), nil
	}

	var s StateFile
	if err := json.Unmarshal(data, &s); err != nil {
		return CreateInitialState(), nil
	}
	return s, nil
}

// ResolveState resolves state for a specific spec, or the active state if no spec given.
func ResolveState(root string, specName *string) (StateFile, error) {
	if specName == nil {
		return ReadState(root)
	}

	// Check if spec exists
	specDirPath := filepath.Join(root, paths.SpecDir(*specName))
	if _, err := os.Stat(specDirPath); err != nil {
		return StateFile{}, fmt.Errorf("spec '%s' not found. Run `tddmaster spec list` to see available specs", *specName)
	}

	// Try per-spec state first, fall back to active state if matching
	specState, err := ReadSpecState(root, *specName)
	if err != nil {
		return StateFile{}, err
	}
	if specState.Spec != nil && *specState.Spec == *specName {
		return specState, nil
	}

	// Check if active state matches
	activeState, err := ReadState(root)
	if err != nil {
		return StateFile{}, err
	}
	if activeState.Spec != nil && *activeState.Spec == *specName {
		return activeState, nil
	}

	// Return the per-spec state even if spec field is null (freshly created)
	specState.Spec = specName
	return specState, nil
}

// WriteState writes the state file, creating directories as needed.
func WriteState(root string, s StateFile) error {
	filePath := filepath.Join(root, stateFile)

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return WriteFileAtomic(filePath, data, 0o644)
}

// =============================================================================
// Spec Flag Parsing
// =============================================================================

// ParseSpecFlag parses --spec=<name> from args. Returns nil if not found.
func ParseSpecFlag(args []string) *string {
	for _, arg := range args {
		if strings.HasPrefix(arg, "--spec=") {
			s := arg[len("--spec="):]
			return &s
		}
	}
	return nil
}

// RequireSpecFlagResult is the result of RequireSpecFlag.
type RequireSpecFlagResult struct {
	OK    bool
	Spec  string
	Error string
}

// RequireSpecFlag returns the spec name from args, or an error if not found.
func RequireSpecFlag(args []string) RequireSpecFlagResult {
	spec := ParseSpecFlag(args)
	if spec == nil || len(*spec) == 0 {
		return RequireSpecFlagResult{
			OK:    false,
			Error: "Error: spec name is required. Use `tddmaster spec <name> <command>` format.",
		}
	}
	return RequireSpecFlagResult{OK: true, Spec: *spec}
}

// UsesOldSpecFlag returns true if any arg starts with --spec=.
func UsesOldSpecFlag(args []string) bool {
	for _, a := range args {
		if strings.HasPrefix(a, "--spec=") {
			return true
		}
	}
	return false
}

// =============================================================================
// Active Spec
// =============================================================================

// ReadActiveSpec returns the active spec name from state.json.
func ReadActiveSpec(root string) (*string, error) {
	s, err := ReadState(root)
	if err != nil {
		return nil, err
	}
	return s.Spec, nil
}

// =============================================================================
// Per-Spec State Files (.tddmaster/.state/specs/<name>.json)
// =============================================================================

// ReadSpecState reads the per-spec state file.
func ReadSpecState(root, specName string) (StateFile, error) {
	filePath := filepath.Join(root, paths.SpecStateFile(specName))
	data, err := os.ReadFile(filePath)
	if err != nil {
		return CreateInitialState(), nil
	}

	var s StateFile
	if err := json.Unmarshal(data, &s); err != nil {
		return CreateInitialState(), nil
	}
	return s, nil
}

// WriteSpecState writes the per-spec state file.
func WriteSpecState(root, specName string, s StateFile) error {
	filePath := filepath.Join(root, paths.SpecStateFile(specName))

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return WriteFileAtomic(filePath, data, 0o644)
}

// SpecStateEntry holds a spec name + its state.
type SpecStateEntry struct {
	Name  string
	State StateFile
}

// ListSpecStates lists all spec names that have state files.
func ListSpecStates(root string) ([]SpecStateEntry, error) {
	dirPath := filepath.Join(root, specStatesDir)
	var results []SpecStateEntry

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		// No spec states yet
		return results, nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		specName := strings.TrimSuffix(name, ".json")
		data, err := os.ReadFile(filepath.Join(dirPath, name))
		if err != nil {
			continue
		}
		var s StateFile
		if err := json.Unmarshal(data, &s); err != nil {
			continue
		}
		results = append(results, SpecStateEntry{Name: specName, State: s})
	}
	return results, nil
}

// =============================================================================
// Config (tddmaster section inside .tddmaster/manifest.yml)
// =============================================================================

// ReadManifest reads the tddmaster section from manifest.yml.
func ReadManifest(root string) (*NosManifest, error) {
	filePath := filepath.Join(root, manifestFile)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil
	}

	// Parse into a map so we can extract just the tddmaster key
	var doc map[string]yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, nil
	}

	nosNode, ok := doc["tddmaster"]
	if !ok {
		return nil, nil
	}

	// Decode the tddmaster node directly into NosManifest (respects yaml tags)
	var manifest NosManifest
	if err := nosNode.Decode(&manifest); err != nil {
		return nil, nil
	}
	return &manifest, nil
}

// WriteManifest writes the tddmaster config to manifest.yml, preserving other keys.
func WriteManifest(root string, config NosManifest) error {
	dirPath := filepath.Join(root, TddmasterDir)
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return err
	}

	filePath := filepath.Join(root, manifestFile)

	// Parse existing document if it exists
	var doc yaml.Node
	data, err := os.ReadFile(filePath)
	if err == nil {
		var rootNode yaml.Node
		if err := yaml.Unmarshal(data, &rootNode); err == nil && rootNode.Kind == yaml.DocumentNode {
			doc = *rootNode.Content[0]
		}
	}

	if doc.Kind == 0 {
		doc = yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	}

	// Build a node for the config
	configNode := &yaml.Node{}
	if err := configNode.Encode(config); err != nil {
		return err
	}
	// configNode is a document node, get its content
	if configNode.Kind == yaml.DocumentNode && len(configNode.Content) > 0 {
		configNode = configNode.Content[0]
	}
	configNode.HeadComment = " tddmaster orchestrator — inline comments in this section won't be preserved on next write"

	// Set the tddmaster key in the mapping
	found := false
	for i := 0; i+1 < len(doc.Content); i += 2 {
		if doc.Content[i].Value == "tddmaster" {
			doc.Content[i+1] = configNode
			found = true
			break
		}
	}
	if !found {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "tddmaster", Tag: "!!str"}
		doc.Content = append(doc.Content, keyNode, configNode)
	}

	rootDoc := yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{&doc}}
	out, err := yaml.Marshal(&rootDoc)
	if err != nil {
		return err
	}
	return WriteFileAtomic(filePath, out, 0o644)
}

// =============================================================================
// Concern Files
// =============================================================================

// ReadConcern reads a concern definition by ID.
func ReadConcern(root, concernID string) (*ConcernDefinition, error) {
	filePath := filepath.Join(root, paths.ConcernFile(concernID))
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil
	}

	var c ConcernDefinition
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, nil
	}
	return &c, nil
}

// WriteConcern writes a concern definition to disk.
func WriteConcern(root string, concern ConcernDefinition) error {
	filePath := filepath.Join(root, paths.ConcernFile(concern.ID))

	data, err := json.MarshalIndent(concern, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return WriteFileAtomic(filePath, data, 0o644)
}

// ListConcerns lists all concern definitions.
func ListConcerns(root string) ([]ConcernDefinition, error) {
	dirPath := filepath.Join(root, concernsDir)
	var concerns []ConcernDefinition

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		// Directory doesn't exist yet
		return concerns, nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dirPath, name))
		if err != nil {
			continue
		}
		var c ConcernDefinition
		if err := json.Unmarshal(data, &c); err != nil {
			continue
		}
		concerns = append(concerns, c)
	}
	return concerns, nil
}

// =============================================================================
// Directory Scaffolding
// =============================================================================

// ScaffoldDir creates the full .tddmaster directory structure.
func ScaffoldDir(root string) error {
	dirs := []string{
		TddmasterDir,
		stateDir,
		specStatesDir,
		concernsDir,
		rulesDir,
		specsDir,
		workflowsDir,
		eventsDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			return err
		}
	}

	// .gitignore at .tddmaster/ level — only create if missing
	gitignorePath := filepath.Join(root, paths.TddmasterGitignore())
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		content := "# tddmaster toolchain runtime state — not tracked by git\n.state/\n.sessions/\n.events/\n"
		if err := os.WriteFile(gitignorePath, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// =============================================================================
// State + Spec State (write both atomically)
// =============================================================================

// WriteStateAndSpec writes main state AND the per-spec state file for the
// active spec. state.json is the primary source of truth; per-spec state is a
// derivative view. Each file is written atomically via WriteFileAtomic.
//
// If the per-spec write fails after the primary state was committed, the
// returned error explicitly signals that re-running the command will reconcile
// (the primary state is correct; the per-spec file will be rewritten from it).
func WriteStateAndSpec(root string, s StateFile) error {
	if err := WriteState(root, s); err != nil {
		return err
	}
	if s.Spec != nil {
		if err := WriteSpecState(root, *s.Spec, s); err != nil {
			return fmt.Errorf("primary state committed but per-spec state write failed; re-run command to reconcile: %w", err)
		}
	}
	return nil
}

// =============================================================================
// Sessions
// =============================================================================

// Session represents an active work session.
type Session struct {
	ID           string  `json:"id"`
	Spec         *string `json:"spec"`
	Mode         string  `json:"mode"` // "spec" | "free"
	Phase        *string `json:"phase"`
	PID          int     `json:"pid"`
	StartedAt    string  `json:"startedAt"`
	LastActiveAt string  `json:"lastActiveAt"`
	Tool         string  `json:"tool"`
	ProjectRoot  *string `json:"projectRoot,omitempty"`
}

const staleThresholdMs = 2 * 60 * 60 * 1000 // 2 hours in milliseconds

// CreateSession writes a session file.
func CreateSession(root string, session Session) error {
	dir := filepath.Join(root, sessionsDir)
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return WriteFileAtomic(filepath.Join(dir, session.ID+".json"), data, 0o644)
}

// ReadSession reads a session file by ID.
func ReadSession(root, sessionID string) (*Session, error) {
	filePath := filepath.Join(root, sessionsDir, sessionID+".json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil
	}

	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, nil
	}
	return &s, nil
}

// ListSessions lists all sessions.
func ListSessions(root string) ([]Session, error) {
	dir := filepath.Join(root, sessionsDir)
	var sessions []Session

	entries, err := os.ReadDir(dir)
	if err != nil {
		// no sessions dir
		return sessions, nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		var s Session
		if err := json.Unmarshal(data, &s); err != nil {
			continue // corrupt file, skip
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

// DeleteSession removes a session file. Returns false if not found.
func DeleteSession(root, sessionID string) (bool, error) {
	filePath := filepath.Join(root, sessionsDir, sessionID+".json")
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// UpdateSessionPhase updates the phase and lastActiveAt of a session.
func UpdateSessionPhase(root, sessionID, phase string) error {
	session, err := ReadSession(root, sessionID)
	if err != nil || session == nil {
		return err
	}

	updated := *session
	updated.Phase = &phase
	updated.LastActiveAt = time.Now().UTC().Format(time.RFC3339)

	data, err := json.MarshalIndent(updated, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return WriteFileAtomic(filepath.Join(root, sessionsDir, sessionID+".json"), data, 0o644)
}

// GcStaleSessions removes stale sessions and returns their IDs.
func GcStaleSessions(root string) ([]string, error) {
	sessions, err := ListSessions(root)
	if err != nil {
		return nil, err
	}
	var removed []string
	for _, s := range sessions {
		if IsSessionStale(s) {
			if _, err := DeleteSession(root, s.ID); err == nil {
				removed = append(removed, s.ID)
			}
		}
	}
	return removed, nil
}

// IsSessionStale returns true if the session has been inactive for more than 2 hours.
func IsSessionStale(session Session) bool {
	t, err := time.Parse(time.RFC3339, session.LastActiveAt)
	if err != nil {
		return true
	}
	elapsed := time.Since(t).Milliseconds()
	return elapsed > staleThresholdMs
}

// GenerateSessionId generates an 8-char random hex session ID.
func GenerateSessionId() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%08x", b), nil
}

// =============================================================================
// Existence Checks
// =============================================================================

// IsInitialized checks if the project has been initialized (manifest.yml with tddmaster section).
func IsInitialized(root string) (bool, error) {
	filePath := filepath.Join(root, manifestFile)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false, nil
	}

	var doc map[string]interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return false, nil
	}

	_, ok := doc["tddmaster"]
	return ok, nil
}

// =============================================================================
// Project Root Discovery
// =============================================================================

// FindProjectRoot walks up directory tree to find the nearest directory containing .tddmaster/.
func FindProjectRoot(startDir string) (string, error) {
	dir := startDir
	for depth := 0; depth < 100; depth++ {
		if _, err := os.Stat(filepath.Join(dir, TddmasterDir)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil // filesystem root
		}
		dir = parent
	}
	return "", nil
}

// ResolveProjectRootResult holds the result of ResolveProjectRoot.
type ResolveProjectRootResult struct {
	Root  string
	Found bool
}

// ResolveProjectRoot resolves the tddmaster project root with priority:
//  1. TDDMASTER_PROJECT_ROOT env var
//  2. Walk up from cwd to find .tddmaster/
//  3. Fall back to cwd
func ResolveProjectRoot() (ResolveProjectRootResult, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return ResolveProjectRootResult{}, err
	}

	// 1. Explicit env var
	envRoot := os.Getenv("TDDMASTER_PROJECT_ROOT")
	if envRoot != "" {
		if _, err := os.Stat(filepath.Join(envRoot, TddmasterDir)); err == nil {
			return ResolveProjectRootResult{Root: envRoot, Found: true}, nil
		}
		// env var set but .tddmaster/ not there — fall through to walk-up
	}

	// 2. Walk up from cwd
	found, err := FindProjectRoot(cwd)
	if err != nil {
		return ResolveProjectRootResult{}, err
	}
	if found != "" {
		return ResolveProjectRootResult{Root: found, Found: true}, nil
	}

	// 3. Env var exists but .tddmaster/ not found
	if envRoot != "" {
		return ResolveProjectRootResult{Root: envRoot, Found: false}, nil
	}

	// 4. Nothing found — return cwd (for init command to use)
	return ResolveProjectRootResult{Root: cwd, Found: false}, nil
}
