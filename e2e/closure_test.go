package e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thellimist/clihub/internal/closure"
	"github.com/thellimist/clihub/internal/codegen"
	"github.com/thellimist/clihub/internal/schema"
)

// TestClosureParamsPassedToServer is an end-to-end test that verifies closure
// params are actually sent to the MCP server at runtime. It:
//  1. Compiles a tiny echo-params MCP server
//  2. Generates a CLI with closure config via codegen.Generate
//  3. Compiles the generated CLI
//  4. Runs the generated CLI and asserts the server received correct params
func TestClosureParamsPassedToServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	// Build the echo-params test server
	testServerBin := filepath.Join(t.TempDir(), "echo-params-server")
	buildServer := exec.Command("go", "build", "-o", testServerBin, "./testserver")
	buildServer.Dir = filepath.Join(projectRoot(t), "e2e")
	if out, err := buildServer.CombinedOutput(); err != nil {
		t.Fatalf("build test server: %v\n%s", err, out)
	}

	// Helper to generate+compile a CLI with the given closure config.
	buildCLI := func(t *testing.T, name string, cfg *closure.Config) string {
		t.Helper()
		ctx := codegen.GenerateContext{
			CLIName:       name,
			StdioCommand:  testServerBin,
			ClihubVersion: "test",
			IsHTTP:        false,
			ClosureConfig: cfg,
			Tools: []codegen.ToolDef{
				{
					Name: "echo_params", CommandName: "echo-params",
					Description: "Echoes params",
					Options: []schema.ToolOption{
						{PropertyName: "org_id", FlagName: "org-id", Description: "Organization ID", GoType: "string"},
						{PropertyName: "project_id", FlagName: "project-id", Description: "Project ID", GoType: "string"},
						{PropertyName: "query", FlagName: "query", Description: "Search query", GoType: "string"},
					},
				},
				{
					Name: "create_item", CommandName: "create-item",
					Description: "Creates an item",
					Options: []schema.ToolOption{
						{PropertyName: "org_id", FlagName: "org-id", Description: "Organization ID", GoType: "string"},
						{PropertyName: "project_id", FlagName: "project-id", Description: "Project ID", GoType: "string"},
						{PropertyName: "title", FlagName: "title", Description: "Item title", GoType: "string"},
					},
				},
				{
					Name: "list_items", CommandName: "list-items",
					Description: "Lists items",
					Options: []schema.ToolOption{
						{PropertyName: "org_id", FlagName: "org-id", Description: "Organization ID", GoType: "string"},
						{PropertyName: "query", FlagName: "query", Description: "Search query", GoType: "string"},
					},
				},
			},
		}

		dir := t.TempDir()
		projectDir, err := codegen.Generate(ctx, dir)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		binPath := filepath.Join(t.TempDir(), name)
		build := exec.Command("go", "build", "-o", binPath, ".")
		build.Dir = projectDir
		if out, err := build.CombinedOutput(); err != nil {
			mainGo, _ := os.ReadFile(filepath.Join(projectDir, "main.go"))
			t.Fatalf("build generated CLI: %v\n%s\n\nmain.go:\n%s", err, out, mainGo)
		}
		return binPath
	}

	// runCLI executes the generated CLI and returns the JSON params echoed by the server.
	runCLI := func(t *testing.T, bin string, args ...string) map[string]any {
		t.Helper()
		cmd := exec.Command(bin, args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("run CLI %v: %v\n%s", args, err, out)
		}
		// The output is JSON text from the echo server
		text := strings.TrimSpace(string(out))
		var params map[string]any
		if err := json.Unmarshal([]byte(text), &params); err != nil {
			t.Fatalf("parse server response as JSON: %v\nraw output: %q", err, text)
		}
		return params
	}

	t.Run("hidden_mode_global_param", func(t *testing.T) {
		cfg := &closure.Config{
			Mode: closure.ModeHidden,
			Global: closure.GlobalConfig{
				Params: map[string]any{"org_id": "acme-corp"},
			},
		}
		bin := buildCLI(t, "hidden-global", cfg)

		// Call echo_params without any flags — org_id should be injected.
		params := runCLI(t, bin, "echo-params", "--query", "test")
		if params["org_id"] != "acme-corp" {
			t.Errorf("expected org_id=acme-corp, got %v", params["org_id"])
		}
		if params["query"] != "test" {
			t.Errorf("expected query=test, got %v", params["query"])
		}
	})

	t.Run("hidden_mode_tool_specific_param", func(t *testing.T) {
		cfg := &closure.Config{
			Mode: closure.ModeHidden,
			Global: closure.GlobalConfig{
				Params: map[string]any{"org_id": "acme-corp"},
			},
			Tools: map[string]closure.ToolConfig{
				"create_item": {Params: map[string]any{"project_id": "PROJ-123"}},
			},
		}
		bin := buildCLI(t, "hidden-tool", cfg)

		// create_item should have both org_id (global) and project_id (tool-specific)
		params := runCLI(t, bin, "create-item", "--title", "hello")
		if params["org_id"] != "acme-corp" {
			t.Errorf("expected org_id=acme-corp, got %v", params["org_id"])
		}
		if params["project_id"] != "PROJ-123" {
			t.Errorf("expected project_id=PROJ-123, got %v", params["project_id"])
		}
		if params["title"] != "hello" {
			t.Errorf("expected title=hello, got %v", params["title"])
		}
	})

	t.Run("hidden_mode_tool_isolation", func(t *testing.T) {
		cfg := &closure.Config{
			Mode: closure.ModeHidden,
			Global: closure.GlobalConfig{
				Params: map[string]any{"org_id": "acme-corp"},
			},
			Tools: map[string]closure.ToolConfig{
				"create_item": {Params: map[string]any{"project_id": "PROJ-123"}},
			},
		}
		bin := buildCLI(t, "hidden-isolation", cfg)

		// list_items should NOT have project_id (only set for create_item)
		params := runCLI(t, bin, "list-items", "--query", "search")
		if params["org_id"] != "acme-corp" {
			t.Errorf("expected org_id=acme-corp, got %v", params["org_id"])
		}
		if _, exists := params["project_id"]; exists {
			t.Errorf("project_id should not leak to list_items, got %v", params["project_id"])
		}
	})

	t.Run("hidden_mode_injected_without_user_flags", func(t *testing.T) {
		cfg := &closure.Config{
			Mode: closure.ModeHidden,
			Global: closure.GlobalConfig{
				Params: map[string]any{"org_id": "acme-corp"},
			},
		}
		bin := buildCLI(t, "hidden-no-flags", cfg)

		// Call with NO flags at all — org_id should still be injected.
		params := runCLI(t, bin, "echo-params")
		if params["org_id"] != "acme-corp" {
			t.Errorf("expected org_id=acme-corp even without user flags, got %v", params["org_id"])
		}
	})

	t.Run("default_mode_override", func(t *testing.T) {
		cfg := &closure.Config{
			Mode: closure.ModeDefault,
			Global: closure.GlobalConfig{
				Params: map[string]any{"org_id": "acme-corp"},
			},
			Tools: map[string]closure.ToolConfig{
				"create_item": {Params: map[string]any{"project_id": "PROJ-123"}},
			},
		}
		bin := buildCLI(t, "default-override", cfg)

		// User overrides org_id via flag — should use user value, not closure default.
		params := runCLI(t, bin, "create-item", "--org-id", "user-org", "--title", "hello")
		if params["org_id"] != "user-org" {
			t.Errorf("expected org_id=user-org (user override), got %v", params["org_id"])
		}
		// project_id should still come from closure default since user didn't override.
		if params["project_id"] != "PROJ-123" {
			t.Errorf("expected project_id=PROJ-123 (closure default), got %v", params["project_id"])
		}
	})

	t.Run("default_mode_uses_closure_defaults", func(t *testing.T) {
		cfg := &closure.Config{
			Mode: closure.ModeDefault,
			Global: closure.GlobalConfig{
				Params: map[string]any{"org_id": "acme-corp"},
			},
		}
		bin := buildCLI(t, "default-defaults", cfg)

		// Call without overriding — closure defaults should be sent.
		params := runCLI(t, bin, "echo-params", "--query", "test")
		if params["org_id"] != "acme-corp" {
			t.Errorf("expected org_id=acme-corp (closure default), got %v", params["org_id"])
		}
	})
}

// projectRoot returns the root of the clihub project.
func projectRoot(t *testing.T) string {
	t.Helper()
	// Walk up from this test file to find go.mod
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (go.mod)")
		}
		dir = parent
	}
}
