package nameutil

import (
	"strings"
	"unicode"
)

// Slugify converts a string into a URL/CLI-safe slug.
// It lowercases the input, replaces non-alphanumeric characters with dashes,
// collapses consecutive dashes, and trims leading/trailing dashes.
func Slugify(s string) string {
	s = strings.ToLower(s)

	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}

	// Collapse consecutive dashes.
	result := b.String()
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}

	// Trim leading/trailing dashes.
	result = strings.Trim(result, "-")

	return result
}
