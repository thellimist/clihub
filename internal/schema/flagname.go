package schema

import "strings"

// ToFlagName converts a camelCase or PascalCase property name to a kebab-case
// CLI flag name.
//
// Rules:
//   - Insert "-" before an uppercase letter that follows a lowercase letter.
//   - Insert "-" between consecutive uppercase letters and a following lowercase
//     letter (for acronyms like "HTTP" in "getHTTPResponse" → "get-http-response").
//   - Lowercase everything.
//   - Replace underscores with dashes.
func ToFlagName(propertyName string) string {
	if propertyName == "" {
		return ""
	}

	var b strings.Builder
	runes := []rune(propertyName)
	for i, r := range runes {
		if isUpper(r) {
			if i > 0 {
				prev := runes[i-1]
				if isLower(prev) {
					// camelCase boundary: aB → a-b
					b.WriteRune('-')
				} else if isUpper(prev) && i+1 < len(runes) && isLower(runes[i+1]) {
					// Acronym boundary: ABc → A-Bc (e.g., "HTMLParser" → "HTML-Parser")
					b.WriteRune('-')
				}
			}
			b.WriteRune(toLower(r))
		} else if r == '_' {
			b.WriteRune('-')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isUpper(r rune) bool { return r >= 'A' && r <= 'Z' }
func isLower(r rune) bool { return r >= 'a' && r <= 'z' }
func toLower(r rune) rune {
	if isUpper(r) {
		return r + ('a' - 'A')
	}
	return r
}
