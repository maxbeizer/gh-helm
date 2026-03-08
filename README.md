# max-ops

**Autonomous developer agents backed by GitHub as the single source of truth.**

max-ops is a platform where every project gets an agent that does the work, and every manager gets an agent that watches the work. No Jira, no Linear, no Notion — just repos, issues, projects, and PRs.

## The Idea

Software teams generate enormous amounts of signal — PRs merged, issues opened, reviews completed, branches stalled. Today that signal lives in dashboards nobody checks, standups nobody remembers, and 1-1 docs written in a Friday panic.

max-ops puts agents on both sides:

- **Project Agents** do the engineering work. A developer points one at an issue, it breaks it down, writes code, pushes draft PRs. The developer reviews, tests, and merges.
- **Manager Agents** watch the work. They monitor who's shipping what, flag blockers, map contributions to performance pillars, and write observations to 1-1 repos so prep happens passively.

GitHub is the only source of truth. Every decision, every status change, every observation lives in issues, PRs, and markdown files in repos.

## How It Works

```
┌─────────────────────────────────────────────────────────────┐
│                        max-ops                               │
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

## Project Agent

Each project repo gets its own agent, configured with a simple `max-ops.yaml`:

```yaml
# max-ops.yaml (lives in the project repo root)
project:
  board: 25                          # GitHub Projects v2 number
  owner: myorg                       # Project owner

agent:
  hubber: maxbeizer                  # Developer running the agent
  model: gpt-4o                      # AI model (via GitHub Models)
  max-per-hour: 3                    # Guardrail

notifications:
  channel: slack                     # slack | teams | discord | github
  ops-channel: "#project-alpha-ops"  # Where PR-ready notifications go

source-of-truth: docs/SOURCE_OF_TRUTH.md  # Living project goals document
```

### Workflow

```
Developer                          Project Agent
    │                                   │
    ├─ Assigns issue to agent ─────────►│
    │                                   ├─ Claims issue (In Progress)
    │                                   ├─ Reads issue + source of truth
    │                                   ├─ Breaks into sub-tasks if needed
    │                                   ├─ Writes code
    │                                   ├─ Runs tests
    │                                   ├─ Pushes draft PR
    │                                   ├─ Links PR to issue
    │                                   │
    │◄─ Draft PR ready ────────────────┤
    │                                   │
    ├─ Reviews code                     │
    ├─ Tests locally                    │
    ├─ Marks PR ready for review ──────►│
    │                                   ├─ Posts to ops channel:
    │                                   │  "PR #42 ready for review
    │                                   │   by @maxbeizer — adds auth flow"
    │                                   │
    ├─ Gets reviews from team           │
    ├─ Merges ─────────────────────────►│
    │                                   ├─ Checks: does this change
    │                                   │  project direction?
    │                                   ├─ If yes: updates SOURCE_OF_TRUTH.md
    │                                   │  with new decision + rationale
    │                                   ├─ Moves item to Done
    │                                   └─ Ready for next issue
    │
    ├─ Points agent at next issue ─────►
```

### Running Modes

**Local (interactive pairing):**
```bash
max-ops project start --issue 42
```
Agent works in your local repo. You watch, review, iterate in real-time.

**Codespace (fire and forget):**
```bash
max-ops project start --issue 42 --codespace
```
Agent spins up a Codespace (or uses a pre-configured one), works overnight. You wake up to draft PRs.

**Daemon (continuous):**
```bash
max-ops project daemon
```
Agent polls the project board continuously. Issues land in "Ready" → agent picks them up. Runs in a Codespace indefinitely.

### Source of Truth Document

Every project has a `SOURCE_OF_TRUTH.md` (path configurable):

```markdown
# Project Alpha — Source of Truth

## Goals
- Ship user authentication by Q2
- Reduce API latency to <100ms p99
- Achieve 95% test coverage

## Key Decisions
- **2026-03-08**: Using JWT for stateless auth (PR #42)
- **2026-03-05**: PostgreSQL over SQLite for production (#38)
- **2026-03-01**: React over Vue for frontend (#30)

## Outcomes
- [x] OAuth provider integration (PR #42, merged 2026-03-08)
- [ ] Session management + refresh tokens
- [ ] Login UI components

## Architecture Notes
Auth flow uses OAuth2 with PKCE. Tokens stored in httpOnly cookies.
Refresh rotation on every access token renewal.
```

The agent reads this before starting work (context), and flags updates when merged PRs change the project's direction. A human reviews and approves the update — the agent proposes, never unilaterally rewrites goals.

## Manager Agent

The manager agent runs separately, configured across all your projects and reports:

```yaml
# manager-ops.yaml
manager:
  hubber: maxbeizer
  
projects:
  - owner: myorg
    board: 25
    name: "Project Alpha"
  - owner: myorg  
    board: 30
    name: "Project Beta"

team:
  - handle: sarah
    1-1-repo: maxbeizer/sarah-1-1
    pillars: [reliability, velocity]     # Areas of focus
  - handle: alex
    1-1-repo: maxbeizer/alex-1-1
    pillars: [security, developer-experience]
  - handle: jordan
    1-1-repo: maxbeizer/jordan-1-1
    pillars: [velocity, mentorship]

pillars:
  reliability:
    description: "System stability, incident reduction, monitoring coverage"
    signals: ["bug fixes", "test coverage", "monitoring PRs", "incident response"]
  velocity:
    description: "Feature delivery speed, PR cycle time, unblocking others"
    signals: ["PRs merged", "cycle time", "reviews completed", "issues closed"]
  security:
    description: "Vulnerability remediation, secure coding practices, audit compliance"
    signals: ["security PRs", "dependency updates", "audit findings addressed"]
  developer-experience:
    description: "Tooling, documentation, onboarding, developer productivity"
    signals: ["docs PRs", "tooling improvements", "onboarding guides"]
  mentorship:
    description: "Code review quality, knowledge sharing, pair programming"
    signals: ["review depth", "review turnaround", "docs authored"]

notifications:
  channel: slack
  ops-channel: "#engineering-ops"

schedule:
  pulse: "0 9 * * 1"        # Team pulse every Monday 9am
  prep: "0 8 * * *"         # 1-1 prep morning of scheduled meetings
```

### What the Manager Agent Does

**Continuous monitoring:**
- Watches assigned issues across all configured projects
- Tracks who's working on what, what's stalled, what's shipping

**1-1 repo observations:**
The agent posts structured observations to each report's 1-1 repo as issues or comments:

```markdown
## 📊 Week of Mar 3–7, 2026

### Activity
- 4 PRs merged (2 in Project Alpha, 2 in Project Beta)
- 2 reviews completed (avg turnaround: 3h)
- 1 issue opened, 3 closed

### Pillar Impact: Reliability
- PR #42: Fixed auth token refresh race condition — direct reliability improvement
- PR #45: Added integration tests for payment flow — test coverage +8%

### Pillar Impact: Velocity  
- PR cycle time: 1.2 days (team avg: 2.5 days) ⭐
- Unblocked @jordan on #60 with review

### Observations
- Strong week on reliability — auth fix was high-impact
- Might be worth discussing: no security-related work in 3 weeks (assigned pillar)

### Suggested Topics for 1-1
- Acknowledge auth fix impact
- Discuss security pillar balance
- Career: ready for more cross-team visibility?
```

**The manager decides what to keep.** The agent surfaces observations. The manager curates what matters before the 1-1. Some observations become talking points, others get dismissed. Over time the agent learns what the manager values.

**Pillar mapping:**
The manager defines performance pillars (or uses org defaults). The agent infers which pillar a PR/issue maps to using:
1. Labels (highest signal)
2. Repository (e.g., monitoring repo → reliability)
3. File paths (e.g., `tests/` → reliability, `docs/` → developer-experience)
4. AI inference from PR title/description (fallback)

Managers bring their own values — the pillars are fully configurable. A security-focused team has different pillars than a product team.

## Architecture

```
max-ops/
├── cmd/
│   ├── project.go          # Project agent commands
│   ├── manager.go          # Manager agent commands
│   ├── daemon.go           # Daemon mode
│   └── root.go
├── internal/
│   ├── agent/
│   │   ├── project.go      # Project agent logic
│   │   ├── manager.go      # Manager agent logic
│   │   └── loop.go         # Poll/claim/work loop
│   ├── github/
│   │   ├── projects.go     # Projects v2 API
│   │   ├── issues.go       # Issues API
│   │   ├── models.go       # GitHub Models API (AI)
│   │   └── search.go       # Search API
│   ├── config/
│   │   ├── project.go      # max-ops.yaml parsing
│   │   └── manager.go      # manager-ops.yaml parsing
│   ├── notifications/
│   │   ├── slack.go         # Slack integration
│   │   ├── teams.go         # Teams integration
│   │   ├── discord.go       # Discord integration
│   │   └── notifier.go      # Interface
│   ├── pillars/
│   │   ├── mapper.go        # Map work → pillars
│   │   └── report.go        # Generate pillar reports
│   ├── sot/
│   │   └── source.go        # Source of truth document management
│   └── output/
│       └── output.go        # JSON/table/jq output
├── max-ops.yaml.example
├── manager-ops.yaml.example
├── go.mod
├── Makefile
└── README.md
```

## Commands

### Project Agent
```bash
max-ops project init                    # Create max-ops.yaml in current repo
max-ops project start --issue 42        # Work one issue (local)
max-ops project start --issue 42 --codespace  # Work in Codespace
max-ops project daemon                  # Continuous: poll board, work items
max-ops project status                  # Current agent status
max-ops project sot                     # Show source of truth document
max-ops project sot propose "New decision"  # Propose source of truth update
```

### Manager Agent
```bash
max-ops manager init                    # Create manager-ops.yaml
max-ops manager start                   # Start monitoring
max-ops manager pulse                   # One-shot team pulse
max-ops manager prep <handle>           # Generate 1-1 prep for one person
max-ops manager prep --all              # Prep for all reports
max-ops manager observe                 # One-shot: generate observations, post to 1-1 repos
max-ops manager pillars                 # Show pillar definitions
max-ops manager report <handle>         # Full report card for a team member
```

### Shared
```bash
max-ops config show                     # Show current config
max-ops version                         # Version info
```

## Integration with gh-planning

`gh-planning` is the individual developer's CLI tool. `max-ops` is the orchestration platform. They complement each other:

| gh-planning | max-ops |
|-------------|---------|
| `status` — what's on my plate | `project status` — what's the agent doing |
| `track` — create + track issue | `project start` — agent works the issue |
| `breakdown` — split an issue | Built into project agent workflow |
| `handoff` — session transition | Built into agent claim/complete cycle |
| `prep` — 1-1 preparation | `manager prep` — automated, pillar-aware |
| `team` — team dashboard | `manager pulse` — deeper, with pillar mapping |

A developer might use `gh planning` for manual work and `max-ops project` for delegating to the agent. A manager uses `max-ops manager` for oversight.

## Open Source Philosophy

This is open source. The core platform is general-purpose:
- **Pillars are configurable.** Your org's performance framework plugs in.
- **Notifications are pluggable.** Slack, Teams, Discord, or just GitHub.
- **AI is pluggable.** GitHub Models, or bring your own.
- **No vendor lock-in.** It's GitHub APIs + Go. That's it.

The goal is to make every engineering team more effective by putting agents on both sides of the work — the doing and the observing — with GitHub as the shared truth.

## Implementation Phases

### Phase 1: Project Agent MVP
- [ ] `max-ops.yaml` config format
- [ ] `project init` — scaffold config
- [ ] `project start --issue N` — claim, break down, code, draft PR (local mode)
- [ ] Source of truth document reading/proposing
- [ ] Slack notification on PR ready

### Phase 2: Manager Agent MVP
- [ ] `manager-ops.yaml` config format
- [ ] `manager observe` — one-shot observation generation
- [ ] `manager prep` — 1-1 prep with pillar mapping
- [ ] 1-1 repo posting
- [ ] Pillar definition + inference

### Phase 3: Daemon + Codespace
- [ ] `project daemon` — continuous poll/claim/work loop
- [ ] `.devcontainer` setup for one-click Codespace
- [ ] Guardrails (max-per-hour, label filters)
- [ ] Failure handling (comment + skip)

### Phase 4: Polish
- [ ] `manager report` — full report card
- [ ] `manager pulse` — scheduled team health
- [ ] Notification channel plugins (Teams, Discord)
- [ ] Learning: agent improves pillar mapping over time
- [ ] `--json` / `--jq` everywhere

### Phase 5: Community
- [ ] Documentation site
- [ ] Example configs for common setups
- [ ] Pillar templates (SRE team, product team, platform team)
- [ ] Plugin system for custom notification channels

## License

MIT
