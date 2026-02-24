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
	Version int                         `json:"version"`
	Servers map[string]ServerCredential `json:"servers"`
}

// ServerCredential holds auth info for a single server.
// v2 schema uses AuthType to identify the provider; v1 used Type.
type ServerCredential struct {
	// v2 field — identifies which AuthProvider to instantiate
	AuthType string `json:"auth_type,omitempty"`
	// v1 field — kept for backwards compat during migration
	Type string `json:"type,omitempty"`

	// bearer_token / api_key
	Token      string `json:"token,omitempty"`
	HeaderName string `json:"header_name,omitempty"` // api_key header name

	// basic_auth
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`

	// oauth2
	AccessToken  string     `json:"access_token,omitempty"`
	RefreshToken string     `json:"refresh_token,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	ClientID     string     `json:"client_id,omitempty"`
	Scope        string     `json:"scope,omitempty"`

	// s2s_oauth2
	ClientSecret  string `json:"client_secret,omitempty"`
	TokenEndpoint string `json:"token_endpoint,omitempty"`

	// google_sa
	KeyFile string   `json:"key_file,omitempty"`
	Scopes  []string `json:"scopes,omitempty"`
}

// ResolveAuthType returns the effective auth type, handling both v1 and v2 schemas.
func (sc *ServerCredential) ResolveAuthType() string {
	if sc.AuthType != "" {
		return sc.AuthType
	}
	// v1 fallback
	switch sc.Type {
	case "bearer":
		return "bearer_token"
	case "oauth":
		return "oauth2"
	default:
		return sc.Type
	}
}

// LoadCredentials reads and parses a credentials file at the given path.
// If the file does not exist, it returns an empty CredentialsFile with
// Version=2 and an empty Servers map (not an error).
// v1 credentials are auto-migrated to v2 on load.
func LoadCredentials(path string) (*CredentialsFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &CredentialsFile{
				Version: 2,
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

	// Auto-migrate v1 → v2
	if creds.Version < 2 {
		migrateV1ToV2(&creds)
		// Save migrated file back (transparent upgrade)
		_ = SaveCredentials(path, &creds)
	}

	return &creds, nil
}

// migrateV1ToV2 converts v1 credential entries to v2 format.
func migrateV1ToV2(creds *CredentialsFile) {
	creds.Version = 2
	for url, sc := range creds.Servers {
		if sc.AuthType != "" {
			continue // Already v2
		}
		switch sc.Type {
		case "bearer":
			sc.AuthType = "bearer_token"
		case "oauth":
			sc.AuthType = "oauth2"
		default:
			if sc.Token != "" {
				sc.AuthType = "bearer_token"
			}
		}
		creds.Servers[url] = sc
	}
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
	authType := sc.ResolveAuthType()
	if authType == "oauth2" {
		return sc.AccessToken
	}
	return sc.Token
}

// GetOAuthCredential returns the full OAuth credential for the given server URL,
// or nil if none exists or the type is not oauth2.
func GetOAuthCredential(creds *CredentialsFile, serverURL string) *ServerCredential {
	if creds.Servers == nil {
		return nil
	}
	sc, ok := creds.Servers[serverURL]
	if !ok {
		return nil
	}
	if sc.ResolveAuthType() != "oauth2" {
		return nil
	}
	return &sc
}

// SetOAuthTokens stores OAuth tokens for the given server URL using v2 format.
func SetOAuthTokens(creds *CredentialsFile, serverURL, accessToken, refreshToken, clientID, scope string, expiresAt *time.Time) {
	if creds.Servers == nil {
		creds.Servers = make(map[string]ServerCredential)
	}
	if creds.Version < 2 {
		creds.Version = 2
	}
	creds.Servers[serverURL] = ServerCredential{
		AuthType:     "oauth2",
		Type:         "oauth", // kept for any v1 readers
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

// SetToken stores a bearer token for the given server URL using v2 format.
func SetToken(creds *CredentialsFile, serverURL, token string) {
	if creds.Servers == nil {
		creds.Servers = make(map[string]ServerCredential)
	}
	if creds.Version < 2 {
		creds.Version = 2
	}
	creds.Servers[serverURL] = ServerCredential{
		AuthType: "bearer_token",
		Type:     "bearer", // kept for any v1 readers
		Token:    token,
	}
}
