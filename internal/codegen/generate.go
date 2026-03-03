package codegen

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// Generate creates a Go project (main.go + go.mod) in the given output directory.
// If outputDir is empty, a temporary directory is created and its path returned.
// Returns the project directory path.
func Generate(ctx GenerateContext, outputDir string) (string, error) {
	return generateProject(mainTemplate, goModTemplate, ctx, outputDir)
}

// GenerateOpenAPI creates a Go project from an OpenAPI spec context.
// The generated CLI makes direct HTTP calls (no mcp-go dependency).
func GenerateOpenAPI(ctx OpenAPIGenerateContext, outputDir string) (string, error) {
	return generateProject(openAPIMainTemplate, openAPIGoModTemplate, ctx, outputDir)
}

func generateProject(mainTmpl, modTmpl interface{ Execute(w io.Writer, data any) error }, data any, outputDir string) (string, error) {
	if outputDir == "" {
		dir, err := os.MkdirTemp("", "clihub-*")
		if err != nil {
			return "", fmt.Errorf("create temp dir: %w", err)
		}
		outputDir = dir
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}

	// Write main.go
	mainPath := filepath.Join(outputDir, "main.go")
	mainFile, err := os.Create(mainPath)
	if err != nil {
		return outputDir, fmt.Errorf("create main.go: %w", err)
	}
	defer mainFile.Close()

	if err := mainTmpl.Execute(mainFile, data); err != nil {
		return outputDir, fmt.Errorf("render main.go template: %w", err)
	}

	// Write go.mod
	modPath := filepath.Join(outputDir, "go.mod")
	modFile, err := os.Create(modPath)
	if err != nil {
		return outputDir, fmt.Errorf("create go.mod: %w", err)
	}
	defer modFile.Close()

	if err := modTmpl.Execute(modFile, data); err != nil {
		return outputDir, fmt.Errorf("render go.mod template: %w", err)
	}

	// Run go mod tidy to download dependencies and generate go.sum
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = outputDir
	if out, err := tidyCmd.CombinedOutput(); err != nil {
		return outputDir, fmt.Errorf("go mod tidy failed: %s\n%s", err, string(out))
	}

	return outputDir, nil
}
