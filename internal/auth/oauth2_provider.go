package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// OAuth2Provider provides OAuth2 authentication via stored tokens and
// supports interactive browser flow for initial authentication.
type OAuth2Provider struct {
	// ServerURL is the MCP server URL (used for credential store lookup).
	ServerURL string
	// CredPath is the path to the credentials file.
	CredPath string
	// ClientID is a pre-registered OAuth client ID (optional; skips DCR if set).
	ClientID string
	// ClientSecret is a pre-registered OAuth client secret (optional).
	ClientSecret string
	// Verbose is an optional logging function.
	Verbose func(format string, args ...interface{})

	// cachedToken is the access token from the last successful auth.
	cachedToken string
}

func (p *OAuth2Provider) GetHeaders(_ context.Context) (map[string]string, error) {
	token := p.cachedToken
	if token == "" {
		// Try loading from credential store
		token = p.loadToken()
	}
	if token == "" {
		return nil, nil
	}
	return map[string]string{"Authorization": "Bearer " + token}, nil
}

func (p *OAuth2Provider) OnUnauthorized(ctx context.Context, _ *http.Response) (bool, error) {
	// Try token refresh
	if p.CredPath == "" {
		return false, nil
	}
	creds, err := LoadCredentials(p.CredPath)
	if err != nil {
		return false, nil
	}
	sc := GetOAuthCredential(creds, p.ServerURL)
	if sc == nil {
		return false, nil
	}
	if sc.RefreshToken == "" {
		return false, fmt.Errorf("access token expired and no refresh token available")
	}

	// Discover token endpoint
	tokenEndpoint := sc.TokenEndpoint
	if tokenEndpoint == "" {
		// Try to discover from auth server metadata
		endpoint, err := discoverTokenEndpoint(ctx, p.ServerURL)
		if err != nil {
			return false, fmt.Errorf("cannot refresh: %w", err)
		}
		tokenEndpoint = endpoint
	}

	// Attempt refresh
	httpClient := &http.Client{Timeout: 30 * time.Second}
	tokenResp, err := RefreshAccessToken(ctx, httpClient, tokenEndpoint, sc.ClientID, sc.RefreshToken)
	if err != nil {
		return false, fmt.Errorf("token refresh failed: %w", err)
	}

	// Update credential store
	var expiresAt *time.Time
	if tokenResp.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		expiresAt = &t
	}
	refreshToken := tokenResp.RefreshToken
	if refreshToken == "" {
		refreshToken = sc.RefreshToken // keep old refresh token if not rotated
	}
	SetOAuthTokens(creds, p.ServerURL, tokenResp.AccessToken, refreshToken, sc.ClientID, sc.Scope, expiresAt)
	if tokenEndpoint != "" {
		// Store token endpoint for future refreshes
		entry := creds.Servers[p.ServerURL]
		entry.TokenEndpoint = tokenEndpoint
		creds.Servers[p.ServerURL] = entry
	}
	_ = SaveCredentials(p.CredPath, creds)

	p.cachedToken = tokenResp.AccessToken
	return true, nil
}

// RunInteractiveFlow runs the interactive OAuth2 browser flow and stores the tokens.
func (p *OAuth2Provider) RunInteractiveFlow(ctx context.Context) (string, error) {
	cfg := FlowConfig{
		ServerURL:    p.ServerURL,
		ClientID:     p.ClientID,
		ClientSecret: p.ClientSecret,
		Verbose:      p.Verbose,
	}

	tokens, err := Authenticate(ctx, cfg)
	if err != nil {
		return "", err
	}

	// Save tokens to credential store
	if p.CredPath != "" {
		creds, loadErr := LoadCredentials(p.CredPath)
		if loadErr == nil {
			SetOAuthTokens(creds, p.ServerURL, tokens.AccessToken, tokens.RefreshToken, tokens.ClientID, tokens.Scope, &tokens.ExpiresAt)
			_ = SaveCredentials(p.CredPath, creds)
		}
	}

	p.cachedToken = tokens.AccessToken
	return tokens.AccessToken, nil
}

func (p *OAuth2Provider) loadToken() string {
	if p.CredPath == "" {
		return ""
	}
	creds, err := LoadCredentials(p.CredPath)
	if err != nil {
		return ""
	}
	token := GetToken(creds, p.ServerURL)
	if token != "" {
		p.cachedToken = token
	}
	return token
}

// discoverTokenEndpoint finds the token endpoint for a server using OAuth metadata discovery.
func discoverTokenEndpoint(ctx context.Context, serverURL string) (string, error) {
	httpClient := &http.Client{Timeout: 30 * time.Second}

	// Try protected resource metadata first to find auth server
	resMeta, err := FetchProtectedResourceMetadata(ctx, httpClient, serverURL)
	var authServerURL string
	if err == nil && len(resMeta.AuthorizationServers) > 0 {
		authServerURL = resMeta.AuthorizationServers[0]
	} else {
		// Fallback to server root
		authServerURL = serverURL
	}

	// Fetch auth server metadata
	authMeta, err := FetchAuthServerMetadata(ctx, httpClient, authServerURL)
	if err != nil {
		return "", fmt.Errorf("could not discover token endpoint: %w", err)
	}
	return authMeta.TokenEndpoint, nil
}
