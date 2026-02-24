package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestS2SOAuth2Provider_ImplementsInterface(t *testing.T) {
	var _ AuthProvider = &S2SOAuth2Provider{}
}

func TestS2SOAuth2Provider_GetHeaders_NoToken(t *testing.T) {
	p := &S2SOAuth2Provider{}
	headers, err := p.GetHeaders(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(headers) != 0 {
		t.Errorf("expected no headers, got %v", headers)
	}
}

func TestS2SOAuth2Provider_GetHeaders_CachedToken(t *testing.T) {
	p := &S2SOAuth2Provider{cachedToken: "s2s-token"}
	headers, err := p.GetHeaders(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := headers["Authorization"]; got != "Bearer s2s-token" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer s2s-token")
	}
}

func TestS2SOAuth2Provider_Authenticate(t *testing.T) {
	// Mock token endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if got := r.FormValue("grant_type"); got != "client_credentials" {
			t.Errorf("grant_type = %q, want %q", got, "client_credentials")
		}
		if got := r.FormValue("client_id"); got != "test-client" {
			t.Errorf("client_id = %q, want %q", got, "test-client")
		}
		if got := r.FormValue("client_secret"); got != "test-secret" {
			t.Errorf("client_secret = %q, want %q", got, "test-secret")
		}
		if got := r.FormValue("scope"); got != "read write" {
			t.Errorf("scope = %q, want %q", got, "read write")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "new-s2s-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer server.Close()

	p := &S2SOAuth2Provider{
		ClientID:      "test-client",
		ClientSecret:  "test-secret",
		TokenEndpoint: server.URL,
		Scope:         "read write",
	}

	token, err := p.Authenticate(context.Background())
	if err != nil {
		t.Fatalf("Authenticate error: %v", err)
	}
	if token != "new-s2s-token" {
		t.Errorf("token = %q, want %q", token, "new-s2s-token")
	}

	// Verify cached
	if p.cachedToken != "new-s2s-token" {
		t.Errorf("cachedToken = %q, want %q", p.cachedToken, "new-s2s-token")
	}

	// GetHeaders should now return the token
	headers, err := p.GetHeaders(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := headers["Authorization"]; got != "Bearer new-s2s-token" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer new-s2s-token")
	}
}

func TestS2SOAuth2Provider_Authenticate_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_client",
			"error_description": "Client authentication failed",
		})
	}))
	defer server.Close()

	p := &S2SOAuth2Provider{
		ClientID:      "bad-client",
		ClientSecret:  "bad-secret",
		TokenEndpoint: server.URL,
	}

	_, err := p.Authenticate(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "S2S OAuth2: invalid_client — Client authentication failed" {
		t.Errorf("error = %q", got)
	}
}

func TestS2SOAuth2Provider_Authenticate_MissingAccessToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"token_type": "Bearer",
		})
	}))
	defer server.Close()

	p := &S2SOAuth2Provider{
		ClientID:      "client",
		ClientSecret:  "secret",
		TokenEndpoint: server.URL,
	}

	_, err := p.Authenticate(context.Background())
	if err == nil {
		t.Fatal("expected error for missing access_token")
	}
}

func TestS2SOAuth2Provider_OnUnauthorized(t *testing.T) {
	// Mock token endpoint that succeeds on retry
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "refreshed-token",
			"token_type":   "Bearer",
		})
	}))
	defer server.Close()

	p := &S2SOAuth2Provider{
		ClientID:      "client",
		ClientSecret:  "secret",
		TokenEndpoint: server.URL,
	}

	retry, err := p.OnUnauthorized(context.Background(), nil)
	if err != nil {
		t.Fatalf("OnUnauthorized error: %v", err)
	}
	if !retry {
		t.Error("expected retry=true after re-auth")
	}
	if p.cachedToken != "refreshed-token" {
		t.Errorf("cachedToken = %q, want %q", p.cachedToken, "refreshed-token")
	}
}

func TestS2SOAuth2Provider_Authenticate_NoEndpointNoServer(t *testing.T) {
	p := &S2SOAuth2Provider{
		ClientID:     "client",
		ClientSecret: "secret",
		ServerURL:    "https://nonexistent.example.com",
	}

	_, err := p.Authenticate(context.Background())
	if err == nil {
		t.Fatal("expected error when no token endpoint and no discoverable server")
	}
}
