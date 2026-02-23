package cmd

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/clihub/clihub/internal/auth"
	"github.com/clihub/clihub/internal/codegen"
	"github.com/clihub/clihub/internal/compile"
	"github.com/clihub/clihub/internal/gocheck"
	"github.com/clihub/clihub/internal/mcp"
	"github.com/clihub/clihub/internal/nameutil"
	"github.com/clihub/clihub/internal/schema"
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

	filterTools := make([]toolfilter.Tool, len(tools))
	for i, t := range tools {
		filterTools[i] = toolfilter.Tool{Name: t.Name, Description: t.Description}
	}

	filtered, err := toolfilter.FilterTools(filterTools, include, exclude)
	if err != nil {
		return err
	}

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

	// REQ-13a: Save credentials if requested
	if flagSaveCredentials && flagAuthToken != "" {
		serverURL := flagURL
		if serverURL != "" {
			credPath := auth.DefaultCredentialsPath()
			creds, err := auth.LoadCredentials(credPath)
			if err != nil {
				return fmt.Errorf("load credentials: %w", err)
			}
			auth.SetToken(creds, serverURL, flagAuthToken)
			if err := auth.SaveCredentials(credPath, creds); err != nil {
				return fmt.Errorf("save credentials: %w", err)
			}
			info("Saved credentials to %s", credPath)
		}
	}

	// Process tool schemas
	verbose("Processing tool schemas...")
	toolDefs, err := processToolSchemas(finalTools)
	if err != nil {
		return err
	}

	// Build codegen context
	genCtx := codegen.GenerateContext{
		CLIName:       cliName,
		Tools:         toolDefs,
		ClihubVersion: appVersion,
		IsHTTP:        flagURL != "",
	}

	if flagURL != "" {
		genCtx.ServerURL = flagURL
	} else {
		parts, _ := nameutil.SplitCommand(flagStdio)
		if len(parts) > 0 {
			genCtx.StdioCommand = parts[0]
			if len(parts) > 1 {
				genCtx.StdioArgs = parts[1:]
			}
		}
		// Extract env keys (not values)
		for _, env := range flagEnv {
			if idx := strings.Index(env, "="); idx > 0 {
				genCtx.EnvKeys = append(genCtx.EnvKeys, env[:idx])
			}
		}
	}

	// Generate Go project
	verbose("Generating Go project...")
	projectDir, err := codegen.Generate(genCtx, "")
	if err != nil {
		return fmt.Errorf("code generation failed: %w", err)
	}

	// Track temp dir for cleanup
	cleanupDir := projectDir
	defer func() {
		if cleanupDir != "" {
			os.RemoveAll(cleanupDir)
		}
	}()

	verbose("Generated project at %s", projectDir)

	// Compile for current platform
	goos, goarch := compile.CurrentPlatform()
	verbose("Compiling %s for %s/%s...", cliName, goos, goarch)

	binaryPath, err := compile.Compile(projectDir, flagOutput, cliName, goos, goarch)
	if err != nil {
		// Preserve temp dir on failure
		cleanupDir = ""
		return fmt.Errorf("%s\nGenerated source preserved at: %s", err, projectDir)
	}

	verbose("Compiled binary at %s", binaryPath)

	// Smoke test
	verbose("Running smoke test...")
	if err := compile.SmokeTest(binaryPath); err != nil {
		cleanupDir = ""
		return fmt.Errorf("%s\nGenerated source preserved at: %s", err, projectDir)
	}
	verbose("Smoke test passed")

	// Print summary
	if !flagQuiet {
		fmt.Printf("Generated %s from %s (%d tools)\n", cliName, target, len(finalTools))
		fmt.Printf("Binary: %s\n", binaryPath)
	}

	return nil
}

// processToolSchemas converts MCP tools to codegen tool definitions.
func processToolSchemas(tools []mcp.Tool) ([]codegen.ToolDef, error) {
	defs := make([]codegen.ToolDef, 0, len(tools))
	for _, t := range tools {
		options, err := schema.ExtractOptions(t.InputSchema)
		if err != nil {
			return nil, fmt.Errorf("schema processing for tool %q: %w", t.Name, err)
		}

		commandName := schema.ToFlagName(strings.ReplaceAll(t.Name, "_", "-"))
		if commandName == "" {
			commandName = t.Name
		}

		defs = append(defs, codegen.ToolDef{
			Name:        t.Name,
			CommandName: commandName,
			Description: t.Description,
			Options:     options,
		})
	}
	return defs, nil
}

// createTransport creates the appropriate MCP transport based on flags.
func createTransport() (mcp.Transport, string, error) {
	if flagURL != "" {
		transport := mcp.NewHTTPTransport(flagURL, flagAuthToken)
		return transport, flagURL, nil
	}

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

// info prints a message unless --quiet is set.
func info(format string, args ...interface{}) {
	if !flagQuiet {
		fmt.Printf(format+"\n", args...)
	}
}

func validateFlags() error {
	if flagURL == "" && flagStdio == "" {
		return fmt.Errorf("provide --url or --stdio to specify the MCP server")
	}

	if flagURL != "" && flagStdio != "" {
		return fmt.Errorf("--url and --stdio cannot be used together")
	}

	if flagIncludeTools != "" && flagExcludeTools != "" {
		return fmt.Errorf("--include-tools and --exclude-tools cannot be used together")
	}

	if flagVerbose && flagQuiet {
		return fmt.Errorf("--verbose and --quiet cannot be used together")
	}

	for _, env := range flagEnv {
		if !strings.Contains(env, "=") {
			return fmt.Errorf("invalid --env format %q: expected KEY=VALUE", env)
		}
	}

	return nil
}
