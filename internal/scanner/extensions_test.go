package scanner_test

import (
	"testing"

	"github.com/grzadr/gomper/internal/scanner"
)

func TestLookupLanguage(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		expectedLang string
		expectedKnown bool
	}{
		{
			name:         "Known Go extension",
			path:         "main.go",
			expectedLang: "go",
			expectedKnown: true,
		},
		{
			name:         "Known TypeScript extension",
			path:         "src/auth/jwt.ts",
			expectedLang: "typescript",
			expectedKnown: true,
		},
		{
			name:         "Known special filename Makefile",
			path:         "Makefile",
			expectedLang: "makefile",
			expectedKnown: true,
		},
		{
			name:         "Unknown extension .xyz",
			path:         "data/config.xyz",
			expectedLang: "xyz",
			expectedKnown: false,
		},
		{
			name:         "No extension",
			path:         "bin/executable",
			expectedLang: "text",
			expectedKnown: false,
		},
		{
			name:         "Trailing dot extension",
			path:         "data/file.",
			expectedLang: "text",
			expectedKnown: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lang, known := scanner.LookupLanguage(tt.path)
			if lang != tt.expectedLang {
				t.Errorf("expected lang %q, got %q", tt.expectedLang, lang)
			}
			if known != tt.expectedKnown {
				t.Errorf("expected known %v, got %v", tt.expectedKnown, known)
			}
		})
	}
}
