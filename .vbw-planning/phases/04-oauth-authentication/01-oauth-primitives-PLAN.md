# Plan 01: OAuth Primitives (PKCE, Metadata, Registration, Token Exchange)

## Goal

Build pure library functions for OAuth protocol primitives — all stateless, testable with httptest.NewServer.

## Wave

1 (no dependencies)

## Tasks

### T1: Define OAuth types

Create `internal/oauth/types.go`:
- ProtectedResourceMetadata (RFC 9728): authorization_servers
- AuthServerMetadata (RFC 8414): issuer, authorization_endpoint, token_endpoint, registration_endpoint, scopes_supported, code_challenge_methods_supported
- ClientRegistration (RFC 7591 response): client_id, client_secret
- TokenResponse: access_token, token_type, expires_in, refresh_token, scope
- OAuthTokens: access_token, refresh_token, expires_at, client_id, scope

### T2: Implement PKCE utilities

Create `internal/oauth/pkce.go`:
- `GenerateCodeVerifier()` — crypto/rand, 43-128 chars, unreserved charset per RFC 7636
- `GenerateCodeChallenge(verifier)` — SHA256 + base64url (RawURLEncoding, no padding)

### T3: Implement metadata discovery

Create `internal/oauth/metadata.go`:
- `FetchProtectedResourceMetadata(ctx, client, serverURL)` — GET `<origin>/.well-known/oauth-protected-resource`
- `FetchAuthServerMetadata(ctx, client, authServerURL)` — GET `<origin>/.well-known/oauth-authorization-server`
- URL construction: extract origin (scheme+host), not append to full path

### T4: Implement dynamic client registration

Create `internal/oauth/registration.go`:
- `RegisterClient(ctx, client, registrationEndpoint, redirectURI)` — POST with client_name="clihub", redirect_uris, grant_types=[authorization_code, refresh_token], response_types=[code], token_endpoint_auth_method=none, scope=mcp:tools

### T5: Implement token exchange

Create `internal/oauth/token.go`:
- `ExchangeCode(ctx, client, tokenEndpoint, params)` — POST form-encoded: grant_type=authorization_code, code, redirect_uri, client_id, code_verifier
- `RefreshAccessToken(ctx, client, tokenEndpoint, clientID, refreshToken)` — POST form-encoded: grant_type=refresh_token

### T6: Unit tests

- `pkce_test.go`: verifier length, charset, uniqueness, challenge known vector, no padding
- `metadata_test.go`: success, 404, invalid JSON, URL construction
- `registration_test.go`: success, server error, verify POST body
- `token_test.go`: exchange success, error response, verify form params, refresh success

## Acceptance Criteria

1. PKCE generates valid pairs per RFC 7636
2. Metadata discovery constructs correct .well-known URLs
3. Client registration sends correct POST body per RFC 7591
4. Token exchange sends correct form-encoded parameters
5. All functions handle HTTP errors gracefully
6. All tests pass

## Estimated Complexity

Medium
