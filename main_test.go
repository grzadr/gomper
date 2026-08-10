package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMain(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"gomper", "profiles"}
	main()
}

func TestMain_Error(t *testing.T) {
	if os.Getenv("BE_CRASHING_MAIN") == "1" {
		os.Args = []string{"gomper", "list", "--invalid-flag-that-does-not-exist"}
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestMain_Error")
	cmd.Env = append(os.Environ(), "BE_CRASHING_MAIN=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected main to exit with error status, got success")
	}

	if !strings.Contains(string(out), "Error:") && !strings.Contains(string(out), "unknown flag") {
		t.Errorf("expected error output on stderr, got: %s", string(out))
	}
}

