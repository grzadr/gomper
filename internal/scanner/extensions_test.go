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
			name:         "Known Terraform tf extension",
			path:         "infra/main.tf",
			expectedLang: "terraform",
			expectedKnown: true,
		},
		{
			name:         "Known Terraform tfvars extension",
			path:         "infra/terraform.tfvars",
			expectedLang: "terraform",
			expectedKnown: true,
		},
		{
			name:         "Known Terraform tftpl extension",
			path:         "templates/userdata.tftpl",
			expectedLang: "terraform",
			expectedKnown: true,
		},
		{
			name:         "Known HCL extension",
			path:         "terragrunt.hcl",
			expectedLang: "hcl",
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

func TestListFormats(t *testing.T) {
	exts, specials, err := scanner.ListFormats()
	if err != nil {
		t.Fatalf("unexpected error from ListFormats: %v", err)
	}

	if len(exts) != len(scanner.SupportedExtensions) {
		t.Errorf("expected %d extensions, got %d", len(scanner.SupportedExtensions), len(exts))
	}

	if len(specials) != len(scanner.SpecialFilenames) {
		t.Errorf("expected %d special filenames, got %d", len(scanner.SpecialFilenames), len(specials))
	}

	// Verify sorting of extensions
	for i := 1; i < len(exts); i++ {
		if exts[i-1].Extension >= exts[i].Extension {
			t.Errorf("extensions not sorted: %s >= %s", exts[i-1].Extension, exts[i].Extension)
		}
	}

	// Verify sorting of specials
	for i := 1; i < len(specials); i++ {
		if specials[i-1].Filename >= specials[i].Filename {
			t.Errorf("specials not sorted: %s >= %s", specials[i-1].Filename, specials[i].Filename)
		}
	}

	// Verify known mappings exist
	foundGo := false
	for _, ext := range exts {
		if ext.Extension == ".go" && ext.Language == "go" {
			foundGo = true
			break
		}
	}
	if !foundGo {
		t.Errorf("expected .go extension to be mapped to go")
	}

	foundMakefile := false
	for _, spec := range specials {
		if spec.Filename == "makefile" && spec.Language == "makefile" {
			foundMakefile = true
			break
		}
	}
	if !foundMakefile {
		t.Errorf("expected makefile special filename to be mapped to makefile")
	}
}

