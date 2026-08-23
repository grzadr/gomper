package cmd

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/grzadr/gomper/internal/scanner"
)

func TestFormatsCommand_Error(t *testing.T) {
	origList := listFormatsFunc
	listFormatsFunc = func() ([]scanner.FormatEntry, []scanner.SpecialFileEntry, error) {
		return nil, nil, errors.New("simulated list formats failure")
	}
	defer func() { listFormatsFunc = origList }()

	formatsCmd := NewFormatsCommand()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	formatsCmd.SetOut(outBuf)
	formatsCmd.SetErr(errBuf)

	err := formatsCmd.Execute()
	if err == nil {
		t.Fatalf("expected error from formats command, got nil")
	}
	if !strings.Contains(err.Error(), "simulated list formats failure") {
		t.Errorf("expected error message mentioning simulated list formats failure, got: %v", err)
	}
}

type errOutWriter struct {
	failOnWrite int
	writes      int
}

func (e *errOutWriter) Write(p []byte) (int, error) {
	e.writes++
	if e.failOnWrite == 0 || e.writes >= e.failOnWrite {
		return 0, os.ErrPermission
	}
	return len(p), nil
}

func TestFormatsCommand_JSONAndQuery(t *testing.T) {
	t.Run("Output valid JSON with --json flag", func(t *testing.T) {
		formatsCmd := NewFormatsCommand()
		outBuf := new(bytes.Buffer)
		formatsCmd.SetOut(outBuf)
		formatsCmd.SetArgs([]string{"--json"})

		if err := formatsCmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		out := outBuf.String()
		if !strings.Contains(out, `"supported_formats"`) || !strings.Contains(out, `"special_filenames"`) {
			t.Errorf("expected JSON structure with supported_formats and special_filenames, got: %s", out)
		}
	})

	t.Run("Query supported_formats array", func(t *testing.T) {
		formatsCmd := NewFormatsCommand()
		outBuf := new(bytes.Buffer)
		formatsCmd.SetOut(outBuf)
		formatsCmd.SetArgs([]string{"--query", "/supported_formats"})

		if err := formatsCmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		out := outBuf.String()
		if !strings.Contains(out, `"extension"`) || !strings.Contains(out, `"language"`) {
			t.Errorf("expected format entries array in output, got: %s", out)
		}
	})

	t.Run("Query special_filenames array", func(t *testing.T) {
		formatsCmd := NewFormatsCommand()
		outBuf := new(bytes.Buffer)
		formatsCmd.SetOut(outBuf)
		formatsCmd.SetArgs([]string{"--query", "/special_filenames"})

		if err := formatsCmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		out := outBuf.String()
		if !strings.Contains(out, `"filename"`) || !strings.Contains(out, `"language"`) {
			t.Errorf("expected special filename entries array in output, got: %s", out)
		}
	})

	t.Run("Query specific format entry and properties", func(t *testing.T) {
		// Entry object
		cmd1 := NewFormatsCommand()
		buf1 := new(bytes.Buffer)
		cmd1.SetOut(buf1)
		cmd1.SetArgs([]string{"--query", "/supported_formats/0"})
		if err := cmd1.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf1.String(), `"extension"`) {
			t.Errorf("expected format entry object, got: %s", buf1.String())
		}

		// Entry extension
		cmd2 := NewFormatsCommand()
		buf2 := new(bytes.Buffer)
		cmd2.SetOut(buf2)
		cmd2.SetArgs([]string{"--query", "/supported_formats/0/extension"})
		if err := cmd2.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if buf2.Len() == 0 {
			t.Errorf("expected non-empty extension")
		}

		// Entry language
		cmd3 := NewFormatsCommand()
		buf3 := new(bytes.Buffer)
		cmd3.SetOut(buf3)
		cmd3.SetArgs([]string{"--query", "/supported_formats/0/language"})
		if err := cmd3.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if buf3.Len() == 0 {
			t.Errorf("expected non-empty language")
		}
	})

	t.Run("Query specific special filename entry and properties", func(t *testing.T) {
		// Special entry object
		cmd1 := NewFormatsCommand()
		buf1 := new(bytes.Buffer)
		cmd1.SetOut(buf1)
		cmd1.SetArgs([]string{"--query", "/special_filenames/0"})
		if err := cmd1.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf1.String(), `"filename"`) {
			t.Errorf("expected special filename object, got: %s", buf1.String())
		}

		// Special filename property
		cmd2 := NewFormatsCommand()
		buf2 := new(bytes.Buffer)
		cmd2.SetOut(buf2)
		cmd2.SetArgs([]string{"--query", "/special_filenames/0/filename"})
		if err := cmd2.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if buf2.Len() == 0 {
			t.Errorf("expected non-empty filename")
		}

		// Special language property
		cmd3 := NewFormatsCommand()
		buf3 := new(bytes.Buffer)
		cmd3.SetOut(buf3)
		cmd3.SetArgs([]string{"--query", "/special_filenames/0/language"})
		if err := cmd3.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if buf3.Len() == 0 {
			t.Errorf("expected non-empty language")
		}
	})

	t.Run("Query error paths and pointer validation", func(t *testing.T) {
		testCases := []struct {
			name        string
			query       string
			errContains string
		}{
			{"Invalid RFC 6901 pointer syntax", "not_a_valid_pointer", "invalid json pointer query"},
			{"Root token not found", "/unknown_root_key", "not found in root"},
			{"Invalid supported_formats non-integer index", "/supported_formats/abc", "invalid or out-of-range index"},
			{"Out of range supported_formats index", "/supported_formats/99999", "invalid or out-of-range index"},
			{"Invalid special_filenames non-integer index", "/special_filenames/abc", "invalid or out-of-range index"},
			{"Out of range special_filenames index", "/special_filenames/99999", "invalid or out-of-range index"},
			{"Unknown field on FormatEntry", "/supported_formats/0/unknown_field", "unknown field"},
			{"Unknown field on SpecialFileEntry", "/special_filenames/0/unknown_field", "unknown field"},
			{"Cannot navigate into scalar string value", "/supported_formats/0/extension/cannot_nest", "cannot navigate into value of type"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				c := NewFormatsCommand()
				c.SetOut(new(bytes.Buffer))
				c.SetArgs([]string{"--query", tc.query})
				err := c.Execute()
				if err == nil {
					t.Fatalf("expected error for query %q, got nil", tc.query)
				}
				if !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("expected error containing %q, got: %v", tc.errContains, err)
				}
			})
		}
	})

	t.Run("JSON streaming writer error", func(t *testing.T) {
		formatsCmd := NewFormatsCommand()
		formatsCmd.SetOut(new(errOutWriter{failOnWrite: 1}))
		formatsCmd.SetArgs([]string{"--json"})

		if err := formatsCmd.Execute(); err == nil {
			t.Fatalf("expected error when streaming to failing writer, got nil")
		}
	})
}

func TestResolveFormatsPointer_RootAndEmpty(t *testing.T) {
	data := FormatsOutput{
		SupportedFormats: []scanner.FormatEntry{{Extension: ".go", Language: "go"}},
	}
	for _, ptr := range []string{"", "/"} {
		got, err := resolveFormatsPointer(data, ptr)
		if err != nil {
			t.Fatalf("ptr=%q: unexpected error: %v", ptr, err)
		}
		if got == nil {
			t.Errorf("ptr=%q: expected non-nil result", ptr)
		}
	}
}
