package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
)

func TestGenerateCodeVerifier_Length(t *testing.T) {
	v, err := GenerateCodeVerifier()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(v) != codeVerifierLength {
		t.Errorf("got length %d, want %d", len(v), codeVerifierLength)
	}
}

func TestGenerateCodeVerifier_CharacterSet(t *testing.T) {
	v, err := GenerateCodeVerifier()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range v {
		if !strings.ContainsRune(unreservedChars, c) {
			t.Errorf("invalid character %q in verifier", c)
		}
	}
}

func TestGenerateCodeVerifier_Uniqueness(t *testing.T) {
	v1, _ := GenerateCodeVerifier()
	v2, _ := GenerateCodeVerifier()
	if v1 == v2 {
		t.Error("two verifiers are identical")
	}
}

func TestGenerateCodeChallenge_KnownVector(t *testing.T) {
	// Known test: SHA256("test") = 9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08
	verifier := "test"
	got := GenerateCodeChallenge(verifier)
	h := sha256.Sum256([]byte(verifier))
	want := base64.RawURLEncoding.EncodeToString(h[:])
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGenerateCodeChallenge_NoPadding(t *testing.T) {
	v, _ := GenerateCodeVerifier()
	challenge := GenerateCodeChallenge(v)
	if strings.Contains(challenge, "=") {
		t.Error("challenge contains padding character '='")
	}
}
