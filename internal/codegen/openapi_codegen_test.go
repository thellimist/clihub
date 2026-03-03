package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thellimist/clihub/internal/schema"
)

func TestGenerateOpenAPIProducesValidGo(t *testing.T) {
	ctx := OpenAPIGenerateContext{
		CLIName:       "petstore",
		BaseURL:       "https://petstore.example.com/api/v1",
		Title:         "Petstore",
		ClihubVersion: "test",
		Operations: []OpenAPIToolDef{
			{
				OperationID: "listPets",
				CommandName: "list-pets",
				Description: "List all pets",
				Method:      "GET",
				Path:        "/pets",
				QueryParams: []schema.ToolOption{
					{
						PropertyName: "limit",
						FlagName:     "limit",
						Description:  "Max items to return",
						GoType:       "int",
						DefaultValue: float64(20),
					},
					{
						PropertyName: "status",
						FlagName:     "status",
						Description:  "Filter by status",
						GoType:       "string",
						EnumValues:   []string{"available", "pending", "sold"},
					},
				},
			},
			{
				OperationID: "createPet",
				CommandName: "create-pet",
				Description: "Create a pet",
				Method:      "POST",
				Path:        "/pets",
				HasBody:     true,
				BodyParams: []schema.ToolOption{
					{
						PropertyName: "name",
						FlagName:     "name",
						Description:  "Pet name",
						Required:     true,
						GoType:       "string",
					},
					{
						PropertyName: "tag",
						FlagName:     "tag",
						Description:  "Optional tag",
						GoType:       "string",
					},
					{
						PropertyName: "age",
						FlagName:     "age",
						Description:  "Age in years",
						GoType:       "int",
					},
				},
			},
			{
				OperationID: "getPet",
				CommandName: "get-pet",
				Description: "Get a pet by ID",
				Method:      "GET",
				Path:        "/pets/{petId}",
				PathParams: []schema.ToolOption{
					{
						PropertyName: "petId",
						FlagName:     "pet-id",
						Description:  "Pet identifier",
						Required:     true,
						GoType:       "string",
					},
				},
			},
			{
				OperationID: "deletePet",
				CommandName: "delete-pet",
				Description: "Delete a pet",
				Method:      "DELETE",
				Path:        "/pets/{petId}",
				PathParams: []schema.ToolOption{
					{
						PropertyName: "petId",
						FlagName:     "pet-id",
						Description:  "Pet identifier",
						Required:     true,
						GoType:       "string",
					},
				},
			},
		},
	}

	dir := t.TempDir()
	projectDir, err := GenerateOpenAPI(ctx, dir)
	if err != nil {
		t.Fatalf("GenerateOpenAPI failed: %v", err)
	}

	// Verify files exist.
	for _, f := range []string{"main.go", "go.mod", "go.sum"} {
		path := filepath.Join(projectDir, f)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %s to exist: %v", f, err)
		}
	}

	// Verify no mcp-go dependency.
	modBytes, err := os.ReadFile(filepath.Join(projectDir, "go.mod"))
	if err != nil {
		t.Fatalf("read generated go.mod: %v", err)
	}
	if strings.Contains(string(modBytes), "mcp-go") {
		t.Fatalf("generated go.mod should NOT contain mcp-go dependency:\n%s", string(modBytes))
	}

	// Verify go.mod has go 1.24.
	if !strings.Contains(string(modBytes), "\ngo 1.24") {
		t.Fatalf("generated go.mod must declare go 1.24.x, got:\n%s", string(modBytes))
	}

	// Verify generated Go code passes go vet.
	vetCmd := exec.Command("go", "vet", "./...")
	vetCmd.Dir = projectDir
	if out, err := vetCmd.CombinedOutput(); err != nil {
		mainGo, _ := os.ReadFile(filepath.Join(projectDir, "main.go"))
		t.Fatalf("go vet failed: %v\nOutput: %s\nGenerated main.go:\n%s", err, string(out), string(mainGo))
	}

	// Verify it compiles.
	buildCmd := exec.Command("go", "build", "-o", filepath.Join(t.TempDir(), "petstore"), ".")
	buildCmd.Dir = projectDir
	if out, err := buildCmd.CombinedOutput(); err != nil {
		mainGo, _ := os.ReadFile(filepath.Join(projectDir, "main.go"))
		t.Fatalf("go build failed: %v\nOutput: %s\nGenerated main.go:\n%s", err, string(out), string(mainGo))
	}
}

func TestGenerateOpenAPIWithAllParamTypes(t *testing.T) {
	ctx := OpenAPIGenerateContext{
		CLIName:       "allparams",
		BaseURL:       "https://api.example.com",
		Title:         "All Params API",
		ClihubVersion: "test",
		Operations: []OpenAPIToolDef{
			{
				OperationID: "complexOp",
				CommandName: "complex-op",
				Description: "Operation with all param types",
				Method:      "POST",
				Path:        "/items/{itemId}",
				HasBody:     true,
				PathParams: []schema.ToolOption{
					{PropertyName: "itemId", FlagName: "item-id", GoType: "string", Required: true},
				},
				QueryParams: []schema.ToolOption{
					{PropertyName: "verbose", FlagName: "verbose", GoType: "bool"},
					{PropertyName: "tags", FlagName: "tags", GoType: "[]string"},
				},
				HeaderParams: []schema.ToolOption{
					{PropertyName: "X-Request-ID", FlagName: "x-request-id", GoType: "string"},
				},
				BodyParams: []schema.ToolOption{
					{PropertyName: "title", FlagName: "title", GoType: "string", Required: true},
					{PropertyName: "count", FlagName: "count", GoType: "int"},
					{PropertyName: "score", FlagName: "score", GoType: "float64"},
				},
			},
		},
	}

	dir := t.TempDir()
	projectDir, err := GenerateOpenAPI(ctx, dir)
	if err != nil {
		t.Fatalf("GenerateOpenAPI failed: %v", err)
	}

	buildCmd := exec.Command("go", "build", "-o", filepath.Join(t.TempDir(), "allparams"), ".")
	buildCmd.Dir = projectDir
	if out, err := buildCmd.CombinedOutput(); err != nil {
		mainGo, _ := os.ReadFile(filepath.Join(projectDir, "main.go"))
		t.Fatalf("go build failed: %v\nOutput: %s\nGenerated main.go:\n%s", err, string(out), string(mainGo))
	}
}

func TestGenerateOpenAPINoOperations(t *testing.T) {
	ctx := OpenAPIGenerateContext{
		CLIName:       "empty",
		BaseURL:       "https://api.example.com",
		Title:         "Empty API",
		ClihubVersion: "test",
		Operations:    nil,
	}

	dir := t.TempDir()
	projectDir, err := GenerateOpenAPI(ctx, dir)
	if err != nil {
		t.Fatalf("GenerateOpenAPI failed: %v", err)
	}

	buildCmd := exec.Command("go", "build", "-o", filepath.Join(t.TempDir(), "empty"), ".")
	buildCmd.Dir = projectDir
	if out, err := buildCmd.CombinedOutput(); err != nil {
		mainGo, _ := os.ReadFile(filepath.Join(projectDir, "main.go"))
		t.Fatalf("go build failed: %v\nOutput: %s\nGenerated main.go:\n%s", err, string(out), string(mainGo))
	}
}
