package openapi

import "github.com/getkin/kin-openapi/openapi3"

// mapSchemaType maps a kin-openapi Schema to a Go type string.
//
// Supported mappings:
//   - string → "string"
//   - integer → "int"
//   - number → "float64"
//   - boolean → "bool"
//   - array of string → "[]string"
//   - array of integer → "[]int"
//   - object/unknown → "string" (use --from-json for complex types)
func mapSchemaType(s *openapi3.Schema) string {
	if s == nil {
		return "string"
	}

	typ := s.Type
	if typ == nil {
		return "string"
	}

	types := typ.Slice()
	if len(types) == 0 {
		return "string"
	}

	// Handle nullable types: pick first non-"null" entry.
	t := types[0]
	for _, candidate := range types {
		if candidate != "null" {
			t = candidate
			break
		}
	}

	switch t {
	case "string":
		return "string"
	case "integer":
		return "int"
	case "number":
		return "float64"
	case "boolean":
		return "bool"
	case "array":
		return mapArraySchemaType(s.Items)
	default:
		return "string"
	}
}

// mapArraySchemaType determines the Go slice type from an array's items schema.
func mapArraySchemaType(items *openapi3.SchemaRef) string {
	if items == nil || items.Value == nil {
		return "[]string"
	}

	typ := items.Value.Type
	if typ == nil {
		return "[]string"
	}

	types := typ.Slice()
	if len(types) == 0 {
		return "[]string"
	}

	switch types[0] {
	case "integer":
		return "[]int"
	default:
		return "[]string"
	}
}
