package auth

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// LoadCredentials tests
// ---------------------------------------------------------------------------

func TestLoadCredentials_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")

	// Write a v1 file — should be auto-migrated to v2 on load
	v1Data := `{"version":1,"servers":{"https://mcp.example.com":{"token":"tok123","type":"bearer"}}}`
	if err := os.WriteFile(path, []byte(v1Data), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := LoadCredentials(path)
	if err != nil {
		t.Fatalf("LoadCredentials returned error: %v", err)
	}
	if got.Version != 2 {
		t.Errorf("Version = %d, want 2 (auto-migrated)", got.Version)
	}
	sc, ok := got.Servers["https://mcp.example.com"]
	if !ok {
		t.Fatal("server entry not found")
	}
	if sc.Token != "tok123" {
		t.Errorf("Token = %q, want %q", sc.Token, "tok123")
	}
	if sc.AuthType != "bearer_token" {
		t.Errorf("AuthType = %q, want %q", sc.AuthType, "bearer_token")
	}
}

func TestLoadCredentials_NonExistentFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "does-not-exist.json")

	got, err := LoadCredentials(path)
	if err != nil {
		t.Fatalf("LoadCredentials returned error for missing file: %v", err)
	}
	if got.Version != 2 {
		t.Errorf("Version = %d, want 2", got.Version)
	}
	if got.Servers == nil {
		t.Fatal("Servers map is nil, want empty map")
	}
	if len(got.Servers) != 0 {
		t.Errorf("Servers has %d entries, want 0", len(got.Servers))
	}
}

func TestLoadCredentials_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")

	if err := os.WriteFile(path, []byte("{not json!!}"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadCredentials(path)
	if err == nil {
		t.Fatal("LoadCredentials should return error for malformed JSON")
	}
}

// ---------------------------------------------------------------------------
// SaveCredentials tests
// ---------------------------------------------------------------------------

func TestSaveCredentials_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "credentials.json")

	creds := &CredentialsFile{
		Version: 2,
		Servers: map[string]ServerCredential{
			"https://example.com": {AuthType: "bearer_token", Token: "abc"},
		},
	}
	if err := SaveCredentials(path, creds); err != nil {
		t.Fatalf("SaveCredentials returned error: %v", err)
	}

	// Verify the file was created.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read saved file: %v", err)
	}
	var loaded CredentialsFile
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("cannot parse saved file: %v", err)
	}
	if loaded.Version != 2 {
		t.Errorf("Version = %d, want 2", loaded.Version)
	}
	if loaded.Servers["https://example.com"].Token != "abc" {
		t.Errorf("Token = %q, want %q", loaded.Servers["https://example.com"].Token, "abc")
	}
}

func TestSaveCredentials_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")

	creds := &CredentialsFile{Version: 2, Servers: make(map[string]ServerCredential)}
	if err := SaveCredentials(path, creds); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	perm := info.Mode().Perm()
	if perm != fs.FileMode(0600) {
		t.Errorf("file permissions = %04o, want 0600", perm)
	}
}

// ---------------------------------------------------------------------------
// GetToken / SetToken tests
// ---------------------------------------------------------------------------

func TestSetAndGetToken(t *testing.T) {
	creds := &CredentialsFile{
		Version: 2,
		Servers: make(map[string]ServerCredential),
	}
	SetToken(creds, "https://example.com", "mytoken")

	got := GetToken(creds, "https://example.com")
	if got != "mytoken" {
		t.Errorf("GetToken = %q, want %q", got, "mytoken")
	}

	// Verify auth_type is set correctly (v2).
	sc := creds.Servers["https://example.com"]
	if sc.AuthType != "bearer_token" {
		t.Errorf("AuthType = %q, want %q", sc.AuthType, "bearer_token")
	}
}

func TestGetToken_EmptyMap(t *testing.T) {
	creds := &CredentialsFile{
		Version: 2,
		Servers: make(map[string]ServerCredential),
	}
	got := GetToken(creds, "https://nonexistent.com")
	if got != "" {
		t.Errorf("GetToken = %q, want empty string", got)
	}
}

func TestGetToken_NilMap(t *testing.T) {
	creds := &CredentialsFile{Version: 2}
	got := GetToken(creds, "https://example.com")
	if got != "" {
		t.Errorf("GetToken with nil map = %q, want empty string", got)
	}
}

func TestSetToken_NilServersMap(t *testing.T) {
	creds := &CredentialsFile{Version: 2}
	SetToken(creds, "https://example.com", "tok")

	if creds.Servers == nil {
		t.Fatal("Servers should be initialized, got nil")
	}
	got := GetToken(creds, "https://example.com")
	if got != "tok" {
		t.Errorf("GetToken = %q, want %q", got, "tok")
	}
}

// ---------------------------------------------------------------------------
// OAuth credential tests
// ---------------------------------------------------------------------------

func TestSetAndGetOAuthTokens(t *testing.T) {
	creds := &CredentialsFile{Version: 2, Servers: make(map[string]ServerCredential)}
	exp := time.Now().Add(1 * time.Hour)
	SetOAuthTokens(creds, "https://mcp.notion.com", "access-1", "refresh-1", "client-1", "mcp:tools", &exp)

	sc := creds.Servers["https://mcp.notion.com"]
	if sc.AuthType != "oauth2" {
		t.Errorf("AuthType = %q, want %q", sc.AuthType, "oauth2")
	}
	if sc.AccessToken != "access-1" {
		t.Errorf("AccessToken = %q, want %q", sc.AccessToken, "access-1")
	}
	if sc.RefreshToken != "refresh-1" {
		t.Errorf("RefreshToken = %q, want %q", sc.RefreshToken, "refresh-1")
	}
	if sc.ClientID != "client-1" {
		t.Errorf("ClientID = %q, want %q", sc.ClientID, "client-1")
	}
}

func TestGetToken_OAuthType(t *testing.T) {
	creds := &CredentialsFile{Version: 2, Servers: make(map[string]ServerCredential)}
	exp := time.Now().Add(1 * time.Hour)
	SetOAuthTokens(creds, "https://mcp.example.com", "oauth-access", "refresh", "cid", "", &exp)

	got := GetToken(creds, "https://mcp.example.com")
	if got != "oauth-access" {
		t.Errorf("GetToken = %q, want %q", got, "oauth-access")
	}
}

func TestGetOAuthCredential(t *testing.T) {
	creds := &CredentialsFile{Version: 2, Servers: make(map[string]ServerCredential)}
	exp := time.Now().Add(1 * time.Hour)
	SetOAuthTokens(creds, "https://example.com", "a", "r", "c", "s", &exp)

	sc := GetOAuthCredential(creds, "https://example.com")
	if sc == nil {
		t.Fatal("expected non-nil credential")
	}
	if sc.ClientID != "c" {
		t.Errorf("ClientID = %q, want %q", sc.ClientID, "c")
	}

	// Bearer type should return nil
	SetToken(creds, "https://bearer.com", "tok")
	if got := GetOAuthCredential(creds, "https://bearer.com"); got != nil {
		t.Error("expected nil for bearer credential")
	}
}

func TestIsTokenExpired_NotExpired(t *testing.T) {
	exp := time.Now().Add(1 * time.Hour)
	sc := ServerCredential{ExpiresAt: &exp}
	if IsTokenExpired(sc) {
		t.Error("token should not be expired")
	}
}

func TestIsTokenExpired_Expired(t *testing.T) {
	exp := time.Now().Add(-1 * time.Hour)
	sc := ServerCredential{ExpiresAt: &exp}
	if !IsTokenExpired(sc) {
		t.Error("token should be expired")
	}
}

func TestIsTokenExpired_NoExpiry(t *testing.T) {
	sc := ServerCredential{}
	if IsTokenExpired(sc) {
		t.Error("token without expiry should not be expired")
	}
}

func TestCredentialsBackwardCompatibility(t *testing.T) {
	// Old-style v1 JSON with only token and type — should auto-migrate to v2
	data := `{"version":1,"servers":{"https://old.com":{"token":"old-tok","type":"bearer"}}}`
	dir := t.TempDir()
	path := filepath.Join(dir, "creds.json")
	os.WriteFile(path, []byte(data), 0600)

	creds, err := LoadCredentials(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.Version != 2 {
		t.Errorf("Version = %d, want 2 (auto-migrated)", creds.Version)
	}
	got := GetToken(creds, "https://old.com")
	if got != "old-tok" {
		t.Errorf("got %q, want %q", got, "old-tok")
	}
	sc := creds.Servers["https://old.com"]
	if sc.AuthType != "bearer_token" {
		t.Errorf("AuthType = %q, want %q", sc.AuthType, "bearer_token")
	}
}

func TestV1OAuthMigration(t *testing.T) {
	data := `{"version":1,"servers":{"https://mcp.notion.com":{"type":"oauth","access_token":"at","refresh_token":"rt","client_id":"cid","scope":"mcp:tools"}}}`
	dir := t.TempDir()
	path := filepath.Join(dir, "creds.json")
	os.WriteFile(path, []byte(data), 0600)

	creds, err := LoadCredentials(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.Version != 2 {
		t.Errorf("Version = %d, want 2", creds.Version)
	}
	sc := creds.Servers["https://mcp.notion.com"]
	if sc.AuthType != "oauth2" {
		t.Errorf("AuthType = %q, want %q", sc.AuthType, "oauth2")
	}
	if sc.AccessToken != "at" {
		t.Errorf("AccessToken = %q, want %q", sc.AccessToken, "at")
	}
	if sc.RefreshToken != "rt" {
		t.Errorf("RefreshToken = %q, want %q", sc.RefreshToken, "rt")
	}
	got := GetToken(creds, "https://mcp.notion.com")
	if got != "at" {
		t.Errorf("GetToken = %q, want %q", got, "at")
	}
}

func TestV2Format(t *testing.T) {
	// Test that a v2 file loads without migration
	data := `{"version":2,"servers":{"https://api.example.com":{"auth_type":"api_key","token":"key123","header_name":"X-Auth"}}}`
	dir := t.TempDir()
	path := filepath.Join(dir, "creds.json")
	os.WriteFile(path, []byte(data), 0600)

	creds, err := LoadCredentials(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.Version != 2 {
		t.Errorf("Version = %d, want 2", creds.Version)
	}
	sc := creds.Servers["https://api.example.com"]
	if sc.AuthType != "api_key" {
		t.Errorf("AuthType = %q, want %q", sc.AuthType, "api_key")
	}
	if sc.Token != "key123" {
		t.Errorf("Token = %q, want %q", sc.Token, "key123")
	}
	if sc.HeaderName != "X-Auth" {
		t.Errorf("HeaderName = %q, want %q", sc.HeaderName, "X-Auth")
	}
}

// ---------------------------------------------------------------------------
// DefaultCredentialsPath tests
// ---------------------------------------------------------------------------

func TestDefaultCredentialsPath_WithoutEnv(t *testing.T) {
	t.Setenv("CLIHUB_CREDENTIALS_FILE", "")

	got := DefaultCredentialsPath()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".clihub", "credentials.json")
	if got != want {
		t.Errorf("DefaultCredentialsPath() = %q, want %q", got, want)
	}
}

func TestDefaultCredentialsPath_WithEnv(t *testing.T) {
	t.Setenv("CLIHUB_CREDENTIALS_FILE", "/custom/path/creds.json")

	got := DefaultCredentialsPath()
	if got != "/custom/path/creds.json" {
		t.Errorf("DefaultCredentialsPath() = %q, want %q", got, "/custom/path/creds.json")
	}
}

// ---------------------------------------------------------------------------
// LookupToken tests
// ---------------------------------------------------------------------------

func TestLookupToken_FlagWins(t *testing.T) {
	t.Setenv("CLIHUB_AUTH_TOKEN", "env-token")
	t.Setenv("CLIHUB_CREDENTIALS_FILE", "")

	got := LookupToken("flag-token", "https://example.com")
	if got != "flag-token" {
		t.Errorf("LookupToken = %q, want %q", got, "flag-token")
	}
}

func TestLookupToken_EnvVarWins(t *testing.T) {
	t.Setenv("CLIHUB_AUTH_TOKEN", "env-token")
	t.Setenv("CLIHUB_CREDENTIALS_FILE", "")

	got := LookupToken("", "https://example.com")
	if got != "env-token" {
		t.Errorf("LookupToken = %q, want %q", got, "env-token")
	}
}

func TestLookupToken_CredentialsFileFallback(t *testing.T) {
	t.Setenv("CLIHUB_AUTH_TOKEN", "")

	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")
	creds := &CredentialsFile{
		Version: 1,
		Servers: map[string]ServerCredential{
			"https://example.com": {Token: "file-token", Type: "bearer"},
		},
	}
	if err := SaveCredentials(path, creds); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLIHUB_CREDENTIALS_FILE", path)

	got := LookupToken("", "https://example.com")
	if got != "file-token" {
		t.Errorf("LookupToken = %q, want %q", got, "file-token")
	}
}

func TestLookupToken_NoTokenFound(t *testing.T) {
	t.Setenv("CLIHUB_AUTH_TOKEN", "")
	t.Setenv("CLIHUB_CREDENTIALS_FILE", filepath.Join(t.TempDir(), "nonexistent.json"))

	got := LookupToken("", "https://example.com")
	if got != "" {
		t.Errorf("LookupToken = %q, want empty string", got)
	}
}

func TestLookupToken_CredentialsFileEnvOverride(t *testing.T) {
	t.Setenv("CLIHUB_AUTH_TOKEN", "")

	// Create two credentials files with different tokens.
	dir := t.TempDir()
	customPath := filepath.Join(dir, "custom", "creds.json")
	creds := &CredentialsFile{
		Version: 1,
		Servers: map[string]ServerCredential{
			"https://server.io": {Token: "custom-tok", Type: "bearer"},
		},
	}
	if err := SaveCredentials(customPath, creds); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLIHUB_CREDENTIALS_FILE", customPath)

	got := LookupToken("", "https://server.io")
	if got != "custom-tok" {
		t.Errorf("LookupToken = %q, want %q", got, "custom-tok")
	}
}
