package adapter

func init() {
	Register(ClaudeCodeAdapter{})
	Register(CursorAdapter{})
	Register(CodexCLIAdapter{})
	Register(OpenCodeAdapter{})
}
