# Plan 02: Auth & Credentials

## Goal

Implement credential persistence and runtime lookup for both clihub and generated CLIs.

## Wave

1 (parallel with Plan 01 — no dependencies except Phase 1)

## Requirements

REQ-13a, REQ-55, REQ-56, REQ-57

## Tasks

### T1: Implement credentials.json read/write

In `internal/auth/credentials.go`:

Schema: `{"version": 1, "servers": {"<url>": {"token": "...", "type": "bearer"}}}`

- `LoadCredentials(path string) (*CredentialsFile, error)` — read and parse, return empty if not exists
- `SaveCredentials(path string, creds *CredentialsFile) error` — write with 0600 permissions, create dir if needed
- `GetToken(creds *CredentialsFile, serverURL string) string` — lookup token by URL
- `SetToken(creds *CredentialsFile, serverURL, token string)` — set/update token for URL

### T2: Implement credential file path resolution

- Default: `~/.clihub/credentials.json`
- Override via `CLIHUB_CREDENTIALS_FILE` env var (REQ-56)
- `DefaultCredentialsPath() string`

### T3: Implement --save-credentials for clihub

In `cmd/generate.go` (or called from it):
- When `--save-credentials` is set with `--auth-token`:
  - Load or create credentials file
  - Set token for the server URL
  - Save credentials file
  - Print: "Saved credentials to ~/.clihub/credentials.json"
- Without `--auth-token`, `--save-credentials` is a no-op

### T4: Implement LookupToken for generated CLIs

`LookupToken(serverURL string) string` — returns token from (in priority order):
1. `--auth-token` flag value (passed as parameter)
2. `CLIHUB_AUTH_TOKEN` env var
3. `~/.clihub/credentials.json` (or CLIHUB_CREDENTIALS_FILE override) keyed by server URL

### T5: Unit tests

- Read/write credentials file
- Token lookup priority chain
- Path resolution with env override
- Empty/missing file handling
- Directory creation

## Acceptance Criteria

1. `--save-credentials` persists token to credentials file
2. Credentials file uses correct schema and permissions (0600)
3. Token lookup follows priority order: flag → env → file
4. CLIHUB_CREDENTIALS_FILE overrides default path
5. All tests pass

## Estimated Complexity

Low-Medium — file I/O and env var handling
