package nameutil

import (
	"fmt"
	"strings"
)

// SplitCommand splits a shell command string into tokens, respecting
// single quotes (literal), double quotes (with backslash escapes),
// and backslash escaping outside quotes.
func SplitCommand(input string) ([]string, error) {
	var tokens []string
	var current strings.Builder
	inSingle := false
	inDouble := false
	hasContent := false

	runes := []rune(input)
	i := 0

	for i < len(runes) {
		ch := runes[i]

		if inSingle {
			if ch == '\'' {
				inSingle = false
			} else {
				current.WriteRune(ch)
			}
			i++
			continue
		}

		if inDouble {
			if ch == '\\' && i+1 < len(runes) {
				next := runes[i+1]
				// Inside double quotes, backslash only escapes: " \ $
				if next == '"' || next == '\\' || next == '$' {
					current.WriteRune(next)
					i += 2
					continue
				}
				// Otherwise, backslash is literal.
				current.WriteRune(ch)
				i++
				continue
			}
			if ch == '"' {
				inDouble = false
			} else {
				current.WriteRune(ch)
			}
			i++
			continue
		}

		// Outside quotes.
		if ch == '\\' {
			if i+1 < len(runes) {
				current.WriteRune(runes[i+1])
				hasContent = true
				i += 2
				continue
			}
			// Trailing backslash — treat as literal.
			current.WriteRune(ch)
			hasContent = true
			i++
			continue
		}

		if ch == '\'' {
			inSingle = true
			hasContent = true
			i++
			continue
		}

		if ch == '"' {
			inDouble = true
			hasContent = true
			i++
			continue
		}

		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			if current.Len() > 0 || hasContent {
				tokens = append(tokens, current.String())
				current.Reset()
				hasContent = false
			}
			i++
			continue
		}

		current.WriteRune(ch)
		hasContent = true
		i++
	}

	if inSingle || inDouble {
		return nil, fmt.Errorf("unterminated quote in command string")
	}

	if current.Len() > 0 || hasContent {
		tokens = append(tokens, current.String())
	}

	return tokens, nil
}
