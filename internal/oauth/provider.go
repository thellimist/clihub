package oauth

import (
	"context"
	"net/http"
	"time"

	"github.com/clihub/clihub/internal/mcp"
)

// Provider implements the mcp.OAuthProvider interface using the MCP OAuth flow.
type Provider struct {
	HTTPClient   *http.Client
	Verbose      func(format string, args ...interface{})
	OnTokens     func(serverURL string, tokens *OAuthTokens) // Called after successful auth
	ClientID     string                                       // Pre-registered client ID (skips dynamic registration)
	ClientSecret string                                       // Pre-registered client secret
}

// Authenticate runs the OAuth flow and returns an access token.
func (p *Provider) Authenticate(ctx context.Context, serverURL string, hints *mcp.OAuthHints) (string, error) {
	if p.HTTPClient == nil {
		p.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}

	cfg := FlowConfig{
		ServerURL:    serverURL,
		HTTPClient:   p.HTTPClient,
		Verbose:      p.Verbose,
		ClientID:     p.ClientID,
		ClientSecret: p.ClientSecret,
	}

	// Use hints from the 401 WWW-Authenticate header if available
	if hints != nil {
		cfg.ResourceMetadataURL = hints.ResourceMetadataURL
		if hints.Scope != "" {
			cfg.Scope = hints.Scope
		}
	}

	tokens, err := Authenticate(ctx, cfg)
	if err != nil {
		return "", err
	}

	if p.OnTokens != nil {
		p.OnTokens(serverURL, tokens)
	}

	return tokens.AccessToken, nil
}
