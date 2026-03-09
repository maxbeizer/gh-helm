# Pick Up Work

Claim issues and delegate coding work to the gh-helm project agent.

## Usage
- "Pick up the next issue"
- "Start working on issue 2"
- "Claim issue #42 from copilot-atc"
- "What should I work on next?"
- "Run the agent in daemon mode"

## Tools

### Start an issue
```bash
gh helm project start --issue {number} --repo {owner/repo} --json
```

### Suggest work
```bash
gh helm project suggest --repo {owner/repo} --json
```

### Dry run (preview plan without executing)
```bash
gh helm project start --issue {number} --dry-run --json
```

### Continuous daemon mode
```bash
gh helm project daemon --status Ready --max-per-hour 3 --json
```
