# Plan 03: Credential Storage, Transport Integration, and Command Wiring

## Goal

Integrate OAuth flow into existing codebase. Extend credential storage for OAuth tokens, make HTTPTransport react to 401 responses, wire through cmd/generate.go.

## Wave

3 (depends on Plans 01 and 02)

## Tasks

### T1: Extend credential storage for OAuth tokens

Modify `internal/auth/credentials.go`:
- Extend ServerCredential: add AccessToken, RefreshToken, ExpiresAt (*time.Time), ClientID, Scope fields (all omitempty)
- `SetOAuthTokens(creds, serverURL, tokens)` — stores OAuth tokens with type="oauth"
- `GetOAuthTokens(creds, serverURL)` — returns OAuthTokens or nil
- `IsTokenExpired(cred)` — checks ExpiresAt
- Update `GetToken` to return AccessToken for oauth type entries
- Backward compatible: no version bump, omitempty on new fields

### T2: Add OAuthProvider interface to HTTP transport

Modify `internal/mcp/http.go`:
- Define `OAuthProvider` interface: `Authenticate(ctx, serverURL) (string, error)`
- Add `oauthProvider` and `oauthAttempted` fields to HTTPTransport
- `NewHTTPTransportWithOAuth(url, authToken, provider)` constructor
- In `Send()`: if 401 and provider set and not already attempted, call provider.Authenticate, set AuthToken, retry once

### T3: Create OAuth provider adapter

Create `internal/oauth/provider.go`:
- Provider struct: HTTPClient, Verbose, CredPath, OnTokens callback
- Implements mcp.OAuthProvider interface
- Logic: check cached tokens → refresh if expired → full flow if no cache
- Calls OnTokens after successful auth for persistence

### T4: Wire OAuth into generate command

Modify `cmd/generate.go`:
- Add `--oauth` flag (bool)
- Validate: --oauth with --stdio → error
- In createTransport: if --oauth, create oauth.Provider, use NewHTTPTransportWithOAuth
- Provider.OnTokens persists to credentials file
- Update help text with OAuth example

### T5: Unit tests

- `internal/auth/auth_test.go` (extend): SetAndGetOAuthTokens, GetToken_OAuthType, IsTokenExpired, backward compat
- `internal/mcp/http_test.go` (new): 401 triggers OAuth, 401 without provider errors, no retry loop, retry only once
- `internal/oauth/provider_test.go`: cached token, expired+refresh, full flow

## Acceptance Criteria

1. OAuth tokens persist to and load from credentials.json
2. Existing bearer credentials unaffected
3. HTTPTransport detects 401 and triggers OAuth exactly once
4. --oauth flag enables OAuth for HTTP servers
5. --oauth with --stdio produces error
6. All existing 119 tests pass
7. New tests pass

## Estimated Complexity

High
