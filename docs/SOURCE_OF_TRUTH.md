# gh-helm — Source of Truth

## Mission

Put autonomous agents on both sides of engineering work — one that does it, one that watches it — with GitHub as the only source of truth.

## Current Focus

Phase 1–4 implementation is structurally complete. Core needs before real-world use:

- Test coverage (zero tests currently)
- Error handling hardening (swallowed errors, missing retries)
- Performance improvements (sequential API calls, hardcoded rate-limit delays)

## Key Decisions

- **2026-03-08**: Renamed from max-ops to gh-helm; now a `gh` CLI extension for free auth and distribution
- **2026-03-08**: Migrated config from YAML to TOML for unambiguous syntax and cleaner nested structures
- **2026-03-08**: Two config files: `helm.toml` (project agent) and `helm-manager.toml` (manager agent)
- **2026-03-08**: State stored in `.helm/` directory within project root
- **2026-03-08**: GitHub CLI (`gh`) used as primary API interface — subprocess per call, trades perf for auth simplicity
- **2026-03-08**: Pillar mapping uses multi-signal priority: labels > repos > file paths > keywords > AI inference

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
- [ ] Test coverage
- [ ] Retry/backoff on API failures
- [ ] Codespace cleanup (delete after use)
- [ ] Structured logging with levels
- [ ] Atomic state file writes (race condition fix)

## Risks & Blockers

- Zero test files — any refactoring is high-risk without coverage
- State files (`.helm/state.json`, `.helm/failures.json`) have race conditions under concurrent agents
- Hardcoded 3-second rate-limit delays make manager commands slow for larger teams
- Codespace leak: `DeleteCodespace()` exists but is never called in daemon flow

## Next Up

- Add unit tests for pillar mapping, config parsing, guardrails, and schedule logic
- Implement retry with exponential backoff on `gh` CLI calls
- Replace hardcoded rate-limit sleep with header-based adaptive delays
- Wire up `DeleteCodespace()` in daemon cleanup
