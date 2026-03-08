# Commands

Complete CLI reference for gh-helm. All commands support `--json` and `--jq` flags for machine-readable output.

## Project Agent

### `gh helm project init`

Create a `helm.toml` config file in the current directory.

```bash
gh helm project init
```

Interactively prompts for project board, owner, and agent settings.

### `gh helm project start`

Work a single issue. The agent claims the issue, generates a plan, writes code, and pushes a draft PR.

```bash
gh helm project start --issue 42
gh helm project start --issue 42 --codespace
```

| Flag | Description |
|------|-------------|
| `--issue` | Issue number to work (required) |
| `--codespace` | Run in a GitHub Codespace instead of locally |

### `gh helm project daemon`

Continuously poll the project board and work issues as they become ready.

```bash
gh helm project daemon
gh helm project daemon --max-per-hour 5 --codespace
```

| Flag | Description |
|------|-------------|
| `--max-per-hour` | Override rate limit from config |
| `--codespace` | Run each issue in its own Codespace |
| `--json-log` | Output structured JSON logs |

### `gh helm project status`

Show what the agent is currently working on.

```bash
gh helm project status
```

### `gh helm project sot`

View or propose updates to the project's source of truth document.

```bash
gh helm project sot                         # View current document
gh helm project sot propose "New decision"  # Propose an update
```

## Manager Agent

### `gh helm manager init`

Create a `helm-manager.toml` config file in the current directory.

```bash
gh helm manager init
```

### `gh helm manager observe`

Generate and post observations for all team members. One-shot: fetches activity, maps to pillars, posts to 1-1 repos.

```bash
gh helm manager observe
gh helm manager observe --since 7d
gh helm manager observe --dry-run
```

| Flag | Description |
|------|-------------|
| `--since` | Look back period (e.g., `7d`, `168h`) |
| `--dry-run` | Generate observations without posting |

### `gh helm manager prep`

Generate 1-1 meeting preparation for a specific team member.

```bash
gh helm manager prep sarah
gh helm manager prep --all
```

| Flag | Description |
|------|-------------|
| `--all` | Generate prep for all team members |
| `--since` | Look back period |

### `gh helm manager pulse`

Generate a team health overview across all projects.

```bash
gh helm manager pulse
gh helm manager pulse --since 14d
```

### `gh helm manager pillars`

Show configured pillar definitions and their signals.

```bash
gh helm manager pillars
```

### `gh helm manager stats`

Show team or individual statistics including PR velocity, cycle time, review turnaround, pillar coverage, and bus factor analysis.

```bash
gh helm manager stats                   # Team-wide stats
gh helm manager stats sarah             # Individual stats
gh helm manager stats --since 90d       # Custom look-back period
```

| Flag | Description |
|------|-------------|
| `--since` | Look back period (e.g. `30d`, `720h`) |

### `gh helm manager report`

Generate a full report card for a team member, including pillar impact, growth tracking, and notable contributions.

```bash
gh helm manager report sarah
gh helm manager report sarah --since 30d
```

### `gh helm manager start`

Start the manager daemon. Runs on the configured schedule (pulse, prep, observe).

```bash
gh helm manager start
gh helm manager start --json-log
```

## Shared Commands

### `gh helm config show`

Display the current `helm.toml` configuration.

```bash
gh helm config show
```

### `gh helm doctor`

Check project health and gh-helm setup.

```bash
gh helm doctor
```

Validates: config file, source of truth, project board access, labels, devcontainer, notifications, auth, and state directory.

### `gh helm upgrade`

Auto-fix issues found by `gh helm doctor`.

```bash
gh helm upgrade
gh helm upgrade --dry-run
```

Creates missing labels, scaffolds devcontainer, initializes state directory, and sets config defaults.

### `gh helm version`

Print the current version.

```bash
gh helm version
```
