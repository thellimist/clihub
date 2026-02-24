package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// ExchangeCode exchanges an authorization code for tokens at the token endpoint.
func ExchangeCode(ctx context.Context, client *http.Client, tokenEndpoint string, params TokenExchangeParams) (*TokenResponse, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {params.Code},
		"redirect_uri":  {params.RedirectURI},
		"client_id":     {params.ClientID},
		"code_verifier": {params.CodeVerifier},
	}

	// Determine auth method: client_secret_post includes secret in body,
	// client_secret_basic uses HTTP Basic auth, none uses just client_id
	authMethod := resolveAuthMethod(params.AuthMethods, params.ClientSecret)

	if authMethod == "client_secret_post" {
		form.Set("client_secret", params.ClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if authMethod == "client_secret_basic" {
		req.SetBasicAuth(params.ClientID, params.ClientSecret)
	}

	return doTokenRequest(client, req)
}

// RefreshAccessToken uses a refresh token to obtain a new access token.
func RefreshAccessToken(ctx context.Context, client *http.Client, tokenEndpoint, clientID, refreshToken string) (*TokenResponse, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {clientID},
		"refresh_token": {refreshToken},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return doTokenRequest(client, req)
}

// resolveAuthMethod picks the auth method based on server support and whether we have a secret.
func resolveAuthMethod(supported []string, clientSecret string) string {
	if clientSecret == "" {
		return "none"
	}
	// If server specifies supported methods, use the first one we can handle
	for _, m := range supported {
		switch m {
		case "client_secret_post", "client_secret_basic":
			return m
		}
	}
	// Default to client_secret_post if we have a secret but no explicit method
	if clientSecret != "" {
		return "client_secret_post"
	}
	return "none"
}

func doTokenRequest(client *http.Client, req *http.Request) (*TokenResponse, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		// Try to parse an OAuth error response
		body, _ := io.ReadAll(resp.Body)
		var errResp struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("token error: %s — %s", errResp.Error, errResp.Description)
		}
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("token response missing access_token")
	}

	return &tokenResp, nil
}
