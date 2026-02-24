package auth

import "testing"

func TestGenerateState_Length(t *testing.T) {
	s, err := GenerateState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 32 bytes → 64 hex chars
	if len(s) != 64 {
		t.Errorf("got length %d, want 64", len(s))
	}
}

func TestGenerateState_Uniqueness(t *testing.T) {
	s1, _ := GenerateState()
	s2, _ := GenerateState()
	if s1 == s2 {
		t.Error("two states are identical")
	}
}
