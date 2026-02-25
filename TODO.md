# clihub — TODO

Prioritized roadmap to cover all edge cases and reach the full vision: **one command to turn any MCP server into a compiled CLI binary that works with every auth method**.

---

## P0 — Security & Data Integrity

These can cause data loss, credential leaks, or security vulnerabilities. Fix before any public release.

- [ ] **Atomic credential file writes** — `SaveCredentials` truncates then writes. If process crashes mid-write, credentials are lost. Write to temp file + rename.
- [ ] **File-level locking on credentials** — Two generated CLIs refreshing tokens simultaneously corrupt `credentials.json`. Use `flock` or equivalent.
- [ ] **Set 0600 permissions on credential file** — Verify permissions on creation and on every load. Warn if world-readable.
- [ ] **Introduce `SecretStore` abstraction** — Add a runtime interface (`Get`, `Set`, `Delete`, `ListByServer`) so auth providers never read/write secrets directly.
- [ ] **Use OS keychain backends by default** — Implement `SecretStore` backends for macOS Keychain, Windows Credential Manager/DPAPI, and Linux Secret Service/KWallet.
- [ ] **Handle missing keychains explicitly** — Detect headless/container/CI environments where keychain is unavailable; fail with actionable guidance or require explicit fallback mode.
- [ ] **Split metadata from secret material** — Keep `credentials.json` for non-secret metadata (`auth_type`, `expires_at`, `scope`, `token_endpoint`, secret references). Store secrets (tokens, passwords, client secrets, API keys) in `SecretStore`.
- [ ] **Credential schema migration for secret refs** — Add v2→v3 migration that moves secrets out of `credentials.json` into `SecretStore`, with rollback-safe behavior.
- [ ] **HTML-escape OAuth callback error output** — `error_description` from server is rendered raw in callback HTML. XSS if server sends malicious content.
- [ ] **Warn on non-HTTPS server URLs** — HTTP MCP servers expose tokens in transit. Print a warning, don't silently accept.
- [ ] **Strip secrets from verbose/error output** — Audit all stderr/log paths (including OAuth/S2S token endpoint error bodies) so tokens, client secrets, API keys, and passwords never leak.
- [ ] **Validate stdio command safety** — Sanitize `--stdio` input to prevent shell injection via `$(...)` or backticks.
- [ ] **Keep generated CLIs security-parity with runtime** — Apply the same locking, atomic writes, permission checks, redaction, and callback escaping in `internal/codegen/main_tmpl.go`.

---

## P1 — Core Functionality Gaps

These prevent clihub from working with real-world MCP servers. The "works with any server" promise breaks without them.

- [ ] **Complex schema support (objects as flags)** — Tools with `type: "object"` params currently require `--raw` JSON. Generate nested flags (e.g. `--config.timeout 30`) or at minimum show the expected JSON structure in `--help`.
- [ ] **`anyOf`/`oneOf`/`allOf` schema handling** — Currently ignored, defaults to string. At minimum, flatten to the most common type and document the limitation.
- [ ] **Required field validation before server call** — Check required fields client-side and show a clear error instead of letting the server return a cryptic `-32602`.
- [ ] **MCP Resources support** — Call `ListResources`, generate `resources list` and `resources read <uri>` subcommands. Many servers are resource-focused (file systems, databases).
- [ ] **MCP Prompts support** — Call `ListPrompts`, generate `prompts list` and `prompts get <name>` subcommands with argument flags.
- [ ] **Server capabilities negotiation** — Read server capabilities from `Initialize` response. Only generate resource/prompt commands if server advertises support. Skip gracefully otherwise.
- [ ] **Retry on 401 in tool calls** — When a tool call returns 401, attempt token refresh and retry once before failing. Currently only pre-flight checks token expiry.
- [ ] **Handle 403 as auth failure** — Some servers return 403 instead of 401. Treat both as auth-related.
- [ ] **Pre-emptive token refresh** — If token expires within 60s, refresh before the request instead of waiting for 401.

---

## P2 — Robustness & Error Handling

These cause confusing failures or silent data issues. Important for production use.

- [ ] **Retry transient failures** — Network timeouts, 500/502/503 errors should retry with exponential backoff (3 attempts, 1s/2s/4s).
- [ ] **Per-step timeouts** — Currently one global timeout covers the entire flow. Separate timeouts for connection, initialization, and tool calls.
- [ ] **Graceful shutdown on SIGINT/SIGTERM** — Clean up temp directories, close connections, don't leave partial files.
- [ ] **Limit output size** — Cap tool response buffering (e.g. 50MB). Stream to stdout for larger responses instead of buffering all in memory.
- [ ] **Validate enum defaults** — If default value isn't in the enum list, warn at generation time.
- [ ] **Number range validation** — Enforce `minimum`/`maximum` from schema in generated code.
- [ ] **String length validation** — Enforce `minLength`/`maxLength` from schema.
- [ ] **Pattern validation** — Compile `pattern` regex at generation time, validate at runtime.
- [ ] **Flag name collision detection** — If a tool parameter name collides with a global flag (`--output`, `--timeout`, `--auth-token`), prefix it to avoid shadowing.
- [ ] **Tool name collision after normalization** — `list-issues` and `list_issues` both become the same cobra command. Detect and suffix with `_2`.
- [ ] **Warn before overwriting existing binary** — `--output` silently overwrites. Add confirmation or `--force` flag.
- [ ] **Error code translation** — Map JSON-RPC error codes (`-32602`, `-32601`, etc.) to human-readable messages.
- [ ] **Capture stderr from stdio servers on tool call failures** — Currently only captured on initialization failure, not on `CallTool`.

---

## P3 — UX & Developer Experience

These make clihub significantly more pleasant to use. High value, moderate effort.

- [ ] **Progress indicator during generation** — Show steps: connecting → discovering tools → generating code → compiling. Even a simple spinner helps.
- [ ] **`--dry-run` flag** — Show what would be generated (tool count, names, auth type) without compiling.
- [ ] **Show defaults in `--help`** — If a schema field has a `default`, display it in the flag description.
- [ ] **Show enum values in `--help`** — List allowed values in flag description, not just in validation error.
- [ ] **Show JSON structure for object params** — When a flag expects JSON, include an example in `--help`.
- [ ] **Shell completion generation** — Generate bash/zsh/fish completions. Cobra supports this natively, just wire it up.
- [ ] **`--version` flag in generated CLIs** — Embed clihub version + generation timestamp. Helps debugging stale binaries.
- [ ] **Parallel cross-compilation** — Compile multiple platforms concurrently instead of sequentially.
- [ ] **Strip debug symbols** — Add `-ldflags="-s -w"` to `go build` for ~30% smaller binaries.
- [ ] **Build caching** — Skip recompilation if tool schema hasn't changed since last generate.
- [ ] **Prune expired credentials + orphaned secrets** — Clean up stale metadata in `credentials.json` and remove matching stale entries from `SecretStore`.
- [ ] **`clihub list`** — Show all previously generated CLIs and their server URLs (scan `~/.clihub/` or output dirs).
- [ ] **Colored output** — Use ANSI colors for errors (red), warnings (yellow), success (green). Respect `NO_COLOR`.

---

## P4 — MCP Spec Completeness

Full spec compliance. Not all servers use these, but completeness matters for the "any MCP server" promise.

- [ ] **Progress tracking (ProgressToken)** — Send `progressToken` with `CallTool` requests, display server progress notifications as a progress bar.
- [ ] **Server notifications** — Register notification handlers for `tools/list_changed`, `resources/list_changed`, `prompts/list_changed`. Only relevant for long-lived sessions (future).
- [ ] **Image content rendering** — Detect `ImageContent` in tool results. Save to temp file and open, or print file path. Don't dump base64 to terminal.
- [ ] **Markdown output format** — Format tool results as markdown when `--output markdown` is used. Currently falls through to text.
- [ ] **Logging level control** — Add `--log-level` flag to set MCP server-side log verbosity (debug, info, warning, error).
- [ ] **Annotated content support** — Render `AnnotatedTextContent` annotations (priority, audience) in output.

---

## P5 — Future

Low priority. Only if there's demand.

- [ ] **Regenerate in-place** — `clihub update ./out/linear` to re-discover tools and recompile without full flags.
- [ ] **Audit log** — Log credential access to `~/.clihub/audit.log` for enterprise compliance.
