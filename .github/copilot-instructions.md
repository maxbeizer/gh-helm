# gh-helm Copilot Instructions

## Project Overview

gh-helm is a `gh` CLI extension providing autonomous developer agents backed by GitHub. It has two agents:
- **Project agent**: claims issues, generates code plans via AI, pushes draft PRs
- **Manager agent**: monitors team activity, maps work to performance pillars, posts observations to 1-1 repos

Read `docs/SOURCE_OF_TRUTH.md` for current goals, key decisions, and known issues before making changes.

## Language & Build

- Go 1.22+
- Build: `go build ./...`
- Test: `go test ./...`
- Lint: standard `go vet ./...`
- Binary name: `gh-helm`

## Code Conventions

- Use `context.Context` for all functions that do I/O or call external processes
- Return errors, never panic. Wrap errors with `fmt.Errorf("context: %w", err)`
- Shell out to `gh` CLI for GitHub API calls (not raw HTTP) — the `internal/github/` package handles this
- Config is TOML (`github.com/BurntSushi/toml`), struct tags use `toml:"field-name"`
- CLI commands use Cobra (`github.com/spf13/cobra`), one file per subcommand in `cmd/`
- All commands must support `--json` and `--jq` flags via `internal/output`

## Package Layout

- `cmd/` — CLI command definitions (thin: parse flags, call internal logic, format output)
- `internal/agent/` — project agent logic (issue → plan → code → PR)
- `internal/manager/` — manager agent logic (observe, prep, pulse, report)
- `internal/github/` — GitHub API wrappers (shell out to `gh`)
- `internal/config/` — TOML config loading/writing
- `internal/notifications/` — pluggable notification dispatch
- `internal/pillars/` — performance pillar mapping
- `internal/guardrails/` — rate limiting and safety checks
- `internal/oneone/` — 1-1 repo observation posting
- `internal/sot/` — source of truth document management
- `internal/doctor/` — project health checks
- `internal/upgrade/` — auto-remediation
- `internal/output/` — JSON/table/jq output formatting

## Testing

- Test files go next to the code they test (`foo_test.go` alongside `foo.go`)
- Use table-driven tests
- Mock `gh` CLI calls by testing the functions that process their output, not the shell calls themselves
- No test framework beyond stdlib `testing`

## Configuration

- Project config: `helm.toml` (parsed by `internal/config/project.go`)
- Manager config: `helm-manager.toml` (parsed by `internal/config/manager.go`)
- State directory: `.helm/` (add to `.gitignore`)

## Key Files

- `docs/SOURCE_OF_TRUTH.md` — living project document, read this first
- `helm.toml.example` — example project config
- `helm-manager.toml.example` — example manager config
