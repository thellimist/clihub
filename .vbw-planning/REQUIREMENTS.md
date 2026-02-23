<!-- Requirements for clihub — Go rewrite of mcporter generate-cli -->

# clihub Requirements

Defined: 2026-02-23
Core value: One command to turn any MCP server into a compiled CLI binary
Source: ../mcporter generate-cli (TypeScript) → clihub (Go)

## CLI Surface

```
clihub generate --url https://mcp.linear.app/mcp
clihub generate --stdio "npx @modelcontextprotocol/server-github"
clihub generate --url https://... --platform linux/amd64,darwin/arm64
clihub generate --url https://... --platform all
clihub generate --url https://... --include-tools create_issue,list_issues
clihub generate --url https://... --exclude-tools dangerous_tool
clihub generate --url https://... --output ./dist/linear
clihub generate --url https://... --auth-token $TOKEN --save-credentials
clihub generate --url https://... --name my-tool
clihub generate --stdio "npx server" --env GITHUB_TOKEN=$TOKEN --env DEBUG=true
clihub generate --url https://... --timeout 60000 --verbose
clihub --version
```

One command. No config files, no daemon, no inspect, no regeneration.

## v1 Requirements

### Prerequisites

- [ ] **REQ-01**: Requires Go toolchain (>= 1.22) installed on the host. On startup, verify `go version` succeeds. If not, error: "Error: Go toolchain not found. Install Go >= 1.22 from https://go.dev/dl/"
- [ ] **REQ-02**: clihub version is injected at build time via `-ldflags "-X main.version=X.Y.Z"`. `clihub --version` prints `clihub vX.Y.Z`

### CLI Entry Point

- [ ] **REQ-03**: Go binary named `clihub` with a single `generate` subcommand. CLI framework decision: cobra (full-featured, ~5MB binary overhead) vs urfave/cli (lighter, ~2MB). Decision: **cobra** — the binary size overhead is acceptable and cobra's subcommand model maps naturally to the generated CLI's per-tool subcommands. Both clihub and generated CLIs use the same framework
- [ ] **REQ-04**: `--url <url>` flag — Streamable HTTP URL of an MCP server to generate a CLI from. Mutually exclusive with `--stdio`. Example: `--url https://mcp.linear.app/mcp`
- [ ] **REQ-05**: `--stdio <command>` flag — shell command that spawns a local MCP server via stdin/stdout. The command string is split shell-style (respecting quotes). Mutually exclusive with `--url`. Example: `--stdio "npx @modelcontextprotocol/server-github"`
- [ ] **REQ-06**: `--name <value>` flag — override the auto-inferred name for the generated CLI. If omitted, the name is inferred from the URL hostname or stdio command. Example: `--name linear` produces a binary called `linear`
- [ ] **REQ-07**: `--output <path>` flag — directory where compiled binaries are written. Defaults to `./out/`. Example: `--output ./dist/linear` writes binaries into `./dist/linear/`
- [ ] **REQ-08**: `--platform <os/arch,...|all>` flag — comma-separated list of GOOS/GOARCH pairs to cross-compile for. Special value `all` expands to all 6 standard targets (REQ-49). Defaults to the current platform only. Example: `--platform linux/amd64,darwin/arm64`
- [ ] **REQ-09**: `--include-tools <tool1,tool2,...>` flag — only include these tools as subcommands in the generated CLI. Comma-separated. Mutually exclusive with `--exclude-tools`. Errors if a listed tool doesn't exist on the server
- [ ] **REQ-10**: `--exclude-tools <tool1,tool2,...>` flag — remove these tools from the generated CLI. Comma-separated. Mutually exclusive with `--include-tools`. Errors if all tools would be excluded
- [ ] **REQ-11**: `--auth-token <token>` flag — static bearer token used during MCP server connection for tool discovery. See REQ-55 for how generated CLIs handle auth at runtime (not embedded in binary)
- [ ] **REQ-12**: `--timeout <ms>` flag — timeout for the MCP connection and tool discovery during generation. Default 30000ms. Example: `--timeout 60000` for slow servers. This is clihub's own timeout, separate from the generated CLI's `--timeout` flag (REQ-41)
- [ ] **REQ-13**: `--env KEY=VALUE` flag — repeatable. Set environment variables for stdio MCP server processes. During generation, the values are passed to the child process for tool discovery. In the generated CLI, only the env var **keys** are embedded (not the values). At runtime, the generated CLI reads each key from `os.Getenv()` and passes it to the stdio child process. This means `--env GITHUB_TOKEN=ghp_secret` uses that token during generation, but the compiled binary only knows it needs `GITHUB_TOKEN` — the actual value comes from the user's runtime environment. Non-secret values like `LOG_LEVEL=debug` work the same way: the key is embedded, the value is read at runtime
- [ ] **REQ-13a**: `--save-credentials` flag — when used with `--auth-token`, persist the token to `~/.clihub/credentials.json` keyed by the server URL. Without this flag, `--auth-token` is used for the current generation only. Prints: "Saved credentials to ~/.clihub/credentials.json"
- [ ] **REQ-14**: `--verbose` flag — show detailed progress during generation: connection status, tool count, code generation steps, compilation per-platform. Default: quiet, only show final output path(s)
- [ ] **REQ-15**: `--quiet` flag — suppress all output except errors. Mutually exclusive with `--verbose`
- [ ] **REQ-16**: Must provide either `--url` or `--stdio` — error if neither is given: "Error: provide --url or --stdio to specify the MCP server"
- [ ] **REQ-17**: `--help` shows usage, all flags with descriptions, and examples
- [ ] **REQ-18**: `clihub --version` prints `clihub vX.Y.Z` (version injected via REQ-02)

### MCP Server Connection

Both HTTP and stdio transports follow the MCP 2025 protocol: initialize handshake first, then tools/list.

- [ ] **REQ-19**: HTTP (Streamable HTTP) connection: send `initialize` JSON-RPC request to the provided URL with client capabilities, wait for `initialized` response, then send `tools/list` to discover tools and their input schemas. This follows the MCP 2025 Streamable HTTP spec — you cannot skip the handshake
- [ ] **REQ-20**: Stdio connection: spawn the command as a child process, communicate over stdin/stdout using JSON-RPC. Send `initialize` handshake, wait for `initialized`, then send `tools/list` to discover tools. Pass `--env` variables (REQ-13) to the child process environment
- [ ] **REQ-21**: Parse the `--stdio` command string using shell-aware splitting — respect single/double quotes and backslash escapes. `"npx -y @org/server"` splits into `["npx", "-y", "@org/server"]`
- [ ] **REQ-22**: Apply `--timeout` (REQ-12, default 30s) to the entire MCP handshake + tool discovery sequence. If the server doesn't respond within the timeout, error: "Error: MCP server did not respond within {timeout}ms"
- [ ] **REQ-23**: Each discovered tool has: name (string), description (string), and inputSchema (JSON Schema object describing the tool's parameters)
- [ ] **REQ-24**: If `--auth-token` is provided, include it as a Bearer token in the `Authorization` header for HTTP connections. For stdio servers, auth-token is not directly applicable — use `--env` to pass server-specific credentials (e.g., `--env GITHUB_TOKEN=$TOKEN`). If `--auth-token` is provided with `--stdio`, print a warning: "Warning: --auth-token is ignored for stdio servers. Use --env to pass credentials"

### Tool Schema Processing

Transforms raw JSON Schema from MCP tools into structured metadata that drives Go code generation.

- [ ] **REQ-25**: For each tool, extract every property from `inputSchema.properties` and build a `ToolOption` struct containing: PropertyName (original JSON key like `issueId`), FlagName (kebab-case CLI flag like `--issue-id`), Description (from schema `description` field), Required (true if property appears in schema's `required` array), GoType (mapped from JSON Schema type per REQ-26), and DefaultValue (from schema `default` field, used as the flag default so users skip common values)
- [ ] **REQ-26**: Map JSON Schema types to Go types: `"string"` → `string`, `"integer"` → `int`, `"number"` → `float64`, `"boolean"` → `bool`, `"array"` with string items → `[]string`, `"array"` with integer items → `[]int`. Anything unrecognized → `string` (user passes JSON, generated CLI deserializes it)
- [ ] **REQ-27**: Handle JSON Schema `enum` fields — when a property has `enum: ["a", "b", "c"]`, the generated CLI validates the flag value against the allowed set at runtime and shows the options in `--help` (e.g., `--status <open|closed|merged>`)
- [ ] **REQ-28**: Handle nullable types like `type: ["string", "null"]` — pick the first non-null type for the Go mapping. The null variant means the field is optional, not that Go needs a *string
- [ ] **REQ-29**: Convert property names to CLI flag names: `camelCase` → `kebab-case` using word boundary detection. Examples: `issueId` → `--issue-id`, `relativePath` → `--relative-path`, `HTMLParser` → `--html-parser`

### Name Inference

When `--name` is not provided, clihub infers a CLI name from the target.

- [ ] **REQ-30**: From HTTP URLs: extract the meaningful part of the hostname. Strip prefixes (`www.`, `api.`, `mcp.`) and known TLDs (`.com`, `.io`, `.app`, `.dev`, `.org`, `.net`). Example: `https://mcp.linear.app/mcp` → `linear`. Fallback to first non-empty path segment if hostname is all generic parts
- [ ] **REQ-31**: From stdio commands: extract the package/tool name from the command. Strip common prefixes for cleaner names: `mcp-server-`, `server-`, `mcp-` (checked in that order, first match wins). Examples: `"npx @modelcontextprotocol/server-github"` → `github`, `"npx mcp-server-postgres"` → `postgres`, `"npx @org/server-redis@latest"` → `redis` (also strip version suffixes like `@latest`, `@1.2.3`)
- [ ] **REQ-32**: Slugify the result: lowercase, replace non-alphanumeric characters with `-`, collapse consecutive dashes, trim leading/trailing dashes. Ensures the name is valid as a binary filename across all platforms

### Tool Filtering

- [ ] **REQ-33**: `--include-tools`: parse the comma-separated list, trim whitespace, deduplicate. After fetching tools from the server, keep only the listed ones. If a listed tool doesn't exist, error with: "Error: tool 'foo' not found on server. Available tools: bar, baz, qux" (include a "Did you mean 'baz'?" hint if Levenshtein distance <= 3)
- [ ] **REQ-34**: `--exclude-tools`: parse the comma-separated list, remove those tools from the full set. Error if removing them would leave zero tools: "Error: all tools excluded — nothing to generate"
- [ ] **REQ-35**: Validate mutual exclusivity: if both `--include-tools` and `--exclude-tools` are provided, error immediately before connecting to the server: "Error: --include-tools and --exclude-tools cannot be used together"

### Go Code Generation

Core of clihub: generating a complete, self-contained Go project that compiles into the target CLI.

- [ ] **REQ-36**: Generate a complete Go project in a temporary directory: `main.go` (full CLI source) and `go.mod` (with `go 1.22` directive and dependencies). The project compiles with `go build` without manual edits. On successful compilation, clean up the temp directory. On failure, preserve it and print: "Error: compilation failed. Generated source preserved at: /tmp/clihub-xxx/ for debugging"
- [ ] **REQ-37**: The generated `main.go` uses cobra for CLI structure. It embeds the MCP server URL/command, tool schemas, and connection logic as Go constants and variables — making the binary fully self-contained. If `--env` flags were provided (REQ-13), embed only the env var **keys** (not values). At runtime, the generated CLI reads each key from the host environment via `os.Getenv()` and passes it to the stdio child process. If a required env var is not set at runtime, the generated CLI prints: "Error: required environment variable <KEY> is not set"
- [ ] **REQ-38**: Generated CLI has one subcommand per MCP tool. Tool name `list_issues` becomes subcommand `list-issues` (kebab-case), with the original underscore name as a cobra alias so both `list-issues` and `list_issues` work
- [ ] **REQ-39**: Each tool subcommand has CLI flags generated from the tool's input schema. Flag names are kebab-case (REQ-29). Flag types use Go's native cobra type bindings: `StringVar`, `IntVar`, `Float64Var`, `BoolVar`, `StringSliceVar`
- [ ] **REQ-40**: Each tool subcommand has a `--raw <json>` flag that accepts a raw JSON string, bypassing all individual flags. Precedence: if `--raw` is provided, it wins — all individual flags are ignored, even if also set. No error on conflict (raw is the escape hatch for complex/nested arguments that don't map to flat CLI flags)
- [ ] **REQ-41**: Generated CLI has global flags: `-t, --timeout <ms>` (per-call timeout, default 30000), `-o, --output <format>` (output format: `text|json|markdown|raw`, default `text`), and `--auth-token <token>` (bearer token for authenticated MCP servers, overrides env var and credentials file). This is the generated CLI's own auth flag — separate from clihub's `--auth-token` which is used during generation only
- [ ] **REQ-42**: Generated CLI's action handler for each tool: parse CLI flags into a `map[string]interface{}`, open MCP connection (handshake + call), invoke `tools/call` via JSON-RPC, format the response per `--output`, print to stdout, close connection
- [ ] **REQ-43**: Output formatters in generated CLI: `text` (human-readable, extracts text content from tool result), `json` (pretty-printed JSON of the tool response), `markdown` (tool result formatted as markdown), `raw` (full unprocessed JSON-RPC response including metadata)
- [ ] **REQ-44**: Generated CLI shows a custom `--help` that lists all tool subcommands with descriptions, shows usage examples for the first 3 tools (auto-generated from schema), and displays global flags
- [ ] **REQ-45**: Generated `main.go` header: `// Code generated by clihub vX.Y.Z. DO NOT EDIT.` — the version is the clihub build version from REQ-02, injected into the code generation template at generation time
- [ ] **REQ-46**: Generated CLI handles the MCP connection lifecycle: connect on command invocation, perform handshake, send the tool call, receive the response, close the connection. No persistent connections, no background processes, no daemons
- [ ] **REQ-47**: Generated CLI uses `context.WithTimeout` for tool calls — if the MCP server doesn't respond within `--timeout`, the context is cancelled and the CLI prints an error: "Error: tool call timed out after {timeout}ms"

### Cross-Compilation

- [ ] **REQ-48**: After generating the Go project, run `go build` for each requested `--platform` target by setting GOOS and GOARCH environment variables. Use `CGO_ENABLED=0` for reproducible cross-compilation
- [ ] **REQ-49**: Six standard platform targets (used by `--platform all`): `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`, `windows/arm64`
- [ ] **REQ-50**: Output binary naming: `<name>-<os>-<arch>` with `.exe` suffix for Windows. Examples: `linear-linux-amd64`, `linear-darwin-arm64`, `linear-windows-arm64.exe`
- [ ] **REQ-51**: Write compiled binaries to the `--output` directory (default `./out/`). Create the directory if it doesn't exist
- [ ] **REQ-52**: When `--platform` is omitted, compile only for the current platform (`runtime.GOOS`/`runtime.GOARCH`). The binary name omits the os-arch suffix: just `<name>` (or `<name>.exe` on Windows)
- [ ] **REQ-53**: Smoke test: if the host platform is in the target list, run the host-platform binary with `--help` and verify exit code 0 within 15 seconds. If the host platform is NOT in the target list (e.g., building `linux/amd64` on macOS), skip the smoke test and print: "Warning: smoke test skipped — no binary for host platform (darwin/arm64)"
- [ ] **REQ-54**: Progress output (when `--verbose`): show each platform as it compiles. Example: `Compiling linux/amd64... done (2.1s)`, `Compiling darwin/arm64... done (1.8s)`. Without `--verbose`, just show final binary paths

### Auth & Credentials

- [ ] **REQ-55**: `--auth-token <token>`: used only during generation for tool discovery (passed as Bearer header for HTTP, or env var for stdio). The token is NOT embedded in the generated binary. Instead, the generated CLI reads credentials at runtime from (in priority order): `--auth-token` flag on the generated CLI, `CLIHUB_AUTH_TOKEN` env var, `~/.clihub/credentials.json` keyed by server URL. This prevents tokens from being extractable via `strings` on a distributed binary
- [ ] **REQ-56**: Generated CLI's credential lookup path (`~/.clihub/credentials.json`) is configurable via `CLIHUB_CREDENTIALS_FILE` env var, so users who don't have clihub installed can point to their own credential file or skip it entirely
- [ ] **REQ-57**: `~/.clihub/credentials.json` stores tokens keyed by server URL. Schema: `{ "version": 1, "servers": { "<url>": { "token": "...", "type": "bearer" } } }`. clihub does NOT silently write to this file. To persist credentials, the user must explicitly pass `--save-credentials` alongside `--auth-token`. Without it, the token is used for the current generation only and not saved. When `--save-credentials` is used, print: "Saved credentials to ~/.clihub/credentials.json". Generated CLIs read from this file at runtime (see REQ-55)

### Error Handling

- [ ] **REQ-58**: All errors print to stderr with a clear message and non-zero exit code. Format: `Error: <message>` (no stack traces, no "fatal", no redundant prefixes)
- [ ] **REQ-59**: Missing required flag: "Error: provide --url or --stdio to specify the MCP server"
- [ ] **REQ-60**: Mutual exclusivity violations: "Error: --include-tools and --exclude-tools cannot be used together", "Error: --verbose and --quiet cannot be used together", "Error: --url and --stdio cannot be used together"
- [ ] **REQ-61**: MCP connection failure: "Error: failed to connect to MCP server at <url>: <reason>"
- [ ] **REQ-62**: MCP handshake failure: "Error: MCP server at <url> did not complete initialization handshake"
- [ ] **REQ-63**: No tools found: "Error: MCP server returned no tools"
- [ ] **REQ-64**: Tool not found (--include-tools): "Error: tool 'foo' not found on server. Available tools: bar, baz, qux"
- [ ] **REQ-65**: All tools excluded: "Error: all tools excluded — nothing to generate"
- [ ] **REQ-66**: Go toolchain missing: "Error: Go toolchain not found. Install Go >= 1.22 from https://go.dev/dl/"
- [ ] **REQ-67**: Compilation failure: "Error: go build failed for <os>/<arch>: <stderr>"
- [ ] **REQ-68**: Smoke test failure: "Error: generated binary failed smoke test (exit code <N>): <stderr>"
- [ ] **REQ-69**: Invalid platform: "Error: invalid platform 'foo/bar'. Valid targets: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64, windows/arm64"
- [ ] **REQ-70**: Timeout: "Error: MCP server did not respond within <timeout>ms"

## Out of Scope

| Item | Reason |
|------|--------|
| `clihub call` command | clihub doesn't call MCPs — generated CLIs do |
| `clihub list` command | Not discovering servers, user provides the target |
| `inspect-cli` command | Nice-to-have, not core for v1 |
| Config file system (`clihub.json`) | No multi-server configs — user passes URL/command directly |
| Daemon / keep-alive | Generated CLIs connect per invocation, no persistent state |
| Regeneration (`--from`, `--dry-run`) | No metadata system needed for v1 |
| Editor imports (Cursor/VS Code detection) | Not applicable |
| TypeScript emit / JS bundling | Gone with Go rewrite |
| Server proxy / SDK patches | JS-specific |
| `--runtime`, `--bundler`, `--minify` flags | Go compiles natively |
| SSE transport | Legacy MCP transport. The 2025 MCP spec uses Streamable HTTP. SSE-only servers should upgrade. If needed later, can be added as an `--sse` flag without breaking changes |
| OAuth flow (`--oauth`) | Complex enough to be its own phase (v1.1). Needs OAuth discovery (RFC 8414), token endpoint, client registration, PKCE. Deferred — `--auth-token` covers most auth-gated servers for v1 |
| OAuth refresh in generated CLIs | Requires embedded OAuth client, token endpoint URL, client ID/secret. Deferred with `--oauth` to v1.1 |

## Traceability

Requirement-to-phase mapping is tracked in ROADMAP.md.

## Go Package Layout (Target)

| Package | Responsibility | Source (mcporter) |
|---------|---------------|-------------------|
| `cmd/` | cobra CLI setup, `generate` command, flag parsing, `--version` | cli.ts, generate-cli-runner.ts, flags.ts |
| `internal/mcp/` | MCP protocol client (Streamable HTTP + stdio), initialize handshake, tools/list | definition.ts, server-utils.ts |
| `internal/schema/` | JSON Schema → Go type mapping, ToolOption extraction, enum/default handling | tools.ts |
| `internal/codegen/` | Go source code generation (main.go + go.mod template) | template.ts |
| `internal/compile/` | Cross-compilation via `go build` with GOOS/GOARCH, smoke test | artifacts.ts (replaced bundling) |
| `internal/nameutil/` | Name inference from URLs and commands, slugify, mcp-server-/server- prefix stripping | name-utils.ts |
| `internal/auth/` | Bearer token handling, credentials.json read/write | oauth-vault.ts (simplified) |
