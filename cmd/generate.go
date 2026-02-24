package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/clihub/clihub/internal/auth"
	"github.com/clihub/clihub/internal/codegen"
	"github.com/clihub/clihub/internal/compile"
	"github.com/clihub/clihub/internal/gocheck"
	"github.com/clihub/clihub/internal/nameutil"
	"github.com/clihub/clihub/internal/oauth"
	"github.com/clihub/clihub/internal/schema"
	"github.com/clihub/clihub/internal/toolfilter"
	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
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
	flagOAuth           bool
	flagClientID        string
	flagClientSecret    string
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

  # With OAuth authentication (interactive browser flow)
  clihub generate --url https://mcp.notion.com/mcp --oauth

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
	f.BoolVar(&flagOAuth, "oauth", false, "use OAuth for authentication (interactive browser flow)")
	f.StringVar(&flagClientID, "client-id", "", "pre-registered OAuth client ID (use with --oauth)")
	f.StringVar(&flagClientSecret, "client-secret", "", "pre-registered OAuth client secret (use with --oauth)")
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

	// Determine target for error messages
	target := flagURL
	if target == "" {
		target = flagStdio
	}

	// Create MCP client via mcp-go SDK
	verbose("Connecting to MCP server...")
	mcpClient, err := createMCPClient()
	if err != nil {
		return err
	}
	defer mcpClient.Close()

	// Start transport (required for HTTP; stdio auto-starts in NewStdioMCPClient)
	timeout := time.Duration(flagTimeout) * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if flagURL != "" {
		if err := mcpClient.Start(ctx); err != nil {
			return fmt.Errorf("failed to connect to MCP server at %s: %s", target, err)
		}
	}

	// REQ-19/20: Initialize handshake
	verbose("Performing MCP handshake...")
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "clihub",
		Version: appVersion,
	}
	initReq.Params.Capabilities = mcp.ClientCapabilities{}

	_, err = mcpClient.Initialize(ctx, initReq)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("MCP server did not respond within %dms", flagTimeout)
		}
		// If OAuth is enabled and we got an auth error, try the OAuth flow
		if flagOAuth && isAuthError(err) {
			verbose("Authentication required, starting OAuth flow...")
			token, oauthErr := runOAuthFlow(ctx)
			if oauthErr != nil {
				return fmt.Errorf("OAuth authentication failed: %w", oauthErr)
			}
			// Recreate client with the new token
			mcpClient.Close()
			mcpClient, err = createHTTPClient(token)
			if err != nil {
				return err
			}
			defer mcpClient.Close()
			if err := mcpClient.Start(ctx); err != nil {
				return fmt.Errorf("failed to connect after OAuth: %s", err)
			}
			_, err = mcpClient.Initialize(ctx, initReq)
			if err != nil {
				return fmt.Errorf("MCP server at %s did not complete initialization handshake\n  %s", target, err)
			}
		} else {
			return fmt.Errorf("MCP server at %s did not complete initialization handshake\n  %s", target, err)
		}
	}
	verbose("Handshake complete")

	// REQ-23: Discover tools
	verbose("Discovering tools...")
	toolsResult, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("MCP server did not respond within %dms", flagTimeout)
		}
		return fmt.Errorf("failed to connect to MCP server at %s: %s", target, err)
	}

	tools := toolsResult.Tools

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

	// Parse platforms
	platforms, err := compile.ParsePlatforms(flagPlatform)
	if err != nil {
		return err
	}
	multiPlatform := len(platforms) > 1

	// Compile for each platform
	var binaries []string
	for _, p := range platforms {
		if flagVerbose {
			fmt.Printf("Compiling %s...", p)
		}

		start := time.Now()
		binaryPath, err := compile.Compile(projectDir, flagOutput, cliName, p, multiPlatform)
		elapsed := time.Since(start)

		if err != nil {
			if flagVerbose {
				fmt.Println(" failed")
			}
			cleanupDir = ""
			return fmt.Errorf("%s\nGenerated source preserved at: %s", err, projectDir)
		}

		if flagVerbose {
			fmt.Printf(" done (%.1fs)\n", elapsed.Seconds())
		}
		binaries = append(binaries, binaryPath)
	}

	// Smart smoke test: only run if host platform is in the target list
	hostGOOS, hostGOARCH := compile.CurrentPlatform()
	hostPlatform := hostGOOS + "/" + hostGOARCH
	var hostBinary string
	for i, p := range platforms {
		if p.String() == hostPlatform {
			hostBinary = binaries[i]
			break
		}
	}

	if hostBinary != "" {
		verbose("Running smoke test...")
		if err := compile.SmokeTest(hostBinary); err != nil {
			cleanupDir = ""
			return fmt.Errorf("%s\nGenerated source preserved at: %s", err, projectDir)
		}
		verbose("Smoke test passed")
	} else {
		verbose("Warning: smoke test skipped — no binary for host platform (%s)", hostPlatform)
	}

	// Print summary
	if !flagQuiet {
		fmt.Printf("Generated %s from %s (%d tools, %d platform", cliName, target, len(finalTools), len(platforms))
		if len(platforms) != 1 {
			fmt.Print("s")
		}
		fmt.Println(")")
		fmt.Println("Binaries:")
		for _, b := range binaries {
			fmt.Printf("  %s\n", b)
		}
	}

	return nil
}

// processToolSchemas converts mcp-go tools to codegen tool definitions.
func processToolSchemas(tools []mcp.Tool) ([]codegen.ToolDef, error) {
	defs := make([]codegen.ToolDef, 0, len(tools))
	for _, t := range tools {
		// Marshal the mcp-go ToolInputSchema to json.RawMessage for schema processing
		inputSchemaJSON, err := json.Marshal(t.InputSchema)
		if err != nil {
			return nil, fmt.Errorf("schema marshaling for tool %q: %w", t.Name, err)
		}

		options, err := schema.ExtractOptions(inputSchemaJSON)
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

// createMCPClient creates the appropriate mcp-go client based on flags.
func createMCPClient() (*mcpclient.Client, error) {
	if flagURL != "" {
		// Resolve auth token
		token := flagAuthToken
		if token == "" && flagOAuth {
			// Try cached token first
			credPath := auth.DefaultCredentialsPath()
			creds, err := auth.LoadCredentials(credPath)
			if err == nil {
				token = auth.GetToken(creds, flagURL)
			}
		}
		return createHTTPClient(token)
	}

	// Stdio transport
	parts, err := nameutil.SplitCommand(flagStdio)
	if err != nil {
		return nil, fmt.Errorf("invalid --stdio command: %s", err)
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("--stdio command is empty")
	}

	command := parts[0]
	var cmdArgs []string
	if len(parts) > 1 {
		cmdArgs = parts[1:]
	}

	c, err := mcpclient.NewStdioMCPClient(command, flagEnv, cmdArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MCP server at %s: %s", flagStdio, err)
	}
	return c, nil
}

// createHTTPClient creates an HTTP-based mcp-go client with the given auth token.
func createHTTPClient(token string) (*mcpclient.Client, error) {
	var opts []transport.StreamableHTTPCOption
	if token != "" {
		opts = append(opts, transport.WithHTTPHeaders(map[string]string{
			"Authorization": "Bearer " + token,
		}))
	}
	c, err := mcpclient.NewStreamableHttpClient(flagURL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client for %s: %s", flagURL, err)
	}
	return c, nil
}

// runOAuthFlow executes the OAuth authentication flow and returns the access token.
func runOAuthFlow(ctx context.Context) (string, error) {
	credPath := auth.DefaultCredentialsPath()
	var verboseFn func(string, ...interface{})
	if flagVerbose {
		verboseFn = func(format string, args ...interface{}) {
			verbose(format, args...)
		}
	}

	cfg := oauth.FlowConfig{
		ServerURL:    flagURL,
		ClientID:     flagClientID,
		ClientSecret: flagClientSecret,
		Verbose:      verboseFn,
	}

	tokens, err := oauth.Authenticate(ctx, cfg)
	if err != nil {
		return "", err
	}

	// Save tokens to credential store
	creds, loadErr := auth.LoadCredentials(credPath)
	if loadErr == nil {
		auth.SetOAuthTokens(creds, flagURL, tokens.AccessToken, tokens.RefreshToken, tokens.ClientID, tokens.Scope, &tokens.ExpiresAt)
		if saveErr := auth.SaveCredentials(credPath, creds); saveErr == nil {
			info("OAuth tokens saved to %s", credPath)
		}
	}

	return tokens.AccessToken, nil
}

// isAuthError checks if an error indicates an authentication failure.
func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "401") ||
		strings.Contains(msg, "Unauthorized") ||
		strings.Contains(msg, "unauthorized")
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

	if flagOAuth && flagStdio != "" {
		return fmt.Errorf("--oauth is not supported for stdio servers")
	}

	if (flagClientID != "" || flagClientSecret != "") && !flagOAuth {
		return fmt.Errorf("--client-id and --client-secret require --oauth")
	}

	for _, env := range flagEnv {
		if !strings.Contains(env, "=") {
			return fmt.Errorf("invalid --env format %q: expected KEY=VALUE", env)
		}
	}

	return nil
}
