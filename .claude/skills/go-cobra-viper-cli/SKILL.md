---
name: go-cobra-viper-cli
description: Architect and bootstrap production-ready, modular Go CLI applications using Cobra and Viper without global state. Use whenever the user wants to scaffold a new Go CLI tool, add subcommands to an existing Cobra app, wire up Viper configuration (flags/env/config file precedence), or refactor a CLI away from package-level globals toward testable factory functions.
---

# Go CLI Architecture with Cobra and Viper

## Skill Overview

This skill provides design patterns, rules, and blueprints for bootstrapping expandable, enterprise-grade Go command-line tools using `spf13/cobra` and `spf13/viper`. It prioritizes testability, Twelve-Factor configuration hierarchy, signal-aware context propagation, and complete decoupling of CLI transport logic from domain business execution.

## Core Architecture Principles

1. **Zero Global State**: Avoid package-level command variables and implicit `init()` setup hooks. Instantiation occurs via factory functions (`NewRootCommand`, `NewServeCommand`) returning isolated command structures.

2. **Explicit Precedence Stack**: Enforce strict resolution priority: Flags > Environment Variables > Configuration File > Application Defaults.

3. **Decoupled Transport and Domain**: Keep Cobra flag parsing and Viper setup inside `cmd/`, and delegate core workflows to `internal/app/` or `internal/config/`.

4. **Signal Context Propagation**: Use Go's `context.Context` derived from OS signals (`SIGINT`, `SIGTERM`) passed into `ExecuteContext(ctx)` for clean cancellation.

5. **In-Memory Execution Testing**: Test command trees in memory by overriding standard streams (`SetOut`, `SetErr`) without spawning subprocesses.

## Key Technical Rules & Anti-Patterns

| Category | Anti-Pattern | Recommended Pattern |
| --- | --- | --- |
| Command Definition | Package-level `var rootCmd = &cobra.Command{...}` | Constructor function `func NewRootCommand() *cobra.Command` |
| Viper Instance | Using global `viper.GetString()` everywhere | Isolated `v := viper.New()` unmarshaled into typed `*config.Config` struct |
| Env Variable Naming | Unmapped kebab-case flags breaking env variable lookup | Configure `v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))` |
| Error Handling | Printing default usage dumps on operational errors | Set `SilenceUsage = true` and `SilenceErrors = true` on the root command |
| Context Handling | Creating `context.Background()` inside command handlers | Accessing `cmd.Context()` derived from `ExecuteContext(ctx)` |

## Workflow & Scaffolding Checklist

1. Define configuration schema with `mapstructure` tags in `internal/config/config.go`.
2. Construct root factory in `cmd/root.go` that initializes an isolated Viper instance, configures environment replacers, binds persistent flags, and unmarshals configuration during `PersistentPreRunE`.
3. Construct subcommand factories in `cmd/` accepting dependency references (`*config.Config`, services, logger).
4. Set up `main.go` with signal-notify context (`SIGINT`, `SIGTERM`) and delegate error handling to the top level.
5. Add unit tests in `cmd/root_test.go` using memory buffers for stdout/stderr assertions.

## Reference Implementations

* `resources/implementation-playbook.md`: Detailed step-by-step implementation guide with complete code scaffolding, covering configuration schema, root/subcommand factories, the signal-aware entrypoint, unit testing, and precedence verification.

Read the playbook whenever you're actually generating the project files rather than just discussing architecture — it has copy-pasteable code for every file in the blueprint.
