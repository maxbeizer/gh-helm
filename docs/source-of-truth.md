# Source of Truth

Every project managed by gh-helm has a source of truth document — a living markdown file that captures the project's goals, key decisions, and current state.

## Purpose

The source of truth serves two roles:

1. **Context for the agent.** Before the project agent starts working an issue, it reads this document to understand the project's goals, constraints, and architectural decisions. This prevents the agent from making changes that conflict with prior decisions.

2. **Living record for humans.** Key decisions get captured as they happen, not reconstructed months later. The document becomes the canonical "why did we do this?" reference.

## Format

The default path is `docs/SOURCE_OF_TRUTH.md` (configurable via `source-of-truth` in `helm.toml`).

```markdown
# Project Name — Source of Truth

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

## How It Evolves

The source of truth is **not static**. It grows as the project evolves:

1. **Agent reads it** before every issue to understand context.
2. **When a PR merges** that changes the project's direction, the agent flags it.
3. **The agent proposes an update** — a new key decision, a checked-off outcome, an architecture note.
4. **A human reviews and approves.** The agent proposes, never unilaterally rewrites goals.

```bash
# View the current source of truth
gh helm project sot

# Propose an update
gh helm project sot propose "Switched from REST to GraphQL for the API layer (PR #55)"
```

## Sections Explained

### Goals
High-level objectives with measurable targets where possible. These rarely change — when they do, it's a significant project decision.

### Key Decisions
Dated entries linking to the PR or issue where the decision was made. This is the "architectural decision record" of the project, kept lightweight.

### Outcomes
Checkbox-style tracking of deliverables. The agent checks these off as PRs merge.

### Architecture Notes
Free-form technical context that helps the agent (and new developers) understand the current system design.

## Tips

- Keep it concise. This isn't a design doc — it's a quick-reference for "what are we building and what have we decided."
- Link to PRs and issues. The source of truth points to where the real discussion happened.
- Review agent proposals critically. The agent is good at noticing when a PR changes direction, but the human decides whether it's worth recording.
- One per project repo. If you have a monorepo, you might want one per major component.
