# Plan 02: Cross-Compilation & Polish

## Goal

Update the generate command for multi-platform compilation with per-platform progress, smart smoke test, and production polish.

## Wave

2 (depends on Plan 01)

## Requirements

REQ-48, REQ-53, REQ-54, REQ-14, REQ-15, REQ-67, REQ-68

## Tasks

### T1: Update generate command for multi-platform

Update `cmd/generate.go`:
- Parse --platform flag using `compile.ParsePlatforms`
- Loop over platforms, compile each
- Use `BinaryName` for correct naming
- Track compiled binaries for summary

### T2: Per-platform verbose progress

When --verbose:
```
Compiling linux/amd64... done (2.1s)
Compiling darwin/arm64... done (1.8s)
```

Time each compilation and display.

### T3: Smart smoke test

- After compilation, check if host platform (runtime.GOOS/runtime.GOARCH) is in the target list
- If yes: run smoke test on the host-platform binary
- If no: print warning "Warning: smoke test skipped — no binary for host platform (<os>/<arch>)" (REQ-53)

### T4: Output summary polish

Default output (not verbose, not quiet):
```
Generated <name> from <url> (<N> tools, <M> platforms)
Binaries:
  ./out/<name>-linux-amd64
  ./out/<name>-darwin-arm64
```

Quiet: nothing on success.

### T5: Error handling

- Compilation failure on any platform: report which platform failed, continue or abort
- Report all binary paths at the end

## Acceptance Criteria

1. `--platform all` produces 6 binaries
2. `--platform linux/amd64,darwin/arm64` produces exactly 2 binaries
3. Binary naming: `<name>-<os>-<arch>` for multi-platform
4. Single platform: just `<name>` (no suffix)
5. `--verbose` shows per-platform progress with timing
6. Smoke test runs on host platform, skips with warning otherwise
7. `--quiet` shows nothing on success
8. Invalid platform produces correct error

## Estimated Complexity

Medium — iteration and conditional logic
