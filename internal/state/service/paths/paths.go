// Package paths defines the on-disk layout for a tddmaster project and
// exposes path helpers used by every I/O subpackage under service/.
package paths

// TddmasterDir is the on-disk project directory (relative to project root).
const TddmasterDir = ".tddmaster"

const (
	StateDir        = TddmasterDir + "/.state"
	StateFilePath   = StateDir + "/state.json"
	ManifestFile    = TddmasterDir + "/manifest.yml"
	ConcernsDir     = TddmasterDir + "/concerns"
	RulesDir        = TddmasterDir + "/rules"
	SpecsDir        = TddmasterDir + "/specs"
	WorkflowsDir    = TddmasterDir + "/workflows"
	SpecStatesDir   = StateDir + "/specs"
	ActiveFile      = StateDir + "/active.json"
	SessionsDir     = TddmasterDir + "/.sessions"
	EventsDir       = TddmasterDir + "/.events"
	gitignoreSuffix = "/.gitignore"
)

// Paths provides path computation helpers rooted at the project directory.
// All methods are pure and stateless.
type Paths struct{}

func (p Paths) EserDir() string       { return TddmasterDir }
func (p Paths) StateDir() string      { return StateDir }
func (p Paths) StateFile() string     { return StateFilePath }
func (p Paths) ManifestFile() string  { return ManifestFile }
func (p Paths) ConcernsDir() string   { return ConcernsDir }
func (p Paths) RulesDir() string      { return RulesDir }
func (p Paths) SpecsDir() string      { return SpecsDir }
func (p Paths) WorkflowsDir() string  { return WorkflowsDir }
func (p Paths) SpecStatesDir() string { return SpecStatesDir }
func (p Paths) ActiveFile() string    { return ActiveFile }
func (p Paths) SessionsDir() string   { return SessionsDir }
func (p Paths) EventsDir() string     { return EventsDir }

func (p Paths) SpecDir(specName string) string {
	return SpecsDir + "/" + specName
}

func (p Paths) SpecFile(specName string) string {
	return SpecsDir + "/" + specName + "/spec.md"
}

func (p Paths) SpecStateFile(specName string) string {
	return SpecStatesDir + "/" + specName + ".json"
}

func (p Paths) ConcernFile(concernID string) string {
	return ConcernsDir + "/" + concernID + ".json"
}

func (p Paths) SessionFile(sessionID string) string {
	return SessionsDir + "/" + sessionID + ".json"
}

func (p Paths) TddmasterGitignore() string {
	return TddmasterDir + gitignoreSuffix
}
