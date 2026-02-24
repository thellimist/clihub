package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegisterClient_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["client_name"] != "clihub" {
			t.Errorf("expected client_name 'clihub', got %v", body["client_name"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ClientRegistration{ClientID: "test-client-123"})
	}))
	defer ts.Close()

	reg, err := RegisterClient(context.Background(), ts.Client(), ts.URL, "http://127.0.0.1:9999/callback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reg.ClientID != "test-client-123" {
		t.Errorf("got client_id %q, want %q", reg.ClientID, "test-client-123")
	}
}

func TestRegisterClient_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	_, err := RegisterClient(context.Background(), ts.Client(), ts.URL, "http://127.0.0.1:9999/callback")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected status 500 in error, got: %s", err)
	}
}

func TestRegisterClient_VerifyRequestBody(t *testing.T) {
	var captured clientRegistrationRequest
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&captured)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ClientRegistration{ClientID: "x"})
	}))
	defer ts.Close()

	RegisterClient(context.Background(), ts.Client(), ts.URL, "http://127.0.0.1:8080/callback")

	if len(captured.RedirectURIs) != 1 || captured.RedirectURIs[0] != "http://127.0.0.1:8080/callback" {
		t.Errorf("unexpected redirect_uris: %v", captured.RedirectURIs)
	}
	if captured.TokenEndpointAuthMethod != "none" {
		t.Errorf("expected token_endpoint_auth_method 'none', got %q", captured.TokenEndpointAuthMethod)
	}
	if captured.Scope != "mcp:tools" {
		t.Errorf("expected scope 'mcp:tools', got %q", captured.Scope)
	}
}
