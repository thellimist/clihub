# Plan 04: Tool Filtering

## Goal

Implement tool filtering by include/exclude lists with validation and "did you mean?" hints using Levenshtein distance.

## Wave

2 (parallel with Plans 02, 03 — depends on Plan 01 for module structure)

## Requirements

REQ-33, REQ-34, REQ-35, REQ-64, REQ-65

## Tasks

### T1: Implement tool filter logic

In `internal/toolfilter/filter.go`:
- `ParseToolList(csv string) []string` — parse comma-separated, trim whitespace, deduplicate
- `FilterTools(tools []mcp.Tool, include, exclude []string) ([]mcp.Tool, error)`
  - If both include and exclude non-empty → error (REQ-35, validated earlier in flags but defense-in-depth)
  - Include mode: keep only listed tools, error if any not found (REQ-33)
  - Exclude mode: remove listed tools, error if all excluded (REQ-34)

### T2: Implement Levenshtein distance

In `internal/toolfilter/levenshtein.go`:
- `LevenshteinDistance(a, b string) int`
- `SuggestTool(name string, available []string) string` — return closest match if distance <= 3

### T3: Error messages with hints

- Tool not found: `"Error: tool 'foo' not found on server. Available tools: bar, baz, qux"` (REQ-64)
- If Levenshtein distance <= 3: append `"Did you mean 'baz'?"`
- All excluded: `"Error: all tools excluded — nothing to generate"` (REQ-65)

### T4: Unit tests

- Parse CSV with whitespace, duplicates
- Include filtering: valid, missing tool, with hint
- Exclude filtering: valid, all excluded
- Levenshtein: exact, close, far
- Table-driven tests

## Acceptance Criteria

1. `ParseToolList("foo, bar, foo")` returns `["foo", "bar"]`
2. Include filter keeps only listed tools
3. Missing tool error includes available tools list
4. Missing tool with close match includes "Did you mean?" hint
5. Exclude filter removes listed tools
6. Excluding all tools produces correct error
7. All tests pass

## Estimated Complexity

Low — straightforward filtering with simple string distance algorithm
