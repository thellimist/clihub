package toolfilter

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// ParseToolList tests
// ---------------------------------------------------------------------------

func TestParseToolList(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "basic comma separated",
			input: "foo, bar, baz",
			want:  []string{"foo", "bar", "baz"},
		},
		{
			name:  "deduplication preserves order",
			input: "foo, bar, foo",
			want:  []string{"foo", "bar"},
		},
		{
			name:  "trim whitespace and skip empty",
			input: "  a , b ,  ",
			want:  []string{"a", "b"},
		},
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseToolList(tc.input)
			if !strSliceEqual(got, tc.want) {
				t.Errorf("ParseToolList(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FilterTools — include mode
// ---------------------------------------------------------------------------

func TestFilterToolsInclude(t *testing.T) {
	allTools := []Tool{
		{Name: "a", Description: "tool a"},
		{Name: "b", Description: "tool b"},
		{Name: "c", Description: "tool c"},
		{Name: "d", Description: "tool d"},
	}

	t.Run("include subset", func(t *testing.T) {
		got, err := FilterTools(allTools, []string{"a", "b"}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wantNames := []string{"a", "b"}
		if !toolNamesEqual(got, wantNames) {
			t.Errorf("got tools %v, want %v", toolNames(got), wantNames)
		}
	})

	t.Run("include unknown tool lists available", func(t *testing.T) {
		_, err := FilterTools(allTools, []string{"x"}, nil)
		if err == nil {
			t.Fatal("expected error for unknown tool")
		}
		if !strings.Contains(err.Error(), "tool 'x' not found") {
			t.Errorf("error should mention tool not found, got: %v", err)
		}
		if !strings.Contains(err.Error(), "Available tools:") {
			t.Errorf("error should list available tools, got: %v", err)
		}
	})

	t.Run("include with close match suggests tool", func(t *testing.T) {
		tools := []Tool{
			{Name: "list_issues"},
			{Name: "create_issue"},
		}
		_, err := FilterTools(tools, []string{"lisst_issues"}, nil)
		if err == nil {
			t.Fatal("expected error for misspelled tool")
		}
		if !strings.Contains(err.Error(), "Did you mean 'list_issues'?") {
			t.Errorf("error should suggest list_issues, got: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// FilterTools — exclude mode
// ---------------------------------------------------------------------------

func TestFilterToolsExclude(t *testing.T) {
	allTools := []Tool{
		{Name: "a", Description: "tool a"},
		{Name: "b", Description: "tool b"},
		{Name: "c", Description: "tool c"},
		{Name: "d", Description: "tool d"},
	}

	t.Run("exclude subset", func(t *testing.T) {
		got, err := FilterTools(allTools, nil, []string{"c", "d"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wantNames := []string{"a", "b"}
		if !toolNamesEqual(got, wantNames) {
			t.Errorf("got tools %v, want %v", toolNames(got), wantNames)
		}
	})

	t.Run("exclude all tools errors", func(t *testing.T) {
		_, err := FilterTools(
			[]Tool{{Name: "a"}, {Name: "b"}, {Name: "c"}},
			nil,
			[]string{"a", "b", "c"},
		)
		if err == nil {
			t.Fatal("expected error when all tools excluded")
		}
		if !strings.Contains(err.Error(), "all tools excluded") {
			t.Errorf("error should mention all tools excluded, got: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// FilterTools — edge cases
// ---------------------------------------------------------------------------

func TestFilterToolsBothIncludeAndExclude(t *testing.T) {
	_, err := FilterTools(nil, []string{"a"}, []string{"b"})
	if err == nil {
		t.Fatal("expected error when both include and exclude provided")
	}
	if !strings.Contains(err.Error(), "cannot be used together") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestFilterToolsNoFilter(t *testing.T) {
	tools := []Tool{{Name: "a"}, {Name: "b"}}
	got, err := FilterTools(tools, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !toolNamesEqual(got, []string{"a", "b"}) {
		t.Errorf("expected all tools returned, got %v", toolNames(got))
	}
}

// ---------------------------------------------------------------------------
// LevenshteinDistance tests
// ---------------------------------------------------------------------------

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "abc", 0},
		{"", "abc", 3},
		{"abc", "", 3},
		{"kitten", "sitting", 3},
		{"list_issues", "lisst_issues", 1},
	}

	for _, tc := range tests {
		t.Run(tc.a+"_vs_"+tc.b, func(t *testing.T) {
			got := LevenshteinDistance(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("LevenshteinDistance(%q, %q) = %d, want %d",
					tc.a, tc.b, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SuggestTool tests
// ---------------------------------------------------------------------------

func TestSuggestTool(t *testing.T) {
	available := []string{"list_issues", "create_issue", "delete_repo"}

	t.Run("close match returns suggestion", func(t *testing.T) {
		got := SuggestTool("lisst_issues", available)
		if got != "list_issues" {
			t.Errorf("SuggestTool returned %q, want %q", got, "list_issues")
		}
	})

	t.Run("far match returns empty", func(t *testing.T) {
		got := SuggestTool("zzzzzzzzzzzzz", available)
		if got != "" {
			t.Errorf("SuggestTool returned %q, want empty string", got)
		}
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func strSliceEqual(a, b []string) bool {
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

func toolNames(tools []Tool) []string {
	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Name
	}
	return names
}

func toolNamesEqual(tools []Tool, want []string) bool {
	return strSliceEqual(toolNames(tools), want)
}
