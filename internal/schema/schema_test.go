package schema

import (
	"encoding/json"
	"testing"
)

// ---------------------------------------------------------------------------
// ToFlagName tests
// ---------------------------------------------------------------------------

func TestToFlagName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"issueId", "issue-id"},
		{"relativePath", "relative-path"},
		{"HTMLParser", "html-parser"},
		{"getHTTPResponse", "get-http-response"},
		{"myURL", "my-url"},
		{"simple", "simple"},
		{"list_items", "list-items"},
		{"ALLCAPS", "allcaps"},
		{"", ""},
		{"a", "a"},
		{"A", "a"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := ToFlagName(tc.input)
			if got != tc.want {
				t.Errorf("ToFlagName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// mapJSONSchemaType tests
// ---------------------------------------------------------------------------

func TestMapJSONSchemaType(t *testing.T) {
	tests := []struct {
		name       string
		schemaType interface{}
		items      map[string]interface{}
		want       string
	}{
		{"string", "string", nil, "string"},
		{"integer", "integer", nil, "int"},
		{"number", "number", nil, "float64"},
		{"boolean", "boolean", nil, "bool"},
		{"array with string items", "array", map[string]interface{}{"type": "string"}, "[]string"},
		{"array with integer items", "array", map[string]interface{}{"type": "integer"}, "[]int"},
		{"array with no items", "array", nil, "[]string"},
		{"array with unknown item type", "array", map[string]interface{}{"type": "object"}, "[]string"},
		{"nullable string", []interface{}{"string", "null"}, nil, "string"},
		{"nullable integer (null first)", []interface{}{"null", "integer"}, nil, "int"},
		{"unknown type", "object", nil, "string"},
		{"nil type", nil, nil, "string"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mapJSONSchemaType(tc.schemaType, tc.items)
			if got != tc.want {
				t.Errorf("mapJSONSchemaType(%v, %v) = %q, want %q", tc.schemaType, tc.items, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ExtractOptions tests
// ---------------------------------------------------------------------------

func TestExtractOptions_FullSchema(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"issueId": {
				"type": "integer",
				"description": "The issue number"
			},
			"labels": {
				"type": "array",
				"items": {"type": "string"},
				"description": "Labels to apply"
			},
			"verbose": {
				"type": "boolean",
				"default": false,
				"description": "Enable verbose output"
			},
			"format": {
				"type": "string",
				"enum": ["json", "text", "csv"],
				"description": "Output format"
			},
			"threshold": {
				"type": "number",
				"default": 0.5,
				"description": "Confidence threshold"
			}
		},
		"required": ["issueId"]
	}`

	opts, err := ExtractOptions(json.RawMessage(schema))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(opts) != 5 {
		t.Fatalf("expected 5 options, got %d", len(opts))
	}

	// First option should be the required one (issueId).
	first := opts[0]
	if first.PropertyName != "issueId" {
		t.Errorf("expected first option to be issueId, got %q", first.PropertyName)
	}
	if first.FlagName != "issue-id" {
		t.Errorf("expected flag name issue-id, got %q", first.FlagName)
	}
	if !first.Required {
		t.Error("expected issueId to be required")
	}
	if first.GoType != "int" {
		t.Errorf("expected GoType int, got %q", first.GoType)
	}
	if first.Description != "The issue number" {
		t.Errorf("expected description 'The issue number', got %q", first.Description)
	}

	// Remaining options should be sorted alphabetically by FlagName.
	expectedOrder := []string{"format", "labels", "threshold", "verbose"}
	for i, name := range expectedOrder {
		if opts[i+1].FlagName != name {
			t.Errorf("opts[%d].FlagName = %q, want %q", i+1, opts[i+1].FlagName, name)
		}
	}

	// Check enum values on format.
	var formatOpt *ToolOption
	for i := range opts {
		if opts[i].PropertyName == "format" {
			formatOpt = &opts[i]
			break
		}
	}
	if formatOpt == nil {
		t.Fatal("format option not found")
	}
	if len(formatOpt.EnumValues) != 3 {
		t.Fatalf("expected 3 enum values, got %d", len(formatOpt.EnumValues))
	}
	if formatOpt.EnumValues[0] != "json" || formatOpt.EnumValues[1] != "text" || formatOpt.EnumValues[2] != "csv" {
		t.Errorf("unexpected enum values: %v", formatOpt.EnumValues)
	}

	// Check default on verbose.
	var verboseOpt *ToolOption
	for i := range opts {
		if opts[i].PropertyName == "verbose" {
			verboseOpt = &opts[i]
			break
		}
	}
	if verboseOpt == nil {
		t.Fatal("verbose option not found")
	}
	if verboseOpt.DefaultValue != false {
		t.Errorf("expected default false, got %v", verboseOpt.DefaultValue)
	}
	if verboseOpt.GoType != "bool" {
		t.Errorf("expected GoType bool, got %q", verboseOpt.GoType)
	}

	// Check labels.
	var labelsOpt *ToolOption
	for i := range opts {
		if opts[i].PropertyName == "labels" {
			labelsOpt = &opts[i]
			break
		}
	}
	if labelsOpt == nil {
		t.Fatal("labels option not found")
	}
	if labelsOpt.GoType != "[]string" {
		t.Errorf("expected GoType []string, got %q", labelsOpt.GoType)
	}
}

func TestExtractOptions_NullableTypes(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"name": {
				"type": ["string", "null"],
				"description": "Optional name"
			},
			"count": {
				"type": ["null", "integer"],
				"description": "Optional count"
			}
		}
	}`

	opts, err := ExtractOptions(json.RawMessage(schema))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(opts) != 2 {
		t.Fatalf("expected 2 options, got %d", len(opts))
	}

	// Find by property name.
	typeMap := make(map[string]string)
	for _, opt := range opts {
		typeMap[opt.PropertyName] = opt.GoType
	}
	if typeMap["name"] != "string" {
		t.Errorf("expected name type string, got %q", typeMap["name"])
	}
	if typeMap["count"] != "int" {
		t.Errorf("expected count type int, got %q", typeMap["count"])
	}
}

func TestExtractOptions_EmptyProperties(t *testing.T) {
	schema := `{"type": "object", "properties": {}}`
	opts, err := ExtractOptions(json.RawMessage(schema))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(opts) != 0 {
		t.Errorf("expected 0 options, got %d", len(opts))
	}
}

func TestExtractOptions_NoProperties(t *testing.T) {
	schema := `{"type": "object"}`
	opts, err := ExtractOptions(json.RawMessage(schema))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts != nil {
		t.Errorf("expected nil, got %v", opts)
	}
}

func TestExtractOptions_NullSchema(t *testing.T) {
	opts, err := ExtractOptions(json.RawMessage("null"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts != nil {
		t.Errorf("expected nil, got %v", opts)
	}
}

func TestExtractOptions_EmptySchema(t *testing.T) {
	opts, err := ExtractOptions(json.RawMessage(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts != nil {
		t.Errorf("expected nil, got %v", opts)
	}
}

func TestExtractOptions_MissingType(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"unknown": {
				"description": "No type specified"
			}
		}
	}`
	opts, err := ExtractOptions(json.RawMessage(schema))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(opts) != 1 {
		t.Fatalf("expected 1 option, got %d", len(opts))
	}
	if opts[0].GoType != "string" {
		t.Errorf("expected GoType string for missing type, got %q", opts[0].GoType)
	}
}

func TestExtractOptions_SortOrder(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"zeta": {"type": "string"},
			"alpha": {"type": "string"},
			"beta": {"type": "string"},
			"gamma": {"type": "string"}
		},
		"required": ["gamma", "alpha"]
	}`

	opts, err := ExtractOptions(json.RawMessage(schema))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(opts) != 4 {
		t.Fatalf("expected 4 options, got %d", len(opts))
	}

	// Required first, alphabetical: alpha, gamma. Then non-required: beta, zeta.
	expected := []struct {
		flag     string
		required bool
	}{
		{"alpha", true},
		{"gamma", true},
		{"beta", false},
		{"zeta", false},
	}
	for i, exp := range expected {
		if opts[i].FlagName != exp.flag {
			t.Errorf("opts[%d].FlagName = %q, want %q", i, opts[i].FlagName, exp.flag)
		}
		if opts[i].Required != exp.required {
			t.Errorf("opts[%d].Required = %v, want %v", i, opts[i].Required, exp.required)
		}
	}
}
