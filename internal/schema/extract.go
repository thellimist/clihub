package schema

import (
	"encoding/json"
	"fmt"
	"sort"
)

// ExtractOptions parses a JSON Schema inputSchema and returns a sorted slice
// of ToolOption values — one per property.
//
// Sort order: required options first, then alphabetical by FlagName within each
// group.
//
// Edge cases:
//   - nil or empty inputSchema → returns empty slice, no error
//   - Missing "properties" → returns empty slice, no error
//   - Missing "type" on a property → defaults to "string"
func ExtractOptions(inputSchema json.RawMessage) ([]ToolOption, error) {
	if len(inputSchema) == 0 || string(inputSchema) == "null" {
		return nil, nil
	}

	var root map[string]interface{}
	if err := json.Unmarshal(inputSchema, &root); err != nil {
		return nil, fmt.Errorf("schema: failed to parse inputSchema: %w", err)
	}

	propsRaw, ok := root["properties"]
	if !ok {
		return nil, nil
	}
	properties, ok := propsRaw.(map[string]interface{})
	if !ok {
		return nil, nil
	}
	if len(properties) == 0 {
		return nil, nil
	}

	// Build required set.
	requiredSet := make(map[string]bool)
	if reqRaw, ok := root["required"]; ok {
		if reqArr, ok := reqRaw.([]interface{}); ok {
			for _, v := range reqArr {
				if s, ok := v.(string); ok {
					requiredSet[s] = true
				}
			}
		}
	}

	options := make([]ToolOption, 0, len(properties))
	for name, propRaw := range properties {
		prop, ok := propRaw.(map[string]interface{})
		if !ok {
			continue
		}

		opt := ToolOption{
			PropertyName: name,
			FlagName:     ToFlagName(name),
			Required:     requiredSet[name],
		}

		// Description.
		if desc, ok := prop["description"].(string); ok {
			opt.Description = desc
		}

		// Type mapping.
		schemaType := prop["type"] // may be nil, string, or []interface{}
		var items map[string]interface{}
		if itemsRaw, ok := prop["items"].(map[string]interface{}); ok {
			items = itemsRaw
		}
		if schemaType == nil {
			opt.GoType = "string"
		} else {
			opt.GoType = mapJSONSchemaType(schemaType, items)
		}

		// Default value.
		if def, ok := prop["default"]; ok {
			opt.DefaultValue = def
		}

		// Enum values.
		if enumRaw, ok := prop["enum"].([]interface{}); ok {
			vals := make([]string, 0, len(enumRaw))
			for _, v := range enumRaw {
				vals = append(vals, fmt.Sprintf("%v", v))
			}
			opt.EnumValues = vals
		}

		options = append(options, opt)
	}

	// Sort: required first, then alphabetical by FlagName.
	sort.Slice(options, func(i, j int) bool {
		if options[i].Required != options[j].Required {
			return options[i].Required
		}
		return options[i].FlagName < options[j].FlagName
	})

	return options, nil
}
