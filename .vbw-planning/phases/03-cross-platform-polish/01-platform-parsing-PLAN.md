# Plan 01: Platform Parsing & Validation

## Goal

Parse --platform flag, handle "all" shorthand, validate platform strings, and define binary naming rules.

## Wave

1 (no dependencies except Phase 2)

## Requirements

REQ-08, REQ-49, REQ-50, REQ-52, REQ-69

## Tasks

### T1: Platform parsing and validation

In `internal/compile/platform.go`:

- `ParsePlatforms(platformFlag string) ([]Platform, error)`
  - Split on commas, trim whitespace
  - Handle "all" → expand to 6 standard targets
  - Validate each: must be "os/arch" format
  - Validate against valid targets set
  - Error: "invalid platform 'foo/bar'. Valid targets: linux/amd64, ..." (REQ-69)

- `Platform` struct: `GOOS string, GOARCH string`

- `ValidPlatforms`: the 6 standard targets (REQ-49):
  linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64, windows/arm64

### T2: Binary naming

- `BinaryName(name string, p Platform, multiPlatform bool) string`
  - Single platform (multiPlatform=false): just `<name>` (or `<name>.exe` on windows) (REQ-52)
  - Multi-platform (multiPlatform=true): `<name>-<os>-<arch>` with `.exe` for windows (REQ-50)
  - Examples: `linear`, `linear-linux-amd64`, `linear-windows-arm64.exe`

### T3: Unit tests

- Parse "all" → 6 platforms
- Parse "linux/amd64,darwin/arm64" → 2 platforms
- Parse single platform → 1 platform
- Invalid platform → error with valid list
- Binary naming: single, multi, windows .exe

## Acceptance Criteria

1. "all" expands to 6 platforms
2. Invalid platforms produce error with valid targets list
3. Binary naming follows convention
4. All tests pass

## Estimated Complexity

Low — string parsing and validation
