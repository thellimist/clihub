<!-- VBW PROJECT TEMPLATE (ARTF-04) -->

# clihub

## What This Is

A Go CLI that turns any MCP server into a compiled, standalone command-line tool. Give it an MCP server URL or stdio command — it connects, discovers tools, generates a self-contained Go CLI, and cross-compiles it to native binaries. One command: `clihub generate`.

## Core Value

`clihub generate --url https://mcp.linear.app/mcp` → compiled `linear` binary with subcommands for every tool.

## Requirements

### Active

- [ ] 71 requirements in REQUIREMENTS.md (REQ-01 to REQ-70 + REQ-13a, reviewed 2026-02-23)

### Out of Scope

- OAuth flow (deferred to v1.1)
- SSE transport (Streamable HTTP only)
- Config system, daemon, inspect, regeneration
- GUI / web dashboard

## Context

Go rewrite of ../mcporter's generate-cli. Strips down to one command, replaces JS bundling with Go cross-compilation (6 GOOS/GOARCH targets). Bearer token auth only — OAuth deferred.

## Constraints

- **Language:** Go (>= 1.22)
- **Scope:** `clihub generate` only
- **Output:** Compiled native binaries (not source code)
- **Auth:** Bearer tokens for v1, OAuth in v1.1
- **MCP Transport:** Streamable HTTP + stdio (no SSE)

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Go + cobra framework | Native binaries, ~5MB overhead acceptable | Active |
| Single `generate` command | All other mcporter commands out of scope | Active |
| No token embedding | Security — tokens extractable via `strings` on binary | Active |
| --env embeds keys only | Same security concern — values from runtime os.Getenv() | Active |
| Explicit --save-credentials | No silent credential writes to disk | Active |
| OAuth deferred to v1.1 | Complex (RFC 8414 discovery, PKCE, token endpoints) | Active |
| SSE excluded | 2025 MCP spec uses Streamable HTTP | Active |
| 3-phase roadmap | scaffold → codegen → cross-platform | Active |

---
*Last updated: 2026-02-23 after requirements review*
