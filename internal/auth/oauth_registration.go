package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type clientRegistrationRequest struct {
	ClientName                string   `json:"client_name"`
	RedirectURIs              []string `json:"redirect_uris"`
	GrantTypes                []string `json:"grant_types"`
	ResponseTypes             []string `json:"response_types"`
	TokenEndpointAuthMethod   string   `json:"token_endpoint_auth_method"`
	Scope                     string   `json:"scope"`
}

// RegisterClient performs RFC 7591 dynamic client registration.
func RegisterClient(ctx context.Context, client *http.Client, registrationEndpoint, redirectURI, scope string) (*ClientRegistration, error) {
	body := clientRegistrationRequest{
		ClientName:              "clihub",
		RedirectURIs:            []string{redirectURI},
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		ResponseTypes:           []string{"code"},
		TokenEndpointAuthMethod: "none",
		Scope:                   scope,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", registrationEndpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("client registration request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("client registration failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var reg ClientRegistration
	if err := json.NewDecoder(resp.Body).Decode(&reg); err != nil {
		return nil, fmt.Errorf("parse registration response: %w", err)
	}

	if reg.ClientID == "" {
		return nil, fmt.Errorf("server returned empty client_id")
	}

	return &reg, nil
}
