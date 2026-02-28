package closure

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "closure.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad_ValidConfig(t *testing.T) {
	path := writeTestFile(t, `{
		"mode": "hidden",
		"global": {
			"params": {
				"org_id": "acme-corp",
				"team_id": "engineering"
			}
		},
		"tools": {
			"create_issue": {
				"params": {
					"project_id": "PROJ-123",
					"labels": ["bug", "urgent"],
					"metadata": {"source": "cli", "version": 2}
				}
			},
			"list_issues": {
				"params": {
					"status": "open"
				}
			}
		}
	}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Mode != ModeHidden {
		t.Errorf("mode = %q, want %q", cfg.Mode, ModeHidden)
	}

	if got := cfg.Global.Params["org_id"]; got != "acme-corp" {
		t.Errorf("global org_id = %v, want %q", got, "acme-corp")
	}

	// Verify complex types survived deserialization.
	tc := cfg.Tools["create_issue"]
	labels, ok := tc.Params["labels"].([]any)
	if !ok {
		t.Fatalf("labels is not []any: %T", tc.Params["labels"])
	}
	if len(labels) != 2 || labels[0] != "bug" || labels[1] != "urgent" {
		t.Errorf("labels = %v, want [bug urgent]", labels)
	}

	meta, ok := tc.Params["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("metadata is not map[string]any: %T", tc.Params["metadata"])
	}
	if meta["source"] != "cli" {
		t.Errorf("metadata.source = %v, want %q", meta["source"], "cli")
	}
	if meta["version"] != float64(2) {
		t.Errorf("metadata.version = %v, want 2", meta["version"])
	}
}

func TestLoad_DefaultMode(t *testing.T) {
	path := writeTestFile(t, `{
		"global": {"params": {"key": "val"}}
	}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Mode != ModeHidden {
		t.Errorf("mode = %q, want %q (default)", cfg.Mode, ModeHidden)
	}
}

func TestLoad_ModeDefault(t *testing.T) {
	path := writeTestFile(t, `{"mode": "default"}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Mode != ModeDefault {
		t.Errorf("mode = %q, want %q", cfg.Mode, ModeDefault)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("/nonexistent/closure.json")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	path := writeTestFile(t, `{not valid json}`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestLoad_UnknownMode(t *testing.T) {
	path := writeTestFile(t, `{"mode": "banana"}`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for unknown mode, got nil")
	}
}

func TestLoad_EmptyGlobalParamName(t *testing.T) {
	path := writeTestFile(t, `{
		"global": {"params": {"": "value"}}
	}`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for empty global param name, got nil")
	}
}

func TestLoad_EmptyToolParamName(t *testing.T) {
	path := writeTestFile(t, `{
		"tools": {"mytool": {"params": {"": "value"}}}
	}`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for empty tool param name, got nil")
	}
}

func TestMerge_GlobalOnly(t *testing.T) {
	cfg := &Config{
		Global: GlobalConfig{
			Params: map[string]any{"org": "acme", "team": "eng"},
		},
	}

	merged := cfg.Merge("unknown_tool")
	if len(merged) != 2 {
		t.Errorf("len = %d, want 2", len(merged))
	}
	if merged["org"] != "acme" {
		t.Errorf("org = %v, want %q", merged["org"], "acme")
	}
}

func TestMerge_ToolOverridesGlobal(t *testing.T) {
	cfg := &Config{
		Global: GlobalConfig{
			Params: map[string]any{"org": "acme", "verbose": true},
		},
		Tools: map[string]ToolConfig{
			"deploy": {
				Params: map[string]any{"org": "override-org", "env": "prod"},
			},
		},
	}

	merged := cfg.Merge("deploy")
	if merged["org"] != "override-org" {
		t.Errorf("org = %v, want %q", merged["org"], "override-org")
	}
	if merged["verbose"] != true {
		t.Errorf("verbose = %v, want true", merged["verbose"])
	}
	if merged["env"] != "prod" {
		t.Errorf("env = %v, want %q", merged["env"], "prod")
	}
}

func TestMerge_NoGlobalNoTool(t *testing.T) {
	cfg := &Config{}
	merged := cfg.Merge("anything")
	if len(merged) != 0 {
		t.Errorf("len = %d, want 0", len(merged))
	}
}

func TestParamNames(t *testing.T) {
	cfg := &Config{
		Global: GlobalConfig{
			Params: map[string]any{"org": "acme", "team": "eng"},
		},
		Tools: map[string]ToolConfig{
			"deploy": {
				Params: map[string]any{"team": "override", "env": "prod"},
			},
		},
	}

	names := cfg.ParamNames("deploy")
	expected := []string{"env", "org", "team"}
	if len(names) != len(expected) {
		t.Fatalf("names = %v, want %v", names, expected)
	}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("names[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

func TestParamNames_UnknownTool(t *testing.T) {
	cfg := &Config{
		Global: GlobalConfig{
			Params: map[string]any{"org": "acme"},
		},
	}

	names := cfg.ParamNames("unknown")
	if len(names) != 1 || names[0] != "org" {
		t.Errorf("names = %v, want [org]", names)
	}
}

// --- ParseSetEntries tests ---

func TestParseSetEntries_Basic(t *testing.T) {
	params, err := ParseSetEntries([]string{"org=acme", "team=eng"})
	if err != nil {
		t.Fatal(err)
	}
	if params["org"] != "acme" {
		t.Errorf("org = %v, want %q", params["org"], "acme")
	}
	if params["team"] != "eng" {
		t.Errorf("team = %v, want %q", params["team"], "eng")
	}
}

func TestParseSetEntries_JSONValue(t *testing.T) {
	params, err := ParseSetEntries([]string{`labels=["bug","urgent"]`})
	if err != nil {
		t.Fatal(err)
	}
	labels, ok := params["labels"].([]any)
	if !ok {
		t.Fatalf("labels is %T, want []any", params["labels"])
	}
	if len(labels) != 2 || labels[0] != "bug" || labels[1] != "urgent" {
		t.Errorf("labels = %v", labels)
	}
}

func TestParseSetEntries_ValueWithEquals(t *testing.T) {
	params, err := ParseSetEntries([]string{"query=a=b=c"})
	if err != nil {
		t.Fatal(err)
	}
	if params["query"] != "a=b=c" {
		t.Errorf("query = %v, want %q", params["query"], "a=b=c")
	}
}

func TestParseSetEntries_EmptyValue(t *testing.T) {
	params, err := ParseSetEntries([]string{"key="})
	if err != nil {
		t.Fatal(err)
	}
	if params["key"] != "" {
		t.Errorf("key = %v, want empty string", params["key"])
	}
}

func TestParseSetEntries_MissingEquals(t *testing.T) {
	_, err := ParseSetEntries([]string{"noequals"})
	if err == nil {
		t.Fatal("expected error for missing =")
	}
}

func TestParseSetEntries_EmptyKey(t *testing.T) {
	_, err := ParseSetEntries([]string{"=value"})
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}

// --- ParseSetToolEntries tests ---

func TestParseSetToolEntries_Basic(t *testing.T) {
	tools, err := ParseSetToolEntries([]string{"create_issue.project_id=PROJ-123"})
	if err != nil {
		t.Fatal(err)
	}
	if tools["create_issue"]["project_id"] != "PROJ-123" {
		t.Errorf("got %v", tools["create_issue"]["project_id"])
	}
}

func TestParseSetToolEntries_JSONValue(t *testing.T) {
	tools, err := ParseSetToolEntries([]string{`mytool.meta={"a":1}`})
	if err != nil {
		t.Fatal(err)
	}
	meta, ok := tools["mytool"]["meta"].(map[string]any)
	if !ok {
		t.Fatalf("meta is %T, want map[string]any", tools["mytool"]["meta"])
	}
	if meta["a"] != float64(1) {
		t.Errorf("meta.a = %v, want 1", meta["a"])
	}
}

func TestParseSetToolEntries_MultipleSameTool(t *testing.T) {
	tools, err := ParseSetToolEntries([]string{
		"deploy.env=prod",
		"deploy.region=us-east",
	})
	if err != nil {
		t.Fatal(err)
	}
	if tools["deploy"]["env"] != "prod" {
		t.Errorf("env = %v", tools["deploy"]["env"])
	}
	if tools["deploy"]["region"] != "us-east" {
		t.Errorf("region = %v", tools["deploy"]["region"])
	}
}

func TestParseSetToolEntries_MissingDot(t *testing.T) {
	_, err := ParseSetToolEntries([]string{"nodot=value"})
	if err == nil {
		t.Fatal("expected error for missing dot")
	}
}

func TestParseSetToolEntries_EmptyToolName(t *testing.T) {
	_, err := ParseSetToolEntries([]string{".key=value"})
	if err == nil {
		t.Fatal("expected error for empty tool name")
	}
}

func TestParseSetToolEntries_MissingEquals(t *testing.T) {
	_, err := ParseSetToolEntries([]string{"tool.keyonly"})
	if err == nil {
		t.Fatal("expected error for missing =")
	}
}

func TestParseSetToolEntries_EmptyKey(t *testing.T) {
	_, err := ParseSetToolEntries([]string{"tool.=value"})
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}

// --- MergeOverrides tests ---

func TestMergeOverrides_NilConfig(t *testing.T) {
	cfg, err := MergeOverrides(nil, []string{"org=acme"}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mode != ModeHidden {
		t.Errorf("mode = %q, want %q", cfg.Mode, ModeHidden)
	}
	if cfg.Global.Params["org"] != "acme" {
		t.Errorf("org = %v, want %q", cfg.Global.Params["org"], "acme")
	}
}

func TestMergeOverrides_CLIOverridesFile(t *testing.T) {
	fileCfg := &Config{
		Mode: ModeHidden,
		Global: GlobalConfig{
			Params: map[string]any{"org": "file-org", "team": "file-team"},
		},
		Tools: map[string]ToolConfig{
			"deploy": {Params: map[string]any{"env": "staging"}},
		},
	}

	cfg, err := MergeOverrides(fileCfg,
		[]string{"org=cli-org"},
		[]string{"deploy.env=prod"},
		"",
	)
	if err != nil {
		t.Fatal(err)
	}

	// CLI --set overrides file global.
	if cfg.Global.Params["org"] != "cli-org" {
		t.Errorf("org = %v, want %q", cfg.Global.Params["org"], "cli-org")
	}
	// File value preserved when no override.
	if cfg.Global.Params["team"] != "file-team" {
		t.Errorf("team = %v, want %q", cfg.Global.Params["team"], "file-team")
	}
	// CLI --set-tool overrides file tool param.
	if cfg.Tools["deploy"].Params["env"] != "prod" {
		t.Errorf("deploy.env = %v, want %q", cfg.Tools["deploy"].Params["env"], "prod")
	}
	// Original config not mutated.
	if fileCfg.Global.Params["org"] != "file-org" {
		t.Errorf("original config mutated: org = %v", fileCfg.Global.Params["org"])
	}
}

func TestMergeOverrides_ModeOverride(t *testing.T) {
	fileCfg := &Config{Mode: ModeHidden}

	cfg, err := MergeOverrides(fileCfg, nil, nil, "default")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mode != ModeDefault {
		t.Errorf("mode = %q, want %q", cfg.Mode, ModeDefault)
	}
}

func TestMergeOverrides_InvalidMode(t *testing.T) {
	_, err := MergeOverrides(nil, nil, nil, "banana")
	if err == nil {
		t.Fatal("expected error for invalid mode override")
	}
}

func TestMergeOverrides_NewToolFromCLI(t *testing.T) {
	cfg, err := MergeOverrides(nil, nil, []string{"newtool.key=val"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Tools["newtool"].Params["key"] != "val" {
		t.Errorf("newtool.key = %v, want %q", cfg.Tools["newtool"].Params["key"], "val")
	}
}

func TestMergeOverrides_AllEmpty(t *testing.T) {
	cfg, err := MergeOverrides(nil, nil, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.Mode != ModeHidden {
		t.Errorf("mode = %q, want %q", cfg.Mode, ModeHidden)
	}
}
