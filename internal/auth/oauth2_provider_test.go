package auth

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOAuth2Provider_GetHeaders_CachedToken(t *testing.T) {
	p := &OAuth2Provider{cachedToken: "my-token"}
	headers, err := p.GetHeaders(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := headers["Authorization"]; got != "Bearer my-token" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer my-token")
	}
}

func TestOAuth2Provider_GetHeaders_FromCredStore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")

	// Create a credential store with an OAuth2 token
	exp := time.Now().Add(1 * time.Hour)
	creds := &CredentialsFile{
		Version: 2,
		Servers: map[string]ServerCredential{
			"https://example.com": {
				AuthType:    "oauth2",
				AccessToken: "stored-token",
				ExpiresAt:   &exp,
			},
		},
	}
	if err := SaveCredentials(path, creds); err != nil {
		t.Fatal(err)
	}

	p := &OAuth2Provider{
		ServerURL: "https://example.com",
		CredPath:  path,
	}
	headers, err := p.GetHeaders(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := headers["Authorization"]; got != "Bearer stored-token" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer stored-token")
	}
}

func TestOAuth2Provider_GetHeaders_NoToken(t *testing.T) {
	p := &OAuth2Provider{
		ServerURL: "https://example.com",
		CredPath:  filepath.Join(t.TempDir(), "nonexistent.json"),
	}
	headers, err := p.GetHeaders(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(headers) != 0 {
		t.Errorf("expected no headers, got %v", headers)
	}
}

func TestOAuth2Provider_OnUnauthorized_NoRefreshToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")

	creds := &CredentialsFile{
		Version: 2,
		Servers: map[string]ServerCredential{
			"https://example.com": {
				AuthType:    "oauth2",
				AccessToken: "expired-token",
			},
		},
	}
	if err := SaveCredentials(path, creds); err != nil {
		t.Fatal(err)
	}

	p := &OAuth2Provider{
		ServerURL: "https://example.com",
		CredPath:  path,
	}
	retry, err := p.OnUnauthorized(context.Background(), nil)
	if retry {
		t.Error("expected no retry without refresh token")
	}
	if err == nil {
		t.Error("expected error about missing refresh token")
	}
}

func TestOAuth2Provider_OnUnauthorized_NoCreds(t *testing.T) {
	p := &OAuth2Provider{
		ServerURL: "https://example.com",
		CredPath:  filepath.Join(t.TempDir(), "nonexistent.json"),
	}
	retry, err := p.OnUnauthorized(context.Background(), nil)
	if retry {
		t.Error("expected no retry without credentials")
	}
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOAuth2Provider_OnUnauthorized_NoCredPath(t *testing.T) {
	p := &OAuth2Provider{}
	retry, err := p.OnUnauthorized(context.Background(), nil)
	if retry {
		t.Error("expected no retry without cred path")
	}
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOAuth2Provider_LoadToken_SetsCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")

	exp := time.Now().Add(1 * time.Hour)
	creds := &CredentialsFile{
		Version: 2,
		Servers: map[string]ServerCredential{
			"https://example.com": {
				AuthType:    "oauth2",
				AccessToken: "cached-from-file",
				ExpiresAt:   &exp,
			},
		},
	}
	SaveCredentials(path, creds)

	p := &OAuth2Provider{
		ServerURL: "https://example.com",
		CredPath:  path,
	}
	// First call loads from store
	p.GetHeaders(context.Background())
	if p.cachedToken != "cached-from-file" {
		t.Errorf("cachedToken = %q, want %q", p.cachedToken, "cached-from-file")
	}

	// Modify credential store — cached value should persist
	creds.Servers["https://example.com"] = ServerCredential{
		AuthType:    "oauth2",
		AccessToken: "new-token",
		ExpiresAt:   &exp,
	}
	SaveCredentials(path, creds)

	headers, _ := p.GetHeaders(context.Background())
	if got := headers["Authorization"]; got != "Bearer cached-from-file" {
		t.Errorf("should use cached token, got %q", got)
	}
}

func TestDiscoverTokenEndpoint_NoMetadata(t *testing.T) {
	// This will fail because there's no real server, but it should not panic
	_, err := discoverTokenEndpoint(context.Background(), "https://nonexistent.example.com")
	if err == nil {
		t.Error("expected error for nonexistent server")
	}
}

// TestOAuth2Provider_ImplementsInterface ensures OAuth2Provider satisfies AuthProvider.
func TestOAuth2Provider_ImplementsInterface(t *testing.T) {
	var _ AuthProvider = &OAuth2Provider{}
}

// TestOAuth2Provider_CredPathFromEnv tests loading with env-based cred path.
func TestOAuth2Provider_CredPathFromEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")

	exp := time.Now().Add(1 * time.Hour)
	creds := &CredentialsFile{
		Version: 2,
		Servers: map[string]ServerCredential{
			"https://api.example.com": {
				AuthType:    "oauth2",
				AccessToken: "env-token",
				ExpiresAt:   &exp,
			},
		},
	}
	SaveCredentials(path, creds)

	// Set env and check DefaultCredentialsPath
	t.Setenv("CLIHUB_CREDENTIALS_FILE", path)

	credPath := DefaultCredentialsPath()
	p := &OAuth2Provider{
		ServerURL: "https://api.example.com",
		CredPath:  credPath,
	}
	headers, err := p.GetHeaders(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := headers["Authorization"]; got != "Bearer env-token" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer env-token")
	}

	// Clean up env
	os.Unsetenv("CLIHUB_CREDENTIALS_FILE")
}
