package auth

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateState returns a cryptographically random state string for CSRF protection.
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
