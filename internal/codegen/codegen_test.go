package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thellimist/clihub/internal/closure"
	"github.com/thellimist/clihub/internal/schema"
)

func TestGenerateProducesValidGo(t *testing.T) {
	ctx := GenerateContext{
		CLIName:       "testcli",
		ServerURL:     "https://example.com/mcp",
		ClihubVersion: "test",
		IsHTTP:        true,
		Tools: []ToolDef{
			{
				Name:        "list_items",
				CommandName: "list-items",
				Description: "List all items",
				Options: []schema.ToolOption{
					{
						PropertyName: "query",
						FlagName:     "query",
						Description:  "Search query",
						Required:     true,
						GoType:       "string",
					},
					{
						PropertyName: "limit",
						FlagName:     "limit",
						Description:  "Max results",
						GoType:       "int",
						DefaultValue: float64(10),
					},
					{
						PropertyName: "verbose",
						FlagName:     "verbose",
						Description:  "Show details",
						GoType:       "bool",
					},
					{
						PropertyName: "tags",
						FlagName:     "tags",
						Description:  "Filter tags",
						GoType:       "[]string",
					},
					{
						PropertyName: "status",
						FlagName:     "status",
						Description:  "Status filter",
						GoType:       "string",
						EnumValues:   []string{"open", "closed", "merged"},
					},
				},
			},
			{
				Name:        "create_item",
				CommandName: "create-item",
				Description: "Create a new item",
				Options: []schema.ToolOption{
					{
						PropertyName: "title",
						FlagName:     "title",
						Description:  "Item title",
						Required:     true,
						GoType:       "string",
					},
				},
			},
		},
	}

	dir := t.TempDir()
	projectDir, err := Generate(ctx, dir)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify files exist
	for _, f := range []string{"main.go", "go.mod", "go.sum"} {
		path := filepath.Join(projectDir, f)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %s to exist: %v", f, err)
		}
	}

	// Keep generated go.mod Go version aligned with dependency minimum.
	modBytes, err := os.ReadFile(filepath.Join(projectDir, "go.mod"))
	if err != nil {
		t.Fatalf("read generated go.mod: %v", err)
	}
	if !strings.Contains(string(modBytes), "\ngo 1.24") {
		t.Fatalf("generated go.mod must declare go 1.24.x, got:\n%s", string(modBytes))
	}

	// Verify generated Go code passes go vet
	vetCmd := exec.Command("go", "vet", "./...")
	vetCmd.Dir = projectDir
	if out, err := vetCmd.CombinedOutput(); err != nil {
		// Read main.go for debugging
		mainGo, _ := os.ReadFile(filepath.Join(projectDir, "main.go"))
		t.Fatalf("go vet failed: %v\nOutput: %s\nGenerated main.go:\n%s", err, string(out), string(mainGo))
	}

	// Verify it compiles
	buildCmd := exec.Command("go", "build", "-o", filepath.Join(t.TempDir(), "testcli"), ".")
	buildCmd.Dir = projectDir
	if out, err := buildCmd.CombinedOutput(); err != nil {
		mainGo, _ := os.ReadFile(filepath.Join(projectDir, "main.go"))
		t.Fatalf("go build failed: %v\nOutput: %s\nGenerated main.go:\n%s", err, string(out), string(mainGo))
	}
}

func TestGenerateStdioMode(t *testing.T) {
	ctx := GenerateContext{
		CLIName:       "testcli",
		StdioCommand:  "npx",
		StdioArgs:     []string{"-y", "@org/server"},
		EnvKeys:       []string{"API_KEY"},
		ClihubVersion: "test",
		IsHTTP:        false,
		Tools: []ToolDef{
			{
				Name:        "hello",
				CommandName: "hello",
				Description: "Say hello",
				Options: []schema.ToolOption{
					{
						PropertyName: "name",
						FlagName:     "name",
						Description:  "Name to greet",
						GoType:       "string",
					},
				},
			},
		},
	}

	dir := t.TempDir()
	projectDir, err := Generate(ctx, dir)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify it compiles
	buildCmd := exec.Command("go", "build", "-o", filepath.Join(t.TempDir(), "testcli"), ".")
	buildCmd.Dir = projectDir
	if out, err := buildCmd.CombinedOutput(); err != nil {
		mainGo, _ := os.ReadFile(filepath.Join(projectDir, "main.go"))
		t.Fatalf("go build failed: %v\nOutput: %s\nGenerated main.go:\n%s", err, string(out), string(mainGo))
	}
}

func TestGenerateWithRawBooleanOptionCompiles(t *testing.T) {
	ctx := GenerateContext{
		CLIName:       "rawtest",
		ServerURL:     "https://example.com/mcp",
		ClihubVersion: "test",
		IsHTTP:        true,
		Tools: []ToolDef{
			{
				Name:        "vault",
				CommandName: "vault",
				Description: "Tool with raw boolean parameter",
				Options: []schema.ToolOption{
					{
						PropertyName: "raw",
						FlagName:     "raw",
						Description:  "Return raw result",
						GoType:       "bool",
					},
					{
						PropertyName: "query",
						FlagName:     "query",
						Description:  "Query text",
						GoType:       "string",
					},
				},
			},
		},
	}

	dir := t.TempDir()
	projectDir, err := Generate(ctx, dir)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	buildCmd := exec.Command("go", "build", "-o", filepath.Join(t.TempDir(), "rawtest"), ".")
	buildCmd.Dir = projectDir
	if out, err := buildCmd.CombinedOutput(); err != nil {
		mainGo, _ := os.ReadFile(filepath.Join(projectDir, "main.go"))
		t.Fatalf("go build failed: %v\nOutput: %s\nGenerated main.go:\n%s", err, string(out), string(mainGo))
	}
}

func TestGenerateWithClosureHiddenMode(t *testing.T) {
	ctx := GenerateContext{
		CLIName:       "closurecli",
		ServerURL:     "https://example.com/mcp",
		ClihubVersion: "test",
		IsHTTP:        true,
		ClosureConfig: &closure.Config{
			Mode: closure.ModeHidden,
			Global: closure.GlobalConfig{
				Params: map[string]any{"org_id": "acme-corp"},
			},
			Tools: map[string]closure.ToolConfig{
				"create_item": {
					Params: map[string]any{"project_id": "PROJ-123"},
				},
			},
		},
		Tools: []ToolDef{
			{
				Name:        "list_items",
				CommandName: "list-items",
				Description: "List all items",
				Options: []schema.ToolOption{
					{
						PropertyName: "query",
						FlagName:     "query",
						Description:  "Search query",
						GoType:       "string",
					},
					{
						PropertyName: "org_id",
						FlagName:     "org-id",
						Description:  "Organization ID",
						GoType:       "string",
					},
				},
			},
			{
				Name:        "create_item",
				CommandName: "create-item",
				Description: "Create a new item",
				Options: []schema.ToolOption{
					{
						PropertyName: "title",
						FlagName:     "title",
						Description:  "Item title",
						GoType:       "string",
					},
					{
						PropertyName: "org_id",
						FlagName:     "org-id",
						Description:  "Organization ID",
						GoType:       "string",
					},
					{
						PropertyName: "project_id",
						FlagName:     "project-id",
						Description:  "Project ID",
						GoType:       "string",
					},
				},
			},
		},
	}

	dir := t.TempDir()
	projectDir, err := Generate(ctx, dir)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Read generated main.go for content checks
	mainGo, err := os.ReadFile(filepath.Join(projectDir, "main.go"))
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	src := string(mainGo)

	// Hidden mode: org_id flag should NOT be generated for list_items
	// The closure params should be injected via JSON unmarshal
	if !strings.Contains(src, "closureParams") {
		t.Error("expected closure param injection code in generated source")
	}

	// Verify it compiles
	buildCmd := exec.Command("go", "build", "-o", filepath.Join(t.TempDir(), "closurecli"), ".")
	buildCmd.Dir = projectDir
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\nOutput: %s\nGenerated main.go:\n%s", err, string(out), src)
	}
}

func TestGenerateWithClosureDefaultMode(t *testing.T) {
	ctx := GenerateContext{
		CLIName:       "closuredefault",
		ServerURL:     "https://example.com/mcp",
		ClihubVersion: "test",
		IsHTTP:        true,
		ClosureConfig: &closure.Config{
			Mode: closure.ModeDefault,
			Global: closure.GlobalConfig{
				Params: map[string]any{"org_id": "acme-corp"},
			},
			Tools: map[string]closure.ToolConfig{
				"create_item": {
					Params: map[string]any{"project_id": "PROJ-123"},
				},
			},
		},
		Tools: []ToolDef{
			{
				Name:        "list_items",
				CommandName: "list-items",
				Description: "List all items",
				Options: []schema.ToolOption{
					{
						PropertyName: "query",
						FlagName:     "query",
						Description:  "Search query",
						GoType:       "string",
					},
					{
						PropertyName: "org_id",
						FlagName:     "org-id",
						Description:  "Organization ID",
						GoType:       "string",
					},
				},
			},
			{
				Name:        "create_item",
				CommandName: "create-item",
				Description: "Create a new item",
				Options: []schema.ToolOption{
					{
						PropertyName: "title",
						FlagName:     "title",
						Description:  "Item title",
						GoType:       "string",
					},
					{
						PropertyName: "org_id",
						FlagName:     "org-id",
						Description:  "Organization ID",
						GoType:       "string",
					},
					{
						PropertyName: "project_id",
						FlagName:     "project-id",
						Description:  "Project ID",
						GoType:       "string",
					},
				},
			},
		},
	}

	dir := t.TempDir()
	projectDir, err := Generate(ctx, dir)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	mainGo, err := os.ReadFile(filepath.Join(projectDir, "main.go"))
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	src := string(mainGo)

	// Default mode: flags should still be generated, with closure value as default
	// org_id flag should have "acme-corp" as default
	if !strings.Contains(src, `"acme-corp"`) {
		t.Error("expected closure default value 'acme-corp' in generated source for default mode")
	}

	// Closure injection should use "if _, exists := params[k]; !exists" pattern
	if !strings.Contains(src, "exists") {
		t.Error("expected existence check for default mode closure injection")
	}

	// Verify it compiles
	buildCmd := exec.Command("go", "build", "-o", filepath.Join(t.TempDir(), "closuredefault"), ".")
	buildCmd.Dir = projectDir
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\nOutput: %s\nGenerated main.go:\n%s", err, string(out), src)
	}
}

func TestGenerateWithClosureComplexTypes(t *testing.T) {
	ctx := GenerateContext{
		CLIName:       "closurecomplex",
		ServerURL:     "https://example.com/mcp",
		ClihubVersion: "test",
		IsHTTP:        true,
		ClosureConfig: &closure.Config{
			Mode: closure.ModeHidden,
			Global: closure.GlobalConfig{
				Params: map[string]any{
					"settings": map[string]any{"theme": "dark", "count": float64(5)},
					"tags":     []any{"alpha", "beta"},
					"enabled":  true,
				},
			},
		},
		Tools: []ToolDef{
			{
				Name:        "do_thing",
				CommandName: "do-thing",
				Description: "Do a thing",
				Options: []schema.ToolOption{
					{
						PropertyName: "name",
						FlagName:     "name",
						Description:  "Name",
						GoType:       "string",
					},
				},
			},
		},
	}

	dir := t.TempDir()
	projectDir, err := Generate(ctx, dir)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify it compiles (complex JSON types are embedded correctly)
	buildCmd := exec.Command("go", "build", "-o", filepath.Join(t.TempDir(), "closurecomplex"), ".")
	buildCmd.Dir = projectDir
	if out, err := buildCmd.CombinedOutput(); err != nil {
		mainGo, _ := os.ReadFile(filepath.Join(projectDir, "main.go"))
		t.Fatalf("go build failed: %v\nOutput: %s\nGenerated main.go:\n%s", err, string(out), string(mainGo))
	}
}

func TestTemplateFunctions(t *testing.T) {
	tests := []struct {
		name     string
		fn       func() string
		expected string
	}{
		{"varName simple", func() string { return toVarName("query") }, "flagQuery"},
		{"varName kebab", func() string { return toVarName("team-id") }, "flagTeamId"},
		{"varName multi", func() string { return toVarName("my-long-name") }, "flagMyLongName"},
		{"funcName underscore", func() string { return toFuncName("list_issues") }, "ListIssues"},
		{"funcName dash", func() string { return toFuncName("list-issues") }, "ListIssues"},
		{"cobraFlag string", func() string { return cobraFlagType("string") }, "StringVar"},
		{"cobraFlag int", func() string { return cobraFlagType("int") }, "IntVar"},
		{"cobraFlag bool", func() string { return cobraFlagType("bool") }, "BoolVar"},
		{"cobraFlag float64", func() string { return cobraFlagType("float64") }, "Float64Var"},
		{"cobraFlag []string", func() string { return cobraFlagType("[]string") }, "StringSliceVar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn()
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}
