package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
)

// AuthProvider defines the interface for MCP authentication providers.
// Each provider knows how to produce HTTP headers for a specific auth method.
type AuthProvider interface {
	// GetHeaders returns the HTTP headers to include in MCP requests.
	GetHeaders(ctx context.Context) (map[string]string, error)
	// OnUnauthorized is called when the server returns 401. It can attempt
	// re-authentication and return retry=true to retry the request.
	OnUnauthorized(ctx context.Context, resp *http.Response) (retry bool, err error)
}

// NoAuthProvider provides no authentication headers.
type NoAuthProvider struct{}

func (p *NoAuthProvider) GetHeaders(_ context.Context) (map[string]string, error) {
	return nil, nil
}

func (p *NoAuthProvider) OnUnauthorized(_ context.Context, _ *http.Response) (bool, error) {
	return false, nil
}

// BearerTokenProvider provides Bearer token authentication.
type BearerTokenProvider struct {
	Token string
}

func (p *BearerTokenProvider) GetHeaders(_ context.Context) (map[string]string, error) {
	if p.Token == "" {
		return nil, nil
	}
	return map[string]string{"Authorization": "Bearer " + p.Token}, nil
}

func (p *BearerTokenProvider) OnUnauthorized(_ context.Context, _ *http.Response) (bool, error) {
	return false, nil
}

// APIKeyProvider provides API key authentication via a custom header.
type APIKeyProvider struct {
	Token      string
	HeaderName string // Defaults to "X-API-Key" if empty
}

func (p *APIKeyProvider) GetHeaders(_ context.Context) (map[string]string, error) {
	if p.Token == "" {
		return nil, nil
	}
	name := p.HeaderName
	if name == "" {
		name = "X-API-Key"
	}
	return map[string]string{name: p.Token}, nil
}

func (p *APIKeyProvider) OnUnauthorized(_ context.Context, _ *http.Response) (bool, error) {
	return false, nil
}

// BasicAuthProvider provides HTTP Basic authentication.
type BasicAuthProvider struct {
	Username string
	Password string
}

func (p *BasicAuthProvider) GetHeaders(_ context.Context) (map[string]string, error) {
	if p.Username == "" && p.Password == "" {
		return nil, nil
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(p.Username + ":" + p.Password))
	return map[string]string{"Authorization": "Basic " + encoded}, nil
}

func (p *BasicAuthProvider) OnUnauthorized(_ context.Context, _ *http.Response) (bool, error) {
	return false, nil
}

// NewProvider creates an AuthProvider from an auth_type string and credentials.
func NewProvider(authType string, cred ServerCredential) (AuthProvider, error) {
	switch authType {
	case "no_auth", "none", "":
		return &NoAuthProvider{}, nil
	case "bearer_token", "bearer":
		return &BearerTokenProvider{Token: cred.Token}, nil
	case "api_key":
		return &APIKeyProvider{Token: cred.Token, HeaderName: cred.HeaderName}, nil
	case "basic_auth", "basic":
		return &BasicAuthProvider{Username: cred.Username, Password: cred.Password}, nil
	default:
		return nil, fmt.Errorf("unknown auth type: %q", authType)
	}
}
