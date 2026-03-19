# Project Agent

The project agent is an autonomous developer that claims issues, writes code, and pushes draft PRs — all backed by GitHub as the single source of truth.

## Configuration

Each project repo gets its own agent, configured with a `helm.toml` in the repo root:

```toml
version = 1

[project]
board = 25                              # GitHub Projects v2 number
owner = "myorg"                         # Project/org owner

[agent]
user = "octocat"                    # Developer running the agent
model = "gpt-4o"                        # AI model (via GitHub Models)
max-per-hour = 3                        # Guardrail

[notifications]
channel = "slack"                       # slack | teams | discord | github
ops-channel = "#project-alpha-ops"      # Where PR-ready notifications go
webhook-url = "https://hooks.slack.com/services/..."

source-of-truth = "docs/SOURCE_OF_TRUTH.md"  # Living project goals document

[filters]
status = "Ready"                        # Project board status to watch
labels = ["agent-ready"]                # Only claim issues with these labels
```

Run `gh helm project init` to scaffold this file interactively.

## Workflow

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

## Running Modes

### Local (interactive pairing)

```bash
gh helm project start --issue 42
```

Agent works in your local repo. You watch, review, iterate in real-time.

### Codespace (fire and forget)

```bash
gh helm project start --issue 42 --codespace
```

Agent spins up a Codespace (or uses a pre-configured one), works overnight. You wake up to draft PRs.

### Daemon (continuous)

```bash
gh helm project daemon --status Ready --max-per-hour 3 --codespace
```

Agent polls the project board continuously. Issues land in "Ready" → agent picks them up. Optional filters/guardrails let you scope by status/label, and `--codespace` spins up a workspace per PR branch.

## Source of Truth Document

Every project has a `SOURCE_OF_TRUTH.md` (path configurable). The agent reads this before starting work for context, and proposes updates when merged PRs change the project's direction. A human reviews and approves — the agent proposes, never unilaterally rewrites goals.

See [Source of Truth](source-of-truth.md) for the full format and philosophy.

## Commands

```bash
gh helm project init                    # Create helm.toml in current repo
gh helm project start --issue 42        # Work one issue (local)
gh helm project start --issue 42 --codespace  # Work in Codespace
gh helm project daemon                  # Continuous: poll board, work items
gh helm project status                  # Current agent status
gh helm project sot                     # Show source of truth document
gh helm project sot propose "New decision"  # Propose source of truth update
```

All commands support `--json` and `--jq` for machine-readable output.
