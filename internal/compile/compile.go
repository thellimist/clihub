package compile

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// Compile runs go build for the given project directory and target platform.
// Returns the path to the compiled binary.
func Compile(projectDir, outputDir, name string, p Platform, multiPlatform bool) (string, error) {
	binaryName := BinaryName(name, p, multiPlatform)

	// Make output path absolute so go build writes to the right place
	absOutput, err := filepath.Abs(outputDir)
	if err != nil {
		return "", fmt.Errorf("resolve output dir: %w", err)
	}

	binaryPath := filepath.Join(absOutput, binaryName)
	if err := os.MkdirAll(absOutput, 0755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = projectDir
	cmd.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		"GOOS="+p.GOOS,
		"GOARCH="+p.GOARCH,
	)

	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("go build failed for %s: %s", p, string(out))
	}

	return binaryPath, nil
}

// SmokeTest runs the compiled binary with --help and verifies exit code 0.
func SmokeTest(binaryPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("generated binary failed smoke test (timed out after 15s)")
		}
		exitCode := 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		return fmt.Errorf("generated binary failed smoke test (exit code %d): %s", exitCode, string(out))
	}

	return nil
}

// CurrentPlatform returns the current GOOS and GOARCH.
func CurrentPlatform() (string, string) {
	return runtime.GOOS, runtime.GOARCH
}
