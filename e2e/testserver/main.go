// Package main implements a minimal MCP stdio server for E2E testing.
// Each tool echoes back the received params as JSON text content,
// allowing tests to assert exactly which params were sent.
package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	s := server.NewMCPServer("echo-params", "1.0.0")

	// echo_params: echoes all received arguments as JSON
	s.AddTool(
		mcp.NewTool("echo_params",
			mcp.WithDescription("Echoes all received params as JSON"),
			mcp.WithString("org_id", mcp.Description("Organization ID")),
			mcp.WithString("project_id", mcp.Description("Project ID")),
			mcp.WithString("query", mcp.Description("Search query")),
		),
		echoHandler,
	)

	// create_item: similar tool but named differently for per-tool closure testing
	s.AddTool(
		mcp.NewTool("create_item",
			mcp.WithDescription("Creates an item (echoes params)"),
			mcp.WithString("org_id", mcp.Description("Organization ID")),
			mcp.WithString("project_id", mcp.Description("Project ID")),
			mcp.WithString("title", mcp.Description("Item title")),
		),
		echoHandler,
	)

	// list_items: another tool to test that tool-specific params don't leak
	s.AddTool(
		mcp.NewTool("list_items",
			mcp.WithDescription("Lists items (echoes params)"),
			mcp.WithString("org_id", mcp.Description("Organization ID")),
			mcp.WithString("query", mcp.Description("Search query")),
		),
		echoHandler,
	)

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("server error: %v\n", err)
	}
}

func echoHandler(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	data, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("marshal args: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}
