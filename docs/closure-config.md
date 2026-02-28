# Closure Config

Bake parameters into generated CLIs at build time so end users don't need to supply them.

## Use Cases

- **Org-specific defaults** — set `org=acme` once, every invocation inherits it
- **Team configs** — different closure files per team, same MCP server
- **Project-scoped CLIs** — generate a CLI that always targets a specific project or workspace

## Config File Format

A closure config is a JSON file with three top-level keys:

```json
{
  "mode": "hidden",
  "global": {
    "params": {
      "org": "acme",
      "environment": "production"
    }
  },
  "tools": {
    "create-issue": {
      "params": {
        "project": "BACKEND"
      }
    }
  }
}
```

| Key | Description |
|-----|-------------|
| `mode` | `"hidden"` (default) or `"default"` — controls how params are exposed |
| `global.params` | Key-value pairs injected into every tool call |
| `tools.<name>.params` | Key-value pairs injected into a specific tool (overrides global on conflict) |

## Modes

### Hidden (default)

Parameters are baked in silently. The generated CLI does not expose them as flags — end users can't see or override them. Hidden params also override matching keys when using `--from-json`.

### Default

Parameters become CLI flags with the closure value as the default. End users see the flags and can override them.

## CLI Flags

### `--closure <path>`

Load a closure config file:

```bash
clihub generate --url https://mcp.example.com --closure closure.json
```

### `--set key=value`

Set a global closure param inline (repeatable). Overrides values from `--closure` file:

```bash
clihub generate --url https://mcp.example.com --set org=acme --set env=prod
```

### `--set-tool toolname.key=value`

Set a tool-specific closure param inline (repeatable). Overrides values from `--closure` file:

```bash
clihub generate --url https://mcp.example.com --set-tool create-issue.project=BACKEND
```

### `--closure-mode hidden|default`

Override the closure mode. Takes priority over the mode in the config file:

```bash
clihub generate --url https://mcp.example.com --closure closure.json --closure-mode default
```

## Combining File + CLI Overrides

CLI flags (`--set`, `--set-tool`, `--closure-mode`) always override values from the `--closure` file. You can use `--set` and `--set-tool` without a `--closure` file — they create a config from scratch.

Precedence (highest to lowest):

1. `--closure-mode` overrides mode from file
2. `--set` overrides global params from file
3. `--set-tool` overrides tool-specific params from file
4. `--closure` file values

## Merge Behavior

When a tool is called, params are merged in this order:

1. `global.params` applied first
2. `tools.<name>.params` override on conflict

For example, with `global.params.org=acme` and `tools.create-issue.params.org=other`, `create-issue` gets `org=other` while all other tools get `org=acme`.

## Interaction with `--from-json`

In **hidden** mode, closure params silently override matching keys in `--from-json` input. The user's JSON is parsed first, then closure values are written on top.

In **default** mode, closure params only fill in keys that the user didn't provide (via flags or `--from-json`).

## Complex Values

Param values can be strings, numbers, booleans, arrays, or nested objects.

In a config file, use native JSON types:

```json
{
  "global": {
    "params": {
      "labels": ["bug", "critical"],
      "metadata": { "source": "clihub", "version": 2 }
    }
  }
}
```

With `--set`, pass JSON strings for non-string values:

```bash
clihub generate --url https://mcp.example.com \
  --set 'labels=["bug","critical"]' \
  --set org=acme
```

Plain strings (like `org=acme`) stay as strings. Values that parse as valid JSON (like `["bug","critical"]`) are deserialized automatically.

## Full Example

Create a closure config for a Linear MCP server that always targets a specific team:

```json
{
  "mode": "hidden",
  "global": {
    "params": {
      "teamId": "TEAM-123"
    }
  },
  "tools": {
    "create-issue": {
      "params": {
        "priority": 2
      }
    }
  }
}
```

Generate the CLI:

```bash
clihub generate --url https://mcp.linear.app/mcp --closure linear-closure.json
```

Use it — `teamId` is injected automatically, `priority` defaults to 2 for `create-issue`:

```bash
./out/linear create-issue --title "Fix login bug"
# teamId=TEAM-123 and priority=2 are sent automatically

./out/linear list-issues
# teamId=TEAM-123 is sent automatically
```

Or override at generation time with CLI flags:

```bash
clihub generate --url https://mcp.linear.app/mcp \
  --set teamId=TEAM-456 \
  --set-tool create-issue.priority=1 \
  --closure-mode default
```
