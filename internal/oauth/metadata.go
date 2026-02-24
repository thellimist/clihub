package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// FetchProtectedResourceMetadata fetches the RFC 9728 protected resource metadata
// from serverURL/.well-known/oauth-protected-resource.
func FetchProtectedResourceMetadata(ctx context.Context, client *http.Client, serverURL string) (*ProtectedResourceMetadata, error) {
	wellKnown, err := wellKnownURL(serverURL, "oauth-protected-resource")
	if err != nil {
		return nil, fmt.Errorf("build discovery URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", wellKnown, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch protected resource metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("protected resource metadata returned %d", resp.StatusCode)
	}

	var meta ProtectedResourceMetadata
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, fmt.Errorf("parse protected resource metadata: %w", err)
	}

	if len(meta.AuthorizationServers) == 0 {
		return nil, fmt.Errorf("no authorization_servers in protected resource metadata")
	}

	return &meta, nil
}

// FetchAuthServerMetadata fetches the RFC 8414 authorization server metadata
// from authServerURL/.well-known/oauth-authorization-server.
func FetchAuthServerMetadata(ctx context.Context, client *http.Client, authServerURL string) (*AuthServerMetadata, error) {
	wellKnown, err := wellKnownURL(authServerURL, "oauth-authorization-server")
	if err != nil {
		return nil, fmt.Errorf("build discovery URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", wellKnown, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch auth server metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auth server metadata returned %d", resp.StatusCode)
	}

	var meta AuthServerMetadata
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, fmt.Errorf("parse auth server metadata: %w", err)
	}

	if meta.AuthorizationEndpoint == "" {
		return nil, fmt.Errorf("missing authorization_endpoint in auth server metadata")
	}
	if meta.TokenEndpoint == "" {
		return nil, fmt.Errorf("missing token_endpoint in auth server metadata")
	}

	return &meta, nil
}

// wellKnownURL constructs a .well-known URL from the origin of the given URL.
// e.g., https://mcp.notion.com/mcp → https://mcp.notion.com/.well-known/<suffix>
func wellKnownURL(rawURL, suffix string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s://%s/.well-known/%s", u.Scheme, u.Host, suffix), nil
}
