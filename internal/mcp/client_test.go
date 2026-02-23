package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

// mockTransport records requests and returns canned responses.
type mockTransport struct {
	requests  []*JSONRPCRequest
	responses []*JSONRPCResponse
	errors    []error
	callIndex int
	closed    bool
}

func (m *mockTransport) Send(_ context.Context, req *JSONRPCRequest) (*JSONRPCResponse, error) {
	m.requests = append(m.requests, req)
	idx := m.callIndex
	m.callIndex++
	if idx < len(m.errors) && m.errors[idx] != nil {
		return nil, m.errors[idx]
	}
	if idx < len(m.responses) {
		return m.responses[idx], nil
	}
	return nil, fmt.Errorf("no response configured for call %d", idx)
}

func (m *mockTransport) Close() error {
	m.closed = true
	return nil
}

func TestInitialize(t *testing.T) {
	initResult := InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities:    map[string]any{"tools": map[string]any{}},
		ServerInfo: ServerInfo{
			Name:    "test-server",
			Version: "1.0.0",
		},
	}
	resultJSON, err := json.Marshal(initResult)
	if err != nil {
		t.Fatalf("marshal init result: %v", err)
	}

	mock := &mockTransport{
		responses: []*JSONRPCResponse{
			{JSONRPC: "2.0", ID: 1, Result: resultJSON},
			{JSONRPC: "2.0", ID: 2},  // response for notifications/initialized
		},
	}

	client := NewClient(mock)
	result, err := client.Initialize(context.Background(), "clihub", "0.1.0")
	if err != nil {
		t.Fatalf("Initialize() error: %v", err)
	}

	// Verify the result was parsed correctly.
	if result.ProtocolVersion != "2024-11-05" {
		t.Errorf("ProtocolVersion = %q, want %q", result.ProtocolVersion, "2024-11-05")
	}
	if result.ServerInfo.Name != "test-server" {
		t.Errorf("ServerInfo.Name = %q, want %q", result.ServerInfo.Name, "test-server")
	}
	if result.ServerInfo.Version != "1.0.0" {
		t.Errorf("ServerInfo.Version = %q, want %q", result.ServerInfo.Version, "1.0.0")
	}

	// Verify two requests were sent: initialize + notifications/initialized.
	if len(mock.requests) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(mock.requests))
	}

	// Verify the initialize request.
	initReq := mock.requests[0]
	if initReq.Method != "initialize" {
		t.Errorf("first request method = %q, want %q", initReq.Method, "initialize")
	}
	if initReq.JSONRPC != "2.0" {
		t.Errorf("first request jsonrpc = %q, want %q", initReq.JSONRPC, "2.0")
	}

	// Verify the initialize params.
	paramsJSON, err := json.Marshal(initReq.Params)
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}
	var params InitializeParams
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}
	if params.ProtocolVersion != "2024-11-05" {
		t.Errorf("params.ProtocolVersion = %q, want %q", params.ProtocolVersion, "2024-11-05")
	}
	if params.ClientInfo.Name != "clihub" {
		t.Errorf("params.ClientInfo.Name = %q, want %q", params.ClientInfo.Name, "clihub")
	}
	if params.ClientInfo.Version != "0.1.0" {
		t.Errorf("params.ClientInfo.Version = %q, want %q", params.ClientInfo.Version, "0.1.0")
	}

	// Verify the notifications/initialized request.
	notifReq := mock.requests[1]
	if notifReq.Method != "notifications/initialized" {
		t.Errorf("second request method = %q, want %q", notifReq.Method, "notifications/initialized")
	}
}

func TestListTools(t *testing.T) {
	toolsResult := ToolsListResult{
		Tools: []Tool{
			{
				Name:        "get-weather",
				Description: "Get weather for a location",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}},"required":["location"]}`),
			},
		},
	}
	resultJSON, err := json.Marshal(toolsResult)
	if err != nil {
		t.Fatalf("marshal tools result: %v", err)
	}

	mock := &mockTransport{
		responses: []*JSONRPCResponse{
			{JSONRPC: "2.0", ID: 1, Result: resultJSON},
		},
	}

	client := NewClient(mock)
	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools() error: %v", err)
	}

	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0].Name != "get-weather" {
		t.Errorf("tool name = %q, want %q", tools[0].Name, "get-weather")
	}
	if tools[0].Description != "Get weather for a location" {
		t.Errorf("tool description = %q, want %q", tools[0].Description, "Get weather for a location")
	}

	// Verify the request.
	if len(mock.requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(mock.requests))
	}
	if mock.requests[0].Method != "tools/list" {
		t.Errorf("request method = %q, want %q", mock.requests[0].Method, "tools/list")
	}
}

func TestListToolsMultiple(t *testing.T) {
	toolsResult := ToolsListResult{
		Tools: []Tool{
			{
				Name:        "get-weather",
				Description: "Get weather for a location",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}}}`),
			},
			{
				Name:        "search",
				Description: "Search the web",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"},"limit":{"type":"integer"}},"required":["query"]}`),
			},
			{
				Name:        "calculate",
				Description: "",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"expression":{"type":"string"}}}`),
			},
		},
	}
	resultJSON, err := json.Marshal(toolsResult)
	if err != nil {
		t.Fatalf("marshal tools result: %v", err)
	}

	mock := &mockTransport{
		responses: []*JSONRPCResponse{
			{JSONRPC: "2.0", ID: 1, Result: resultJSON},
		},
	}

	client := NewClient(mock)
	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools() error: %v", err)
	}

	if len(tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(tools))
	}

	// Verify each tool.
	expected := []struct {
		name        string
		description string
	}{
		{"get-weather", "Get weather for a location"},
		{"search", "Search the web"},
		{"calculate", ""},
	}
	for i, exp := range expected {
		if tools[i].Name != exp.name {
			t.Errorf("tool[%d].Name = %q, want %q", i, tools[i].Name, exp.name)
		}
		if tools[i].Description != exp.description {
			t.Errorf("tool[%d].Description = %q, want %q", i, tools[i].Description, exp.description)
		}
		// Verify InputSchema is valid JSON.
		if !json.Valid(tools[i].InputSchema) {
			t.Errorf("tool[%d].InputSchema is not valid JSON", i)
		}
	}
}

func TestInitializeJSONRPCError(t *testing.T) {
	mock := &mockTransport{
		responses: []*JSONRPCResponse{
			{
				JSONRPC: "2.0",
				ID:      1,
				Error: &JSONRPCError{
					Code:    -32600,
					Message: "invalid request",
				},
			},
		},
	}

	client := NewClient(mock)
	_, err := client.Initialize(context.Background(), "clihub", "0.1.0")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "initialize: server error -32600: invalid request" {
		t.Errorf("error = %q, want %q", got, "initialize: server error -32600: invalid request")
	}
}

func TestListToolsJSONRPCError(t *testing.T) {
	mock := &mockTransport{
		responses: []*JSONRPCResponse{
			{
				JSONRPC: "2.0",
				ID:      1,
				Error: &JSONRPCError{
					Code:    -32601,
					Message: "method not found",
				},
			},
		},
	}

	client := NewClient(mock)
	_, err := client.ListTools(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "tools/list: server error -32601: method not found" {
		t.Errorf("error = %q, want %q", got, "tools/list: server error -32601: method not found")
	}
}

func TestInitializeTransportError(t *testing.T) {
	mock := &mockTransport{
		errors: []error{fmt.Errorf("connection refused")},
	}

	client := NewClient(mock)
	_, err := client.Initialize(context.Background(), "clihub", "0.1.0")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "initialize: connection refused" {
		t.Errorf("error = %q, want %q", got, "initialize: connection refused")
	}
}

func TestListToolsTransportError(t *testing.T) {
	mock := &mockTransport{
		errors: []error{fmt.Errorf("timeout")},
	}

	client := NewClient(mock)
	_, err := client.ListTools(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "tools/list: timeout" {
		t.Errorf("error = %q, want %q", got, "tools/list: timeout")
	}
}

func TestClientClose(t *testing.T) {
	mock := &mockTransport{}
	client := NewClient(mock)

	if err := client.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
	if !mock.closed {
		t.Error("expected transport to be closed")
	}
}

func TestClientIDIncrement(t *testing.T) {
	toolsResult := ToolsListResult{Tools: []Tool{}}
	resultJSON, _ := json.Marshal(toolsResult)

	mock := &mockTransport{
		responses: []*JSONRPCResponse{
			{JSONRPC: "2.0", ID: 1, Result: resultJSON},
			{JSONRPC: "2.0", ID: 2, Result: resultJSON},
			{JSONRPC: "2.0", ID: 3, Result: resultJSON},
		},
	}

	client := NewClient(mock)
	ctx := context.Background()

	_, _ = client.ListTools(ctx)
	_, _ = client.ListTools(ctx)
	_, _ = client.ListTools(ctx)

	if len(mock.requests) != 3 {
		t.Fatalf("expected 3 requests, got %d", len(mock.requests))
	}

	for i, req := range mock.requests {
		expectedID := i + 1
		if req.ID != expectedID {
			t.Errorf("request[%d].ID = %d, want %d", i, req.ID, expectedID)
		}
	}
}
