package auth

import (
	"context"
	"encoding/base64"
	"testing"
)

func TestNoAuthProvider(t *testing.T) {
	p := &NoAuthProvider{}
	headers, err := p.GetHeaders(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(headers) != 0 {
		t.Errorf("expected no headers, got %v", headers)
	}

	retry, err := p.OnUnauthorized(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retry {
		t.Error("expected no retry")
	}
}

func TestBearerTokenProvider(t *testing.T) {
	p := &BearerTokenProvider{Token: "my-secret-token"}
	headers, err := p.GetHeaders(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := headers["Authorization"]; got != "Bearer my-secret-token" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer my-secret-token")
	}
}

func TestBearerTokenProvider_EmptyToken(t *testing.T) {
	p := &BearerTokenProvider{Token: ""}
	headers, err := p.GetHeaders(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(headers) != 0 {
		t.Errorf("expected no headers for empty token, got %v", headers)
	}
}

func TestAPIKeyProvider_DefaultHeader(t *testing.T) {
	p := &APIKeyProvider{Token: "key-123"}
	headers, err := p.GetHeaders(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := headers["X-API-Key"]; got != "key-123" {
		t.Errorf("X-API-Key = %q, want %q", got, "key-123")
	}
}

func TestAPIKeyProvider_CustomHeader(t *testing.T) {
	p := &APIKeyProvider{Token: "key-456", HeaderName: "X-Custom-Auth"}
	headers, err := p.GetHeaders(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := headers["X-Custom-Auth"]; got != "key-456" {
		t.Errorf("X-Custom-Auth = %q, want %q", got, "key-456")
	}
	if _, ok := headers["X-API-Key"]; ok {
		t.Error("should not have default X-API-Key header when custom header is set")
	}
}

func TestAPIKeyProvider_EmptyToken(t *testing.T) {
	p := &APIKeyProvider{Token: ""}
	headers, err := p.GetHeaders(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(headers) != 0 {
		t.Errorf("expected no headers for empty token, got %v", headers)
	}
}

func TestBasicAuthProvider(t *testing.T) {
	p := &BasicAuthProvider{Username: "user", Password: "pass"}
	headers, err := p.GetHeaders(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass"))
	if got := headers["Authorization"]; got != expected {
		t.Errorf("Authorization = %q, want %q", got, expected)
	}
}

func TestBasicAuthProvider_EmptyCredentials(t *testing.T) {
	p := &BasicAuthProvider{}
	headers, err := p.GetHeaders(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(headers) != 0 {
		t.Errorf("expected no headers for empty credentials, got %v", headers)
	}
}

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name     string
		authType string
		cred     ServerCredential
		wantErr  bool
		wantType string
	}{
		{
			name:     "no_auth",
			authType: "no_auth",
			wantType: "*auth.NoAuthProvider",
		},
		{
			name:     "none",
			authType: "none",
			wantType: "*auth.NoAuthProvider",
		},
		{
			name:     "empty string",
			authType: "",
			wantType: "*auth.NoAuthProvider",
		},
		{
			name:     "bearer_token",
			authType: "bearer_token",
			cred:     ServerCredential{Token: "tok"},
			wantType: "*auth.BearerTokenProvider",
		},
		{
			name:     "bearer shorthand",
			authType: "bearer",
			cred:     ServerCredential{Token: "tok"},
			wantType: "*auth.BearerTokenProvider",
		},
		{
			name:     "api_key",
			authType: "api_key",
			cred:     ServerCredential{Token: "key", HeaderName: "X-Auth"},
			wantType: "*auth.APIKeyProvider",
		},
		{
			name:     "basic_auth",
			authType: "basic_auth",
			cred:     ServerCredential{Username: "u", Password: "p"},
			wantType: "*auth.BasicAuthProvider",
		},
		{
			name:     "basic shorthand",
			authType: "basic",
			cred:     ServerCredential{Username: "u", Password: "p"},
			wantType: "*auth.BasicAuthProvider",
		},
		{
			name:     "unknown type",
			authType: "unknown",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewProvider(tt.authType, tt.cred)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p == nil {
				t.Fatal("provider is nil")
			}
		})
	}
}

func TestNewProvider_BearerTokenHeaders(t *testing.T) {
	p, err := NewProvider("bearer_token", ServerCredential{Token: "factory-tok"})
	if err != nil {
		t.Fatal(err)
	}
	headers, err := p.GetHeaders(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := headers["Authorization"]; got != "Bearer factory-tok" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer factory-tok")
	}
}

func TestNewProvider_APIKeyHeaders(t *testing.T) {
	p, err := NewProvider("api_key", ServerCredential{Token: "api-key-val", HeaderName: "X-Custom"})
	if err != nil {
		t.Fatal(err)
	}
	headers, err := p.GetHeaders(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := headers["X-Custom"]; got != "api-key-val" {
		t.Errorf("X-Custom = %q, want %q", got, "api-key-val")
	}
}
