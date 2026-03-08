# Configuration

gh-helm uses TOML config files. There are two: one for project agents and one for manager agents.

## Project Config: `helm.toml`

Lives in the project repo root. Created by `gh helm project init`.

```toml
version = 1

[project]
board = 25                              # GitHub Projects v2 number
owner = "myorg"                         # Project/org owner

[agent]
hubber = "maxbeizer"                    # Developer running the agent
model = "gpt-4o"                        # AI model (via GitHub Models API)
max-per-hour = 3                        # Max issues claimed per rolling hour

[notifications]
channel = "slack"                       # slack | teams | discord | github
ops-channel = "#project-alpha-ops"      # Where PR-ready notifications post
webhook-url = "https://hooks.slack.com/services/..."

source-of-truth = "docs/SOURCE_OF_TRUTH.md"  # Living project goals document

[filters]
status = "Ready"                        # Project board status to watch
labels = ["agent-ready"]                # Only claim issues with these labels
```

### Field Reference

| Section | Field | Required | Default | Description |
|---------|-------|----------|---------|-------------|
| — | `version` | yes | — | Config schema version (currently `1`) |
| `project` | `board` | yes | — | GitHub Projects v2 board number |
| `project` | `owner` | yes | — | GitHub org or user that owns the project |
| `agent` | `hubber` | yes | — | GitHub handle of the developer running the agent |
| `agent` | `model` | no | `gpt-4o` | AI model via GitHub Models API |
| `agent` | `max-per-hour` | no | `3` | Rate limit: max issues claimed per rolling hour |
| `notifications` | `channel` | no | `slack` | Notification channel: `slack`, `teams`, `discord`, `github` |
| `notifications` | `ops-channel` | no | — | Channel/room name for notifications |
| `notifications` | `webhook-url` | no | — | Webhook URL (required for Slack/Teams/Discord) |
| — | `source-of-truth` | no | `docs/SOURCE_OF_TRUTH.md` | Path to the project's source of truth document |
| `filters` | `status` | no | — | Only claim items with this project board status |
| `filters` | `labels` | no | — | Only claim issues with all of these labels |

## Manager Config: `helm-manager.toml`

Lives wherever you run the manager agent. Created by `gh helm manager init`.

```toml
version = 1

[manager]
hubber = "maxbeizer"

[[projects]]
owner = "myorg"
board = 25
name = "Project Alpha"

[[projects]]
owner = "myorg"
board = 30
name = "Project Beta"

[[team]]
handle = "sarah"
one-one-repo = "maxbeizer/sarah-1-1"
pillars = ["reliability", "velocity"]

[[team]]
handle = "alex"
one-one-repo = "maxbeizer/alex-1-1"
pillars = ["security", "developer-experience"]

[pillars.reliability]
description = "System stability, incident reduction, monitoring coverage"
signals = ["bug fixes", "test coverage", "monitoring PRs", "incident response"]
repos = ["myorg/monitoring", "myorg/alerts"]
labels = ["bug", "reliability", "testing"]

[pillars.velocity]
description = "Feature delivery speed, PR cycle time, unblocking others"
signals = ["PRs merged", "cycle time", "reviews completed", "issues closed"]

[notifications]
channel = "slack"
ops-channel = "#engineering-ops"

[schedule]
pulse = "0 9 * * 1"          # Team pulse every Monday 9am (cron, UTC)
prep = "0 8 * * *"           # 1-1 prep daily
observe = "0 2 * * 5"        # Weekly observations Friday night
```

### Field Reference

| Section | Field | Required | Default | Description |
|---------|-------|----------|---------|-------------|
| — | `version` | yes | — | Config schema version (currently `1`) |
| `manager` | `hubber` | yes | — | GitHub handle of the manager |
| `projects` | `owner` | yes | — | Org/user owning the project |
| `projects` | `board` | yes | — | Projects v2 board number |
| `projects` | `name` | yes | — | Human-readable project name |
| `team` | `handle` | yes | — | Team member's GitHub handle |
| `team` | `one-one-repo` | no | — | Repo where observations are posted (e.g., `user/1-1-repo`) |
| `team` | `pillars` | no | — | Performance pillars assigned to this person |
| `pillars.<name>` | `description` | yes | — | What this pillar measures |
| `pillars.<name>` | `signals` | yes | — | Human-readable signal descriptions |
| `pillars.<name>` | `repos` | no | — | Repos that strongly signal this pillar |
| `pillars.<name>` | `labels` | no | — | Labels that map to this pillar |
| `pillars.<name>` | `paths` | no | — | File path patterns that signal this pillar |
| `schedule` | `pulse` | no | — | Cron expression for team pulse |
| `schedule` | `prep` | no | — | Cron expression for 1-1 prep |
| `schedule` | `observe` | no | — | Cron expression for observations |

## State Directory: `.helm/`

gh-helm stores runtime state in `.helm/` within the project root:

- `.helm/state.json` — current agent state (what it's working on)
- `.helm/failures.json` — log of failed items

This directory is created automatically on first run, or via `gh helm upgrade`. Add `.helm/` to your `.gitignore`.
