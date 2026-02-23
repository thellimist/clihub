package gocheck

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var versionRe = regexp.MustCompile(`go(\d+)\.(\d+)`)

// Check verifies that the Go toolchain is installed and meets the minimum
// version requirement (>= 1.22). Returns the version string on success.
func Check() (string, error) {
	out, err := exec.Command("go", "version").Output()
	if err != nil {
		return "", fmt.Errorf("Go toolchain not found. Install Go >= 1.22 from https://go.dev/dl/")
	}

	version := strings.TrimSpace(string(out))
	matches := versionRe.FindStringSubmatch(version)
	if len(matches) < 3 {
		return version, nil // can't parse, assume ok
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])

	if major < 1 || (major == 1 && minor < 22) {
		return "", fmt.Errorf("Go toolchain version %d.%d is too old. Install Go >= 1.22 from https://go.dev/dl/", major, minor)
	}

	return version, nil
}
