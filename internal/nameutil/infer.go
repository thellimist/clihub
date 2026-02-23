package nameutil

import (
	"net"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

// InferName infers a short CLI-friendly name from a URL or command string.
func InferName(urlOrCommand string, isURL bool) string {
	if isURL {
		return inferFromURL(urlOrCommand)
	}
	return inferFromCommand(urlOrCommand)
}

// genericPrefixes are hostname segments to strip from the left.
var genericPrefixes = map[string]bool{
	"www": true,
	"api": true,
	"mcp": true,
}

// knownTLDs are hostname segments to strip from the right.
var knownTLDs = map[string]bool{
	"com": true,
	"io":  true,
	"app": true,
	"dev": true,
	"org": true,
	"net": true,
}

// versionSuffix matches @latest or @1.2.3 style version suffixes.
var versionSuffix = regexp.MustCompile(`@[^/]*$`)

// inferFromURL extracts a meaningful name from a URL's hostname.
func inferFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return ""
	}

	// Strip port if present.
	hostname := u.Hostname()

	// If hostname is localhost or an IP address, skip straight to path fallback.
	isLocal := hostname == "localhost" || net.ParseIP(hostname) != nil

	if !isLocal {
		parts := strings.Split(hostname, ".")

		// Strip generic prefixes from the left.
		for len(parts) > 0 && genericPrefixes[parts[0]] {
			parts = parts[1:]
		}

		// Strip known TLD suffixes from the right.
		for len(parts) > 0 && knownTLDs[parts[len(parts)-1]] {
			parts = parts[:len(parts)-1]
		}

		if len(parts) > 0 {
			// Take the last remaining segment (closest to TLD).
			name := parts[len(parts)-1]
			result := Slugify(name)
			if result != "" {
				return result
			}
		}
	}

	// Fallback: first non-empty path segment.
	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
	for _, p := range pathParts {
		slug := Slugify(p)
		if slug != "" {
			return slug
		}
	}

	return ""
}

// inferFromCommand extracts a name from a shell command string.
func inferFromCommand(command string) string {
	tokens, err := SplitCommand(command)
	if err != nil || len(tokens) == 0 {
		return ""
	}

	// Find the package/tool token: scan from the end for a token that looks
	// like a package name (contains `/`, starts with `@`, or doesn't start with `-`).
	token := ""
	for i := len(tokens) - 1; i >= 0; i-- {
		t := tokens[i]
		if strings.Contains(t, "/") || strings.HasPrefix(t, "@") {
			token = t
			break
		}
		if !strings.HasPrefix(t, "-") {
			token = t
			break
		}
	}

	if token == "" {
		return ""
	}

	// Handle scoped packages: @org/package-name
	name := token
	if strings.HasPrefix(name, "@") && strings.Contains(name, "/") {
		idx := strings.Index(name, "/")
		name = name[idx+1:]
	}

	// Strip version suffixes like @latest or @1.2.3.
	name = versionSuffix.ReplaceAllString(name, "")

	// Strip file extensions (e.g., .js, .py, .ts).
	ext := filepath.Ext(name)
	if ext != "" {
		name = strings.TrimSuffix(name, ext)
	}

	// Normalize underscores to dashes so prefix stripping works uniformly.
	name = strings.ReplaceAll(name, "_", "-")

	// Strip known prefixes in order (first match wins).
	prefixes := []string{"mcp-server-", "server-", "mcp-"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) {
			name = strings.TrimPrefix(name, prefix)
			break
		}
	}

	return Slugify(name)
}
