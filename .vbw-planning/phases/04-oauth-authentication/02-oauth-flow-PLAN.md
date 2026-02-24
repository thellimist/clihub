# Plan 02: Callback Server, Browser Opener, and OAuth Flow Orchestration

## Goal

Build the interactive OAuth components (local HTTP server, browser launch) and the orchestrator that ties all Plan 01 primitives into a complete authentication flow.

## Wave

2 (depends on Plan 01)

## Tasks

### T1: Implement local callback server

Create `internal/oauth/callback.go`:
- CallbackServer struct with Port, server, result channel
- `Start()` — listen on 127.0.0.1:0 (random port)
- `RedirectURI()` — returns http://127.0.0.1:<port>/callback
- `WaitForCallback(ctx, expectedState)` — blocks until callback received, validates state
- `Close()` — shuts down server
- /callback handler extracts ?code=&state=, renders success HTML, sends result on channel
- Handles ?error= parameter for OAuth errors

### T2: Implement browser opener

Create `internal/oauth/browser.go`:
- `OpenBrowser(url)` — darwin: `open`, linux: `xdg-open`, windows: `cmd /c start "" <url>`
- Best-effort, returns error if launch fails

### T3: Implement state parameter generation

Create `internal/oauth/state.go`:
- `GenerateState()` — crypto/rand 32 bytes → hex string

### T4: Implement OAuth flow orchestrator

Create `internal/oauth/flow.go`:
- `Authenticate(ctx, cfg FlowConfig) (*OAuthTokens, error)`
- FlowConfig: ServerURL, HTTPClient, Verbose
- Full flow:
  1. FetchProtectedResourceMetadata
  2. FetchAuthServerMetadata (from first authorization_server)
  3. Start CallbackServer
  4. RegisterClient with callback RedirectURI
  5. GenerateCodeVerifier + GenerateCodeChallenge
  6. GenerateState
  7. Build authorization URL with client_id, redirect_uri, code_challenge, code_challenge_method=S256, state, scope
  8. OpenBrowser + print fallback URL
  9. WaitForCallback (with state validation)
  10. ExchangeCode
  11. Return OAuthTokens with computed ExpiresAt

### T5: Unit tests

- `callback_test.go`: start & port, successful callback, state validation, error response, context cancellation
- `state_test.go`: length (64 hex chars), uniqueness
- `flow_test.go`: full flow with mock servers + simulated browser callback, metadata failure, token exchange failure

## Acceptance Criteria

1. Callback server starts on random port and receives authorization codes
2. State parameter validation prevents CSRF
3. Browser opener works cross-platform
4. Flow orchestrator connects all pieces in correct order
5. Timeout/cancellation propagates correctly
6. All tests pass

## Estimated Complexity

High
