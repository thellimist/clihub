package mcp

import "context"

// Transport defines the interface for sending JSON-RPC requests to an MCP server.
type Transport interface {
	// Send sends a JSON-RPC request and returns the response.
	Send(ctx context.Context, req *JSONRPCRequest) (*JSONRPCResponse, error)
	// Close releases any resources held by the transport.
	Close() error
}
