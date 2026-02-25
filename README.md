# clihub

<p align="center">
  <img src="assets/cli_vs_mcp.png" alt="MCP vs CLI" width="600">
</p>

Turn any [MCP server](https://modelcontextprotocol.io/) into a compiled CLI binary. One command. Designed for agents.

```
clihub generate --url https://mcp.linear.app/mcp
# → ./out/linear (6.5MB static binary, zero runtime dependencies)
```

Every tool the server exposes becomes a subcommand with flags derived from its JSON Schema. Auth is handled automatically — OAuth, bearer tokens, API keys, and more.

## Install

```bash
go install github.com/clihub/clihub@latest
```

Or build from source:

```bash
git clone https://github.com/clihub/clihub.git
cd clihub
go build -o clihub .
```

Requires Go 1.21+.

## Quick Start

### HTTP MCP server

```bash
clihub generate --url https://mcp.linear.app/mcp
./out/linear list-teams
```

### HTTP server with OAuth

```bash
clihub generate --url https://mcp.notion.com/mcp --oauth
./out/notion notion-search --query "meeting notes"
```

### Stdio MCP server

```bash
clihub generate --stdio "npx @modelcontextprotocol/server-github" \
  --env GITHUB_TOKEN=$GITHUB_TOKEN
./out/github list-repos --owner octocat
```

### Cross-compile

```bash
clihub generate --url https://mcp.stripe.com \
  --platform linux/amd64,darwin/arm64,windows/amd64
```

## How It Works

1. **Connect** to the MCP server (HTTP or stdio)
2. **Discover** all tools via `tools/list`
3. **Generate** a Go CLI with one subcommand per tool
4. **Compile** to a static binary for your target platform(s)

The generated binary is standalone — no runtime dependencies, no config files, no clihub needed.

## Authentication

Generated CLIs handle auth at runtime. For HTTP servers, the `auth` subcommand manages credentials:

```bash
# OAuth browser flow (auto-discovers endpoints via RFC 9728 / RFC 8414)
./out/linear auth

# Manual bearer token
./out/linear auth --token $TOKEN

# Check status
./out/linear auth status

# Logout
./out/linear auth logout
```

Auth is resolved in this order:

1. `--auth-token` flag
2. `--auth-type` + related flags (`--auth-header-name`, `--auth-key-file`, etc.)
3. `CLIHUB_AUTH_TOKEN` environment variable
4. `~/.clihub/credentials.json` (persisted from previous auth)
5. Unauthenticated

### Supported Auth Types

| Type | Description |
|------|-------------|
| OAuth 2.0 | Browser flow with PKCE, auto-discovery, token refresh |
| Dynamic Client Registration | RFC 7591 — auto-registers OAuth clients |
| Bearer Token | Static `Authorization: Bearer <token>` |
| API Key | Custom header (e.g. `X-API-Key`) |
| Basic Auth | Username/password |
| S2S OAuth 2.0 | Client credentials grant (no browser) |
| Google Service Account | JWT signed with service account key |

## Flags

```
clihub generate [flags]

Connection:
  --url string              Streamable HTTP URL of an MCP server
  --stdio string            Shell command that spawns a stdio MCP server
  --timeout int             Connection timeout in ms (default 30000)
  --env strings             Environment variables for stdio servers (KEY=VALUE)

Output:
  --name string             Override the inferred binary name
  --output string           Output directory (default "./out/")
  --platform string         Target GOOS/GOARCH pairs or 'all' (default current platform)

Auth:
  --oauth                   Use OAuth for authentication (browser flow)
  --auth-token string       Bearer token
  --auth-type string        Auth type: bearer, api_key, basic, none
  --auth-header-name string Custom header for api_key auth (default X-API-Key)
  --auth-key-file string    Path to Google service account JSON key
  --client-id string        Pre-registered OAuth client ID
  --client-secret string    Pre-registered OAuth client secret
  --save-credentials        Persist auth token to credential store

Filtering:
  --include-tools string    Only include these tools (comma-separated)
  --exclude-tools string    Exclude these tools (comma-separated)

Other:
  --verbose                 Show detailed progress
  --quiet                   Suppress all output except errors
```

## Project Structure

```
cmd/              CLI commands (root, generate)
internal/
  auth/           Auth providers, OAuth flow, credential store
  codegen/        Go template for generated CLIs
  compile/        Go compiler invocation, cross-compilation
  nameutil/       Binary name inference from URLs/commands
  schema/         JSON Schema → Go flag mapping
  toolfilter/     Tool include/exclude with fuzzy matching
  gocheck/        Go installation detection
main.go           Entry point
```

## Contributing

1. Fork the repo
2. Create a feature branch (`git checkout -b feature/my-change`)
3. Make your changes
4. Run tests: `go test ./...`
5. Commit with conventional format: `feat(scope): description`
6. Open a PR

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o clihub .
```

## License

[MIT](LICENSE)
