# Agent Status

Check what the gh-helm agent is working on and validate project setup.

## Usage
- "What is the agent working on?"
- "Show agent status"
- "Is the project set up correctly?"
- "Show me the config"
- "Validate my helm setup"

## Tools

### Current agent status
```bash
gh helm project status --json
```

### Show configuration
```bash
gh helm config show --json
```

### Validate setup
```bash
gh helm doctor --json
```

### Auto-fix issues
```bash
gh helm upgrade --json
```

### View source of truth
```bash
gh helm project sot --json
```
