package openapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/thellimist/clihub/internal/schema"
)

// LoadSpec loads an OpenAPI 3.x spec from a URL or local file path.
// authHeaders are added to the HTTP request when fetching from a URL.
func LoadSpec(ctx context.Context, source string, authHeaders map[string]string) (*openapi3.T, error) {
	var data []byte
	var err error

	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		data, err = fetchURL(ctx, source, authHeaders)
		if err != nil {
			return nil, fmt.Errorf("fetch OpenAPI spec: %w", err)
		}
	} else {
		data, err = os.ReadFile(source)
		if err != nil {
			return nil, fmt.Errorf("read OpenAPI spec: %w", err)
		}
	}

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	doc, err := loader.LoadFromData(data)
	if err != nil {
		return nil, fmt.Errorf("parse OpenAPI spec: %w", err)
	}

	if err := doc.Validate(ctx); err != nil {
		return nil, fmt.Errorf("validate OpenAPI spec: %w", err)
	}

	return doc, nil
}

// ExtractOperations walks an OpenAPI spec and returns one Operation per
// endpoint. Operations are sorted by command name.
func ExtractOperations(doc *openapi3.T) ([]Operation, error) {
	var ops []Operation

	for path, pathItem := range doc.Paths.Map() {
		for method, op := range pathItem.Operations() {
			if op == nil {
				continue
			}

			cmdName := operationCommandName(method, path, op.OperationID)

			summary := op.Summary
			if summary == "" {
				summary = op.Description
			}
			// Truncate long descriptions for CLI short help.
			if len(summary) > 120 {
				summary = summary[:117] + "..."
			}

			operation := Operation{
				OperationID: op.OperationID,
				CommandName: cmdName,
				Summary:     summary,
				Method:      method,
				Path:        path,
			}

			// Extract parameters (path, query, header).
			for _, paramRef := range op.Parameters {
				if paramRef.Value == nil {
					continue
				}
				p := paramRef.Value
				opt := paramToOption(p)

				switch p.In {
				case "path":
					opt.Required = true // Path params are always required.
					operation.PathParams = append(operation.PathParams, opt)
				case "query":
					operation.QueryParams = append(operation.QueryParams, opt)
				case "header":
					operation.HeaderParams = append(operation.HeaderParams, opt)
				}
			}

			// Extract request body (top-level properties only).
			if op.RequestBody != nil && op.RequestBody.Value != nil {
				body := op.RequestBody.Value
				operation.HasBody = true

				// Prefer application/json content type.
				if content, ok := body.Content["application/json"]; ok && content.Schema != nil {
					bodyOpts := extractBodyParams(content.Schema)
					operation.BodyParams = bodyOpts
				}
			}

			ops = append(ops, operation)
		}
	}

	// Sort by command name for deterministic output.
	sort.Slice(ops, func(i, j int) bool {
		return ops[i].CommandName < ops[j].CommandName
	})

	deduplicateCommandNames(ops)

	return ops, nil
}

// BaseURL extracts the first server URL from the spec, or returns empty string.
func BaseURL(doc *openapi3.T) string {
	if doc.Servers != nil && len(doc.Servers) > 0 {
		return doc.Servers[0].URL
	}
	return ""
}

// paramToOption converts an OpenAPI parameter to a ToolOption.
func paramToOption(p *openapi3.Parameter) schema.ToolOption {
	opt := schema.ToolOption{
		PropertyName: p.Name,
		FlagName:     schema.ToFlagName(p.Name),
		Description:  p.Description,
		Required:     p.Required,
		GoType:       "string",
	}

	if p.Schema != nil && p.Schema.Value != nil {
		s := p.Schema.Value
		opt.GoType = mapSchemaType(s)

		if s.Default != nil {
			opt.DefaultValue = s.Default
		}

		if len(s.Enum) > 0 {
			for _, e := range s.Enum {
				opt.EnumValues = append(opt.EnumValues, fmt.Sprintf("%v", e))
			}
		}
	}

	return opt
}

// extractBodyParams extracts top-level properties from a request body schema.
func extractBodyParams(schemaRef *openapi3.SchemaRef) []schema.ToolOption {
	if schemaRef == nil || schemaRef.Value == nil {
		return nil
	}
	s := schemaRef.Value

	// Build required set.
	requiredSet := make(map[string]bool, len(s.Required))
	for _, name := range s.Required {
		requiredSet[name] = true
	}

	var opts []schema.ToolOption
	for name, propRef := range s.Properties {
		if propRef == nil || propRef.Value == nil {
			continue
		}
		prop := propRef.Value

		opt := schema.ToolOption{
			PropertyName: name,
			FlagName:     schema.ToFlagName(name),
			Description:  prop.Description,
			Required:     requiredSet[name],
			GoType:       mapSchemaType(prop),
		}

		if prop.Default != nil {
			opt.DefaultValue = prop.Default
		}

		if len(prop.Enum) > 0 {
			for _, e := range prop.Enum {
				opt.EnumValues = append(opt.EnumValues, fmt.Sprintf("%v", e))
			}
		}

		opts = append(opts, opt)
	}

	// Sort: required first, then alphabetical.
	sort.Slice(opts, func(i, j int) bool {
		if opts[i].Required != opts[j].Required {
			return opts[i].Required
		}
		return opts[i].FlagName < opts[j].FlagName
	})

	return opts
}

// fetchURL fetches a URL with optional auth headers.
func fetchURL(ctx context.Context, rawURL string, authHeaders map[string]string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, application/yaml, */*")

	for k, v := range authHeaders {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}
