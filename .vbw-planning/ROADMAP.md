<!-- Go rewrite of mcporter generate-cli → clihub generate -->

# clihub Roadmap

## Overview

Build `clihub generate` in 3 phases. One command that takes an MCP server URL or stdio command, connects to it, discovers tools, generates a self-contained Go CLI, and cross-compiles it to native binaries. Auth is integrated throughout (bearer tokens only — OAuth deferred to v1.1).

## Phases

- [x] Phase 1: CLI Scaffold & MCP Connection
- [x] Phase 2: Go Code Generation & Compilation
- [x] Phase 3: Cross-Platform Output & Polish
- [x] Phase 4: OAuth Authentication

## Phase Details

### Phase 1: CLI Scaffold & MCP Connection

**Goal:** Working `clihub generate` command that connects to MCP servers (Streamable HTTP + stdio), performs the MCP handshake, and discovers tools with their schemas
**Depends on:** None

**Requirements:**
- REQ-01, REQ-02: Go prerequisite check, version injection via -ldflags
- REQ-03 to REQ-18: All CLI flags (--url, --stdio, --name, --output, --platform, --include-tools, --exclude-tools, --auth-token, --timeout, --env, --verbose, --quiet, --help, --version)
- REQ-19 to REQ-24: MCP connection (Streamable HTTP + stdio), initialize handshake, tools/list, timeout, auth header
- REQ-21: Shell-aware command string splitting for --stdio
- REQ-30 to REQ-32: Name inference (URL hostname, stdio command with mcp-server-/server- stripping, slugify)
- REQ-33 to REQ-35: Tool filtering (include/exclude with validation)
- REQ-58 to REQ-70: Error handling (all flag, connection, handshake, tool errors)

**Success Criteria:**
1. `go build` produces a `clihub` binary; `clihub --version` prints version
2. `clihub generate --help` shows all flags with descriptions and examples
3. `clihub generate --url https://mcp.context7.com/mcp` connects (handshake + tools/list), prints discovered tools
4. `clihub generate --stdio "npx ..."` spawns the process and discovers tools
5. `clihub generate --stdio "npx server" --env GITHUB_TOKEN=$TOKEN` passes env to child
6. Name inference: `--url https://mcp.linear.app/mcp` → `linear`; `--stdio "npx @mcp/server-github"` → `github`
7. Tool filtering: --include-tools and --exclude-tools work with "did you mean?" hints
8. All error paths produce correct messages per REQ-58 to REQ-70

### Phase 2: Go Code Generation & Compilation

**Goal:** Generate complete Go projects from discovered tools and compile them into working binaries
**Depends on:** Phase 1

**Requirements:**
- REQ-25 to REQ-29: Tool schema processing (ToolOption struct, JSON Schema → Go types, enums, defaults, nullable, camelCase → kebab-case)
- REQ-36 to REQ-47: Go code generation (main.go + go.mod, cobra subcommands, embedded MCP client, --raw flag precedence, global flags incl --auth-token, action handler, output formatters, help, header comment, connection lifecycle, context.WithTimeout)
- REQ-51, REQ-52: Write to --output directory, single-platform compilation (current platform only)
- REQ-13a, REQ-55 to REQ-57: --save-credentials flag, auth in generated CLIs (runtime credential lookup, not embedded tokens)
- REQ-66, REQ-67: Go toolchain and compilation errors

**Success Criteria:**
1. `clihub generate --url https://mcp.context7.com/mcp` produces a compiled binary in `./out/`
2. Generated binary `--help` shows all tool subcommands with descriptions
3. Generated binary can call MCP tools: `./out/context7 resolve-library --name react` returns results
4. Output formats work: `--output text`, `--output json`, `--output markdown`, `--output raw`
5. All JSON Schema types map correctly: string, int, float64, bool, []string, enum
6. `--raw '{"key":"value"}'` ignores all individual flags and sends raw JSON
7. Generated CLI reads auth from `CLIHUB_AUTH_TOKEN` env var or `~/.clihub/credentials.json` (not embedded)
8. Temp dir cleaned on success; preserved with path printed on failure
9. Smoke test passes (always runs in Phase 2 since target is host platform)
10. `--save-credentials` persists token to ~/.clihub/credentials.json with confirmation message

### Phase 3: Cross-Platform Output & Polish

**Goal:** Cross-compile to all 6 platform targets, progress output, and production polish
**Depends on:** Phase 2

**Requirements:**
- REQ-08: `--platform all` shorthand expanding to 6 targets (REQ-49)
- REQ-48 to REQ-50: GOOS/GOARCH cross-compilation with CGO_ENABLED=0, binary naming (<name>-<os>-<arch>)
- REQ-53: Smoke test host-platform check (skip with warning if host not in target list)
- REQ-54: Verbose compilation progress per platform
- REQ-14, REQ-15: --verbose progress output, --quiet mode
- REQ-67 to REQ-69: Compilation failure, smoke test failure, invalid platform errors

**Success Criteria:**
1. `clihub generate --url <url> --platform all` produces 6 binaries in `./out/`
2. Binary naming: `linear-linux-amd64`, `linear-darwin-arm64`, `linear-windows-amd64.exe`
3. `--platform linux/amd64,darwin/arm64` produces exactly 2 binaries
4. `--verbose` shows per-platform progress: `Compiling linux/amd64... done (2.1s)`
5. `--quiet` shows nothing on success
6. Smoke test runs on host-platform binary; skips with warning if host not in target list
7. `--output ./dist/my-tool` writes to custom directory
8. `--platform foo/bar` → "Error: invalid platform 'foo/bar'. Valid targets: ..."

### Phase 4: OAuth Authentication

**Goal:** Full MCP OAuth support — browser-based authorization code flow with PKCE, dynamic client registration, and token persistence
**Depends on:** Phase 1 (MCP transport), Phase 2 (credential storage)

**Requirements:**
- OAuth metadata discovery (RFC 8414 + RFC 9728)
- Dynamic client registration (RFC 7591)
- PKCE (RFC 7636) code_verifier/code_challenge
- Local callback server for authorization redirect
- Browser opening (darwin/linux/windows)
- Token exchange and refresh
- Credential storage extension for OAuth tokens
- HTTP transport 401 interception and retry
- `--oauth` flag on generate command

**Success Criteria:**
1. `clihub generate --url https://mcp.notion.com/mcp --oauth` opens browser, user authorizes, tools are discovered
2. OAuth tokens persisted to `~/.clihub/credentials.json`
3. Second run reuses cached token without browser
4. Expired token triggers automatic refresh
5. `--oauth` with `--stdio` produces clear error
6. Existing `--auth-token` flow unaffected

## Progress

| Phase | Plans Complete | Status | Completed |
|-------|---------------|--------|-----------|
| 1 - CLI Scaffold & MCP Connection | 5/5 | built | 2026-02-23 |
| 2 - Go Code Generation & Compilation | 4/4 | built | 2026-02-23 |
| 3 - Cross-Platform Output & Polish | 2/2 | built | 2026-02-23 |
| 4 - OAuth Authentication | 3/3 | built | 2026-02-24 |

---
*Last updated: 2026-02-23 after Phase 3 implementation*
