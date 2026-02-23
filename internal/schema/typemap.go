package schema

// mapJSONSchemaType maps a JSON Schema type (and optional items) to a Go type string.
//
// It handles:
//   - Basic types: string, integer, number, boolean
//   - Array types: checks items.type for element type
//   - Nullable types: when type is an array like ["string", "null"], picks the first non-"null" type
//   - Unrecognized types default to "string"
func mapJSONSchemaType(schemaType interface{}, items map[string]interface{}) string {
	switch t := schemaType.(type) {
	case string:
		return mapSingleType(t, items)
	case []interface{}:
		// Nullable type: pick the first non-"null" entry.
		for _, v := range t {
			s, ok := v.(string)
			if ok && s != "null" {
				return mapSingleType(s, items)
			}
		}
		// All entries were "null" or unrecognized; fall through to default.
		return "string"
	default:
		return "string"
	}
}

// mapSingleType maps a single JSON Schema type string to a Go type.
func mapSingleType(t string, items map[string]interface{}) string {
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
		return mapArrayType(items)
	default:
		return "string"
	}
}

// mapArrayType determines the Go slice type based on the items schema.
func mapArrayType(items map[string]interface{}) string {
	if items == nil {
		return "[]string"
	}
	itemType, ok := items["type"].(string)
	if !ok {
		return "[]string"
	}
	switch itemType {
	case "string":
		return "[]string"
	case "integer":
		return "[]int"
	default:
		return "[]string"
	}
}
