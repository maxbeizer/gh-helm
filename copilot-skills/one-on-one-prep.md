# One-on-One Prep

Generate 1-1 meeting prep and report cards for team members.

## Usage
- "Prep for my 1-1 with sarah"
- "Generate a report card for alex"
- "What has sarah been working on?"
- "Show contributions for the team"

## Tools

### 1-1 meeting prep
```bash
gh helm manager prep {github_handle} --json
```

### Full report card
```bash
gh helm manager report {github_handle} --json
```

### Individual stats
```bash
gh helm manager stats --handle {github_handle} --json
```
