package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var (
	flagURL             string
	flagStdio           string
	flagName            string
	flagOutput          string
	flagPlatform        string
	flagIncludeTools    string
	flagExcludeTools    string
	flagAuthToken       string
	flagTimeout         int
	flagEnv             []string
	flagSaveCredentials bool
	flagVerbose         bool
	flagQuiet           bool
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a CLI from an MCP server",
	Long: `Generate a compiled CLI binary from an MCP server.

clihub connects to an MCP server, discovers its tools, generates a Go CLI
with one subcommand per tool, and compiles it to a native binary.

Examples:
  # From an HTTP MCP server
  clihub generate --url https://mcp.linear.app/mcp

  # From a stdio MCP server
  clihub generate --stdio "npx @modelcontextprotocol/server-github"

  # With authentication
  clihub generate --url https://mcp.example.com/mcp --auth-token $TOKEN

  # Cross-compile for multiple platforms
  clihub generate --url https://mcp.example.com/mcp --platform linux/amd64,darwin/arm64

  # Filter tools
  clihub generate --url https://mcp.example.com/mcp --include-tools create_issue,list_issues

  # Pass environment variables to stdio server
  clihub generate --stdio "npx server" --env GITHUB_TOKEN=$TOKEN --env DEBUG=true`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runGenerate,
}

func init() {
	f := generateCmd.Flags()
	f.StringVar(&flagURL, "url", "", "Streamable HTTP URL of an MCP server")
	f.StringVar(&flagStdio, "stdio", "", "shell command that spawns a local MCP server via stdin/stdout")
	f.StringVar(&flagName, "name", "", "override the auto-inferred name for the generated CLI")
	f.StringVar(&flagOutput, "output", "./out/", "directory where compiled binaries are written")
	f.StringVar(&flagPlatform, "platform", runtime.GOOS+"/"+runtime.GOARCH, "comma-separated GOOS/GOARCH pairs or 'all'")
	f.StringVar(&flagIncludeTools, "include-tools", "", "only include these tools (comma-separated)")
	f.StringVar(&flagExcludeTools, "exclude-tools", "", "exclude these tools (comma-separated)")
	f.StringVar(&flagAuthToken, "auth-token", "", "bearer token for authenticated MCP servers")
	f.IntVar(&flagTimeout, "timeout", 30000, "timeout in milliseconds for MCP connection")
	f.StringSliceVar(&flagEnv, "env", nil, "environment variables for stdio servers (KEY=VALUE, repeatable)")
	f.BoolVar(&flagSaveCredentials, "save-credentials", false, "persist auth token to ~/.clihub/credentials.json")
	f.BoolVar(&flagVerbose, "verbose", false, "show detailed progress during generation")
	f.BoolVar(&flagQuiet, "quiet", false, "suppress all output except errors")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	if err := validateFlags(); err != nil {
		return err
	}

	// Placeholder: full implementation in Plan 05
	fmt.Fprintf(os.Stderr, "Error: generate command not yet fully implemented\n")
	return fmt.Errorf("generate command not yet fully implemented")
}

func validateFlags() error {
	// REQ-16: Must provide either --url or --stdio
	if flagURL == "" && flagStdio == "" {
		return fmt.Errorf("provide --url or --stdio to specify the MCP server")
	}

	// REQ-60: Mutual exclusivity checks
	if flagURL != "" && flagStdio != "" {
		return fmt.Errorf("--url and --stdio cannot be used together")
	}

	if flagIncludeTools != "" && flagExcludeTools != "" {
		return fmt.Errorf("--include-tools and --exclude-tools cannot be used together")
	}

	if flagVerbose && flagQuiet {
		return fmt.Errorf("--verbose and --quiet cannot be used together")
	}

	// Validate --env format
	for _, env := range flagEnv {
		if !strings.Contains(env, "=") {
			return fmt.Errorf("invalid --env format %q: expected KEY=VALUE", env)
		}
	}

	return nil
}
