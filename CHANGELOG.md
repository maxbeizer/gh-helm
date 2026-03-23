# Changelog

## v0.2.2

### Bug Fixes

- **Models API retry with backoff** — The models API now retries up to 3 times with exponential backoff on transient errors (5xx, timeouts). Timeout increased from 60s to 120s. Specific error messages for rate limiting (429), timeouts (408/504), and server errors. Closes #27.
- **Handle readline errors in project init** — All interactive prompts now properly check for and report read errors instead of silently accepting empty input. Closes #28.

## v0.2.1

### Bug Fixes

- **Removed 3-second sleep from `doctor` auth check** — `gh helm doctor` is now ~27 seconds faster.
- **Fixed `runCmd` ignoring `name` parameter** — Renamed to `runGit` and removed the unused parameter. Closes #25.
- **Fixed doctor hint text** — Now correctly says `gh helm doctor --fix` instead of `gh-helm upgrade`.

### Improvements

- **Added `WantsJSON()` tests** — Output package now has full test coverage for JSON detection.

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
