package oauth

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
		if !strings.HasSuffix(r.URL.Path, "/.well-known/oauth-protected-resource") {
			t.Errorf("unexpected path: %s", r.URL.Path)
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
		if !strings.HasSuffix(r.URL.Path, "/.well-known/oauth-authorization-server") {
			t.Errorf("unexpected path: %s", r.URL.Path)
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

func TestWellKnownURL_Construction(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://mcp.notion.com/mcp", "https://mcp.notion.com/.well-known/oauth-protected-resource"},
		{"https://api.example.com/v1/mcp", "https://api.example.com/.well-known/oauth-protected-resource"},
		{"https://example.com", "https://example.com/.well-known/oauth-protected-resource"},
	}

	for _, tt := range tests {
		got, err := wellKnownURL(tt.input, "oauth-protected-resource")
		if err != nil {
			t.Errorf("wellKnownURL(%q): %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("wellKnownURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
