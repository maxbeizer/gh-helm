# Changelog

## v0.2.0

### Improvements

- **Human-friendly `project status` output** — Shows formatted session info, issues worked, and PRs created instead of raw Go struct dump. `--json`/`--jq` flags still available for machine-readable output.
- **`upgrade` / `doctor --fix` scaffolds `helm.toml`** — Running `gh helm upgrade` or `gh helm doctor --fix` in a repo without `helm.toml` now creates one with sensible defaults instead of silently skipping.
- **Relaxed config validation** — `project.board` and `project.owner` are now optional, so a minimal `helm.toml` works without project board integration.

### Bug Fixes

- **Fixed zero-time display in status** — No longer shows `0001-01-01 00:00:00` when no session has run; omits the field entirely.

## v0.1.0

### Features

- **Project context detection** — Agent auto-detects project language from manifest files (`go.mod`, `package.json`, `Cargo.toml`, etc.) and includes it in the AI prompt so generated code matches the project's language and conventions.
- **Enriched AI prompt** — System prompt now includes detected language, manifest contents, `.gitignore`, and a proper file tree via `git ls-tree`.
- **Targeted git staging** — Agent stages only planned files instead of `git add .`, preventing accidental commits of binaries and build artifacts.
