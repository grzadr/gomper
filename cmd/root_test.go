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

	t.Run("Executes successfully listing files in directory while excluding directories", func(t *testing.T) {
		tempDir := t.TempDir()
		subDir := filepath.Join(tempDir, "subfolder")
		_ = os.Mkdir(subDir, 0755)
		testFile := filepath.Join(tempDir, "test.txt")
		subFile := filepath.Join(subDir, "sub.txt")
		if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		if err := os.WriteFile(subFile, []byte("sub content"), 0644); err != nil {
			t.Fatalf("failed to create sub file: %v", err)
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
		if !strings.Contains(output, "test.txt") || !strings.Contains(output, "sub.txt") {
			t.Errorf("expected output to contain test.txt and sub.txt, got: %q", output)
		}
		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, line := range lines {
			if line == "subfolder" || line == "subfolder/" {
				t.Errorf("expected directory 'subfolder' to be excluded from list output, got line: %q in output: %q", line, output)
			}
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

	t.Run("Executes with embedded profile flag --profile terraform", func(t *testing.T) {
		tempDir := t.TempDir()
		tfFile := filepath.Join(tempDir, "main.tf")
		stateFile := filepath.Join(tempDir, "terraform.tfstate")
		varsFile := filepath.Join(tempDir, "secret.tfvars")
		tfDir := filepath.Join(tempDir, ".terraform")
		_ = os.Mkdir(tfDir, 0755)
		_ = os.WriteFile(tfFile, []byte("resource \"null_resource\" \"x\" {}"), 0644)
		_ = os.WriteFile(stateFile, []byte("{}"), 0644)
		_ = os.WriteFile(varsFile, []byte("secret = 1"), 0644)
		_ = os.WriteFile(filepath.Join(tfDir, "lock.hcl"), []byte(""), 0644)

		rootCmd := cmd.NewRootCommand(nil)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		rootCmd.SetOut(outBuf)
		rootCmd.SetErr(errBuf)

		rootCmd.SetArgs([]string{"list", tempDir, "--profile", "terraform"})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := outBuf.String()
		if strings.Contains(output, "terraform.tfstate") || strings.Contains(output, "secret.tfvars") || strings.Contains(output, ".terraform") {
			t.Errorf("expected state, vars, and .terraform directory to be ignored by terraform profile, got output: %q", output)
		}
		if !strings.Contains(output, "main.tf") {
			t.Errorf("expected main.tf to be listed, got output: %q", output)
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

	t.Run("Executes with --ignore-dir / -D flag filtering out target directories", func(t *testing.T) {
		tempDir := t.TempDir()
		binDir := filepath.Join(tempDir, "bin")
		covDir := filepath.Join(tempDir, "coverage")
		srcDir := filepath.Join(tempDir, "src")
		_ = os.Mkdir(binDir, 0755)
		_ = os.Mkdir(covDir, 0755)
		_ = os.Mkdir(srcDir, 0755)

		_ = os.WriteFile(filepath.Join(binDir, "app"), []byte("bin"), 0755)
		_ = os.WriteFile(filepath.Join(covDir, "cov.out"), []byte("cov"), 0644)
		_ = os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main"), 0644)

		rootCmd := cmd.NewRootCommand(nil)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		rootCmd.SetOut(outBuf)
		rootCmd.SetErr(errBuf)

		rootCmd.SetArgs([]string{"list", tempDir, "-D", "bin", "--ignore-dir", "coverage"})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := outBuf.String()
		if strings.Contains(output, "bin") || strings.Contains(output, "coverage") {
			t.Errorf("expected bin and coverage directories to be ignored, got output: %q", output)
		}
		if !strings.Contains(output, "main.go") {
			t.Errorf("expected main.go to be included, got output: %q", output)
		}
	})

	t.Run("Loads ignore_dir from custom YAML configuration file", func(t *testing.T) {
		tempDir := t.TempDir()
		targetDir := filepath.Join(tempDir, "target")
		_ = os.Mkdir(targetDir, 0755)

		binDir := filepath.Join(targetDir, "bin")
		_ = os.Mkdir(binDir, 0755)
		_ = os.WriteFile(filepath.Join(binDir, "app"), []byte("bin"), 0755)

		goFile := filepath.Join(targetDir, "main.go")
		_ = os.WriteFile(goFile, []byte("package main"), 0644)

		configContent := "paths:\n  - " + targetDir + "\nignore_dir:\n  - bin\n"
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
			t.Fatalf("unexpected error: %v", err)
		}

		output := outBuf.String()
		if strings.Contains(output, "bin") {
			t.Errorf("expected bin directory to be filtered by ignore_dir loaded from YAML, got: %q", output)
		}
		if !strings.Contains(output, "main.go") {
			t.Errorf("expected main.go to be listed, got: %q", output)
		}
	})

	t.Run("Implicitly ignores binary files in list command", func(t *testing.T) {
		tempDir := t.TempDir()
		textFile := filepath.Join(tempDir, "script.py")
		binFile := filepath.Join(tempDir, "executable")

		_ = os.WriteFile(textFile, []byte("print('hello')\n"), 0644)
		_ = os.WriteFile(binFile, []byte{0x7f, 'E', 'L', 'F', 0x00, 0x00, 0x01}, 0755)

		rootCmd := cmd.NewRootCommand(nil)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		rootCmd.SetOut(outBuf)
		rootCmd.SetErr(errBuf)
		rootCmd.SetArgs([]string{"list", tempDir})

		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("unexpected error running list: %v", err)
		}

		output := outBuf.String()
		if !strings.Contains(output, "script.py") {
			t.Errorf("expected script.py to be listed, got: %q", output)
		}
		if strings.Contains(output, "executable") {
			t.Errorf("expected binary file 'executable' to be implicitly ignored, got: %q", output)
		}
	})

	t.Run("Executes list with --format detailed", func(t *testing.T) {
		tempDir := t.TempDir()
		codeFile := filepath.Join(tempDir, "main.go")
		_ = os.WriteFile(codeFile, []byte("package main\n\nfunc main() {}\n"), 0644)

		rootCmd := cmd.NewRootCommand(nil)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		rootCmd.SetOut(outBuf)
		rootCmd.SetErr(errBuf)

		rootCmd.SetArgs([]string{"list", tempDir, "--format", "detailed"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("unexpected error running list --format detailed: %v", err)
		}

		output := outBuf.String()
		if !strings.Contains(output, "FILE") || !strings.Contains(output, "EXTENSION") || !strings.Contains(output, "SIZE") || !strings.Contains(output, "LINES") || !strings.Contains(output, "TOKENS") {
			t.Errorf("expected detailed headers in output, got: %q", output)
		}
		if !strings.Contains(output, "main.go") || !strings.Contains(output, ".go") {
			t.Errorf("expected main.go and .go in detailed output, got: %q", output)
		}
	})

	t.Run("Executes list with --format standard", func(t *testing.T) {
		tempDir := t.TempDir()
		codeFile := filepath.Join(tempDir, "app.py")
		_ = os.WriteFile(codeFile, []byte("print('hello')\n"), 0644)

		rootCmd := cmd.NewRootCommand(nil)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		rootCmd.SetOut(outBuf)
		rootCmd.SetErr(errBuf)

		rootCmd.SetArgs([]string{"list", tempDir, "--format", "standard"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("unexpected error running list --format standard: %v", err)
		}

		output := outBuf.String()
		if !strings.Contains(output, "app.py") {
			t.Errorf("expected app.py in standard output, got: %q", output)
		}
		if strings.Contains(output, "EXTENSION") {
			t.Errorf("expected no detailed headers in standard output, got: %q", output)
		}
	})

	t.Run("Fails cleanly when invalid --format is provided", func(t *testing.T) {
		tempDir := t.TempDir()
		rootCmd := cmd.NewRootCommand(nil)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		rootCmd.SetOut(outBuf)
		rootCmd.SetErr(errBuf)

		rootCmd.SetArgs([]string{"list", tempDir, "--format", "unsupported_format"})
		err := rootCmd.Execute()
		if err == nil {
			t.Fatalf("expected error for unsupported format, got nil")
		}
		if !strings.Contains(err.Error(), "unsupported list format") {
			t.Errorf("expected unsupported list format error, got: %v", err)
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

func TestCLI_FormatsCommand(t *testing.T) {
	rootCmd := cmd.NewRootCommand(nil)
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)

	rootCmd.SetArgs([]string{"formats"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error running formats subcommand: %v", err)
	}

	output := outBuf.String()
	if !strings.Contains(output, "Supported file formats:") {
		t.Errorf("expected formats output containing 'Supported file formats:', got: %q", output)
	}
	if !strings.Contains(output, "- .go (go)") {
		t.Errorf("expected formats output containing '- .go (go)', got: %q", output)
	}
	if !strings.Contains(output, "Special filenames:") {
		t.Errorf("expected formats output containing 'Special filenames:', got: %q", output)
	}
	if !strings.Contains(output, "- makefile (makefile)") {
		t.Errorf("expected formats output containing '- makefile (makefile)', got: %q", output)
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

	t.Run("Implicitly ignores binary files in markdown and xml dump", func(t *testing.T) {
		tempDir := t.TempDir()
		textFile := filepath.Join(tempDir, "main.go")
		binFile := filepath.Join(tempDir, "binary_blob")

		_ = os.WriteFile(textFile, []byte("package main\n\nfunc main() {}\n"), 0644)
		_ = os.WriteFile(binFile, []byte{0x00, 0xff, 0xfe, 0x00}, 0755)

		// Test Markdown Dump
		mdCmd := cmd.NewRootCommand(nil)
		mdOut := new(bytes.Buffer)
		mdCmd.SetOut(mdOut)
		mdCmd.SetArgs([]string{"dump", tempDir, "--format", "markdown"})
		if err := mdCmd.Execute(); err != nil {
			t.Fatalf("unexpected error running markdown dump: %v", err)
		}
		mdResult := mdOut.String()
		if !strings.Contains(mdResult, "main.go") {
			t.Errorf("expected markdown dump to contain main.go")
		}
		if strings.Contains(mdResult, "binary_blob") {
			t.Errorf("expected markdown dump to omit binary_blob")
		}

		// Test XML Dump
		xmlCmd := cmd.NewRootCommand(nil)
		xmlOut := new(bytes.Buffer)
		xmlCmd.SetOut(xmlOut)
		xmlCmd.SetArgs([]string{"dump", tempDir, "--format", "xml"})
		if err := xmlCmd.Execute(); err != nil {
			t.Fatalf("unexpected error running xml dump: %v", err)
		}
		xmlResult := xmlOut.String()
		if !strings.Contains(xmlResult, "main.go") {
			t.Errorf("expected xml dump to contain main.go")
		}
		if strings.Contains(xmlResult, "binary_blob") {
			t.Errorf("expected xml dump to omit binary_blob")
		}
	})
}

func TestCLI_NameFilterFlag(t *testing.T) {
	t.Run("Executes list with --name / -n flag filtering whole file name", func(t *testing.T) {
		tempDir := t.TempDir()
		_ = os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main"), 0644)
		_ = os.WriteFile(filepath.Join(tempDir, "utils.js"), []byte("console.log()"), 0644)
		_ = os.WriteFile(filepath.Join(tempDir, "README.md"), []byte("# Readme"), 0644)

		rootCmd := cmd.NewRootCommand(nil)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		rootCmd.SetOut(outBuf)
		rootCmd.SetErr(errBuf)

		rootCmd.SetArgs([]string{"list", tempDir, "-n", ".*\\.go"})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := outBuf.String()
		if !strings.Contains(output, "main.go") {
			t.Errorf("expected main.go in output, got: %q", output)
		}
		if strings.Contains(output, "utils.js") || strings.Contains(output, "README.md") {
			t.Errorf("expected utils.js and README.md to be filtered by -n flag, got: %q", output)
		}
	})

	t.Run("Loads name_filter from YAML config file", func(t *testing.T) {
		tempDir := t.TempDir()
		_ = os.WriteFile(filepath.Join(tempDir, "app.py"), []byte("print('hi')"), 0644)
		_ = os.WriteFile(filepath.Join(tempDir, "notes.txt"), []byte("notes"), 0644)

		configContent := "paths:\n  - " + tempDir + "\nname_filter:\n  - '.*\\.py'\n"
		configFile := filepath.Join(tempDir, "name-config.yaml")
		_ = os.WriteFile(configFile, []byte(configContent), 0644)

		rootCmd := cmd.NewRootCommand(nil)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		rootCmd.SetOut(outBuf)
		rootCmd.SetErr(errBuf)

		rootCmd.SetArgs([]string{"list", "--config", configFile})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := outBuf.String()
		if !strings.Contains(output, "app.py") {
			t.Errorf("expected app.py in output, got: %q", output)
		}
		if strings.Contains(output, "notes.txt") {
			t.Errorf("expected notes.txt to be filtered by name_filter from config, got: %q", output)
		}
	})

	t.Run("Executes dump with --name flag filtering output", func(t *testing.T) {
		tempDir := t.TempDir()
		_ = os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main"), 0644)
		_ = os.WriteFile(filepath.Join(tempDir, "data.json"), []byte("{}"), 0644)

		rootCmd := cmd.NewRootCommand(nil)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		rootCmd.SetOut(outBuf)
		rootCmd.SetErr(errBuf)

		rootCmd.SetArgs([]string{"dump", tempDir, "--name", ".*\\.go"})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := outBuf.String()
		if !strings.Contains(output, "main.go") {
			t.Errorf("expected main.go in dump output, got: %q", output)
		}
		if strings.Contains(output, "data.json") {
			t.Errorf("expected data.json to be filtered out, got: %q", output)
		}
	})
}

func TestCLI_VersionFlag(t *testing.T) {
	t.Run("Displays version with --version flag", func(t *testing.T) {
		rootCmd := cmd.NewRootCommand(nil)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		rootCmd.SetOut(outBuf)
		rootCmd.SetErr(errBuf)

		rootCmd.SetArgs([]string{"--version"})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error running --version: %v", err)
		}

		output := outBuf.String()
		if !strings.Contains(output, "gomper version") {
			t.Errorf("expected output to contain 'gomper version', got: %q", output)
		}
	})

	t.Run("Displays version with -v shorthand flag", func(t *testing.T) {
		rootCmd := cmd.NewRootCommand(nil)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		rootCmd.SetOut(outBuf)
		rootCmd.SetErr(errBuf)

		rootCmd.SetArgs([]string{"-v"})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error running -v: %v", err)
		}

		output := outBuf.String()
		if !strings.Contains(output, "gomper version") {
			t.Errorf("expected output to contain 'gomper version', got: %q", output)
		}
	})
}



