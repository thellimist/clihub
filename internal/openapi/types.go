package openapi

import "github.com/thellimist/clihub/internal/schema"

// Operation represents a single API endpoint extracted from an OpenAPI spec.
type Operation struct {
	OperationID  string              // Original operationId from spec
	CommandName  string              // Kebab-case command name
	Summary      string              // Human-readable summary
	Method       string              // HTTP method: GET, POST, PUT, PATCH, DELETE
	Path         string              // URL path with {param} placeholders
	PathParams   []schema.ToolOption // Parameters from URL path
	QueryParams  []schema.ToolOption // Query string parameters
	HeaderParams []schema.ToolOption // Header parameters
	BodyParams   []schema.ToolOption // Top-level request body properties
	HasBody      bool                // True if operation accepts a request body
}
