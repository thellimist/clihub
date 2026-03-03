package openapi

import (
	"strings"

	"github.com/thellimist/clihub/internal/schema"
)

// operationCommandName derives a kebab-case command name for an operation.
//
// Strategy:
//  1. If operationId is present, kebab-case it.
//  2. Otherwise, build from method + path segments (stripping {params}).
//
// The method and path are used as fallback only when operationId is empty.
func operationCommandName(method, path, operationID string) string {
	if operationID != "" {
		return schema.ToFlagName(strings.ReplaceAll(operationID, "_", "-"))
	}

	// Build from method + path segments
	segments := strings.Split(strings.Trim(path, "/"), "/")
	var parts []string
	for _, seg := range segments {
		if seg == "" {
			continue
		}
		// Skip path parameters like {id}
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			continue
		}
		parts = append(parts, seg)
	}

	name := strings.ToLower(method)
	if len(parts) > 0 {
		name += "-" + strings.Join(parts, "-")
	}
	return name
}

// deduplicateCommandNames appends the HTTP method as a suffix to resolve
// name collisions (e.g., GET /pets and POST /pets both produce "pets").
func deduplicateCommandNames(ops []Operation) {
	counts := make(map[string]int)
	for _, op := range ops {
		counts[op.CommandName]++
	}
	for i := range ops {
		if counts[ops[i].CommandName] > 1 {
			ops[i].CommandName = ops[i].CommandName + "-" + strings.ToLower(ops[i].Method)
		}
	}
}
