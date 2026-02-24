package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExchangeCode_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Form.Get("grant_type") != "authorization_code" {
			t.Errorf("expected grant_type 'authorization_code', got %q", r.Form.Get("grant_type"))
		}
		if r.Form.Get("code") != "test-code" {
			t.Errorf("expected code 'test-code', got %q", r.Form.Get("code"))
		}
		json.NewEncoder(w).Encode(TokenResponse{
			AccessToken:  "access-123",
			TokenType:    "Bearer",
			ExpiresIn:    3600,
			RefreshToken: "refresh-456",
		})
	}))
	defer ts.Close()

	resp, err := ExchangeCode(context.Background(), ts.Client(), ts.URL, TokenExchangeParams{
		Code:         "test-code",
		RedirectURI:  "http://127.0.0.1:9999/callback",
		ClientID:     "client-1",
		CodeVerifier: "verifier-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AccessToken != "access-123" {
		t.Errorf("got access_token %q, want %q", resp.AccessToken, "access-123")
	}
	if resp.RefreshToken != "refresh-456" {
		t.Errorf("got refresh_token %q, want %q", resp.RefreshToken, "refresh-456")
	}
}

func TestExchangeCode_ErrorResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_grant",
			"error_description": "code expired",
		})
	}))
	defer ts.Close()

	_, err := ExchangeCode(context.Background(), ts.Client(), ts.URL, TokenExchangeParams{
		Code: "bad-code", RedirectURI: "x", ClientID: "x", CodeVerifier: "x",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid_grant") {
		t.Errorf("expected 'invalid_grant' in error, got: %s", err)
	}
}

func TestExchangeCode_VerifyFormParams(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("expected form-urlencoded, got %q", r.Header.Get("Content-Type"))
		}
		r.ParseForm()
		if r.Form.Get("code_verifier") != "my-verifier" {
			t.Errorf("expected code_verifier 'my-verifier', got %q", r.Form.Get("code_verifier"))
		}
		if r.Form.Get("redirect_uri") != "http://localhost/cb" {
			t.Errorf("expected redirect_uri 'http://localhost/cb', got %q", r.Form.Get("redirect_uri"))
		}
		json.NewEncoder(w).Encode(TokenResponse{AccessToken: "x", TokenType: "Bearer"})
	}))
	defer ts.Close()

	ExchangeCode(context.Background(), ts.Client(), ts.URL, TokenExchangeParams{
		Code: "c", RedirectURI: "http://localhost/cb", ClientID: "cid", CodeVerifier: "my-verifier",
	})
}

func TestRefreshAccessToken_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Form.Get("grant_type") != "refresh_token" {
			t.Errorf("expected grant_type 'refresh_token', got %q", r.Form.Get("grant_type"))
		}
		if r.Form.Get("refresh_token") != "old-refresh" {
			t.Errorf("expected refresh_token 'old-refresh', got %q", r.Form.Get("refresh_token"))
		}
		json.NewEncoder(w).Encode(TokenResponse{
			AccessToken:  "new-access",
			TokenType:    "Bearer",
			RefreshToken: "new-refresh",
		})
	}))
	defer ts.Close()

	resp, err := RefreshAccessToken(context.Background(), ts.Client(), ts.URL, "cid", "old-refresh")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AccessToken != "new-access" {
		t.Errorf("got %q, want %q", resp.AccessToken, "new-access")
	}
}
