---
name: git-commit-per-step
trigger: always_on
description: Enforces an atomic git stage and commit after every discrete implementation step.
---

# Git Commit Policy for Incremental Steps

After completing each discrete implementation step, refactoring task, or test addition, you must verify and commit the changes immediately.

## Execution Requirements

1. **Step Boundary Definition**: A single step consists of an isolated change (e.g., adding a struct and interface, implementing a specific method, writing a table-driven test case, or refactoring a package).
2. **Pre-Commit Verification**: Run verification commands before staging:
   - `go vet ./...`
   - `go test -race ./...`
   - `make build`
   - Ensure the repository builds cleanly with zero diagnostics.
3. **Stage Changes**: Stage only the files associated with the completed step:
   ```bash
   git add .
