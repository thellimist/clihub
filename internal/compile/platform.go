package compile

import (
	"fmt"
	"strings"
)

// Platform represents a GOOS/GOARCH target.
type Platform struct {
	GOOS   string
	GOARCH string
}

func (p Platform) String() string {
	return p.GOOS + "/" + p.GOARCH
}

// ValidPlatforms are the 6 standard cross-compilation targets.
var ValidPlatforms = []Platform{
	{"linux", "amd64"},
	{"linux", "arm64"},
	{"darwin", "amd64"},
	{"darwin", "arm64"},
	{"windows", "amd64"},
	{"windows", "arm64"},
}

var validPlatformSet map[string]bool

func init() {
	validPlatformSet = make(map[string]bool, len(ValidPlatforms))
	for _, p := range ValidPlatforms {
		validPlatformSet[p.String()] = true
	}
}

// ParsePlatforms parses the --platform flag value into a list of platforms.
// Handles "all" shorthand and validates each platform.
func ParsePlatforms(platformFlag string) ([]Platform, error) {
	platformFlag = strings.TrimSpace(platformFlag)
	if platformFlag == "all" {
		result := make([]Platform, len(ValidPlatforms))
		copy(result, ValidPlatforms)
		return result, nil
	}

	parts := strings.Split(platformFlag, ",")
	var platforms []Platform
	seen := make(map[string]bool)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if !validPlatformSet[part] {
			return nil, fmt.Errorf("invalid platform '%s'. Valid targets: %s", part, validTargetsList())
		}

		if seen[part] {
			continue
		}
		seen[part] = true

		s := strings.SplitN(part, "/", 2)
		platforms = append(platforms, Platform{GOOS: s[0], GOARCH: s[1]})
	}

	if len(platforms) == 0 {
		return nil, fmt.Errorf("no platforms specified")
	}

	return platforms, nil
}

func validTargetsList() string {
	names := make([]string, len(ValidPlatforms))
	for i, p := range ValidPlatforms {
		names[i] = p.String()
	}
	return strings.Join(names, ", ")
}

// BinaryName returns the output binary name for a platform.
// Single platform: just <name> (or <name>.exe on windows).
// Multi-platform: <name>-<os>-<arch> (with .exe for windows).
func BinaryName(name string, p Platform, multiPlatform bool) string {
	if multiPlatform {
		result := fmt.Sprintf("%s-%s-%s", name, p.GOOS, p.GOARCH)
		if p.GOOS == "windows" {
			result += ".exe"
		}
		return result
	}

	if p.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}
