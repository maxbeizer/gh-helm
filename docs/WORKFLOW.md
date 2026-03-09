# Workflow Guide: helm project start vs planning claim

## Overview

`gh-helm` provides two distinct approaches to issue work. Understanding when to use each — and how they complement each other — prevents confusion and keeps your workflow efficient.

## `helm project start` — Autonomous Agent

**What it does:** Claims an issue, generates an AI code plan, writes code, and opens a draft PR — all autonomously.

**When to use it:**
- You want the agent to do the coding work end-to-end
- The issue is well-scoped with clear acceptance criteria
- You're comfortable reviewing a draft PR rather than writing code yourself

**Example:**
```bash
gh helm project start --issue 42
gh helm project start --issue 42 --codespace  # also spins up a Codespace
gh helm project start --issue 42 --dry-run    # preview the plan without executing
```

**What happens:**
1. Fetches the issue details
2. Generates a code plan via AI
3. Creates a branch and writes code
4. Opens a draft PR linking back to the issue
5. (Optionally) creates a Codespace on the PR branch

## `planning claim` — Manual Tracking

**What it does:** Marks an issue as claimed for a session so you (or an agent) can track progress manually. The human or a different agent does the actual work.

**When to use it:**
- You're doing the work yourself and want session tracking
- The work requires human judgment that the autonomous agent can't handle
- You're using a different tool (e.g., Copilot in the editor) for code generation
- You want to claim an issue to prevent others from picking it up

## How They Work Together

The two approaches are complementary, not competing:

| Scenario | Use |
|----------|-----|
| Straightforward bug fix with clear repro | `helm project start` |
| Complex architectural change | `planning claim` + manual work |
| Documentation update | Either — `helm project start` handles docs well |
| Issue triage and claiming | `planning claim` to reserve, then decide approach |
| Fire-and-forget overnight work | `helm project start --codespace` |

### Combined workflow example

1. Use `planning claim` to reserve an issue during triage
2. Review the issue complexity
3. If automatable: `helm project start --issue <N>` to hand off to the agent
4. If not: do the work yourself, using `helm project sot` to track decisions

## SOT and Config: Always Available

Regardless of which approach you use for coding, these commands are always useful:

- `gh helm project sot` — read the Source of Truth document
- `gh helm project sot propose --decision "..."` — propose a SOT update
- `gh helm project sot sync` — reconcile SOT with current issue state
- `gh helm config` — view/manage project configuration

## Summary

- **`helm project start`** = "agent, do this work for me"
- **`planning claim`** = "I'm working on this, track my progress"
- Use SOT commands with either workflow to keep documentation current
