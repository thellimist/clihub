# Plan 05: Generate Command Integration

## Goal

Wire the generate command to connect all components: parse flags → check Go toolchain → connect to MCP → handshake → discover tools → apply filters → infer name → print results. End-to-end flow with proper error handling.

## Wave

3 (depends on Plans 01, 02, 03, 04)

## Requirements

REQ-01, REQ-13 (env passing), REQ-24 (auth warning for stdio), REQ-30-32 (name inference integration), REQ-33-35 (filter integration), all error REQs

## Tasks

### T1: Implement generate command run function

In `cmd/generate.go`, implement the `RunE` function that orchestrates:
1. Validate flags (already done in Plan 01, call validation)
2. Check Go toolchain: run `go version`, parse output (REQ-01, REQ-66)
3. Warn if `--auth-token` used with `--stdio` (REQ-24)
4. Create MCP client (HTTP or stdio based on flags)
5. Pass `--env` values to stdio transport environment (REQ-13)
6. Connect + handshake + discover tools
7. Apply tool filtering (include/exclude)
8. Infer name if `--name` not provided
9. Print discovery summary (tool count, names, inferred CLI name)
10. Close connection

### T2: Go toolchain check

In `internal/gocheck/check.go`:
- `CheckGoToolchain() (string, error)` — run `go version`, parse version string
- Verify >= 1.22
- Error: "Error: Go toolchain not found. Install Go >= 1.22 from https://go.dev/dl/" (REQ-66)

### T3: Verbose/quiet output handling

- `--verbose`: print each step (connecting, handshaking, discovering, filtering)
- `--quiet`: suppress all non-error output
- Default: print final summary only

### T4: End-to-end integration test

- Test with a mock or test MCP server if feasible
- Test error paths: no server, timeout, no tools, bad tool names
- Test name inference integration
- Test verbose/quiet modes

### T5: Phase 1 success criteria verification

After integration, the generate command should satisfy:
1. `clihub generate --url <url>` connects and prints discovered tools
2. `clihub generate --stdio <cmd>` spawns process and discovers tools
3. Name inference works end-to-end
4. Tool filtering works end-to-end
5. All error messages match specifications

## Acceptance Criteria

1. `clihub generate --url https://mcp.context7.com/mcp` connects, handshakes, lists tools
2. `clihub generate --stdio "npx ..."` spawns process and discovers tools
3. `clihub generate --url <url> --name custom` uses provided name
4. `clihub generate --url <url>` with no `--name` infers name from URL
5. Tool filtering with `--include-tools` and `--exclude-tools` works
6. Go toolchain check runs before MCP connection
7. `--verbose` shows progress, `--quiet` suppresses output
8. All error paths produce correct messages

## Estimated Complexity

Medium-High — integration of all components with proper error flow
