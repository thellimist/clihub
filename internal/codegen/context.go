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

// OpenAPIGenerateContext holds all data needed to generate an OpenAPI CLI project.
type OpenAPIGenerateContext struct {
	CLIName       string           // Generated CLI binary name
	BaseURL       string           // Default base URL from spec servers[0]
	Title         string           // API title from spec info
	Operations    []OpenAPIToolDef // Operations derived from spec
	ClihubVersion string           // clihub version for header comment
}

// OpenAPIToolDef represents a single API endpoint for code generation.
type OpenAPIToolDef struct {
	OperationID  string              // Original operationId
	CommandName  string              // Kebab-case command name
	Description  string              // Operation summary
	Method       string              // HTTP method
	Path         string              // URL path with {param} placeholders
	PathParams   []schema.ToolOption // Path parameters
	QueryParams  []schema.ToolOption // Query parameters
	HeaderParams []schema.ToolOption // Header parameters
	BodyParams   []schema.ToolOption // Request body properties
	HasBody      bool                // True if operation has a request body
}

// AllParams returns all parameters for this operation in flag order:
// path params first, then query, then header, then body.
func (t OpenAPIToolDef) AllParams() []schema.ToolOption {
	var all []schema.ToolOption
	all = append(all, t.PathParams...)
	all = append(all, t.QueryParams...)
	all = append(all, t.HeaderParams...)
	all = append(all, t.BodyParams...)
	return all
}
