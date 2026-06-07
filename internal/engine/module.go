package engine

type ModuleID string

type ModuleDef struct {
	ID    ModuleID
	Steps []StepDef
}

type ModuleProgress struct {
	Module ModuleID       `json:"module"`
	Steps  []StepProgress `json:"steps"`
}
