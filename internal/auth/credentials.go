package auth

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// CredentialsFile represents the ~/.clihub/credentials.json file.
type CredentialsFile struct {
	Version int                          `json:"version"`
	Servers map[string]ServerCredential  `json:"servers"`
}

// ServerCredential holds auth info for a single server.
type ServerCredential struct {
	Token string `json:"token"`
	Type  string `json:"type"` // Always "bearer" for now
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
func GetToken(creds *CredentialsFile, serverURL string) string {
	if creds.Servers == nil {
		return ""
	}
	sc, ok := creds.Servers[serverURL]
	if !ok {
		return ""
	}
	return sc.Token
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
