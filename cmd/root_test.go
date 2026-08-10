package cmd_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grzadr/gomper/cmd"
)

func TestCLI_ListCommand(t *testing.T) {
	t.Run("Requires at least one path argument or config path", func(t *testing.T) {
		rootCmd := cmd.NewRootCommand(nil)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		rootCmd.SetOut(outBuf)
		rootCmd.SetErr(errBuf)

		rootCmd.SetArgs([]string{"list"})
		err := rootCmd.Execute()
		if err == nil {
			t.Fatalf("expected error when running list without arguments or config paths, got nil")
		}
		if !strings.Contains(err.Error(), "requires at least 1 path") {
			t.Errorf("expected path requirement error message, got: %v", err)
		}
	})

	t.Run("Fails on invalid YAML config file syntax", func(t *testing.T) {
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "invalid.yaml")
		_ = os.WriteFile(configFile, []byte("paths: [unclosed_yaml"), 0644)

		rootCmd := cmd.NewRootCommand(nil)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		rootCmd.SetOut(outBuf)
		rootCmd.SetErr(errBuf)

		rootCmd.SetArgs([]string{"list", tempDir, "--config", configFile})
		err := rootCmd.Execute()
		if err == nil {
			t.Fatalf("expected error for invalid YAML syntax, got nil")
		}
		if !strings.Contains(err.Error(), "failed to read configuration file") {
			t.Errorf("expected config read error message, got: %v", err)
		}
	})

	t.Run("Fails on invalid type in YAML config file", func(t *testing.T) {
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "badtype.yaml")
		_ = os.WriteFile(configFile, []byte("format: [invalid_type_array]"), 0644)

		rootCmd := cmd.NewRootCommand(nil)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		rootCmd.SetOut(outBuf)
		rootCmd.SetErr(errBuf)

		rootCmd.SetArgs([]string{"list", tempDir, "--config", configFile})
		err := rootCmd.Execute()
		if err == nil {
			t.Fatalf("expected error for bad type in YAML, got nil")
		}
		if !strings.Contains(err.Error(), "unable to decode configuration") {
			t.Errorf("expected config decode error message, got: %v", err)
		}
	})

	t.Run("Loads paths and profiles from custom YAML configuration file", func(t *testing.T) {
		tempDir := t.TempDir()
		targetDir := filepath.Join(tempDir, "target")
		_ = os.Mkdir(targetDir, 0755)

		goFile := filepath.Join(targetDir, "main.go")
		exeFile := filepath.Join(targetDir, "app.exe")
		_ = os.WriteFile(goFile, []byte("package main"), 0644)
		_ = os.WriteFile(exeFile, []byte("binary"), 0755)

		configContent := "paths:\n  - " + targetDir + "\nprofiles:\n  - go\n"
		configFile := filepath.Join(tempDir, "custom-config.yaml")
		if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}

		rootCmd := cmd.NewRootCommand(nil)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		rootCmd.SetOut(outBuf)
		rootCmd.SetErr(errBuf)

		rootCmd.SetArgs([]string{"list", "--config", configFile})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error running list with custom YAML config: %v", err)
		}

		output := outBuf.String()
		if !strings.Contains(output, "main.go") {
			t.Errorf("expected output to contain main.go from YAML paths, got: %q", output)
		}
		if strings.Contains(output, "app.exe") {
			t.Errorf("expected app.exe to be filtered by 'go' profile loaded from YAML, got: %q", output)
		}
	})

	t.Run("CLI positional argument overrides config file paths", func(t *testing.T) {
		tempDir := t.TempDir()
		configTarget := filepath.Join(tempDir, "config_target")
		cliTarget := filepath.Join(tempDir, "cli_target")
		_ = os.Mkdir(configTarget, 0755)
		_ = os.Mkdir(cliTarget, 0755)

		_ = os.WriteFile(filepath.Join(configTarget, "config.txt"), []byte("config"), 0644)
		_ = os.WriteFile(filepath.Join(cliTarget, "cli.txt"), []byte("cli"), 0644)

		configContent := "paths:\n  - " + configTarget + "\n"
		configFile := filepath.Join(tempDir, "config.yaml")
		_ = os.WriteFile(configFile, []byte(configContent), 0644)

		rootCmd := cmd.NewRootCommand(nil)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		rootCmd.SetOut(outBuf)
		rootCmd.SetErr(errBuf)

		rootCmd.SetArgs([]string{"list", cliTarget, "--config", configFile})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := outBuf.String()
		if !strings.Contains(output, "cli.txt") {
			t.Errorf("expected CLI positional arg path file cli.txt to be listed, got: %q", output)
		}
		if strings.Contains(output, "config.txt") {
			t.Errorf("expected config path file config.txt to be overridden by CLI arg, got: %q", output)
		}
	})

	t.Run("Executes successfully listing files in directory", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		rootCmd := cmd.NewRootCommand(nil)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		rootCmd.SetOut(outBuf)
		rootCmd.SetErr(errBuf)

		rootCmd.SetArgs([]string{"list", tempDir})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := outBuf.String()
		if !strings.Contains(output, "test.txt") {
			t.Errorf("expected output to contain test.txt, got: %q", output)
		}
	})

	t.Run("Executes with embedded profile flag --profile go", func(t *testing.T) {
		tempDir := t.TempDir()
		goFile := filepath.Join(tempDir, "main.go")
		exeFile := filepath.Join(tempDir, "app.exe")
		vendorDir := filepath.Join(tempDir, "vendor")
		_ = os.Mkdir(vendorDir, 0755)
		_ = os.WriteFile(goFile, []byte("package main"), 0644)
		_ = os.WriteFile(exeFile, []byte("binary"), 0755)

		rootCmd := cmd.NewRootCommand(nil)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		rootCmd.SetOut(outBuf)
		rootCmd.SetErr(errBuf)

		rootCmd.SetArgs([]string{"list", tempDir, "--profile", "go"})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := outBuf.String()
		if strings.Contains(output, "app.exe") || strings.Contains(output, "vendor") {
			t.Errorf("expected app.exe and vendor to be ignored by go profile, got output: %q", output)
		}
		if !strings.Contains(output, "main.go") {
			t.Errorf("expected main.go to be listed, got output: %q", output)
		}
	})

	t.Run("Fails cleanly when invalid profile is provided", func(t *testing.T) {
		tempDir := t.TempDir()
		rootCmd := cmd.NewRootCommand(nil)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		rootCmd.SetOut(outBuf)
		rootCmd.SetErr(errBuf)

		rootCmd.SetArgs([]string{"list", tempDir, "--profile", "unknown_profile"})
		err := rootCmd.Execute()
		if err == nil {
			t.Fatalf("expected error for unknown profile, got nil")
		}
		if !strings.Contains(err.Error(), "unknown ignore profile \"unknown_profile\"") {
			t.Errorf("expected unknown profile error message, got: %v", err)
		}
	})

	t.Run("Executes with --ignore flag filtering out matching entries", func(t *testing.T) {
		tempDir := t.TempDir()
		goFile := filepath.Join(tempDir, "main.go")
		txtFile := filepath.Join(tempDir, "notes.txt")
		_ = os.WriteFile(goFile, []byte("package main"), 0644)
		_ = os.WriteFile(txtFile, []byte("notes"), 0644)

		rootCmd := cmd.NewRootCommand(nil)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		rootCmd.SetOut(outBuf)
		rootCmd.SetErr(errBuf)

		rootCmd.SetArgs([]string{"list", tempDir, "--ignore", `\.go$`})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := outBuf.String()
		if strings.Contains(output, "main.go") {
			t.Errorf("expected main.go to be ignored, got output: %q", output)
		}
		if !strings.Contains(output, "notes.txt") {
			t.Errorf("expected notes.txt to be included, got output: %q", output)
		}
	})
}

func TestCLI_ProfilesCommand(t *testing.T) {
	rootCmd := cmd.NewRootCommand(nil)
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)

	rootCmd.SetArgs([]string{"profiles"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error running profiles subcommand: %v", err)
	}

	output := outBuf.String()
	if !strings.Contains(output, "Available ignore profiles:") || !strings.Contains(output, "- go") {
		t.Errorf("expected available profiles output containing '- go', got: %q", output)
	}
}

func TestCLI_DumpCommand(t *testing.T) {
	t.Run("Requires at least one path argument or config path", func(t *testing.T) {
		rootCmd := cmd.NewRootCommand(nil)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		rootCmd.SetOut(outBuf)
		rootCmd.SetErr(errBuf)

		rootCmd.SetArgs([]string{"dump"})
		err := rootCmd.Execute()
		if err == nil {
			t.Fatalf("expected error when running dump without arguments or config paths, got nil")
		}
		if !strings.Contains(err.Error(), "requires at least 1 path") {
			t.Errorf("expected argument requirement error message, got: %v", err)
		}
	})

	t.Run("Executes default markdown dump with one path argument", func(t *testing.T) {
		tempDir := t.TempDir()
		_ = os.WriteFile(filepath.Join(tempDir, "doc.md"), []byte("doc"), 0644)

		rootCmd := cmd.NewRootCommand(nil)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		rootCmd.SetOut(outBuf)
		rootCmd.SetErr(errBuf)

		rootCmd.SetArgs([]string{"dump", tempDir})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := outBuf.String()
		if !strings.Contains(output, "Codebase Context") || !strings.Contains(output, "doc.md") {
			t.Errorf("expected output to indicate markdown format dump and file, got: %q", output)
		}
	})

	t.Run("Executes dump with custom format and output file flags", func(t *testing.T) {
		tempDir := t.TempDir()
		_ = os.WriteFile(filepath.Join(tempDir, "data.xml"), []byte("<xml/>"), 0644)
		outFile := filepath.Join(tempDir, "export.xml")

		rootCmd := cmd.NewRootCommand(nil)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		rootCmd.SetOut(outBuf)
		rootCmd.SetErr(errBuf)

		rootCmd.SetArgs([]string{"dump", tempDir, "--format", "xml", "--output", outFile, "--instructions", "Test instructions"})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := outBuf.String()
		if !strings.Contains(output, "xml format to") {
			t.Errorf("expected output info message, got: %q", output)
		}

		content, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatalf("expected output file %s to be created: %v", outFile, err)
		}
		xmlContent := string(content)
		if !strings.Contains(xmlContent, `<codebase_context version="1.0">`) || !strings.Contains(xmlContent, "Test instructions") {
			t.Errorf("expected XML dump with user instructions, got: %s", xmlContent)
		}
	})

	t.Run("Executes dump with --ignore-dotfiles flag", func(t *testing.T) {
		tempDir := t.TempDir()
		_ = os.WriteFile(filepath.Join(tempDir, ".env"), []byte("SECRET=123"), 0644)
		_ = os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main"), 0644)

		rootCmd := cmd.NewRootCommand(nil)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		rootCmd.SetOut(outBuf)
		rootCmd.SetErr(errBuf)

		rootCmd.SetArgs([]string{"dump", tempDir, "--format", "xml", "--ignore-dotfiles"})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := outBuf.String()
		if strings.Contains(output, ".env") {
			t.Errorf("expected .env to be excluded with --ignore-dotfiles, got: %s", output)
		}
		if !strings.Contains(output, "main.go") {
			t.Errorf("expected main.go to be present in dump output, got: %s", output)
		}
	})

	t.Run("Fails when custom config file does not exist", func(t *testing.T) {
		rootCmd := cmd.NewRootCommand(nil)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		rootCmd.SetOut(outBuf)
		rootCmd.SetErr(errBuf)

		rootCmd.SetArgs([]string{"dump", ".", "--config", "/nonexistent/config/file.yaml"})
		err := rootCmd.Execute()
		if err == nil {
			t.Fatalf("expected error for non-existent config file, got nil")
		}
		if !strings.Contains(err.Error(), "failed to read configuration file") {
			t.Errorf("expected config read error message, got: %v", err)
		}
	})
}

