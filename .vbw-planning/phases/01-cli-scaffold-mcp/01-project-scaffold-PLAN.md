# Plan 01: Project Scaffold & CLI Flags

## Goal

Set up the Go module, directory structure, cobra root command with `--version`/`--help`, and the `generate` subcommand with all flags defined and validated.

## Wave

1 (foundation — all other plans depend on this)

## Requirements

REQ-01, REQ-02, REQ-03, REQ-04, REQ-05, REQ-06, REQ-07, REQ-08, REQ-09, REQ-10, REQ-11, REQ-12, REQ-13, REQ-13a, REQ-14, REQ-15, REQ-16, REQ-17, REQ-18, REQ-58, REQ-59, REQ-60, REQ-66

## Tasks

### T1: Initialize Go module and directory structure

Create `go.mod` with `go 1.22` directive. Create directory layout:
- `cmd/` — cobra CLI setup
- `internal/mcp/` — MCP protocol client (placeholder)
- `internal/schema/` — JSON Schema processing (placeholder, Phase 2)
- `internal/codegen/` — Code generation (placeholder, Phase 2)
- `internal/compile/` — Compilation (placeholder, Phase 2)
- `internal/nameutil/` — Name inference
- `internal/auth/` — Bearer token handling
- `main.go` — entry point

### T2: Implement root command and version

- Root cobra command: `clihub`
- `--version` flag: prints `clihub vX.Y.Z`
- Version injected via `-ldflags "-X main.version=X.Y.Z"` (REQ-02)
- Build-time default `dev` if not set

### T3: Implement generate subcommand with all flags

Define all flags on the `generate` command:
- `--url <url>` (string) — Streamable HTTP URL
- `--stdio <command>` (string) — stdio command
- `--name <value>` (string) — override inferred name
- `--output <path>` (string, default `./out/`)
- `--platform <os/arch,...|all>` (string, default current platform)
- `--include-tools <tools>` (string) — comma-separated
- `--exclude-tools <tools>` (string) — comma-separated
- `--auth-token <token>` (string) — bearer token
- `--timeout <ms>` (int, default 30000)
- `--env KEY=VALUE` (string slice, repeatable)
- `--save-credentials` (bool)
- `--verbose` (bool)
- `--quiet` (bool)

### T4: Implement flag validation

- REQ-16: Require either `--url` or `--stdio`
- REQ-60: Mutual exclusivity: `--url`/`--stdio`, `--include-tools`/`--exclude-tools`, `--verbose`/`--quiet`
- REQ-01/REQ-66: Go toolchain check (`go version`)
- All errors to stderr with `Error: <message>` format (REQ-58)

### T5: Help output

- `clihub generate --help` shows all flags with descriptions
- Include usage examples in the long description (REQ-17)

## Acceptance Criteria

1. `go build` produces a `clihub` binary
2. `clihub --version` prints `clihub vdev`
3. `clihub generate --help` shows all flags
4. `clihub generate` (no flags) prints "Error: provide --url or --stdio to specify the MCP server"
5. `clihub generate --url x --stdio y` prints mutual exclusivity error
6. `clihub generate --verbose --quiet --url x` prints mutual exclusivity error

## Estimated Complexity

Medium — mostly flag wiring and validation boilerplate
