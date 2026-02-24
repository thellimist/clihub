package auth

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// CredentialsFile represents the ~/.clihub/credentials.json file.
type CredentialsFile struct {
	Version int                          `json:"version"`
	Servers map[string]ServerCredential  `json:"servers"`
}

// ServerCredential holds auth info for a single server.
type ServerCredential struct {
	Type         string     `json:"type"`                    // "bearer" or "oauth"
	Token        string     `json:"token,omitempty"`         // For bearer type
	AccessToken  string     `json:"access_token,omitempty"`  // For oauth type
	RefreshToken string     `json:"refresh_token,omitempty"` // For oauth type
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`    // For oauth type
	ClientID     string     `json:"client_id,omitempty"`     // For oauth type
	Scope        string     `json:"scope,omitempty"`         // For oauth type
}

// LoadCredentials reads and parses a credentials file at the given path.
// If the file does not exist, it returns an empty CredentialsFile with
// Version=1 and an empty Servers map (not an error).
func LoadCredentials(path string) (*CredentialsFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &CredentialsFile{
				Version: 1,
				Servers: make(map[string]ServerCredential),
			}, nil
		}
		return nil, err
	}

	var creds CredentialsFile
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}
	if creds.Servers == nil {
		creds.Servers = make(map[string]ServerCredential)
	}
	return &creds, nil
}

// SaveCredentials writes the credentials to the given path. It creates
// the parent directory with 0700 permissions if needed, and writes the
// file with 0600 permissions (owner-only read/write).
func SaveCredentials(path string, creds *CredentialsFile) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// GetToken returns the token for the given server URL, or an empty
// string if the server is not found in the credentials.
// For OAuth credentials, returns the access_token.
func GetToken(creds *CredentialsFile, serverURL string) string {
	if creds.Servers == nil {
		return ""
	}
	sc, ok := creds.Servers[serverURL]
	if !ok {
		return ""
	}
	if sc.Type == "oauth" {
		return sc.AccessToken
	}
	return sc.Token
}

// GetOAuthCredential returns the full OAuth credential for the given server URL,
// or nil if none exists or the type is not "oauth".
func GetOAuthCredential(creds *CredentialsFile, serverURL string) *ServerCredential {
	if creds.Servers == nil {
		return nil
	}
	sc, ok := creds.Servers[serverURL]
	if !ok || sc.Type != "oauth" {
		return nil
	}
	return &sc
}

// SetOAuthTokens stores OAuth tokens for the given server URL.
func SetOAuthTokens(creds *CredentialsFile, serverURL, accessToken, refreshToken, clientID, scope string, expiresAt *time.Time) {
	if creds.Servers == nil {
		creds.Servers = make(map[string]ServerCredential)
	}
	creds.Servers[serverURL] = ServerCredential{
		Type:         "oauth",
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		ClientID:     clientID,
		Scope:        scope,
	}
}

// IsTokenExpired returns true if the credential has an expires_at in the past.
// Returns false if there is no expiry set.
func IsTokenExpired(sc ServerCredential) bool {
	if sc.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*sc.ExpiresAt)
}

// SetToken stores a bearer token for the given server URL. It
// initializes the Servers map if it is nil.
func SetToken(creds *CredentialsFile, serverURL, token string) {
	if creds.Servers == nil {
		creds.Servers = make(map[string]ServerCredential)
	}
	creds.Servers[serverURL] = ServerCredential{
		Token: token,
		Type:  "bearer",
	}
}
