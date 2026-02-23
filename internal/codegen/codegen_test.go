package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/clihub/clihub/internal/schema"
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
