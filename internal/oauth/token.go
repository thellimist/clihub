package oauth

import (
	"context"
	"encoding/json"
	"fmt"
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

	return postTokenRequest(ctx, client, tokenEndpoint, form)
}

// RefreshAccessToken uses a refresh token to obtain a new access token.
func RefreshAccessToken(ctx context.Context, client *http.Client, tokenEndpoint, clientID, refreshToken string) (*TokenResponse, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {clientID},
		"refresh_token": {refreshToken},
	}

	return postTokenRequest(ctx, client, tokenEndpoint, form)
}

func postTokenRequest(ctx context.Context, client *http.Client, tokenEndpoint string, form url.Values) (*TokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Try to parse an OAuth error response
		var errResp struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
		}
		if json.NewDecoder(resp.Body).Decode(&errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("token error: %s — %s", errResp.Error, errResp.Description)
		}
		return nil, fmt.Errorf("token endpoint returned %d", resp.StatusCode)
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
