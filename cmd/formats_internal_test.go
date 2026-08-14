package cmd

import (
	"bytes"
	"errors"
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
