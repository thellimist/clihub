package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

// StdioTransport implements the Transport interface by communicating with an
// MCP server over stdin/stdout of a child process.
type StdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	reader *bufio.Reader
	mu     sync.Mutex
}

// NewStdioTransport creates a new StdioTransport. Call Start() to spawn the process.
// The env slice should contain entries in KEY=VALUE format; these are merged with
// the current process environment.
func NewStdioTransport(command string, args []string, env []string) *StdioTransport {
	cmd := exec.Command(command, args...)
	cmd.Env = mergeEnv(os.Environ(), env)
	cmd.Stderr = os.Stderr
	return &StdioTransport{cmd: cmd}
}

// Start spawns the child process and sets up stdin/stdout pipes.
func (t *StdioTransport) Start() error {
	var err error

	t.stdin, err = t.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := t.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	t.reader = bufio.NewReader(stdout)

	if err := t.cmd.Start(); err != nil {
		return fmt.Errorf("start process: %w", err)
	}

	return nil
}

// Send writes a JSON-RPC request to the child process stdin and reads a
// JSON-RPC response from stdout. A mutex ensures only one request is in
// flight at a time.
func (t *StdioTransport) Send(_ context.Context, req *JSONRPCRequest) (*JSONRPCResponse, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Write request as a single line terminated by newline.
	data = append(data, '\n')
	if _, err := t.stdin.Write(data); err != nil {
		return nil, fmt.Errorf("write to stdin: %w", err)
	}

	// Read one line of response from stdout.
	line, err := t.reader.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("read from stdout: %w", err)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &resp, nil
}

// Close shuts down the child process by closing stdin, killing the process,
// and waiting for it to exit.
func (t *StdioTransport) Close() error {
	if t.stdin != nil {
		t.stdin.Close()
	}
	if t.cmd.Process != nil {
		t.cmd.Process.Kill()
	}
	// Wait to reap the process and avoid zombies. Ignore the error since
	// we killed the process.
	_ = t.cmd.Wait()
	return nil
}

// mergeEnv merges base environment variables with overrides. If an override
// key already exists in base, the override value wins.
func mergeEnv(base, overrides []string) []string {
	env := make(map[string]string, len(base)+len(overrides))
	order := make([]string, 0, len(base)+len(overrides))

	parseKV := func(entries []string) {
		for _, entry := range entries {
			key, _, found := cutString(entry, "=")
			if !found {
				continue
			}
			if _, exists := env[key]; !exists {
				order = append(order, key)
			}
			env[key] = entry
		}
	}

	parseKV(base)
	parseKV(overrides)

	result := make([]string, 0, len(order))
	for _, key := range order {
		result = append(result, env[key])
	}
	return result
}

// cutString splits s around the first instance of sep, returning the text
// before and after sep. If sep is not found, it returns s, "", false.
func cutString(s, sep string) (before, after string, found bool) {
	for i := 0; i+len(sep) <= len(s); i++ {
		if s[i:i+len(sep)] == sep {
			return s[:i], s[i+len(sep):], true
		}
	}
	return s, "", false
}
