package auth

import (
	"testing"
)

func TestParseWWWAuthenticate_Empty(t *testing.T) {
	challenges := ParseWWWAuthenticate("")
	if len(challenges) != 0 {
		t.Errorf("expected 0 challenges, got %d", len(challenges))
	}
}

func TestParseWWWAuthenticate_BearerSimple(t *testing.T) {
	challenges := ParseWWWAuthenticate(`Bearer realm="example.com"`)
	if len(challenges) != 1 {
		t.Fatalf("expected 1 challenge, got %d", len(challenges))
	}
	if challenges[0].Scheme != "Bearer" {
		t.Errorf("scheme = %q, want Bearer", challenges[0].Scheme)
	}
	if challenges[0].Realm != "example.com" {
		t.Errorf("realm = %q, want example.com", challenges[0].Realm)
	}
}

func TestParseWWWAuthenticate_BearerWithScope(t *testing.T) {
	challenges := ParseWWWAuthenticate(`Bearer realm="example.com", scope="read write"`)
	if len(challenges) != 1 {
		t.Fatalf("expected 1 challenge, got %d", len(challenges))
	}
	if challenges[0].Scope != "read write" {
		t.Errorf("scope = %q, want \"read write\"", challenges[0].Scope)
	}
}

func TestParseWWWAuthenticate_BearerWithResourceMetadata(t *testing.T) {
	challenges := ParseWWWAuthenticate(`Bearer realm="mcp.example.com", resource_metadata="https://mcp.example.com/.well-known/oauth-protected-resource"`)
	if len(challenges) != 1 {
		t.Fatalf("expected 1 challenge, got %d", len(challenges))
	}
	if challenges[0].ResourceMetadata != "https://mcp.example.com/.well-known/oauth-protected-resource" {
		t.Errorf("resource_metadata = %q", challenges[0].ResourceMetadata)
	}
}

func TestParseWWWAuthenticate_BearerWithError(t *testing.T) {
	challenges := ParseWWWAuthenticate(`Bearer error="invalid_token", error_description="The access token expired"`)
	if len(challenges) != 1 {
		t.Fatalf("expected 1 challenge, got %d", len(challenges))
	}
	if challenges[0].Error != "invalid_token" {
		t.Errorf("error = %q, want invalid_token", challenges[0].Error)
	}
	if challenges[0].ErrorDescription != "The access token expired" {
		t.Errorf("error_description = %q", challenges[0].ErrorDescription)
	}
}

func TestParseWWWAuthenticate_SchemeOnly(t *testing.T) {
	challenges := ParseWWWAuthenticate("Bearer")
	if len(challenges) != 1 {
		t.Fatalf("expected 1 challenge, got %d", len(challenges))
	}
	if challenges[0].Scheme != "Bearer" {
		t.Errorf("scheme = %q, want Bearer", challenges[0].Scheme)
	}
}

func TestParseWWWAuthenticate_MultipleChallenges(t *testing.T) {
	challenges := ParseWWWAuthenticate(`Bearer realm="example.com", Basic realm="fallback"`)
	if len(challenges) < 2 {
		t.Fatalf("expected at least 2 challenges, got %d", len(challenges))
	}
	if challenges[0].Scheme != "Bearer" {
		t.Errorf("first scheme = %q, want Bearer", challenges[0].Scheme)
	}
	if challenges[1].Scheme != "Basic" {
		t.Errorf("second scheme = %q, want Basic", challenges[1].Scheme)
	}
}

func TestParseWWWAuthenticate_UnquotedValues(t *testing.T) {
	challenges := ParseWWWAuthenticate(`Bearer realm=example.com, scope=read`)
	if len(challenges) != 1 {
		t.Fatalf("expected 1 challenge, got %d", len(challenges))
	}
	if challenges[0].Realm != "example.com" {
		t.Errorf("realm = %q, want example.com", challenges[0].Realm)
	}
	if challenges[0].Scope != "read" {
		t.Errorf("scope = %q, want read", challenges[0].Scope)
	}
}

func TestParseWWWAuthenticate_AllParams(t *testing.T) {
	header := `Bearer realm="mcp.example.com", scope="mcp:read mcp:write", resource_metadata="https://auth.example.com/.well-known/oauth-protected-resource"`
	challenges := ParseWWWAuthenticate(header)
	if len(challenges) != 1 {
		t.Fatalf("expected 1 challenge, got %d", len(challenges))
	}
	ch := challenges[0]
	if ch.Scheme != "Bearer" {
		t.Errorf("scheme = %q", ch.Scheme)
	}
	if ch.Realm != "mcp.example.com" {
		t.Errorf("realm = %q", ch.Realm)
	}
	if ch.Scope != "mcp:read mcp:write" {
		t.Errorf("scope = %q", ch.Scope)
	}
	if ch.ResourceMetadata != "https://auth.example.com/.well-known/oauth-protected-resource" {
		t.Errorf("resource_metadata = %q", ch.ResourceMetadata)
	}
}

func TestFindBearerChallenge_Found(t *testing.T) {
	challenges := []AuthChallenge{
		{Scheme: "Basic", Realm: "test"},
		{Scheme: "Bearer", Realm: "mcp"},
	}
	ch := FindBearerChallenge(challenges)
	if ch == nil {
		t.Fatal("expected Bearer challenge")
	}
	if ch.Realm != "mcp" {
		t.Errorf("realm = %q, want mcp", ch.Realm)
	}
}

func TestFindBearerChallenge_NotFound(t *testing.T) {
	challenges := []AuthChallenge{
		{Scheme: "Basic", Realm: "test"},
	}
	ch := FindBearerChallenge(challenges)
	if ch != nil {
		t.Error("expected nil for non-Bearer challenges")
	}
}

func TestFindBearerChallenge_CaseInsensitive(t *testing.T) {
	challenges := []AuthChallenge{
		{Scheme: "bearer", Realm: "test"},
	}
	ch := FindBearerChallenge(challenges)
	if ch == nil {
		t.Fatal("expected Bearer challenge (case-insensitive)")
	}
}

func TestParseWWWAuthenticate_EscapedQuotes(t *testing.T) {
	challenges := ParseWWWAuthenticate(`Bearer realm="example \"quoted\""`)
	if len(challenges) != 1 {
		t.Fatalf("expected 1 challenge, got %d", len(challenges))
	}
	if challenges[0].Realm != `example "quoted"` {
		t.Errorf("realm = %q, want %q", challenges[0].Realm, `example "quoted"`)
	}
}
