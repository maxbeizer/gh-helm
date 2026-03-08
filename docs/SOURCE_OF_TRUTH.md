# gh-helm — Source of Truth

## Mission

Put autonomous agents on both sides of engineering work — one that does it, one that watches it — with GitHub as the only source of truth.

## Current Focus

All planned features implemented. Zero open issues. Ready for real-world testing against a live project.

## Key Decisions

- **2026-03-08**: Renamed from max-ops to gh-helm; now a `gh` CLI extension for free auth and distribution
- **2026-03-08**: Migrated config from YAML to TOML for unambiguous syntax and cleaner nested structures
- **2026-03-08**: Two config files: `helm.toml` (project agent) and `helm-manager.toml` (manager agent)
- **2026-03-08**: State stored in `.helm/` directory within project root
- **2026-03-08**: GitHub CLI (`gh`) used as primary API interface — subprocess per call, trades perf for auth simplicity
- **2026-03-08**: Pillar mapping uses multi-signal priority: labels > repos > file paths > keywords > AI inference
- **2026-03-08**: Config versioning with integer `version` field — loaders validate and error with migration message
- **2026-03-08**: Structured logging via `log/slog` with `--verbose` flag for debug output
- **2026-03-08**: Atomic state writes via temp file + rename to prevent concurrent corruption
- **2026-03-08**: `runGh` made injectable for testing via `RunGhFunc` package variable

## Architecture

- Go CLI built with Cobra, distributed as `gh` extension
- GitHub Models API for AI inference (code generation, pillar classification)
- Notifications via pluggable interface (Slack webhooks, GitHub issue comments implemented)
- Cron-like scheduling for manager daemon (pulse/prep/observe)

## Outcomes

- [x] Project agent: claim issue → generate plan → write code → draft PR
- [x] Manager agent: observe → prep → pulse → report with pillar mapping
- [x] Daemon modes for both agents (continuous polling / cron scheduling)
- [x] Codespace integration for fire-and-forget issue work
- [x] Doctor + upgrade commands for project health
- [x] JSON/jq output on all commands
- [x] Test coverage (config, pillars, guardrails, schedule, output — 1025 lines)
- [x] Retry/backoff on API failures (3 attempts, exponential 1s/2s)
- [x] Codespace cleanup (deferred delete after creation)
- [x] Error handling hardened (swallowed errors → logged)
- [x] Rate limiting reduced from 3s to 500ms
- [x] Structured logging with slog + --verbose flag
- [x] Atomic state file writes (temp file + rename)
- [x] Integration tests with mocked gh CLI (7 tests)
- [x] Hubber profile-based work suggestions (`gh helm project suggest`)
- [x] Codespace creation on draft PR (`--codespace` flag + config)
- [x] Manager stats command with bus factor analysis

## Risks & Blockers

- No end-to-end tests yet (would require a real GitHub project board)

## Next Up

- Real-world testing against a live project
- AI-powered pillar inference (currently keyword-only fallback)
- Manager agent learning (adapt to what manager values over time)
