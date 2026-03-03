package openapi

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

const petStoreSpec = `{
  "openapi": "3.0.3",
  "info": { "title": "Petstore", "version": "1.0.0" },
  "servers": [{ "url": "https://petstore.example.com/api/v1" }],
  "paths": {
    "/pets": {
      "get": {
        "operationId": "listPets",
        "summary": "List all pets",
        "parameters": [
          {
            "name": "limit",
            "in": "query",
            "description": "Max items to return",
            "required": false,
            "schema": { "type": "integer", "default": 20 }
          },
          {
            "name": "status",
            "in": "query",
            "description": "Filter by status",
            "schema": { "type": "string", "enum": ["available", "pending", "sold"] }
          }
        ],
        "responses": { "200": { "description": "A list of pets" } }
      },
      "post": {
        "operationId": "createPet",
        "summary": "Create a pet",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["name"],
                "properties": {
                  "name": { "type": "string", "description": "Pet name" },
                  "tag": { "type": "string", "description": "Optional tag" },
                  "age": { "type": "integer", "description": "Age in years" }
                }
              }
            }
          }
        },
        "responses": { "201": { "description": "Pet created" } }
      }
    },
    "/pets/{petId}": {
      "get": {
        "operationId": "getPet",
        "summary": "Get a pet by ID",
        "parameters": [
          {
            "name": "petId",
            "in": "path",
            "required": true,
            "schema": { "type": "string" }
          }
        ],
        "responses": { "200": { "description": "A pet" } }
      },
      "delete": {
        "operationId": "deletePet",
        "summary": "Delete a pet",
        "parameters": [
          {
            "name": "petId",
            "in": "path",
            "required": true,
            "schema": { "type": "string" }
          }
        ],
        "responses": { "204": { "description": "Pet deleted" } }
      }
    }
  }
}`

func TestLoadSpec(t *testing.T) {
	// Write spec to temp file.
	dir := t.TempDir()
	specPath := filepath.Join(dir, "petstore.json")
	if err := os.WriteFile(specPath, []byte(petStoreSpec), 0644); err != nil {
		t.Fatal(err)
	}

	doc, err := LoadSpec(context.Background(), specPath, nil)
	if err != nil {
		t.Fatalf("LoadSpec failed: %v", err)
	}

	if doc.Info.Title != "Petstore" {
		t.Errorf("expected title Petstore, got %q", doc.Info.Title)
	}

	if len(doc.Paths.Map()) != 2 {
		t.Errorf("expected 2 paths, got %d", len(doc.Paths.Map()))
	}
}

func TestExtractOperations(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "petstore.json")
	if err := os.WriteFile(specPath, []byte(petStoreSpec), 0644); err != nil {
		t.Fatal(err)
	}

	doc, err := LoadSpec(context.Background(), specPath, nil)
	if err != nil {
		t.Fatalf("LoadSpec failed: %v", err)
	}

	ops, err := ExtractOperations(doc)
	if err != nil {
		t.Fatalf("ExtractOperations failed: %v", err)
	}

	if len(ops) != 4 {
		t.Fatalf("expected 4 operations, got %d", len(ops))
	}

	// Verify sorted by command name.
	names := make([]string, len(ops))
	for i, op := range ops {
		names[i] = op.CommandName
	}
	for i := 1; i < len(names); i++ {
		if names[i] < names[i-1] {
			t.Errorf("operations not sorted: %v", names)
			break
		}
	}

	// Find specific operations and verify details.
	opMap := make(map[string]Operation)
	for _, op := range ops {
		opMap[op.CommandName] = op
	}

	// listPets
	listPets, ok := opMap["list-pets"]
	if !ok {
		t.Fatal("missing list-pets operation")
	}
	if listPets.Method != "GET" {
		t.Errorf("list-pets method: want GET, got %s", listPets.Method)
	}
	if len(listPets.QueryParams) != 2 {
		t.Errorf("list-pets query params: want 2, got %d", len(listPets.QueryParams))
	}

	// createPet
	createPet, ok := opMap["create-pet"]
	if !ok {
		t.Fatal("missing create-pet operation")
	}
	if !createPet.HasBody {
		t.Error("create-pet should have body")
	}
	if len(createPet.BodyParams) != 3 {
		t.Errorf("create-pet body params: want 3, got %d", len(createPet.BodyParams))
	}
	// Verify required body param is first (sorted).
	if len(createPet.BodyParams) > 0 && createPet.BodyParams[0].PropertyName != "name" {
		t.Errorf("expected first body param to be 'name' (required), got %q", createPet.BodyParams[0].PropertyName)
	}

	// getPet
	getPet, ok := opMap["get-pet"]
	if !ok {
		t.Fatal("missing get-pet operation")
	}
	if len(getPet.PathParams) != 1 {
		t.Errorf("get-pet path params: want 1, got %d", len(getPet.PathParams))
	}
	if len(getPet.PathParams) > 0 && !getPet.PathParams[0].Required {
		t.Error("path param petId should be required")
	}
}

func TestExtractOperationsEnumValues(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "petstore.json")
	if err := os.WriteFile(specPath, []byte(petStoreSpec), 0644); err != nil {
		t.Fatal(err)
	}

	doc, err := LoadSpec(context.Background(), specPath, nil)
	if err != nil {
		t.Fatal(err)
	}

	ops, err := ExtractOperations(doc)
	if err != nil {
		t.Fatal(err)
	}

	for _, op := range ops {
		if op.CommandName != "list-pets" {
			continue
		}
		for _, qp := range op.QueryParams {
			if qp.FlagName == "status" {
				if len(qp.EnumValues) != 3 {
					t.Errorf("status enum: want 3, got %d", len(qp.EnumValues))
				}
				return
			}
		}
		t.Error("status query param not found")
	}
}

func TestBaseURL(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "petstore.json")
	if err := os.WriteFile(specPath, []byte(petStoreSpec), 0644); err != nil {
		t.Fatal(err)
	}

	doc, err := LoadSpec(context.Background(), specPath, nil)
	if err != nil {
		t.Fatal(err)
	}

	got := BaseURL(doc)
	want := "https://petstore.example.com/api/v1"
	if got != want {
		t.Errorf("BaseURL: got %q, want %q", got, want)
	}
}

func TestOperationCommandNameFromID(t *testing.T) {
	tests := []struct {
		method, path, opID string
		want               string
	}{
		{"GET", "/pets", "listPets", "list-pets"},
		{"POST", "/pets", "createPet", "create-pet"},
		{"GET", "/pets/{petId}", "getPetById", "get-pet-by-id"},
		{"DELETE", "/repos/{owner}/{repo}", "deleteRepo", "delete-repo"},
	}

	for _, tt := range tests {
		got := operationCommandName(tt.method, tt.path, tt.opID)
		if got != tt.want {
			t.Errorf("operationCommandName(%q, %q, %q) = %q, want %q", tt.method, tt.path, tt.opID, got, tt.want)
		}
	}
}

func TestOperationCommandNameFallback(t *testing.T) {
	tests := []struct {
		method, path string
		want         string
	}{
		{"GET", "/pets", "get-pets"},
		{"POST", "/pets", "post-pets"},
		{"GET", "/pets/{petId}", "get-pets"},
		{"DELETE", "/repos/{owner}/{repo}/issues", "delete-repos-issues"},
	}

	for _, tt := range tests {
		got := operationCommandName(tt.method, tt.path, "")
		if got != tt.want {
			t.Errorf("operationCommandName(%q, %q, \"\") = %q, want %q", tt.method, tt.path, got, tt.want)
		}
	}
}

func TestDeduplicateCommandNames(t *testing.T) {
	ops := []Operation{
		{CommandName: "pets", Method: "GET"},
		{CommandName: "pets", Method: "POST"},
		{CommandName: "users", Method: "GET"},
	}

	deduplicateCommandNames(ops)

	if ops[0].CommandName != "pets-get" {
		t.Errorf("ops[0].CommandName = %q, want pets-get", ops[0].CommandName)
	}
	if ops[1].CommandName != "pets-post" {
		t.Errorf("ops[1].CommandName = %q, want pets-post", ops[1].CommandName)
	}
	if ops[2].CommandName != "users" {
		t.Errorf("ops[2].CommandName = %q, want users", ops[2].CommandName)
	}
}

func TestMapSchemaType(t *testing.T) {
	tests := []struct {
		name string
		typ  string
		want string
	}{
		{"string", "string", "string"},
		{"integer", "integer", "int"},
		{"number", "number", "float64"},
		{"boolean", "boolean", "bool"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &openapi3.Schema{}
			s.Type = &openapi3.Types{tt.typ}
			got := mapSchemaType(s)
			if got != tt.want {
				t.Errorf("mapSchemaType(%q) = %q, want %q", tt.typ, got, tt.want)
			}
		})
	}
}

func TestMapSchemaTypeArray(t *testing.T) {
	s := &openapi3.Schema{}
	s.Type = &openapi3.Types{"array"}
	s.Items = &openapi3.SchemaRef{
		Value: &openapi3.Schema{},
	}
	s.Items.Value.Type = &openapi3.Types{"string"}

	got := mapSchemaType(s)
	if got != "[]string" {
		t.Errorf("mapSchemaType(array of string) = %q, want []string", got)
	}
}

func TestMapSchemaTypeNil(t *testing.T) {
	got := mapSchemaType(nil)
	if got != "string" {
		t.Errorf("mapSchemaType(nil) = %q, want string", got)
	}
}
