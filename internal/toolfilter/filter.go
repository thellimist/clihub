package toolfilter

import (
	"fmt"
	"strings"
)

// Tool represents an MCP tool for filtering purposes.
type Tool struct {
	Name        string
	Description string
}

// ParseToolList splits a comma-separated string into a deduplicated, trimmed
// list of tool names. Empty entries are removed and order is preserved (first
// occurrence wins on duplicates).
func ParseToolList(csv string) []string {
	if csv == "" {
		return nil
	}

	parts := strings.Split(csv, ",")
	seen := make(map[string]struct{})
	var result []string

	for _, p := range parts {
		name := strings.TrimSpace(p)
		if name == "" {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, name)
	}

	return result
}

// FilterTools applies include or exclude filtering to a list of tools.
//
// Rules:
//   - If both include and exclude are non-empty, an error is returned
//     (defense-in-depth; the flag layer should prevent this).
//   - Include mode: only tools whose Name appears in include are kept.
//     Any include name that does not match a tool produces an error with a
//     suggestion when one is close (Levenshtein <= 3).
//   - Exclude mode: tools whose Name appears in exclude are removed.
//     If that leaves zero tools, an error is returned.
//   - If both slices are empty, all tools are returned unchanged.
func FilterTools(tools []Tool, include, exclude []string) ([]Tool, error) {
	if len(include) > 0 && len(exclude) > 0 {
		return nil, fmt.Errorf("--include-tools and --exclude-tools cannot be used together")
	}

	// No filtering — pass through.
	if len(include) == 0 && len(exclude) == 0 {
		return tools, nil
	}

	// Build a name→tool index and a list of available names.
	byName := make(map[string]Tool, len(tools))
	available := make([]string, 0, len(tools))
	for _, t := range tools {
		byName[t.Name] = t
		available = append(available, t.Name)
	}

	// Include mode.
	if len(include) > 0 {
		var result []Tool
		for _, name := range include {
			t, ok := byName[name]
			if !ok {
				msg := fmt.Sprintf("tool '%s' not found on server. Available tools: %s",
					name, strings.Join(available, ", "))
				if suggestion := SuggestTool(name, available); suggestion != "" {
					msg += fmt.Sprintf(" Did you mean '%s'?", suggestion)
				}
				return nil, fmt.Errorf("%s", msg)
			}
			result = append(result, t)
		}
		return result, nil
	}

	// Exclude mode.
	excludeSet := make(map[string]struct{}, len(exclude))
	for _, name := range exclude {
		excludeSet[name] = struct{}{}
	}

	var result []Tool
	for _, t := range tools {
		if _, skip := excludeSet[t.Name]; !skip {
			result = append(result, t)
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("all tools excluded — nothing to generate")
	}

	return result, nil
}
