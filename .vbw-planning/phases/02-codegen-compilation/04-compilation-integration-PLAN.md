# Plan 04: Compilation, Integration & Smoke Test

## Goal

Compile generated Go projects into binaries, run smoke tests, wire everything into the generate command, and implement --save-credentials.

## Wave

3 (depends on Plans 01, 02, 03)

## Requirements

REQ-36 (temp dir management), REQ-51, REQ-52, REQ-53 (smoke test, Phase 2: always host platform), REQ-13a (save-credentials integration), REQ-67

## Tasks

### T1: Implement compilation pipeline

In `internal/compile/compile.go`:

`Compile(projectDir, outputDir, name, goos, goarch string) (string, error)`:
1. Set CGO_ENABLED=0, GOOS, GOARCH
2. Run `go build -o <outputDir>/<name>` in the project dir
3. Return binary path
4. Error: "go build failed for <os>/<arch>: <stderr>" (REQ-67)

### T2: Implement smoke test

`SmokeTest(binaryPath string) error`:
1. Run `<binary> --help` with 15-second timeout
2. Check exit code is 0
3. Error: "generated binary failed smoke test (exit code <N>): <stderr>" (REQ-68)

### T3: Implement temp directory management

- Create temp dir: `os.MkdirTemp("", "clihub-*")`
- On success: remove temp dir
- On failure: preserve and print path (REQ-36)

### T4: Wire into generate command

Update `cmd/generate.go` to complete the full pipeline:
1. [existing] Validate flags, check Go toolchain, connect, discover, filter, infer name
2. [new] Process tool schemas (Plan 01)
3. [new] Generate Go project (Plan 03)
4. [new] Compile for current platform (this plan)
5. [new] Smoke test
6. [new] Copy binary to output dir
7. [new] Clean up temp dir
8. [new] Print final output (binary path)

### T5: Implement --save-credentials integration

After successful discovery (before codegen):
- If `--save-credentials` and `--auth-token` are both set:
  - Save token to credentials file
  - Print confirmation

### T6: Verbose output for Phase 2 steps

- Schema processing: "Processing tool schemas..."
- Code generation: "Generating Go project..."
- Compilation: "Compiling <name> for <os>/<arch>..."
- Smoke test: "Running smoke test..."
- Output: "Binary written to <path>"

## Acceptance Criteria

1. `clihub generate --url <url>` produces a compiled binary in ./out/
2. Generated binary runs and shows --help
3. Smoke test passes (exit code 0)
4. Temp dir cleaned on success
5. Temp dir preserved with path printed on failure
6. --save-credentials saves token
7. Verbose mode shows compilation progress
8. All error messages match specifications

## Estimated Complexity

Medium-High — integration of all Phase 2 components
