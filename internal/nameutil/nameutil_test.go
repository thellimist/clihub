package nameutil

import (
	"testing"
)

func TestSplitCommand(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{
			name:  "basic splitting",
			input: "npx -y @org/server",
			want:  []string{"npx", "-y", "@org/server"},
		},
		{
			name:  "single quotes",
			input: "sh -c 'echo test'",
			want:  []string{"sh", "-c", "echo test"},
		},
		{
			name:  "double quotes",
			input: `sh -c "echo test"`,
			want:  []string{"sh", "-c", "echo test"},
		},
		{
			name:  "mixed quotes",
			input: `sh -c "it's a test"`,
			want:  []string{"sh", "-c", "it's a test"},
		},
		{
			name:  "backslash escaping outside quotes",
			input: `echo hello\ world`,
			want:  []string{"echo", "hello world"},
		},
		{
			name:  "backslash inside double quotes escapes special chars",
			input: `echo "say \"hello\""`,
			want:  []string{"echo", `say "hello"`},
		},
		{
			name:  "backslash inside double quotes literal for non-special",
			input: `echo "hello\nworld"`,
			want:  []string{"echo", `hello\nworld`},
		},
		{
			name:    "unterminated single quote",
			input:   "echo 'unterminated",
			wantErr: true,
		},
		{
			name:    "unterminated double quote",
			input:   `echo "unterminated`,
			wantErr: true,
		},
		{
			name:  "empty input",
			input: "",
			want:  nil,
		},
		{
			name:  "whitespace only",
			input: "   \t  ",
			want:  nil,
		},
		{
			name:  "extra whitespace between tokens",
			input: "npx   -y   server",
			want:  []string{"npx", "-y", "server"},
		},
		{
			name:  "empty quoted string produces token",
			input: `echo ""`,
			want:  []string{"echo", ""},
		},
		{
			name:  "single quoted empty string",
			input: "echo ''",
			want:  []string{"echo", ""},
		},
		{
			name:  "tab separated",
			input: "cmd\targ1\targ2",
			want:  []string{"cmd", "arg1", "arg2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SplitCommand(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("SplitCommand(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("SplitCommand(%q) unexpected error: %v", tt.input, err)
			}
			if !sliceEqual(got, tt.want) {
				t.Errorf("SplitCommand(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestInferFromURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "mcp.linear.app",
			input: "https://mcp.linear.app/mcp",
			want:  "linear",
		},
		{
			name:  "api.example.com",
			input: "https://api.example.com/v1",
			want:  "example",
		},
		{
			name:  "www.myservice.io",
			input: "https://www.myservice.io/",
			want:  "myservice",
		},
		{
			name:  "plain hostname with com",
			input: "https://github.com/repo",
			want:  "github",
		},
		{
			name:  "subdomain",
			input: "https://mcp.stripe.dev/sse",
			want:  "stripe",
		},
		{
			name:  "localhost fallback to path",
			input: "http://localhost:3000/myservice",
			want:  "myservice",
		},
		{
			name:  "IP address fallback to path",
			input: "http://127.0.0.1:8080/api/v2",
			want:  "api",
		},
		{
			name:  "no meaningful hostname, path fallback",
			input: "https://mcp.com/toolset",
			want:  "toolset",
		},
		{
			name:  "multiple subdomains",
			input: "https://mcp.api.linear.app/sse",
			want:  "linear",
		},
		{
			name:  "invalid URL",
			input: "not-a-url",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferFromURL(tt.input)
			if got != tt.want {
				t.Errorf("inferFromURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestInferFromCommand(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "scoped server-github",
			input: "npx @modelcontextprotocol/server-github",
			want:  "github",
		},
		{
			name:  "mcp-server-postgres",
			input: "npx mcp-server-postgres",
			want:  "postgres",
		},
		{
			name:  "scoped with version suffix",
			input: "npx @org/server-redis@latest",
			want:  "redis",
		},
		{
			name:  "mcp- prefix",
			input: "npx mcp-toolbox",
			want:  "toolbox",
		},
		{
			name:  "version number suffix",
			input: "npx @scope/mcp-server-db@1.2.3",
			want:  "db",
		},
		{
			name:  "node script",
			input: "node server.js",
			want:  "server",
		},
		{
			name:  "python script",
			input: "python -m mcp_server_sqlite",
			want:  "sqlite",
		},
		{
			name:  "plain command",
			input: "my-custom-tool",
			want:  "my-custom-tool",
		},
		{
			name:  "npx with -y flag and scoped",
			input: "npx -y @org/mcp-server-test",
			want:  "test",
		},
		{
			name:  "empty command",
			input: "",
			want:  "",
		},
		{
			name:  "server- prefix strip",
			input: "npx server-mysql",
			want:  "mysql",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferFromCommand(tt.input)
			if got != tt.want {
				t.Errorf("inferFromCommand(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "basic lowercase",
			input: "Hello World",
			want:  "hello-world",
		},
		{
			name:  "special characters",
			input: "my@tool!v2",
			want:  "my-tool-v2",
		},
		{
			name:  "consecutive dashes",
			input: "hello---world",
			want:  "hello-world",
		},
		{
			name:  "leading and trailing dashes",
			input: "--hello--world--",
			want:  "hello-world",
		},
		{
			name:  "already slugified",
			input: "my-tool",
			want:  "my-tool",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only special chars",
			input: "!@#$%",
			want:  "",
		},
		{
			name:  "underscores",
			input: "mcp_server_sqlite",
			want:  "mcp-server-sqlite",
		},
		{
			name:  "mixed case and dots",
			input: "Server.JS",
			want:  "server-js",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Slugify(tt.input)
			if got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestInferName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		isURL bool
		want  string
	}{
		{
			name:  "dispatch to URL",
			input: "https://mcp.linear.app/mcp",
			isURL: true,
			want:  "linear",
		},
		{
			name:  "dispatch to command",
			input: "npx @modelcontextprotocol/server-github",
			isURL: false,
			want:  "github",
		},
		{
			name:  "URL with path fallback",
			input: "http://localhost:3000/myservice",
			isURL: true,
			want:  "myservice",
		},
		{
			name:  "command with mcp-server prefix",
			input: "npx mcp-server-postgres",
			isURL: false,
			want:  "postgres",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InferName(tt.input, tt.isURL)
			if got != tt.want {
				t.Errorf("InferName(%q, %v) = %q, want %q", tt.input, tt.isURL, got, tt.want)
			}
		})
	}
}

// sliceEqual compares two string slices for equality.
func sliceEqual(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
