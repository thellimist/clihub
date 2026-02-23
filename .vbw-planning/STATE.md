<!-- VBW STATE TEMPLATE (ARTF-05) -->

# Project State

## Project Reference

See: .vbw-planning/PROJECT.md (updated 2026-02-23)

**Core value:** One command to turn any MCP server into a compiled CLI binary
**Current focus:** Phase 3 - Cross-Platform Output & Polish

## Current Position

Phase: 3 of 3 (All phases complete)
Plan: 2 of 2 in current phase
Status: Complete
Last activity: 2026-02-23 -- Phase 3 built (2 plans, 119 tests total, 2 commits)

Progress: [####################] 100%

## Accumulated Context

### Decisions

- [bootstrap]: Go, single `generate` command, cobra framework
- [bootstrap]: 71 requirements (REQ-01 to REQ-70 + REQ-13a), 3 phases
- [review]: OAuth deferred to v1.1 — bearer tokens only for v1
- [review]: SSE transport excluded — Streamable HTTP (2025 spec) only
- [review]: Tokens NOT embedded in binaries — generated CLIs read from env/credentials at runtime
- [review]: Temp dir preserved on compilation failure for debugging
- [review]: `mcp-server-`, `server-`, `mcp-` prefixes stripped from inferred names
- [review]: --env embeds only keys (not values) — runtime reads via os.Getenv()
- [review]: --save-credentials required to persist tokens (no silent credential writes)
- [review]: --raw flag takes precedence over individual flags (no error on conflict)
- [review]: Smoke test skipped when host platform not in target list
- [review]: --auth-token with --stdio prints warning, directs user to --env
- [phase1]: HTTP transport handles both JSON and SSE responses with session tracking
- [phase1]: Stdio transport uses mutex-protected stdin/stdout JSON-RPC
- [phase2]: Generated Go projects use text/template with embedded MCP client
- [phase2]: go mod tidy runs in generated project for dependency resolution
- [phase2]: Generated binaries are fully self-contained (~6MB with cobra)

### Pending Todos

None

### Blockers/Concerns

None

## Codebase Profile

- Source files: 28 Go files across 8 packages
- Tests: 119 passing (nameutil: 42, mcp: 9, toolfilter: 16, schema: 20, auth: 16, codegen: 12, compile: 16)
- Stack: Go 1.22+ with cobra
- Reference: ../mcporter (TypeScript source being ported)

### Skills

**Installed:**
- find-skills (global)

**Stack detected:** Go
**Registry available:** yes

## Session Continuity

Last session: 2026-02-23
Stopped at: All phases complete
Resume file: none
