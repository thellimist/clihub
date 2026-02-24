package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchProtectedResourceMetadata_Success(t *testing.T) {
	meta := ProtectedResourceMetadata{
		Resource:             "https://mcp.example.com",
		AuthorizationServers: []string{"https://auth.example.com"},
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/.well-known/oauth-protected-resource") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(meta)
	}))
	defer ts.Close()

	got, err := FetchProtectedResourceMetadata(context.Background(), ts.Client(), ts.URL+"/mcp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.AuthorizationServers) != 1 || got.AuthorizationServers[0] != "https://auth.example.com" {
		t.Errorf("unexpected authorization_servers: %v", got.AuthorizationServers)
	}
}

func TestFetchProtectedResourceMetadata_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	_, err := FetchProtectedResourceMetadata(context.Background(), ts.Client(), ts.URL)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected 404 in error, got: %s", err)
	}
}

func TestFetchProtectedResourceMetadata_NoAuthServers(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer ts.Close()

	_, err := FetchProtectedResourceMetadata(context.Background(), ts.Client(), ts.URL)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no authorization_servers") {
		t.Errorf("expected 'no authorization_servers' error, got: %s", err)
	}
}

func TestFetchAuthServerMetadata_Success(t *testing.T) {
	meta := AuthServerMetadata{
		Issuer:                "https://auth.example.com",
		AuthorizationEndpoint: "https://auth.example.com/authorize",
		TokenEndpoint:         "https://auth.example.com/token",
		RegistrationEndpoint:  "https://auth.example.com/register",
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/.well-known/oauth-authorization-server") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(meta)
	}))
	defer ts.Close()

	got, err := FetchAuthServerMetadata(context.Background(), ts.Client(), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.AuthorizationEndpoint != "https://auth.example.com/authorize" {
		t.Errorf("unexpected authorization_endpoint: %s", got.AuthorizationEndpoint)
	}
	if got.TokenEndpoint != "https://auth.example.com/token" {
		t.Errorf("unexpected token_endpoint: %s", got.TokenEndpoint)
	}
}

func TestFetchAuthServerMetadata_WithPath(t *testing.T) {
	// Simulates Stripe-like server: auth server at /mcp path, well-known at /mcp path
	meta := AuthServerMetadata{
		Issuer:                "https://auth.example.com/mcp",
		AuthorizationEndpoint: "https://auth.example.com/mcp/authorize",
		TokenEndpoint:         "https://auth.example.com/mcp/token",
		RegistrationEndpoint:  "https://auth.example.com/mcp/register",
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only respond to the RFC-compliant path-appended URL
		if r.URL.Path == "/.well-known/oauth-authorization-server/mcp" {
			json.NewEncoder(w).Encode(meta)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	got, err := FetchAuthServerMetadata(context.Background(), ts.Client(), ts.URL+"/mcp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.AuthorizationEndpoint != "https://auth.example.com/mcp/authorize" {
		t.Errorf("unexpected authorization_endpoint: %s", got.AuthorizationEndpoint)
	}
}

func TestFetchAuthServerMetadata_FallbackToPathless(t *testing.T) {
	// Simulates Notion-like server: auth server URL has no path, or path-appended returns 404
	meta := AuthServerMetadata{
		Issuer:                "https://auth.example.com",
		AuthorizationEndpoint: "https://auth.example.com/authorize",
		TokenEndpoint:         "https://auth.example.com/token",
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only respond to the path-less URL
		if r.URL.Path == "/.well-known/oauth-authorization-server" {
			json.NewEncoder(w).Encode(meta)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	got, err := FetchAuthServerMetadata(context.Background(), ts.Client(), ts.URL+"/mcp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.AuthorizationEndpoint != "https://auth.example.com/authorize" {
		t.Errorf("unexpected authorization_endpoint: %s", got.AuthorizationEndpoint)
	}
}

func TestFetchAuthServerMetadata_MissingFields(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"issuer": "x"})
	}))
	defer ts.Close()

	_, err := FetchAuthServerMetadata(context.Background(), ts.Client(), ts.URL)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Errorf("expected 'missing' error, got: %s", err)
	}
}

func TestWellKnownURLs_Construction(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{
			"https://example.com",
			[]string{"https://example.com/.well-known/oauth-protected-resource"},
		},
		{
			"https://example.com/",
			[]string{"https://example.com/.well-known/oauth-protected-resource"},
		},
		{
			"https://mcp.notion.com/mcp",
			[]string{
				"https://mcp.notion.com/.well-known/oauth-protected-resource/mcp",
				"https://mcp.notion.com/.well-known/oauth-protected-resource",
			},
		},
		{
			"https://access.stripe.com/mcp",
			[]string{
				"https://access.stripe.com/.well-known/oauth-protected-resource/mcp",
				"https://access.stripe.com/.well-known/oauth-protected-resource",
			},
		},
	}

	for _, tt := range tests {
		got, err := wellKnownURLs(tt.input, "oauth-protected-resource")
		if err != nil {
			t.Errorf("wellKnownURLs(%q): %v", tt.input, err)
			continue
		}
		if len(got) != len(tt.want) {
			t.Errorf("wellKnownURLs(%q) = %v (len %d), want %v (len %d)", tt.input, got, len(got), tt.want, len(tt.want))
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("wellKnownURLs(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}
