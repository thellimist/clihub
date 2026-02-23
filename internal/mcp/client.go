package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

// Client is a high-level MCP protocol client that uses a Transport to
// communicate with an MCP server.
type Client struct {
	transport Transport
	nextID    int
}

// NewClient creates a new MCP client using the given transport.
func NewClient(transport Transport) *Client {
	return &Client{
		transport: transport,
		nextID:    1,
	}
}

// allocID returns the next request ID and increments the counter.
func (c *Client) allocID() int {
	id := c.nextID
	c.nextID++
	return id
}

// Initialize performs the MCP initialize handshake. It sends an initialize
// request with the given client name and version, then sends a
// notifications/initialized notification.
func (c *Client) Initialize(ctx context.Context, clientName, clientVersion string) (*InitializeResult, error) {
	params := InitializeParams{
		ProtocolVersion: "2024-11-05",
		Capabilities:    map[string]any{},
		ClientInfo: ClientInfo{
			Name:    clientName,
			Version: clientVersion,
		},
	}

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      c.allocID(),
		Method:  "initialize",
		Params:  params,
	}

	resp, err := c.transport.Send(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("initialize: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("initialize: server error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	var result InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("initialize: unmarshal result: %w", err)
	}

	// Send the initialized notification. This is fire-and-forget; we don't
	// require a response but the transport will return one which we ignore.
	notif := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      c.allocID(),
		Method:  "notifications/initialized",
	}
	// Ignore errors from the notification — some transports may not return a
	// response for notifications.
	_, _ = c.transport.Send(ctx, notif)

	return &result, nil
}

// ListTools sends a tools/list request and returns the available tools.
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      c.allocID(),
		Method:  "tools/list",
	}

	resp, err := c.transport.Send(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("tools/list: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("tools/list: server error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	var result ToolsListResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("tools/list: unmarshal result: %w", err)
	}

	return result.Tools, nil
}

// Close closes the underlying transport.
func (c *Client) Close() error {
	return c.transport.Close()
}
