package scanner_test

import (
	"testing"

	"github.com/grzadr/gomper/internal/scanner"
)

func TestListProfiles(t *testing.T) {
	profiles, err := scanner.ListProfiles()
	if err != nil {
		t.Fatalf("unexpected error listing embedded profiles: %v", err)
	}

	if len(profiles) == 0 {
		t.Fatalf("expected embedded profiles, got empty slice")
	}

	profileMap := make(map[string]bool)
	for _, p := range profiles {
		profileMap[p] = true
	}

	expectedProfiles := []string{"generic", "go", "node", "python", "java", "cpp", "rust"}
	for _, expected := range expectedProfiles {
		if !profileMap[expected] {
			t.Errorf("expected embedded profile %q, but not found in %v", expected, profiles)
		}
	}
}

func TestLoadProfilePatterns_Generic(t *testing.T) {
	patterns, err := scanner.LoadProfilePatterns("generic")
	if err != nil {
		t.Fatalf("unexpected error loading 'generic' profile: %v", err)
	}

	if len(patterns) == 0 {
		t.Errorf("expected patterns for 'generic' profile, got empty")
	}

	filter, err := scanner.NewFilter(patterns, false)
	if err != nil {
		t.Fatalf("failed to create filter from loaded patterns: %v", err)
	}

	if !filter.ShouldIgnore(".env", ".env") {
		t.Errorf("expected 'generic' profile to ignore .env")
	}

	if !filter.ShouldIgnore(".env.local", ".env.local") {
		t.Errorf("expected 'generic' profile to ignore .env.local")
	}

	if !filter.ShouldIgnore(".DS_Store", ".DS_Store") {
		t.Errorf("expected 'generic' profile to ignore .DS_Store")
	}

	if !filter.ShouldIgnore(".git", ".git") {
		t.Errorf("expected 'generic' profile to ignore .git")
	}

	if filter.ShouldIgnore("main.go", "main.go") {
		t.Errorf("expected 'generic' profile NOT to ignore main.go")
	}
}

func TestLoadProfilePatterns(t *testing.T) {
	patterns, err := scanner.LoadProfilePatterns("go")
	if err != nil {
		t.Fatalf("unexpected error loading 'go' profile: %v", err)
	}

	if len(patterns) == 0 {
		t.Errorf("expected patterns for 'go' profile, got empty")
	}

	filter, err := scanner.NewFilter(patterns, false)
	if err != nil {
		t.Fatalf("failed to create filter from loaded patterns: %v", err)
	}

	if !filter.ShouldIgnore("main.exe", "main.exe") {
		t.Errorf("expected 'go' profile to ignore main.exe")
	}

	if !filter.ShouldIgnore("vendor", "vendor") {
		t.Errorf("expected 'go' profile to ignore vendor directory")
	}

	if filter.ShouldIgnore("main.go", "main.go") {
		t.Errorf("expected 'go' profile NOT to ignore main.go")
	}
}

func TestLoadProfilePatterns_Unknown(t *testing.T) {
	_, err := scanner.LoadProfilePatterns("unknown_lang")
	if err == nil {
		t.Errorf("expected error for unknown profile, got nil")
	}
}

func TestGitignoreToRegex(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"*.exe", "(^|/).*\\.exe(/|$)"},
		{"node_modules/", "(^|/)node_modules(/|$)"},
		{"# comment", ""},
		{"!negated", ""},
		{"", ""},
		{"/", ""},
		{"file?.txt", "(^|/)file.\\.txt(/|$)"},
		{"file+(1).log", "(^|/)file\\+\\(1\\)\\.log(/|$)"},
		{"a|b^${}", "(^|/)a\\|b\\^\\$\\{\\}(/|$)"},
		{`foo\bar`, `(^|/)foo\\bar(/|$)`},
	}

	for _, tt := range tests {
		got := scanner.GitignoreToRegex(tt.line)
		if got != tt.want {
			t.Errorf("GitignoreToRegex(%q) = %q, want %q", tt.line, got, tt.want)
		}
	}
}

