package app_test

import (
	"testing"

	"github.com/grzadr/gomper/internal/app"
)

func TestOutputFormat_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		format app.OutputFormat
		want   bool
	}{
		{"Markdown lower", app.FormatMarkdown, true},
		{"XML lower", app.FormatXML, true},
		{"JSON lower", app.FormatJSON, true},
		{"Markdown uppercase", app.OutputFormat("MARKDOWN"), true},
		{"XML mixed case", app.OutputFormat("Xml"), true},
		{"JSON uppercase", app.OutputFormat("JSON"), true},
		{"JSON mixed case", app.OutputFormat("Json"), true},
		{"Invalid format yaml", app.OutputFormat("yaml"), false},
		{"Empty format", app.OutputFormat(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.format.IsValid(); got != tt.want {
				t.Errorf("OutputFormat.IsValid(%q) = %v, want %v", tt.format, got, tt.want)
			}
		})
	}
}

func TestParseOutputFormat(t *testing.T) {
	tests := []struct {
		input   string
		want    app.OutputFormat
		wantErr bool
	}{
		{"markdown", app.FormatMarkdown, false},
		{"xml", app.FormatXML, false},
		{"json", app.FormatJSON, false},
		{" MARKDOWN ", app.FormatMarkdown, false},
		{"Xml", app.FormatXML, false},
		{" JSON ", app.FormatJSON, false},
		{"yaml", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := app.ParseOutputFormat(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseOutputFormat(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ParseOutputFormat(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestOutputFormat_SetAndUnmarshal(t *testing.T) {
	var f app.OutputFormat
	if err := f.Set("xml"); err != nil {
		t.Fatalf("unexpected error setting valid format: %v", err)
	}
	if f != app.FormatXML {
		t.Errorf("expected format XML, got %v", f)
	}

	if err := f.Set("json"); err != nil {
		t.Fatalf("unexpected error setting valid json format: %v", err)
	}
	if f != app.FormatJSON {
		t.Errorf("expected format JSON, got %v", f)
	}

	if err := f.Set("invalid"); err == nil {
		t.Errorf("expected error setting invalid format, got nil")
	}

	var f2 app.OutputFormat
	if err := f2.UnmarshalText([]byte("markdown")); err != nil {
		t.Fatalf("unexpected error unmarshaling text: %v", err)
	}
	if f2 != app.FormatMarkdown {
		t.Errorf("expected format Markdown, got %v", f2)
	}

	var f3 app.OutputFormat
	if err := f3.UnmarshalText([]byte("json")); err != nil {
		t.Fatalf("unexpected error unmarshaling text: %v", err)
	}
	if f3 != app.FormatJSON {
		t.Errorf("expected format JSON, got %v", f3)
	}

	if f2.Type() != "format" {
		t.Errorf("expected type 'format', got %q", f2.Type())
	}
}
