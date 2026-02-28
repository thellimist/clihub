package closure

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Mode controls how injected params are exposed to the CLI.
type Mode string

const (
	ModeHidden  Mode = "hidden"  // Params are baked in silently, not exposed as flags.
	ModeDefault Mode = "default" // Params become CLI flags with the closure value as default.
)

// ToolConfig holds per-tool parameter injections.
type ToolConfig struct {
	Params map[string]any `json:"params"`
}

// GlobalConfig holds global parameter injections applied to all tools.
type GlobalConfig struct {
	Params map[string]any `json:"params"`
}

// Config is the top-level closure configuration.
type Config struct {
	Mode   Mode                  `json:"mode"`
	Global GlobalConfig          `json:"global"`
	Tools  map[string]ToolConfig `json:"tools"`
}

// Load reads and validates a closure config from a JSON file at the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("closure: reading config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("closure: parsing config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	// Default mode to hidden if omitted.
	if c.Mode == "" {
		c.Mode = ModeHidden
	}

	switch c.Mode {
	case ModeHidden, ModeDefault:
		// valid
	default:
		return fmt.Errorf("closure: unknown mode %q (must be %q or %q)", c.Mode, ModeHidden, ModeDefault)
	}

	// Validate global param names.
	for name := range c.Global.Params {
		if name == "" {
			return fmt.Errorf("closure: global params contain an empty param name")
		}
	}

	// Validate per-tool param names.
	for tool, tc := range c.Tools {
		for name := range tc.Params {
			if name == "" {
				return fmt.Errorf("closure: tool %q params contain an empty param name", tool)
			}
		}
	}

	return nil
}

// Merge returns the merged params for a given tool.
// Global params are applied first, then tool-specific params override on conflict.
func (c *Config) Merge(toolName string) map[string]any {
	merged := make(map[string]any, len(c.Global.Params))

	for k, v := range c.Global.Params {
		merged[k] = v
	}

	if tc, ok := c.Tools[toolName]; ok {
		for k, v := range tc.Params {
			merged[k] = v
		}
	}

	return merged
}

// ParamNames returns all injected param names for a tool (global + tool-specific),
// sorted alphabetically. Useful for hidden mode to know which flags to skip.
func (c *Config) ParamNames(toolName string) []string {
	seen := make(map[string]struct{})

	for k := range c.Global.Params {
		seen[k] = struct{}{}
	}

	if tc, ok := c.Tools[toolName]; ok {
		for k := range tc.Params {
			seen[k] = struct{}{}
		}
	}

	names := make([]string, 0, len(seen))
	for k := range seen {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// parseValue attempts to parse a string as JSON. If parsing fails, the raw
// string is returned. This allows --set labels='["a","b"]' to produce a
// []any while --set org=acme stays a plain string.
func parseValue(raw string) any {
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return raw
	}
	return v
}

// ParseSetEntries parses --set key=value entries into a param map.
// Each entry is split on the first '='. An error is returned for entries
// missing '=' or having an empty key.
func ParseSetEntries(entries []string) (map[string]any, error) {
	params := make(map[string]any, len(entries))
	for _, entry := range entries {
		idx := strings.Index(entry, "=")
		if idx < 0 {
			return nil, fmt.Errorf("closure: invalid --set %q: expected key=value", entry)
		}
		key := entry[:idx]
		if key == "" {
			return nil, fmt.Errorf("closure: invalid --set %q: empty key", entry)
		}
		params[key] = parseValue(entry[idx+1:])
	}
	return params, nil
}

// ParseSetToolEntries parses --set-tool toolname.key=value entries into a
// per-tool param map. Each entry is split on the first '.' for the tool name,
// then the first '=' for key/value.
func ParseSetToolEntries(entries []string) (map[string]map[string]any, error) {
	tools := make(map[string]map[string]any)
	for _, entry := range entries {
		dotIdx := strings.Index(entry, ".")
		if dotIdx < 0 {
			return nil, fmt.Errorf("closure: invalid --set-tool %q: expected toolname.key=value", entry)
		}
		toolName := entry[:dotIdx]
		if toolName == "" {
			return nil, fmt.Errorf("closure: invalid --set-tool %q: empty tool name", entry)
		}
		rest := entry[dotIdx+1:]
		eqIdx := strings.Index(rest, "=")
		if eqIdx < 0 {
			return nil, fmt.Errorf("closure: invalid --set-tool %q: expected toolname.key=value", entry)
		}
		key := rest[:eqIdx]
		if key == "" {
			return nil, fmt.Errorf("closure: invalid --set-tool %q: empty key", entry)
		}
		if tools[toolName] == nil {
			tools[toolName] = make(map[string]any)
		}
		tools[toolName][key] = parseValue(rest[eqIdx+1:])
	}
	return tools, nil
}

// MergeOverrides merges CLI --set / --set-tool / --closure-mode overrides into
// an existing Config (which may be nil if no --closure file was provided).
// CLI values override file values. The returned Config is always non-nil.
func MergeOverrides(cfg *Config, globalSets []string, toolSets []string, modeOverride string) (*Config, error) {
	// Start from a copy or a fresh config.
	var merged Config
	if cfg != nil {
		merged = *cfg
		// Deep-copy maps so we don't mutate the original.
		merged.Global.Params = copyParams(cfg.Global.Params)
		merged.Tools = copyTools(cfg.Tools)
	}

	// Default mode if not set by file.
	if merged.Mode == "" {
		merged.Mode = ModeHidden
	}

	// Parse and apply --set entries (global params).
	globalParams, err := ParseSetEntries(globalSets)
	if err != nil {
		return nil, err
	}
	if merged.Global.Params == nil {
		merged.Global.Params = make(map[string]any)
	}
	for k, v := range globalParams {
		merged.Global.Params[k] = v
	}

	// Parse and apply --set-tool entries (per-tool params).
	toolParams, err := ParseSetToolEntries(toolSets)
	if err != nil {
		return nil, err
	}
	if merged.Tools == nil {
		merged.Tools = make(map[string]ToolConfig)
	}
	for toolName, params := range toolParams {
		tc := merged.Tools[toolName]
		if tc.Params == nil {
			tc.Params = make(map[string]any)
		}
		for k, v := range params {
			tc.Params[k] = v
		}
		merged.Tools[toolName] = tc
	}

	// Apply --closure-mode override (highest priority).
	if modeOverride != "" {
		m := Mode(modeOverride)
		switch m {
		case ModeHidden, ModeDefault:
			merged.Mode = m
		default:
			return nil, fmt.Errorf("closure: unknown --closure-mode %q (must be %q or %q)", modeOverride, ModeHidden, ModeDefault)
		}
	}

	return &merged, nil
}

func copyParams(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyTools(src map[string]ToolConfig) map[string]ToolConfig {
	if src == nil {
		return nil
	}
	dst := make(map[string]ToolConfig, len(src))
	for name, tc := range src {
		dst[name] = ToolConfig{Params: copyParams(tc.Params)}
	}
	return dst
}
