package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

const codeVerifierLength = 64

// unreserved characters per RFC 7636 appendix B
const unreservedChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"

// GenerateCodeVerifier returns a cryptographically random string suitable for
// use as a PKCE code_verifier per RFC 7636.
func GenerateCodeVerifier() (string, error) {
	b := make([]byte, codeVerifierLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	result := make([]byte, codeVerifierLength)
	for i := range b {
		result[i] = unreservedChars[int(b[i])%len(unreservedChars)]
	}
	return string(result), nil
}

// GenerateCodeChallenge computes the S256 code challenge from a verifier.
// Returns base64url(SHA256(verifier)) with no padding per RFC 7636 appendix B.
func GenerateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}
