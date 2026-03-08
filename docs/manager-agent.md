# Manager Agent

The manager agent monitors team activity, maps contributions to performance pillars, and posts structured observations to 1-1 repos — so prep happens passively instead of in a Friday panic.

## Configuration

The manager agent runs separately, configured across all your projects and reports:

```toml
# helm-manager.toml

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

[[team]]
handle = "jordan"
one-one-repo = "maxbeizer/jordan-1-1"
pillars = ["velocity", "mentorship"]

[pillars.reliability]
description = "System stability, incident reduction, monitoring coverage"
signals = ["bug fixes", "test coverage", "monitoring PRs", "incident response"]

[pillars.velocity]
description = "Feature delivery speed, PR cycle time, unblocking others"
signals = ["PRs merged", "cycle time", "reviews completed", "issues closed"]

[pillars.security]
description = "Vulnerability remediation, secure coding practices, audit compliance"
signals = ["security PRs", "dependency updates", "audit findings addressed"]

[pillars.developer-experience]
description = "Tooling, documentation, onboarding, developer productivity"
signals = ["docs PRs", "tooling improvements", "onboarding guides"]

[pillars.mentorship]
description = "Code review quality, knowledge sharing, pair programming"
signals = ["review depth", "review turnaround", "docs authored"]

[notifications]
channel = "slack"
ops-channel = "#engineering-ops"

[schedule]
pulse = "0 9 * * 1"          # Team pulse every Monday 9am
prep = "0 8 * * *"           # 1-1 prep morning of scheduled meetings
observe = "0 2 * * 5"        # Weekly observations posted Friday night
```

Run `gh helm manager init` to scaffold this file interactively.

## What It Does

### Continuous monitoring

- Watches assigned issues across all configured projects
- Tracks who's working on what, what's stalled, what's shipping

### 1-1 repo observations

The agent posts structured observations to each report's 1-1 repo as issues:

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

### Pillar mapping

The manager defines performance pillars (or uses org defaults). The agent infers which pillar a PR/issue maps to using:

1. **Labels** (highest signal)
2. **Repository** (e.g., monitoring repo → reliability)
3. **File paths** (e.g., `tests/` → reliability, `docs/` → developer-experience)
4. **AI inference** from PR title/description (fallback)

Managers bring their own values — the pillars are fully configurable. A security-focused team has different pillars than a product team. See [Configuration](configuration.md) for pillar config details.

## Commands

```bash
gh helm manager init                    # Create helm-manager.toml
gh helm manager observe                 # One-shot observations
gh helm manager prep <handle>           # 1-1 prep for a report
gh helm manager prep --all              # Prep for all reports
gh helm manager pulse                   # Team health overview
gh helm manager pillars                 # Show pillar definitions
gh helm manager report <handle>         # Report card for a team member
gh helm manager start                   # Scheduled daemon (observe/pulse/prep)
```

All commands support `--json` and `--jq` for machine-readable output.
