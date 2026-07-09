package manifest

type ToolID string

const (
	ToolClaudeCode ToolID = "claude-code"
	ToolCursor     ToolID = "cursor"
	ToolCodexCLI   ToolID = "codex-cli"
	ToolOpenCode   ToolID = "opencode"
)

type CatalogEntry struct {
	Label string
	ID    ToolID
}

var Catalog = []CatalogEntry{
	{Label: "Claude Code", ID: ToolClaudeCode},
	{Label: "Cursor", ID: ToolCursor},
	{Label: "Codex CLI", ID: ToolCodexCLI},
	{Label: "OpenCode", ID: ToolOpenCode},
}
