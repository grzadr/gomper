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
			name:         "Known Bicep extension",
			path:         "infra/main.bicep",
			expectedLang: "bicep",
			expectedKnown: true,
		},
		{
			name:         "Known Bicep parameters extension",
			path:         "infra/main.bicepparam",
			expectedLang: "bicep",
			expectedKnown: true,
		},
		{
			name:         "Bicep file with auxiliary extension",
			path:         "infra/main.bicep.example",
			expectedLang: "bicep",
			expectedKnown: true,
		},
		{
			name:         "Known special filename Makefile",
			path:         "Makefile",
			expectedLang: "makefile",
			expectedKnown: true,
		},
		{
			name:         "Special filename with template extension",
			path:         "Makefile.template",
			expectedLang: "makefile",
			expectedKnown: true,
		},
		{
			name:         "Special filename with example extension",
			path:         "Makefile.example",
			expectedLang: "makefile",
			expectedKnown: true,
		},
		{
			name:         "Special filename with sample extension",
			path:         "Dockerfile.sample",
			expectedLang: "dockerfile",
			expectedKnown: true,
		},
		{
			name:         "Compound YAML example extension",
			path:         "gomper.yaml.example",
			expectedLang: "yaml",
			expectedKnown: true,
		},
		{
			name:         "Compound JSON dist extension",
			path:         "config.json.dist",
			expectedLang: "json",
			expectedKnown: true,
		},
		{
			name:         "Compound TOML default extension",
			path:         "settings.toml.default",
			expectedLang: "toml",
			expectedKnown: true,
		},
		{
			name:         "Compound Shell tmpl extension",
			path:         "scripts/deploy.sh.tmpl",
			expectedLang: "bash",
			expectedKnown: true,
		},
		{
			name:         "Compound HTML tpl extension",
			path:         "templates/index.html.tpl",
			expectedLang: "html",
			expectedKnown: true,
		},
		{
			name:         "Nested multiple auxiliary extensions",
			path:         "configs/app.yaml.tmpl.example",
			expectedLang: "yaml",
			expectedKnown: true,
		},
		{
			name:         "Unknown extension .xyz",
			path:         "data/config.xyz",
			expectedLang: "xyz",
			expectedKnown: false,
		},
		{
			name:         "Unknown extension with auxiliary extension",
			path:         "data/config.xyz.example",
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
			name:         "No extension with auxiliary extension",
			path:         "bin/executable.sample",
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
			t.Parallel()
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

func TestStripAuxiliaryExtensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No auxiliary extension",
			input:    "main.go",
			expected: "main.go",
		},
		{
			name:     "Single .example suffix",
			input:    "gomper.yaml.example",
			expected: "gomper.yaml",
		},
		{
			name:     "Single .template suffix",
			input:    "Makefile.template",
			expected: "Makefile",
		},
		{
			name:     "Single .sample suffix",
			input:    "Dockerfile.sample",
			expected: "Dockerfile",
		},
		{
			name:     "Single .dist suffix",
			input:    "config.json.dist",
			expected: "config.json",
		},
		{
			name:     "Single .default suffix",
			input:    "settings.toml.default",
			expected: "settings.toml",
		},
		{
			name:     "Single .tmpl suffix",
			input:    "deploy.sh.tmpl",
			expected: "deploy.sh",
		},
		{
			name:     "Single .tpl suffix",
			input:    "index.html.tpl",
			expected: "index.html",
		},
		{
			name:     "Multi-layer auxiliary suffixes",
			input:    "app.config.tmpl.example.dist",
			expected: "app.config",
		},
		{
			name:     "Extensionless file with auxiliary suffix",
			input:    "bin/executable.sample",
			expected: "bin/executable",
		},
		{
			name:     "Case-insensitive auxiliary suffix",
			input:    "config.yaml.EXAMPLE",
			expected: "config.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := scanner.StripAuxiliaryExtensions(tt.input)
			if got != tt.expected {
				t.Errorf("StripAuxiliaryExtensions(%q) = %q, want %q", tt.input, got, tt.expected)
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

