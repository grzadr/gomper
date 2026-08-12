# gomper

[![Go Version](https://img.shields.io/badge/Go-1.26%2B-00ADD8?style=flat&logo=go)](https://go.dev)
[![Coverage](https://img.shields.io/badge/Coverage-97.4%25-brightgreen.svg)](Makefile)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

`gomper` is a high-performance Go CLI application designed to inspect directory structures and dump them into formatted Markdown or XML files.

Built with **Go 1.26 range-over-function iterators (`iter.Seq2`)**, Cobra, and Viper, `gomper` adheres to strict zero-global-state architecture, signal-aware context propagation, structured `log/slog` logging, and Twelve-Factor configuration principles.

---

## Features

- **Range-Over-Function Iteration**: Memory-efficient directory traversal using Go 1.26 native `iter.Seq2[Entry, error]` iterators.
- **Embedded Gitignore Profiles**: Preset ignore templates (`generic`, `go`, `node`, `python`, `java`, `cpp`, `rust`) embedded into the binary with `//go:embed`.
- **File Name & Regex Filtering**: Filter files matching custom regular expressions against their whole base file name (`-n` / `--name`), exclude files/directories with custom ignore rules (`-i` / `--ignore`), ignore directories (`-D` / `--ignore-dir`), or hide dotfiles (`-d` / `--ignore-dotfiles`). Filtering is strictly evaluated in 4 steps: 1. ignore dotfiles, 2. ignore directories, 3. name filter, 4. ignore patterns.
- **Atomic File Output**: Transactional file writing via `internal/filetx` using temporary files, fsync, and parent directory sync to prevent partial writes.
- **Token Estimation & Line Numbering**: Automatic token estimation (~4 chars per token) and line numbering (`1 | content...`) tailored for LLM context consumption.
- **CLI Subcommands**:
  - `list`: Inspect matching regular files with optional filtering (`-n`, `-i`, `-p`, `-d`, `-D`) and detailed attribute views (`-l`).
  - `dump`: Export directory structure into Markdown or XML formats (`-f`, `-o`, `-u` / `--instructions`).
  - `profiles`: Display available embedded language ignore templates.
- **YAML Configuration File**: Load target `paths`, `profiles`, `name_filter`, `ignore` patterns, `ignore_dir`, `ignore_dotfiles`, `instructions`, `format`, and `log_level` from `gomper.yaml`.
- **Structured Logging**: Context-aware `log/slog` logger with dynamic level mutation (`slog.LevelVar`).
- **Zero Global State Architecture**: Decoupled CLI transport logic (`cmd/`) and domain execution logic (`internal/app/`, `internal/dumper/`, `internal/filetx/`, `internal/scanner/`, `internal/setup/`).

---

## Installation

### Prerequisites

- [Go 1.26](https://go.dev/dl/) or higher

### Building with Makefile

`gomper` includes a Makefile modeled after `godlv` that runs cleanups, linting (`golangci-lint` or `go vet`), test coverage calculation, minimum threshold verification (fails if coverage < 90%), and binary compilation into `bin/gomper`:

```bash
# Build binary (runs clean, lint, test-coverage, and coverage threshold check automatically)
make build

# Run all unit tests
make test

# Run tests with verbose output
make test-verbose

# Generate test coverage reports and verify 90% minimum threshold
make test-coverage
make check-coverage
make coverage-html

# Run linter
make lint

# Clean build artifacts
make clean
```

---

## Quick Start

### 1. List Files in Directories

List regular files within one or more paths (skipping directory nodes):

```bash
./bin/gomper list ./cmd
```

#### Output
```
dump.go
list.go
profiles.go
root.go
root_test.go
```

#### Options

- **File Name Filter** (`-n`, `--name`): Filter files matching custom regular expressions against their whole base file name (`info.Name()`). Non-matching files are excluded.
  ```bash
  ./bin/gomper list . --name ".*\.go$" --ignore ".*_test\.go$"
  ```

  > **Evaluation Sequence**:
  > Filtering follows a strict 4-step sequence:
  > 1. **Ignore dot files** (`-d` / `--ignore-dotfiles`)
  > 2. **Ignore directories** (`-D` / `--ignore-dir`)
  > 3. **Name filter** (`-n` / `--name`)
  > 4. **Ignore flag & profiles** (`-i` / `--ignore` / `--profile`)

- **Language & Generic Ignore Profiles** (`-p`, `--profile`): Apply preset ignore templates (`generic`, `go`, `node`, `python`, `java`, `cpp`, `rust`). The `generic` profile automatically excludes environment files (`.env`, `.env.*`), OS metadata (`.DS_Store`, `Thumbs.db`), IDE configs (`.vscode/`, `.idea/`), and VCS metadata (`.git/`).
  ```bash
  ./bin/gomper list . --profile generic --profile go
  ```

- **Ignore Directory (Gitignore Convention)** (`-D`, `--ignore-dir`): Filter out directories matching gitignore conventions (e.g. `bin`, `coverage`, `bin/`, `/build`).
  ```bash
  ./bin/gomper list . --ignore-dir bin --ignore-dir coverage
  ```

- **Regex Pattern Ignore** (`-i`, `--ignore`): Filter out files or directories matching custom Perl-compatible regular expressions (supports lookahead and lookbehind assertions).
  ```bash
  ./bin/gomper list . --ignore "_test\.go$" --ignore "node_modules"
  ```

- **Ignore Hidden Dotfiles** (`-d`, `--ignore-dotfiles`): Skip all hidden files and directories starting with `.`.
  ```bash
  ./bin/gomper list . --ignore-dotfiles
  ```

- **Detailed Attributes** (`-l`, `--long`): Display type, size, file mode permissions, and path.
  ```bash
  ./bin/gomper list ./cmd --long
  ```
  ```
  FILE         771 B  -rw-r--r--  dump.go
  FILE         824 B  -rw-r--r--  list.go
  FILE        2191 B  -rw-r--r--  root.go
  FILE        5400 B  -rw-r--r--  root_test.go
  ```

---

### 2. List Available Ignore Profiles

Display all embedded gitignore language templates:

```bash
./bin/gomper profiles
```

#### Output
```
Available ignore profiles:
  - cpp
  - generic
  - go
  - java
  - node
  - python
  - rust
```

---

### 3. Dump Directory Structure

Export target directories into a single file or standard output:

#### Markdown Dump (Default)
```bash
./bin/gomper dump . --format markdown --output structure.md
```

#### XML Dump with Custom User Instructions
```bash
./bin/gomper dump ./cmd -f xml -o structure.xml -u "Refactor package scanner to improve memory efficiency"
```

#### Ignore Dotfiles in Dump
```bash
./bin/gomper dump . -d -f markdown -o context.md
```

---

## YAML Configuration File

`gomper` automatically reads `./gomper.yaml` or `$HOME/gomper.yaml` (or a custom path via `--config <file>`). You can specify default target paths, profiles, format, instructions, custom ignore regexes, and directory ignore rules:

```yaml
# gomper.yaml
paths:
  - ./cmd
  - ./internal

profiles:
  - generic
  - go

name_filter:
  - ".*\\.go$"

ignore:
  - "^tmp/"

ignore_dir:
  - bin
  - coverage

ignore_dotfiles: true

instructions: "Analyze directory structure and provide architectural feedback."

format: markdown
log_level: info
```

When `paths` are specified in `gomper.yaml`, running `./bin/gomper list` or `./bin/gomper dump` without CLI positional arguments will process the configured paths automatically. Positional CLI arguments will override config file paths.

---

## Configuration Hierarchy

`gomper` supports configuration through command-line flags, environment variables, and configuration files (`gomper.yaml` / `gomper.yml`).

| Setting | CLI Flag | Environment Variable | Config YAML Key | Default Value |
| --- | --- | --- | --- | --- |
| Custom Config File | `--config` | - | - | `.` / `$HOME/gomper.yaml` |
| Target Paths | - | - | `paths` | Positional CLI args |
| File Name Filter | `--name`, `-n` | `GOMPER_NAME` | `name` / `name_filter` / `name_filters` | `[]` |
| Ignore Profiles | `--profile`, `-p` | `GOMPER_PROFILE` | `profiles` / `profile` | `[]` |
| Custom Ignore Regex | `--ignore`, `-i` | `GOMPER_IGNORE` | `ignore` | `[]` |
| Ignore Directory | `--ignore-dir`, `-D` | `GOMPER_IGNORE_DIR` | `ignore_dir` / `ignore_dirs` | `[]` |
| Ignore Dotfiles | `--ignore-dotfiles`, `-d` | `GOMPER_IGNORE_DOTFILES` | `ignore_dotfiles` | `false` |
| User Instructions | `--instructions`, `-u` | `GOMPER_INSTRUCTIONS` | `instructions` | `""` |
| Log Level | `--log-level` | `GOMPER_LOG_LEVEL` | `log_level` | `info` |
| Output Format | `--format`, `-f` | `GOMPER_FORMAT` | `format` | `markdown` |
| Output Path | `--output`, `-o` | `GOMPER_OUTPUT` | `output` | `stdout` |


---

## Architecture & Project Structure

```
gomper/
├── cmd/                # Cobra CLI subcommand factories (Zero global state)
│   ├── dump.go         # Dump subcommand factory
│   ├── list.go         # List subcommand factory
│   ├── profiles.go     # Profiles subcommand factory
│   ├── root.go         # Root command & Viper precedence setup
│   └── root_test.go    # In-memory execution unit tests
├── internal/
│   ├── app/            # Core application service & OutputFormat enum
│   │   ├── app.go      # Service interface and runner logic
│   │   ├── app_test.go # Service unit test suite (98.7% coverage)
│   │   ├── format.go   # OutputFormat enum definition & pflag binding
│   │   └── format_test.go
│   ├── config/         # Strongly-typed configuration schema
│   │   ├── config.go   # Config struct & profile helpers
│   │   └── config_test.go
│   ├── dumper/         # Markdown & XML document generator & token estimator
│   │   ├── dumper.go   # XMLDumper, token estimation & directory tree renderer
│   │   └── dumper_test.go
│   ├── filetx/         # Crash-safe atomic transactional file writer
│   │   ├── filetx.go   # WriteAtomically with fsync and directory sync
│   │   └── filetx_test.go
│   ├── scanner/        # File scanner range-over-function iterator & embedded gitignore profiles
│   │   ├── extensions.go # Extension-to-language lookup
│   │   ├── extensions_test.go
│   │   ├── profile.go  # Profile loader & gitignore-to-regex converter
│   │   ├── profiles/   # Embedded gitignore template files (generic, go, node, python, etc.)
│   │   ├── profile_test.go
│   │   ├── scanner.go  # WalkPaths (iter.Seq2[Entry, error])
│   │   └── scanner_test.go
│   └── setup/          # Application setup, slog structured logger & signal context
│       ├── setup.go    # App struct, LevelVar logger & NewContext factory
│       └── setup_test.go
├── bin/                # Output directory for compiled binary (make build)
├── coverage/           # Coverage profiles generated by make test-coverage
├── go.mod              # Go module definition
├── gomper.yaml.example # Example configuration file template
├── LICENSE             # MIT License
├── Makefile            # Build automation script
├── main.go             # Signal-aware entrypoint initializing App
├── main_test.go        # Main entrypoint unit test
└── README.md           # Documentation
```

---

## Testing & Code Quality

`gomper` maintains **97.4% overall statement test coverage** (with near 100% coverage across core internal packages):

| Package | Statement Coverage |
| --- | --- |
| `github.com/grzadr/gomper` (`main`) | **100.0%** |
| `internal/config` | **100.0%** |
| `internal/setup` | **100.0%** |
| `internal/dumper` | **99.5%** |
| `internal/app` | **98.7%** |
| `cmd` | **96.9%** |
| `internal/scanner` | **95.7%** |
| `internal/filetx` | **87.1%** |
| **Total** | **97.4%** |

Run unit tests and coverage analysis:

```bash
make test-verbose
make test-coverage
```

---

## License

This project is licensed under the [MIT License](LICENSE) - see the LICENSE file for details.

