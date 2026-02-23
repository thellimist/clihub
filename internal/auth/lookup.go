package auth

import (
	"os"
	"path/filepath"
)

// DefaultCredentialsPath returns the path to the credentials file.
// It checks the CLIHUB_CREDENTIALS_FILE env var first; if set, that
// path is returned. Otherwise it returns ~/.clihub/credentials.json.
func DefaultCredentialsPath() string {
	if p := os.Getenv("CLIHUB_CREDENTIALS_FILE"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".clihub", "credentials.json")
}

// LookupToken resolves an auth token using the following priority:
//  1. flagToken (from --auth-token flag) — returned if non-empty
//  2. CLIHUB_AUTH_TOKEN env var — returned if set
//  3. Credentials file at DefaultCredentialsPath() — returned if it
//     contains a token for serverURL
//
// Returns an empty string if no token is found.
func LookupToken(flagToken, serverURL string) string {
	// 1. Explicit flag
	if flagToken != "" {
		return flagToken
	}

	// 2. Environment variable
	if t := os.Getenv("CLIHUB_AUTH_TOKEN"); t != "" {
		return t
	}

	// 3. Credentials file
	path := DefaultCredentialsPath()
	if path == "" {
		return ""
	}
	creds, err := LoadCredentials(path)
	if err != nil {
		return ""
	}
	return GetToken(creds, serverURL)
}
