package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// S2SOAuth2Provider provides server-to-server OAuth2 authentication
// using the client_credentials grant (RFC 6749 Section 4.4).
type S2SOAuth2Provider struct {
	// ClientID for the client_credentials grant.
	ClientID string
	// ClientSecret for the client_credentials grant.
	ClientSecret string
	// TokenEndpoint is the OAuth2 token endpoint URL.
	// If empty, it will be discovered from the server URL.
	TokenEndpoint string
	// ServerURL is used for token endpoint discovery if TokenEndpoint is empty.
	ServerURL string
	// Scope is the requested scope (optional).
	Scope string

	cachedToken string
}

func (p *S2SOAuth2Provider) GetHeaders(_ context.Context) (map[string]string, error) {
	if p.cachedToken == "" {
		return nil, nil
	}
	return map[string]string{"Authorization": "Bearer " + p.cachedToken}, nil
}

func (p *S2SOAuth2Provider) OnUnauthorized(ctx context.Context, _ *http.Response) (bool, error) {
	// Re-authenticate with client_credentials
	token, err := p.Authenticate(ctx)
	if err != nil {
		return false, err
	}
	p.cachedToken = token
	return true, nil
}

// Authenticate performs the client_credentials grant and returns the access token.
func (p *S2SOAuth2Provider) Authenticate(ctx context.Context) (string, error) {
	tokenEndpoint := p.TokenEndpoint
	if tokenEndpoint == "" {
		// Discover token endpoint
		endpoint, err := discoverTokenEndpoint(ctx, p.ServerURL)
		if err != nil {
			return "", fmt.Errorf("S2S OAuth2: %w", err)
		}
		tokenEndpoint = endpoint
		p.TokenEndpoint = tokenEndpoint
	}

	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {p.ClientID},
		"client_secret": {p.ClientSecret},
	}
	if p.Scope != "" {
		form.Set("scope", p.Scope)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("S2S OAuth2 token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		var errResp struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
			return "", fmt.Errorf("S2S OAuth2: %s — %s", errResp.Error, errResp.Description)
		}
		return "", fmt.Errorf("S2S OAuth2 token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("parse S2S token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("S2S OAuth2 response missing access_token")
	}

	p.cachedToken = tokenResp.AccessToken
	return tokenResp.AccessToken, nil
}
