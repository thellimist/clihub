package oauth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// FlowConfig configures the OAuth authentication flow.
type FlowConfig struct {
	ServerURL  string
	HTTPClient *http.Client
	Verbose    func(format string, args ...interface{})
}

// Authenticate runs the full MCP OAuth flow:
// 1. Discover protected resource metadata (RFC 9728)
// 2. Discover authorization server metadata (RFC 8414)
// 3. Start local callback server
// 4. Register client dynamically (RFC 7591)
// 5. Generate PKCE verifier/challenge
// 6. Open browser to authorization URL
// 7. Wait for callback with authorization code
// 8. Exchange code for tokens
func Authenticate(ctx context.Context, cfg FlowConfig) (*OAuthTokens, error) {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	log := cfg.Verbose
	if log == nil {
		log = func(string, ...interface{}) {}
	}

	// Step 1: Discover protected resource metadata
	log("Discovering OAuth endpoints...")
	resMeta, err := FetchProtectedResourceMetadata(ctx, cfg.HTTPClient, cfg.ServerURL)
	if err != nil {
		return nil, fmt.Errorf("OAuth discovery failed: %w", err)
	}

	// Step 2: Discover authorization server metadata
	authMeta, err := FetchAuthServerMetadata(ctx, cfg.HTTPClient, resMeta.AuthorizationServers[0])
	if err != nil {
		return nil, fmt.Errorf("OAuth discovery failed: %w", err)
	}
	log("Found authorization server: %s", authMeta.Issuer)

	// Step 3: Start callback server
	callback := &CallbackServer{}
	if err := callback.Start(); err != nil {
		return nil, fmt.Errorf("could not start local callback server: %w", err)
	}
	defer callback.Close()

	// Step 4: Register client (if registration endpoint available)
	var clientID string
	if authMeta.RegistrationEndpoint != "" {
		log("Registering OAuth client...")
		reg, err := RegisterClient(ctx, cfg.HTTPClient, authMeta.RegistrationEndpoint, callback.RedirectURI())
		if err != nil {
			return nil, fmt.Errorf("client registration failed: %w", err)
		}
		clientID = reg.ClientID
	} else {
		return nil, fmt.Errorf("authorization server does not support dynamic client registration")
	}

	// Step 5: Generate PKCE
	verifier, err := GenerateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("generate PKCE verifier: %w", err)
	}
	challenge := GenerateCodeChallenge(verifier)

	// Step 6: Generate state
	state, err := GenerateState()
	if err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}

	// Step 7: Build authorization URL
	authURL, err := buildAuthorizationURL(authMeta.AuthorizationEndpoint, clientID, callback.RedirectURI(), challenge, state)
	if err != nil {
		return nil, err
	}

	// Step 8: Open browser
	log("Opening browser for authentication...")
	if err := OpenBrowser(authURL); err != nil {
		log("Could not open browser automatically")
	}
	fmt.Printf("If the browser doesn't open, visit:\n%s\n\n", authURL)
	fmt.Println("Waiting for authorization...")

	// Step 9: Wait for callback
	code, err := callback.WaitForCallback(ctx, state)
	if err != nil {
		return nil, fmt.Errorf("authorization failed: %w", err)
	}
	log("Authorization code received")

	// Step 10: Exchange code for tokens
	log("Exchanging authorization code for tokens...")
	tokenResp, err := ExchangeCode(ctx, cfg.HTTPClient, authMeta.TokenEndpoint, TokenExchangeParams{
		Code:         code,
		RedirectURI:  callback.RedirectURI(),
		ClientID:     clientID,
		CodeVerifier: verifier,
	})
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	// Build result
	tokens := &OAuthTokens{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ClientID:     clientID,
		Scope:        tokenResp.Scope,
	}
	if tokenResp.ExpiresIn > 0 {
		tokens.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	log("Authentication complete")
	return tokens, nil
}

func buildAuthorizationURL(endpoint, clientID, redirectURI, codeChallenge, state string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("parse authorization endpoint: %w", err)
	}

	q := u.Query()
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("response_type", "code")
	q.Set("code_challenge", codeChallenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", state)
	q.Set("scope", "mcp:tools")
	u.RawQuery = q.Encode()

	return u.String(), nil
}
