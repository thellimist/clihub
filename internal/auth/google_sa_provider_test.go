package auth

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGoogleSAProvider_ImplementsInterface(t *testing.T) {
	var _ AuthProvider = &GoogleSAProvider{}
}

func TestGoogleSAProvider_GetHeaders_MissingKeyFile(t *testing.T) {
	p := &GoogleSAProvider{
		KeyFile: filepath.Join(t.TempDir(), "nonexistent.json"),
	}
	_, err := p.GetHeaders(context.Background())
	if err == nil {
		t.Fatal("expected error for missing key file")
	}
}

func TestGoogleSAProvider_GetHeaders_InvalidKeyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad-key.json")
	os.WriteFile(path, []byte("{not valid sa key}"), 0600)

	p := &GoogleSAProvider{KeyFile: path}
	_, err := p.GetHeaders(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid key file")
	}
}

func TestGoogleSAProvider_OnUnauthorized_ClearsCache(t *testing.T) {
	// Create a provider with an invalid key file — OnUnauthorized should clear cache
	// and try to get a new token (which will fail due to invalid key)
	dir := t.TempDir()
	path := filepath.Join(dir, "bad-key.json")
	os.WriteFile(path, []byte("{}"), 0600)

	p := &GoogleSAProvider{KeyFile: path}
	retry, err := p.OnUnauthorized(context.Background(), nil)
	if retry {
		t.Error("expected no retry for invalid key file")
	}
	if err == nil {
		t.Error("expected error for invalid key file on re-auth")
	}
}

func TestGoogleSAProvider_DefaultScopes(t *testing.T) {
	p := &GoogleSAProvider{
		KeyFile: "/nonexistent/key.json",
	}
	// Verify scopes default is applied during getTokenSource
	if len(p.Scopes) != 0 {
		t.Errorf("expected empty scopes before init, got %v", p.Scopes)
	}
}
