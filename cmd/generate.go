package cmd

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/clihub/clihub/internal/gocheck"
	"github.com/clihub/clihub/internal/mcp"
	"github.com/clihub/clihub/internal/nameutil"
	"github.com/clihub/clihub/internal/toolfilter"
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

	// REQ-01, REQ-66: Check Go toolchain
	verbose("Checking Go toolchain...")
	goVersion, err := gocheck.Check()
	if err != nil {
		return err
	}
	verbose("Found %s", goVersion)

	// REQ-24: Warn if --auth-token used with --stdio
	if flagAuthToken != "" && flagStdio != "" {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: --auth-token is ignored for stdio servers. Use --env to pass credentials\n")
	}

	// Create MCP transport
	verbose("Connecting to MCP server...")
	transport, target, err := createTransport()
	if err != nil {
		return err
	}

	// Create client and run discovery with timeout
	client := mcp.NewClient(transport)
	defer client.Close()

	timeout := time.Duration(flagTimeout) * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// REQ-19/20: Initialize handshake
	verbose("Performing MCP handshake...")
	_, err = client.Initialize(ctx, "clihub", appVersion)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("MCP server did not respond within %dms", flagTimeout)
		}
		return fmt.Errorf("MCP server at %s did not complete initialization handshake", target)
	}
	verbose("Handshake complete")

	// REQ-23: Discover tools
	verbose("Discovering tools...")
	tools, err := client.ListTools(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("MCP server did not respond within %dms", flagTimeout)
		}
		return fmt.Errorf("failed to connect to MCP server at %s: %s", target, err)
	}

	// REQ-63: No tools found
	if len(tools) == 0 {
		return fmt.Errorf("MCP server returned no tools")
	}
	verbose("Discovered %d tools", len(tools))

	// REQ-33-35: Apply tool filtering
	include := toolfilter.ParseToolList(flagIncludeTools)
	exclude := toolfilter.ParseToolList(flagExcludeTools)

	// Convert mcp.Tool to toolfilter.Tool
	filterTools := make([]toolfilter.Tool, len(tools))
	for i, t := range tools {
		filterTools[i] = toolfilter.Tool{Name: t.Name, Description: t.Description}
	}

	filtered, err := toolfilter.FilterTools(filterTools, include, exclude)
	if err != nil {
		return err
	}

	// Map filtered results back to mcp.Tool
	filteredSet := make(map[string]bool, len(filtered))
	for _, ft := range filtered {
		filteredSet[ft.Name] = true
	}
	var finalTools []mcp.Tool
	for _, t := range tools {
		if filteredSet[t.Name] {
			finalTools = append(finalTools, t)
		}
	}

	if len(include) > 0 || len(exclude) > 0 {
		verbose("After filtering: %d tools", len(finalTools))
	}

	// REQ-30-32: Infer name
	cliName := flagName
	if cliName == "" {
		isURL := flagURL != ""
		source := flagURL
		if !isURL {
			source = flagStdio
		}
		cliName = nameutil.InferName(source, isURL)
		if cliName == "" {
			cliName = "mcp-cli"
		}
		verbose("Inferred CLI name: %s", cliName)
	}

	// Print discovery summary (unless quiet)
	if !flagQuiet {
		fmt.Printf("Discovered %d tools from %s\n", len(finalTools), target)
		fmt.Printf("CLI name: %s\n", cliName)
		fmt.Printf("Output: %s\n", flagOutput)
		fmt.Println()
		fmt.Println("Tools:")
		for _, t := range finalTools {
			desc := t.Description
			if len(desc) > 72 {
				desc = desc[:69] + "..."
			}
			if desc != "" {
				fmt.Printf("  - %s: %s\n", t.Name, desc)
			} else {
				fmt.Printf("  - %s\n", t.Name)
			}
		}
		fmt.Println()
		fmt.Println("Phase 1 complete: tool discovery successful.")
		fmt.Println("Code generation and compilation will be available in Phase 2.")
	}

	return nil
}

// createTransport creates the appropriate MCP transport based on flags.
func createTransport() (mcp.Transport, string, error) {
	if flagURL != "" {
		transport := mcp.NewHTTPTransport(flagURL, flagAuthToken)
		return transport, flagURL, nil
	}

	// Stdio transport
	parts, err := nameutil.SplitCommand(flagStdio)
	if err != nil {
		return nil, "", fmt.Errorf("invalid --stdio command: %s", err)
	}
	if len(parts) == 0 {
		return nil, "", fmt.Errorf("--stdio command is empty")
	}

	command := parts[0]
	var cmdArgs []string
	if len(parts) > 1 {
		cmdArgs = parts[1:]
	}

	transport := mcp.NewStdioTransport(command, cmdArgs, flagEnv)
	if err := transport.Start(); err != nil {
		return nil, "", fmt.Errorf("failed to connect to MCP server at %s: %s", flagStdio, err)
	}
	return transport, flagStdio, nil
}

// verbose prints a message if --verbose is set.
func verbose(format string, args ...interface{}) {
	if flagVerbose {
		fmt.Printf(format+"\n", args...)
	}
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
