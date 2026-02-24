package auth

import (
	"strings"
)

// AuthChallenge represents a parsed WWW-Authenticate challenge (RFC 6750).
type AuthChallenge struct {
	Scheme            string // e.g., "Bearer", "Basic"
	Realm             string
	ResourceMetadata  string // resource_metadata URL from MCP servers (RFC 9728)
	Scope             string
	Error             string
	ErrorDescription  string
}

// ParseWWWAuthenticate parses a WWW-Authenticate header value into one or more
// AuthChallenge structs. It handles Bearer challenges with quoted and unquoted
// parameter values per RFC 6750.
func ParseWWWAuthenticate(header string) []AuthChallenge {
	header = strings.TrimSpace(header)
	if header == "" {
		return nil
	}

	var challenges []AuthChallenge

	// Split into individual challenges. Each challenge starts with a scheme name
	// (a token not containing '='). We need to be careful because parameter values
	// can also look like tokens.
	//
	// Strategy: scan for scheme names. A scheme is a word followed by either
	// space+params or end-of-string, and it's NOT part of a key=value pair.
	remaining := header
	for remaining != "" {
		remaining = strings.TrimSpace(remaining)
		if remaining == "" {
			break
		}

		// Extract scheme (first token before space or end)
		scheme, rest := extractToken(remaining)
		if scheme == "" {
			break
		}

		ch := AuthChallenge{Scheme: scheme}
		rest = strings.TrimSpace(rest)

		// Parse parameters (key=value pairs, comma-separated)
		remaining = parseParams(rest, &ch)

		challenges = append(challenges, ch)
	}

	return challenges
}

// FindBearerChallenge returns the first Bearer challenge from the parsed challenges,
// or nil if none exists.
func FindBearerChallenge(challenges []AuthChallenge) *AuthChallenge {
	for i := range challenges {
		if strings.EqualFold(challenges[i].Scheme, "Bearer") {
			return &challenges[i]
		}
	}
	return nil
}

// extractToken extracts the next token (non-whitespace, non-comma, non-equals sequence)
// from the input and returns (token, rest).
func extractToken(s string) (string, string) {
	s = strings.TrimSpace(s)
	i := 0
	for i < len(s) && s[i] != ' ' && s[i] != '\t' && s[i] != ',' && s[i] != '=' {
		i++
	}
	if i == 0 {
		return "", s
	}
	return s[:i], s[i:]
}

// parseParams parses comma-separated key=value parameters from a challenge.
// It returns the remaining unparsed string (which may start with a new scheme).
func parseParams(s string, ch *AuthChallenge) string {
	for {
		s = strings.TrimSpace(s)
		if s == "" {
			return ""
		}

		// Try to extract key=value
		key, rest := extractToken(s)
		if key == "" {
			return s
		}

		rest = strings.TrimSpace(rest)

		// If there's no '=', this is a new scheme name — return for next challenge
		if rest == "" || rest[0] != '=' {
			return s // key is actually the next scheme
		}

		// Skip '='
		rest = rest[1:]
		rest = strings.TrimSpace(rest)

		// Extract value (quoted or unquoted)
		var value string
		if len(rest) > 0 && rest[0] == '"' {
			value, rest = extractQuotedString(rest)
		} else {
			value, rest = extractParamValue(rest)
		}

		setParam(ch, strings.ToLower(key), value)

		// Skip optional comma
		rest = strings.TrimSpace(rest)
		if len(rest) > 0 && rest[0] == ',' {
			rest = rest[1:]
		}
		s = rest
	}
}

// extractQuotedString extracts a double-quoted string value, handling escaped quotes.
func extractQuotedString(s string) (string, string) {
	if len(s) == 0 || s[0] != '"' {
		return "", s
	}

	var b strings.Builder
	i := 1
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			b.WriteByte(s[i+1])
			i += 2
			continue
		}
		if s[i] == '"' {
			return b.String(), s[i+1:]
		}
		b.WriteByte(s[i])
		i++
	}
	// Unterminated quote — return what we have
	return b.String(), ""
}

// extractParamValue extracts an unquoted parameter value (stops at comma or space).
func extractParamValue(s string) (string, string) {
	i := 0
	for i < len(s) && s[i] != ',' && s[i] != ' ' && s[i] != '\t' {
		i++
	}
	return s[:i], s[i:]
}

// setParam assigns a parsed parameter to the appropriate AuthChallenge field.
func setParam(ch *AuthChallenge, key, value string) {
	switch key {
	case "realm":
		ch.Realm = value
	case "scope":
		ch.Scope = value
	case "resource_metadata":
		ch.ResourceMetadata = value
	case "error":
		ch.Error = value
	case "error_description":
		ch.ErrorDescription = value
	}
}
