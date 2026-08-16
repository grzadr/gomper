# Implementation Playbook: Go CLI Application with Cobra & Viper

This playbook provides a step-by-step blueprint to construct a modular, testable, and expandable Go CLI tool.

## Project Structure Blueprint

```
mycli/
├── cmd/
│   ├── root.go
│   └── serve.go
├── internal/
│   ├── app/
│   │   └── server.go
│   └── config/
│       └── config.go
├── go.mod
├── go.sum
└── main.go
```

---

## Step 1: Define Configuration Schema

Create the strongly-typed configuration structure in `internal/config/config.go`. Use `mapstructure` tags to align Viper configuration keys with Go fields.

```go
package config

type Config struct {
	ConfigFile string       `mapstructure:"-"`
	LogLevel   string       `mapstructure:"log_level"`
	Server     ServerConfig `mapstructure:"server"`
}

type ServerConfig struct {
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
	Timeout string `mapstructure:"timeout"`
}
```

---

## Step 2: Implement Root Command Factory & Viper Pipeline

Create `cmd/root.go` using a factory function. This isolates state, binds flags to Viper, transforms environment variable keys, and unmarshals settings inside `PersistentPreRunE`.

```go
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"mycli/internal/config"
)

func NewRootCommand() *cobra.Command {
	v := viper.New()
	cfg := &config.Config{}

	rootCmd := &cobra.Command{
		Use:           "mycli",
		Short:         "Enterprise Scalable Go CLI Application",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initConfig(cmd, v, cfg)
		},
	}

	// Persistent flags (available globally across subcommands)
	rootCmd.PersistentFlags().StringVar(&cfg.ConfigFile, "config", "", "path to custom configuration file")
	rootCmd.PersistentFlags().StringVar(&cfg.LogLevel, "log-level", "info", "logging level (debug, info, warn, error)")

	// Bind persistent flag to Viper
	_ = v.BindPFlag("log_level", rootCmd.PersistentFlags().Lookup("log-level"))

	// Register subcommands
	rootCmd.AddCommand(NewServeCommand(cfg))

	return rootCmd
}

func initConfig(cmd *cobra.Command, v *viper.Viper, cfg *config.Config) error {
	if cfg.ConfigFile != "" {
		v.SetConfigFile(cfg.ConfigFile)
	} else {
		home, err := os.UserHomeDir()
		if err == nil {
			v.AddConfigPath(home)
		}
		v.AddConfigPath(".")
		v.SetConfigName("config")
		v.SetConfigType("yaml")
	}

	// Environment variable transformation (MYCLI_SERVER_PORT -> server.port)
	v.SetEnvPrefix("MYCLI")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	// Default values
	v.SetDefault("server.host", "127.0.0.1")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.timeout", "30s")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read configuration file: %w", err)
		}
	}

	if err := v.Unmarshal(cfg); err != nil {
		return fmt.Errorf("unable to decode configuration: %w", err)
	}

	return nil
}
```

---

## Step 3: Implement Subcommand Factories

Create subcommands in `cmd/serve.go`. Subcommand constructors receive required configuration structs or dependencies directly.

```go
package cmd

import (
	"github.com/spf13/cobra"
	"mycli/internal/app"
	"mycli/internal/config"
)

func NewServeCommand(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the background server daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Access signal-aware context from Cobra execution
			ctx := cmd.Context()

			srv := app.NewServer(cfg.Server.Host, cfg.Server.Port, cfg.LogLevel)
			return srv.Run(ctx)
		},
	}

	// Local command flags
	cmd.Flags().StringVar(&cfg.Server.Host, "host", "127.0.0.1", "server binding interface")
	cmd.Flags().IntVarP(&cfg.Server.Port, "port", "p", 8080, "server listening port")

	return cmd
}
```

Implement the business logic in `internal/app/server.go` decoupled from Cobra:

```go
package app

import (
	"context"
	"fmt"
	"time"
)

type Server struct {
	Host     string
	Port     int
	LogLevel string
}

func NewServer(host string, port int, logLevel string) *Server {
	return &Server{Host: host, Port: port, LogLevel: logLevel}
}

func (s *Server) Run(ctx context.Context) error {
	fmt.Printf("Server executing on %s:%d [Log Level: %s]\n", s.Host, s.Port, s.LogLevel)

	select {
	case <-time.After(500 * time.Millisecond):
		fmt.Println("Server operation finished successfully.")
		return nil
	case <-ctx.Done():
		fmt.Println("Graceful shutdown signal received.")
		return ctx.Err()
	}
}
```

---

## Step 4: Assemble Signal-Aware Entrypoint

In `main.go`, set up signal context interception (`SIGINT`, `SIGTERM`) and pass the context into `rootCmd.ExecuteContext(ctx)`.

```go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"mycli/cmd"
)

func main() {
	// Graceful cancellation context for OS signal delivery
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	rootCmd := cmd.NewRootCommand()
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

---

## Step 5: Unit Testing via Memory Buffers

Construct unit tests in `cmd/root_test.go` without executing compiled binaries or mutating package-level variables.

```go
package cmd_test

import (
	"bytes"
	"testing"

	"mycli/cmd"
)

func TestServeCommand(t *testing.T) {
	rootCmd := cmd.NewRootCommand()

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"serve", "--port", "9090", "--log-level", "debug"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected command execution to succeed, got: %v", err)
	}

	if outBuf.Len() == 0 {
		t.Errorf("expected non-empty output buffer")
	}
}
```

---

## Verification & Precedence Checks

Validate your binary's precedence hierarchy across execution scenarios:

| Test Scenario | Command Execution | Expected Resolution Source |
| --- | --- | --- |
| Default Fallback | `./mycli serve` | Default values (Port 8080) |
| Environment Override | `MYCLI_SERVER_PORT=9000 ./mycli serve` | Environment variable (Port 9000) |
| Flag Priority | `MYCLI_SERVER_PORT=9000 ./mycli serve --port 7000` | Command flag (Port 7000) |
