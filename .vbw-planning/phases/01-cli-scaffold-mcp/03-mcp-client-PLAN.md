# Plan 03: MCP Protocol Client

## Goal

Implement the MCP protocol client supporting Streamable HTTP and stdio transports, including initialize handshake, tools/list discovery, timeout handling, and auth headers.

## Wave

2 (parallel with Plans 02, 04 — depends on Plan 01 for module structure)

## Requirements

REQ-19, REQ-20, REQ-22, REQ-23, REQ-24, REQ-61, REQ-62, REQ-63, REQ-70

## Tasks

### T1: Define MCP types and interfaces

In `internal/mcp/types.go`:
- JSON-RPC request/response types
- `Tool` struct: Name, Description, InputSchema (json.RawMessage)
- `InitializeParams` / `InitializeResult`
- `ToolsListResult`
- Transport interface: `Send(request) → response`, `Close()`

### T2: Implement Streamable HTTP transport

In `internal/mcp/http.go`:
- `HTTPTransport` struct with URL, headers, HTTP client
- Send JSON-RPC requests via POST to the server URL
- Include `Authorization: Bearer <token>` header when auth token provided (REQ-24)
- Parse JSON-RPC responses
- Respect timeout via `context.WithTimeout` (REQ-22)

### T3: Implement stdio transport

In `internal/mcp/stdio.go`:
- `StdioTransport` struct with command, args, env
- Spawn child process with provided env vars
- Communicate via stdin/stdout using JSON-RPC (newline-delimited)
- Handle process lifecycle (start, communicate, kill on close)
- Pass `--env` variables to child process environment (REQ-20)

### T4: Implement MCP client (handshake + tools/list)

In `internal/mcp/client.go`:
- `Client` struct wrapping a Transport
- `Connect()`: send `initialize` request with client info, wait for response (REQ-19)
- `ListTools()`: send `tools/list` request, parse tool array (REQ-23)
- `Close()`: close transport
- Timeout wrapping: apply timeout to entire handshake+discovery sequence (REQ-22)

### T5: Error handling

- Connection failure → "Error: failed to connect to MCP server at <url>: <reason>" (REQ-61)
- Handshake failure → "Error: MCP server at <url> did not complete initialization handshake" (REQ-62)
- No tools → "Error: MCP server returned no tools" (REQ-63)
- Timeout → "Error: MCP server did not respond within <timeout>ms" (REQ-70)

### T6: Unit tests

- Mock transport for testing client logic
- Test handshake sequence
- Test tools/list parsing
- Test timeout behavior
- Test error paths

## Acceptance Criteria

1. HTTP transport can send/receive JSON-RPC messages
2. Stdio transport can spawn a process and communicate via stdin/stdout
3. Client performs initialize handshake and receives server capabilities
4. Client fetches tools list with schemas
5. Timeout cancels operations after specified duration
6. All error messages match REQ specifications
7. Unit tests pass

## Estimated Complexity

High — core protocol implementation with two transport backends
