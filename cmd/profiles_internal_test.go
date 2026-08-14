package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestProfilesCommand_Error(t *testing.T) {
	origList := listProfilesFunc
	listProfilesFunc = func() ([]string, error) {
		return nil, errors.New("simulated list profiles failure")
	}
	defer func() { listProfilesFunc = origList }()

	profilesCmd := NewProfilesCommand()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	profilesCmd.SetOut(outBuf)
	profilesCmd.SetErr(errBuf)

	err := profilesCmd.Execute()
	if err == nil {
		t.Fatalf("expected error from profiles command, got nil")
	}
	if !strings.Contains(err.Error(), "simulated list profiles failure") {
		t.Errorf("expected error message mentioning simulated list profiles failure, got: %v", err)
	}
}
