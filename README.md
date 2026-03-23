# gh-helm

**Autonomous developer agents backed by GitHub as the single source of truth.**

gh-helm is a `gh` CLI extension where every project gets an agent that does the work, and every manager gets an agent that watches the work. No Jira, no Linear, no Notion — just repos, issues, projects, and PRs.

## Install

```bash
gh ext install maxbeizer/gh-helm
```

## Quick Start

### Project Agent — delegate an issue

```bash
# Set up a project
gh helm project init

# Point the agent at an issue
gh helm project start --issue 42
```

The agent claims the issue, reads the project's source of truth, generates a plan, writes code, and pushes a draft PR. You review, test, and merge.

### Manager Agent — watch the work

```bash
# Set up manager config
gh helm manager init

# Generate 1-1 prep for a report
gh helm manager prep sarah

# Team health pulse
gh helm manager pulse
```

The manager agent monitors team activity, maps contributions to configurable performance pillars, and posts structured observations to 1-1 repos.

## How It Works

- **Project Agents** do the engineering work — claim issues, break them down, write code, push draft PRs, update the project's source of truth.
- **Manager Agents** watch the work — track who's shipping what, flag blockers, map contributions to performance pillars, write observations to 1-1 repos.

GitHub is the only source of truth. Every decision, every status change, every observation lives in issues, PRs, and markdown files in repos.

## Documentation

| Doc | Description |
|-----|-------------|
| [**Tutorial Site**](https://maxbeizer.github.io/learn-gh-helm/) | Interactive guide — built by gh-helm itself ([repo](https://github.com/maxbeizer/learn-gh-helm)) |
| [Project Agent](docs/project-agent.md) | Workflow, running modes, configuration |
| [Manager Agent](docs/manager-agent.md) | Team monitoring, pillars, 1-1 observations |
| [Configuration](docs/configuration.md) | Full `helm.toml` and `helm-manager.toml` reference |
| [Commands](docs/commands.md) | Complete CLI reference |
| [Architecture](docs/architecture.md) | Package structure, design decisions |
| [Source of Truth](docs/source-of-truth.md) | Living project document format and philosophy |

## Philosophy

- **Pillars are configurable.** Your org's performance framework plugs in.
- **Notifications are pluggable.** Slack, Teams, Discord, or just GitHub.
- **AI is pluggable.** GitHub Models, or bring your own.
- **No vendor lock-in.** It's GitHub APIs + Go. That's it.

## License

MIT
