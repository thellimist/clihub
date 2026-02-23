package compile

import (
	"strings"
	"testing"
)

func TestParsePlatforms(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantCount int
		wantErr   string
	}{
		{"all expands to 6", "all", 6, ""},
		{"single platform", "linux/amd64", 1, ""},
		{"two platforms", "linux/amd64,darwin/arm64", 2, ""},
		{"with whitespace", " linux/amd64 , darwin/arm64 ", 2, ""},
		{"dedup", "linux/amd64,linux/amd64", 1, ""},
		{"invalid platform", "foo/bar", 0, "invalid platform 'foo/bar'"},
		{"empty string", "", 0, "no platforms specified"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			platforms, err := ParsePlatforms(tt.input)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(platforms) != tt.wantCount {
				t.Errorf("got %d platforms, want %d", len(platforms), tt.wantCount)
			}
		})
	}
}

func TestParsePlatformsAll(t *testing.T) {
	platforms, err := ParsePlatforms("all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := map[string]bool{
		"linux/amd64":   true,
		"linux/arm64":   true,
		"darwin/amd64":  true,
		"darwin/arm64":  true,
		"windows/amd64": true,
		"windows/arm64": true,
	}

	for _, p := range platforms {
		if !expected[p.String()] {
			t.Errorf("unexpected platform: %s", p)
		}
	}
}

func TestBinaryName(t *testing.T) {
	tests := []struct {
		name     string
		cliName  string
		platform Platform
		multi    bool
		want     string
	}{
		{"single linux", "linear", Platform{"linux", "amd64"}, false, "linear"},
		{"single darwin", "linear", Platform{"darwin", "arm64"}, false, "linear"},
		{"single windows", "linear", Platform{"windows", "amd64"}, false, "linear.exe"},
		{"multi linux amd64", "linear", Platform{"linux", "amd64"}, true, "linear-linux-amd64"},
		{"multi darwin arm64", "linear", Platform{"darwin", "arm64"}, true, "linear-darwin-arm64"},
		{"multi windows amd64", "linear", Platform{"windows", "amd64"}, true, "linear-windows-amd64.exe"},
		{"multi windows arm64", "linear", Platform{"windows", "arm64"}, true, "linear-windows-arm64.exe"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BinaryName(tt.cliName, tt.platform, tt.multi)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
