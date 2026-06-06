package manifest

type ToolID string

const ToolClaudeCode ToolID = "claude-code"

type CatalogEntry struct {
	Label string
	ID    ToolID
}

var Catalog = []CatalogEntry{
	{Label: "Claude Code", ID: ToolClaudeCode},
}
