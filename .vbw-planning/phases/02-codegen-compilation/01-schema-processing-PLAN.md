# Plan 01: Tool Schema Processing

## Goal

Transform raw JSON Schema from MCP tools into structured Go metadata that drives code generation. Map JSON Schema types to Go types, handle enums, defaults, nullable types, and convert property names to CLI flags.

## Wave

1 (parallel with Plan 02 — no dependencies except Phase 1)

## Requirements

REQ-25, REQ-26, REQ-27, REQ-28, REQ-29

## Tasks

### T1: Define ToolOption struct

In `internal/schema/schema.go`:

```go
type ToolOption struct {
    PropertyName string   // Original JSON key (e.g., "issueId")
    FlagName     string   // Kebab-case CLI flag (e.g., "issue-id")
    Description  string   // From schema description field
    Required     bool     // True if in schema's required array
    GoType       string   // Go type string: "string", "int", "float64", "bool", "[]string", "[]int"
    DefaultValue any      // From schema default field
    EnumValues   []string // From schema enum field
}
```

### T2: Implement JSON Schema → Go type mapping

Map JSON Schema types to Go types (REQ-26):
- `"string"` → `"string"`
- `"integer"` → `"int"`
- `"number"` → `"float64"`
- `"boolean"` → `"bool"`
- `"array"` with string items → `"[]string"`
- `"array"` with integer items → `"[]int"`
- Anything unrecognized → `"string"` (user passes JSON)

Handle `type: ["string", "null"]` → pick first non-null type (REQ-28).

### T3: Implement camelCase → kebab-case conversion

Convert property names to CLI flag names (REQ-29):
- `issueId` → `issue-id`
- `relativePath` → `relative-path`
- `HTMLParser` → `html-parser`
- Use word boundary detection (uppercase letter after lowercase)

### T4: Implement ExtractOptions function

`ExtractOptions(inputSchema json.RawMessage, toolName string) ([]ToolOption, error)`:
- Parse JSON Schema from inputSchema
- Extract properties and required array
- For each property: build ToolOption with type mapping, flag name, description, required, default, enum
- Sort options: required first, then alphabetical

### T5: Unit tests

Table-driven tests:
- Type mapping: all JSON Schema types including nullable
- camelCase → kebab-case: various patterns including acronyms
- ExtractOptions: full schema parsing with mixed types, enums, defaults, required fields

## Acceptance Criteria

1. All JSON Schema types map correctly to Go types
2. Nullable types use first non-null type
3. camelCase conversion handles word boundaries correctly
4. Enum values are extracted from schema
5. Default values are preserved
6. Required flag is set from schema required array
7. All tests pass

## Estimated Complexity

Medium — JSON schema parsing with type mapping
