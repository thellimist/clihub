package schema

// ToolOption represents a single CLI flag derived from a JSON Schema property.
type ToolOption struct {
	PropertyName string   // Original JSON key (e.g., "issueId")
	FlagName     string   // Kebab-case CLI flag (e.g., "issue-id")
	Description  string   // From schema description field
	Required     bool     // True if property is in schema's required array
	GoType       string   // Go type: "string", "int", "float64", "bool", "[]string", "[]int"
	DefaultValue any      // From schema default field, nil if not set
	EnumValues   []string // From schema enum field, nil if not an enum
}
