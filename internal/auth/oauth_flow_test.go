package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAuthenticate_FullFlow(t *testing.T) {
	// Mock authorization server endpoints
	var capturedClientID string

	mux := http.NewServeMux()

	// Protected resource metadata
	mux.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
		// Will be filled with auth server URL after server starts
	})

	// Auth server metadata (handler set after server URL is known)
	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		// Will be handled by the actual test server
	})

	// Registration endpoint
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		uris := body["redirect_uris"].([]interface{})
		_ = uris[0].(string)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ClientRegistration{ClientID: "test-client-42"})
	})

	// Token endpoint
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		capturedClientID = r.Form.Get("client_id")
		json.NewEncoder(w).Encode(TokenResponse{
			AccessToken:  "access-token-xyz",
			TokenType:    "Bearer",
			ExpiresIn:    3600,
			RefreshToken: "refresh-token-abc",
			Scope:        "mcp:tools",
		})
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Now set up the well-known handlers to return the test server URL
	mux.HandleFunc("/resource-meta", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ProtectedResourceMetadata{
			AuthorizationServers: []string{ts.URL},
		})
	})

	// We need a custom server since we can't modify mux handlers after creation.
	// Instead, create a new server with proper routing.
	ts.Close()

	mux2 := http.NewServeMux()
	var ts2URL string

	mux2.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ProtectedResourceMetadata{
			AuthorizationServers: []string{ts2URL},
		})
	})
	mux2.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(AuthServerMetadata{
			Issuer:                ts2URL,
			AuthorizationEndpoint: ts2URL + "/authorize",
			TokenEndpoint:         ts2URL + "/token",
			RegistrationEndpoint:  ts2URL + "/register",
		})
	})
	mux2.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		uris := body["redirect_uris"].([]interface{})
		_ = uris[0].(string)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ClientRegistration{ClientID: "test-client-42"})
	})
	mux2.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		capturedClientID = r.Form.Get("client_id")
		json.NewEncoder(w).Encode(TokenResponse{
			AccessToken:  "access-token-xyz",
			TokenType:    "Bearer",
			ExpiresIn:    3600,
			RefreshToken: "refresh-token-abc",
			Scope:        "mcp:tools",
		})
	})
	// Authorization endpoint — simulate browser redirect by hitting the callback directly
	mux2.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		redirectURI := r.URL.Query().Get("redirect_uri")
		state := r.URL.Query().Get("state")
		// Simulate the browser redirect to the callback
		go func() {
			time.Sleep(50 * time.Millisecond)
			http.Get(fmt.Sprintf("%s?code=auth-code-123&state=%s", redirectURI, state))
		}()
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "authorize page")
	})

	ts2 := httptest.NewServer(mux2)
	defer ts2.Close()
	ts2URL = ts2.URL

	// Override OpenBrowser to simulate browser visiting the authorize URL
	origOpen := OpenBrowser
	OpenBrowser = func(url string) error {
		// Simulate browser: GET the authorize URL which triggers the callback
		go http.Get(url)
		return nil
	}
	defer func() { OpenBrowser = origOpen }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tokens, err := Authenticate(ctx, FlowConfig{
		ServerURL:  ts2URL + "/mcp",
		HTTPClient: ts2.Client(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tokens.AccessToken != "access-token-xyz" {
		t.Errorf("got access_token %q, want %q", tokens.AccessToken, "access-token-xyz")
	}
	if tokens.RefreshToken != "refresh-token-abc" {
		t.Errorf("got refresh_token %q, want %q", tokens.RefreshToken, "refresh-token-abc")
	}
	if capturedClientID != "test-client-42" {
		t.Errorf("got client_id %q, want %q", capturedClientID, "test-client-42")
	}
	if tokens.ExpiresAt.IsZero() {
		t.Error("expected non-zero expires_at")
	}
}

func TestAuthenticate_MetadataDiscoveryFails(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := Authenticate(ctx, FlowConfig{
		ServerURL:  ts.URL,
		HTTPClient: ts.Client(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "discovery failed") {
		t.Errorf("expected 'discovery failed' error, got: %s", err)
	}
}
