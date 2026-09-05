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

func TestFormatsCommand_JSON(t *testing.T) {
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

	t.Run("JSON streaming writer error", func(t *testing.T) {
		formatsCmd := NewFormatsCommand()
		formatsCmd.SetOut(new(errOutWriter{failOnWrite: 1}))
		formatsCmd.SetArgs([]string{"--json"})

		if err := formatsCmd.Execute(); err == nil {
			t.Fatalf("expected error when streaming to failing writer, got nil")
		}
	})
}
