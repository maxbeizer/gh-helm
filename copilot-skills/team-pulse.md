# Team Pulse

Monitor team health, activity patterns, and performance pillar contributions.

## Usage
- "How's the team doing?"
- "Run a pulse check"
- "Show team stats"
- "What are the velocity trends?"
- "Generate weekly observations"
- "Show pillar definitions"

## Tools

### Team health pulse
```bash
gh helm manager pulse --json
```

### Team/individual stats
```bash
gh helm manager stats --json
gh helm manager stats --handle {github_handle} --json
```

### Generate observations
```bash
gh helm manager observe --json
gh helm manager observe --dry-run --json
```

### View pillar definitions
```bash
gh helm manager pillars --json
```
