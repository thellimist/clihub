package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// FetchProtectedResourceMetadata fetches the RFC 9728 protected resource metadata.
// It tries the RFC-compliant URL with path appended first, then falls back to path-less.
func FetchProtectedResourceMetadata(ctx context.Context, client *http.Client, serverURL string) (*ProtectedResourceMetadata, error) {
	urls, err := wellKnownURLs(serverURL, "oauth-protected-resource")
	if err != nil {
		return nil, fmt.Errorf("build discovery URL: %w", err)
	}

	var lastErr error
	for _, wellKnown := range urls {
		meta, err := fetchResourceMeta(ctx, client, wellKnown)
		if err != nil {
			lastErr = err
			continue
		}
		return meta, nil
	}

	return nil, lastErr
}

func fetchResourceMeta(ctx context.Context, client *http.Client, wellKnown string) (*ProtectedResourceMetadata, error) {
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
		io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("protected resource metadata at %s returned %d", wellKnown, resp.StatusCode)
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

// FetchAuthServerMetadata fetches the RFC 8414 authorization server metadata.
// It tries the RFC-compliant URL with path appended first, then falls back to path-less.
func FetchAuthServerMetadata(ctx context.Context, client *http.Client, authServerURL string) (*AuthServerMetadata, error) {
	urls, err := wellKnownURLs(authServerURL, "oauth-authorization-server")
	if err != nil {
		return nil, fmt.Errorf("build discovery URL: %w", err)
	}

	var lastErr error
	for _, wellKnown := range urls {
		meta, err := fetchAuthMeta(ctx, client, wellKnown)
		if err != nil {
			lastErr = err
			continue
		}
		return meta, nil
	}

	return nil, lastErr
}

func fetchAuthMeta(ctx context.Context, client *http.Client, wellKnown string) (*AuthServerMetadata, error) {
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
		io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("auth server metadata at %s returned %d", wellKnown, resp.StatusCode)
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

// wellKnownURLs returns well-known URLs to try in order.
// Per RFC 8414 Section 3, when the URL has a path, the path is appended after
// the well-known segment. Since some servers don't follow this strictly,
// we return both variants (path-appended first, then path-less) for URLs with paths.
// For URLs without a path, only one URL is returned.
func wellKnownURLs(rawURL, suffix string) ([]string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	path := strings.TrimRight(u.Path, "/")
	base := fmt.Sprintf("%s://%s/.well-known/%s", u.Scheme, u.Host, suffix)
	if path == "" {
		return []string{base}, nil
	}
	// Try RFC-compliant path-appended first, then path-less fallback
	return []string{
		base + path,
		base,
	}, nil
}
