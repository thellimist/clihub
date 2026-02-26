package codegen

import "github.com/thellimist/clihub/internal/schema"

// GenerateContext holds all data needed to generate a CLI project.
type GenerateContext struct {
	CLIName       string    // Generated CLI binary name
	ServerURL     string    // MCP server URL (for HTTP mode)
	StdioCommand  string    // Stdio command (for stdio mode)
	StdioArgs     []string  // Stdio command args
	EnvKeys       []string  // Env var keys to embed (not values)
	Tools         []ToolDef // Tool definitions with options
	ClihubVersion string    // clihub version for header comment
	IsHTTP        bool      // True = HTTP transport, false = stdio
}

// ToolDef represents a single MCP tool for code generation.
type ToolDef struct {
	Name        string              // Original MCP tool name (e.g., "list_issues")
	CommandName string              // Kebab-case command (e.g., "list-issues")
	Description string              // Tool description
	Options     []schema.ToolOption // CLI flag options derived from schema
}
