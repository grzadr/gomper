package app_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/grzadr/gomper/internal/app"
	"github.com/grzadr/gomper/internal/scanner"
)

type dummyFileInfo struct {
	mode os.FileMode
}

func (d dummyFileInfo) Name() string       { return "dummy" }
func (d dummyFileInfo) Size() int64        { return 100 }
func (d dummyFileInfo) Mode() os.FileMode  { return d.mode }
func (d dummyFileInfo) ModTime() time.Time { return time.Now() }
func (d dummyFileInfo) IsDir() bool        { return false }
func (d dummyFileInfo) Sys() any           { return nil }

func TestStandardFormatter(t *testing.T) {
	t.Run("Standard path list output", func(t *testing.T) {
		f := app.NewStandardFormatter(false)
		if f.RequiresMetrics() {
			t.Errorf("expected RequiresMetrics to be false")
		}

		entries := []scanner.Entry{
			{Path: "/root/file1.go", RelPath: "file1.go"},
			{Path: "/root/file2.md", RelPath: "file2.md"},
			{Path: "/root/empty_rel.txt", RelPath: ""},
			{Path: "/root/dot_rel.txt", RelPath: "."},
		}

		out, err := f.Format(entries)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := "file1.go\nfile2.md\n/root/empty_rel.txt\n/root/dot_rel.txt\n"
		if out != expected {
			t.Errorf("expected:\n%q\ngot:\n%q", expected, out)
		}
	})

	t.Run("Long format output with file attributes", func(t *testing.T) {
		f := app.NewStandardFormatter(true)
		entries := []scanner.Entry{
			{
				Path:    "/root/file1.go",
				RelPath: "file1.go",
				Size:    1024,
				Info:    dummyFileInfo{mode: 0644},
			},
			{
				Path:    "/root/no_info.go",
				RelPath: "no_info.go",
				Size:    512,
				Info:    nil,
			},
		}

		out, err := f.Format(entries)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(out, "FILE        1024 B  -rw-r--r--  file1.go") {
			t.Errorf("expected long format line for file1.go, got:\n%s", out)
		}
		if !strings.Contains(out, "FILE         512 B    no_info.go") {
			t.Errorf("expected long format line for no_info.go, got:\n%s", out)
		}
	})
}

func TestDetailedFormatter(t *testing.T) {
	f := app.NewDetailedFormatter()
	if !f.RequiresMetrics() {
		t.Errorf("expected RequiresMetrics to be true")
	}

	t.Run("Empty entries renders header", func(t *testing.T) {
		out, err := f.Format(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(out, "FILE") || !strings.Contains(out, "EXTENSION") || !strings.Contains(out, "SIZE") || !strings.Contains(out, "LINES") || !strings.Contains(out, "TOKENS") {
			t.Errorf("expected header in output, got:\n%s", out)
		}
	})

	t.Run("Formats entries in aligned tabular format with fallbacks", func(t *testing.T) {
		entries := []scanner.Entry{
			{
				Path:      "/root/main.go",
				RelPath:   "main.go",
				Extension: ".go",
				Size:      250,
				Lines:     20,
				Tokens:    45,
			},
			{
				Path:      "/root/nested/doc.md",
				RelPath:   "",
				Extension: "",
				Size:      0,
				Info:      dummyFileInfo{mode: 0644},
				Lines:     80,
				Tokens:    300,
			},
			{
				Path:      "",
				RelPath:   "",
				Extension: "",
				Size:      0,
				Info:      nil,
			},
		}

		out, err := f.Format(entries)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(out, "main.go") || !strings.Contains(out, ".go") || !strings.Contains(out, "250") {
			t.Errorf("expected main.go details in table output, got:\n%s", out)
		}
		if !strings.Contains(out, "/root/nested/doc.md") || !strings.Contains(out, "100") {
			t.Errorf("expected fallback path and info size in table output, got:\n%s", out)
		}
	})
}

func TestNewListFormatter(t *testing.T) {
	tests := []struct {
		format    string
		long      bool
		wantType  string
		wantErr   bool
	}{
		{"", false, "*app.StandardFormatter", false},
		{"standard", false, "*app.StandardFormatter", false},
		{" STANDARD ", true, "*app.StandardFormatter", false},
		{"detailed", false, "*app.DetailedFormatter", false},
		{" Detailed ", false, "*app.DetailedFormatter", false},
		{"json", false, "", true},
		{"invalid", false, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			formatter, err := app.NewListFormatter(tt.format, tt.long)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewListFormatter(%q) error = %v, wantErr %v", tt.format, err, tt.wantErr)
			}
			if !tt.wantErr && formatter == nil {
				t.Fatalf("expected non-nil formatter for format %q", tt.format)
			}
		})
	}
}
