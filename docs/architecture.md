# Architecture

## Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        gh-helm                               │
│                                                              │
│   ┌─────────────────────┐    ┌────────────────────────────┐ │
│   │   Project Agent      │    │   Manager Agent             │ │
│   │                      │    │                             │ │
│   │   • Claims issues    │    │   • Monitors team activity  │ │
│   │   • Breaks down work │    │   • Maps to perf pillars    │ │
│   │   • Writes code      │    │   • Posts to 1-1 repos      │ │
│   │   • Pushes draft PRs │    │   • Flags blockers/wins     │ │
│   │   • Updates source   │    │   • Updates source of truth │ │
│   │     of truth doc     │    │     across projects         │ │
│   └────────┬────────────┘    └──────────┬──────────────────┘ │
│            │                            │                     │
│   ┌────────▼────────────────────────────▼──────────────────┐ │
│   │                    GitHub                               │ │
│   │   Issues · Projects v2 · PRs · Repos · Markdown docs   │ │
│   └────────────────────────┬───────────────────────────────┘ │
│                            │                                  │
│   ┌────────────────────────▼───────────────────────────────┐ │
│   │                Notifications                            │ │
│   │   Slack · Teams · Discord · GitHub (configurable)       │ │
│   └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## Package Structure

```
gh-helm/
├── cmd/
│   ├── root.go                 # Cobra root command
│   ├── project.go              # Project agent subcommand group
│   ├── project_init.go         # gh helm project init
│   ├── project_start.go        # gh helm project start
│   ├── project_daemon.go       # gh helm project daemon
│   ├── project_status.go       # gh helm project status
│   ├── project_sot.go          # gh helm project sot
│   ├── manager.go              # Manager agent subcommand group
│   ├── manager_init.go         # gh helm manager init
│   ├── manager_start.go        # gh helm manager start
│   ├── manager_observe.go      # gh helm manager observe
│   ├── manager_prep.go         # gh helm manager prep
│   ├── manager_pulse.go        # gh helm manager pulse
│   ├── manager_pillars.go      # gh helm manager pillars
│   ├── manager_report.go       # gh helm manager report
│   ├── config.go               # gh helm config show
│   ├── doctor.go               # gh helm doctor
│   ├── upgrade.go              # gh helm upgrade
│   └── version.go              # gh helm version
├── internal/
│   ├── agent/                  # Project agent core logic
│   │   ├── project.go          # Issue → plan → code → PR flow
│   │   ├── codegen.go          # AI-powered code generation
│   │   ├── codespace.go        # Codespace lifecycle management
│   │   ├── daemon.go           # Continuous polling loop
│   │   └── status.go           # Agent state persistence
│   ├── manager/                # Manager agent core logic
│   │   ├── manager.go          # Shared types and helpers
│   │   ├── observe.go          # Weekly observation generation
│   │   ├── prep.go             # 1-1 meeting prep
│   │   ├── pulse.go            # Team health pulse
│   │   ├── report.go           # Full report card generation
│   │   └── daemon.go           # Scheduled cron-like daemon
│   ├── github/                 # GitHub API integration
│   │   ├── gh.go               # gh CLI wrapper
│   │   ├── projects.go         # Projects v2 GraphQL API
│   │   ├── issues.go           # Issues API
│   │   ├── models.go           # GitHub Models API (AI inference)
│   │   ├── search.go           # Search API (PRs, issues, reviews)
│   │   └── repo.go             # Repository operations
│   ├── config/                 # Configuration loading
│   │   ├── project.go          # helm.toml parsing
│   │   └── manager.go          # helm-manager.toml parsing
│   ├── notifications/          # Notification dispatch
│   │   ├── notifier.go         # Notifier interface
│   │   ├── factory.go          # Channel factory
│   │   ├── slack.go            # Slack webhooks
│   │   └── github.go           # GitHub issue comments
│   ├── pillars/                # Performance pillar system
│   │   └── mapper.go           # Map activity → pillars
│   ├── guardrails/             # Rate limiting and safety
│   │   └── guardrails.go       # Sliding window rate limiter
│   ├── oneone/                 # 1-1 repo integration
│   │   └── repo.go             # Post observations to repos
│   ├── sot/                    # Source of truth management
│   │   └── source.go           # Read/propose SOT updates
│   ├── doctor/                 # Health checks
│   │   └── doctor.go           # Project health validation
│   ├── upgrade/                # Auto-remediation
│   │   └── upgrade.go          # Fix project setup issues
│   └── output/                 # Output formatting
│       └── output.go           # JSON/table/jq output
├── docs/                       # Documentation
├── helm.toml.example
├── helm-manager.toml.example
├── go.mod
├── Makefile
└── README.md
```

## Design Decisions

### gh CLI as GitHub interface

gh-helm shells out to the `gh` CLI for all GitHub API interactions rather than using a Go HTTP client directly. This gives us:

- **Free authentication** — uses the user's existing `gh auth` session
- **Correct scopes** — `gh` handles token management
- **Simpler distribution** — as a `gh` extension, installation is `gh ext install`

The tradeoff is subprocess overhead per API call, but for the cadence of agent operations (minutes between actions), this is negligible.

### TOML for configuration

Config files use TOML rather than YAML because:

- Unambiguous syntax (no Norway problem, no implicit type coercion)
- Better support for inline tables and arrays of tables
- Cleaner for the nested pillar definitions

### Pillar mapping

The pillar system uses a multi-signal approach with explicit priority:

1. Labels (explicit, highest confidence)
2. Repository mapping (configured per-pillar)
3. File path patterns (e.g., `tests/` → reliability)
4. Keyword matching (from PR titles/descriptions)
5. AI inference (fallback for ambiguous items)

### State management

Agent state lives in `.helm/` within the project directory. This is simple file-based JSON, suitable for single-agent operation. The state tracks what the agent is currently working on and logs failures for debugging.
