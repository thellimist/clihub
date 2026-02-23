package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// HTTPTransport implements the Transport interface using Streamable HTTP.
// It sends JSON-RPC requests as HTTP POST requests and handles both
// application/json and text/event-stream responses.
type HTTPTransport struct {
	URL        string
	AuthToken  string
	httpClient *http.Client
	sessionID  string
}

// NewHTTPTransport creates a new HTTPTransport targeting the given URL.
// If authToken is non-empty, it will be sent as a Bearer token.
func NewHTTPTransport(url, authToken string) *HTTPTransport {
	return &HTTPTransport{
		URL:        url,
		AuthToken:  authToken,
		httpClient: &http.Client{},
	}
}

// Send sends a JSON-RPC request over HTTP and returns the response.
func (t *HTTPTransport) Send(ctx context.Context, req *JSONRPCRequest) (*JSONRPCResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, t.URL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create http request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")

	if t.AuthToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+t.AuthToken)
	}

	if t.sessionID != "" {
		httpReq.Header.Set("Mcp-Session-Id", t.sessionID)
	}

	resp, err := t.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	// Capture session ID from the response if present.
	if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
		t.sessionID = sid
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("http status %d: %s", resp.StatusCode, string(respBody))
	}

	contentType := resp.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "text/event-stream") {
		return t.parseSSE(resp.Body, req.ID)
	}

	// Default: parse as application/json.
	var rpcResp JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &rpcResp, nil
}

// parseSSE reads an SSE stream and extracts the JSON-RPC response matching the
// given request ID.
func (t *HTTPTransport) parseSSE(r io.Reader, requestID int) (*JSONRPCResponse, error) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var rpcResp JSONRPCResponse
		if err := json.Unmarshal([]byte(data), &rpcResp); err != nil {
			// Skip lines that aren't valid JSON-RPC.
			continue
		}
		if rpcResp.ID == requestID {
			return &rpcResp, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read sse stream: %w", err)
	}
	return nil, fmt.Errorf("no response found for request id %d in sse stream", requestID)
}

// Close is a no-op for HTTP transport since each request is independent.
func (t *HTTPTransport) Close() error {
	return nil
}
