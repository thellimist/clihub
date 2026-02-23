# Plan 02: Name Inference & Shell Splitting

## Goal

Implement name inference from URLs and stdio commands, slugify utility, and shell-aware command string splitting.

## Wave

2 (parallel with Plans 03, 04 — depends on Plan 01 for module structure)

## Requirements

REQ-21, REQ-30, REQ-31, REQ-32

## Tasks

### T1: Shell-aware command splitting

Implement `SplitCommand(input string) ([]string, error)` in `internal/nameutil/`:
- Split on whitespace outside quotes
- Respect single quotes, double quotes, backslash escapes
- Error on unterminated quotes
- Examples:
  - `"npx -y @org/server"` → `["npx", "-y", "@org/server"]`
  - `"sh -c 'echo test'"` → `["sh", "-c", "echo test"]`

### T2: Name inference from HTTP URLs

Implement `InferNameFromURL(rawURL string) string` in `internal/nameutil/`:
- Parse URL, extract hostname
- Strip prefixes: `www.`, `api.`, `mcp.`
- Strip TLD suffixes: `.com`, `.io`, `.app`, `.dev`, `.org`, `.net`
- Take remaining meaningful segment
- Fallback to first non-empty path segment
- Example: `https://mcp.linear.app/mcp` → `linear`

### T3: Name inference from stdio commands

Implement `InferNameFromCommand(command string) string` in `internal/nameutil/`:
- Shell-split the command
- Find the package/tool token (last positional arg, or scoped package)
- Strip prefixes in order: `mcp-server-`, `server-`, `mcp-`
- Strip version suffixes: `@latest`, `@1.2.3`
- Strip scope: `@org/` prefix
- Examples:
  - `"npx @modelcontextprotocol/server-github"` → `github`
  - `"npx mcp-server-postgres"` → `postgres`

### T4: Slugify

Implement `Slugify(s string) string`:
- Lowercase
- Replace non-alphanumeric with `-`
- Collapse consecutive dashes
- Trim leading/trailing dashes

### T5: Unit tests

Test all functions with table-driven tests:
- Shell splitting: basic, quotes, escapes, errors
- URL inference: various URL patterns
- Command inference: npx, node, scoped packages, version suffixes
- Slugify: edge cases

## Acceptance Criteria

1. All unit tests pass
2. `InferNameFromURL("https://mcp.linear.app/mcp")` returns `"linear"`
3. `InferNameFromCommand("npx @modelcontextprotocol/server-github")` returns `"github"`
4. `SplitCommand("npx -y @org/server")` returns `["npx", "-y", "@org/server"]`
5. `Slugify("Hello World!")` returns `"hello-world"`

## Estimated Complexity

Low-Medium — pure utility functions with well-defined behavior
