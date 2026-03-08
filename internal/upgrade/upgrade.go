package upgrade

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/maxbeizer/max-ops/internal/config"
	"github.com/maxbeizer/max-ops/internal/github"
	"gopkg.in/yaml.v3"
)

type ChangeStatus string

const (
	StatusApplied ChangeStatus = "applied"
	StatusSkipped ChangeStatus = "skipped"
)

type Change struct {
	Status  ChangeStatus `json:"status"`
	Message string       `json:"message"`
}

type Result struct {
	Changes []Change `json:"changes"`
	Applied int      `json:"applied"`
	Skipped int      `json:"skipped"`
}

type Options struct {
	DryRun bool
}

var requiredLabels = []string{"agent-ready", "agent-in-progress", "agent-done", "needs-attention"}

var labelColors = map[string]string{
	"agent-ready":       "0E8A16",
	"agent-in-progress": "1D76DB",
	"agent-done":        "5319E7",
	"needs-attention":   "B60205",
}

func Run(ctx context.Context, opts Options) (Result, error) {
	changes := []Change{}

	repo, repoErr := github.CurrentRepo(ctx)
	if repoErr == nil {
		labels, err := github.ListLabels(ctx, repo)
		if err == nil {
			missing := missingLabels(labels, requiredLabels)
			for _, label := range missing {
				if opts.DryRun {
					changes = append(changes, Change{Status: StatusSkipped, Message: "Would create label: " + label})
					continue
				}
				color := labelColors[label]
				desc := "max-ops required label"
				if err := github.CreateLabel(ctx, repo, label, color, desc); err != nil {
					return Result{}, err
				}
				changes = append(changes, Change{Status: StatusApplied, Message: "Created label: " + label})
			}
		} else {
			changes = append(changes, Change{Status: StatusSkipped, Message: "Labels: unable to list"})
		}
	} else {
		changes = append(changes, Change{Status: StatusSkipped, Message: "Labels: repo not detected"})
	}

	cfg, cfgErr := config.Load("max-ops.yaml")
	if cfgErr == nil {
		missingSOT := configFieldMissing("max-ops.yaml", "source-of-truth")
		updated, changed := mergeDefaults(cfg, missingSOT)
		if changed {
			if opts.DryRun {
				changes = append(changes, Change{Status: StatusSkipped, Message: "Config: would update max-ops.yaml"})
			} else {
				if err := config.Write("max-ops.yaml", updated); err != nil {
					return Result{}, err
				}
				changes = append(changes, Change{Status: StatusApplied, Message: "Config: updated max-ops.yaml"})
			}
		} else {
			changes = append(changes, Change{Status: StatusSkipped, Message: "Config: max-ops.yaml already up to date"})
		}
	} else {
		changes = append(changes, Change{Status: StatusSkipped, Message: "Config: max-ops.yaml not found"})
	}

	if err := ensureDevcontainer(opts.DryRun, &changes); err != nil {
		return Result{}, err
	}

	if err := ensureSourceOfTruth(opts.DryRun, &changes); err != nil {
		return Result{}, err
	}

	if err := ensureStateDir(opts.DryRun, &changes); err != nil {
		return Result{}, err
	}

	result := Result{Changes: changes}
	for _, change := range changes {
		if change.Status == StatusApplied {
			result.Applied++
		} else {
			result.Skipped++
		}
	}
	return result, nil
}

func mergeDefaults(cfg config.Config, sourceOfTruthMissing bool) (config.Config, bool) {
	changed := false
	if cfg.Agent.Model == "" {
		cfg.Agent.Model = "gpt-4o"
		changed = true
	}
	if cfg.Agent.MaxPerHour == 0 {
		cfg.Agent.MaxPerHour = 3
		changed = true
	}
	if cfg.Notifications.Channel == "" {
		cfg.Notifications.Channel = "slack"
		changed = true
	}
	if cfg.Notifications.OpsChannel == "" {
		cfg.Notifications.OpsChannel = "#project-alpha-ops"
		changed = true
	}
	if cfg.SourceOfTruth == "" || sourceOfTruthMissing {
		cfg.SourceOfTruth = "docs/SOURCE_OF_TRUTH.md"
		changed = true
	}
	if cfg.Filters.Status == "" {
		cfg.Filters.Status = "Ready"
		changed = true
	}
	if len(cfg.Filters.Labels) == 0 {
		cfg.Filters.Labels = []string{"agent-ready"}
		changed = true
	}
	return cfg, changed
}

func ensureDevcontainer(dryRun bool, changes *[]Change) error {
	path := filepath.Join(".devcontainer", "devcontainer.json")
	if _, err := os.Stat(path); err == nil {
		*changes = append(*changes, Change{Status: StatusSkipped, Message: "DevContainer: already exists"})
		return nil
	}
	if dryRun {
		*changes = append(*changes, Change{Status: StatusSkipped, Message: "DevContainer: would create .devcontainer/devcontainer.json"})
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(devcontainerTemplate), 0o644); err != nil {
		return err
	}
	*changes = append(*changes, Change{Status: StatusApplied, Message: "Created: .devcontainer/devcontainer.json"})
	return nil
}

func ensureSourceOfTruth(dryRun bool, changes *[]Change) error {
	cfg, err := config.Load("max-ops.yaml")
	if err != nil {
		*changes = append(*changes, Change{Status: StatusSkipped, Message: "Source of Truth: config missing"})
		return nil
	}
	path := cfg.SourceOfTruth
	if path == "" {
		path = "docs/SOURCE_OF_TRUTH.md"
	}
	if _, err := os.Stat(path); err == nil {
		*changes = append(*changes, Change{Status: StatusSkipped, Message: "Source of Truth: already exists"})
		return nil
	}
	if dryRun {
		*changes = append(*changes, Change{Status: StatusSkipped, Message: "Source of Truth: would create " + path})
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(sourceOfTruthTemplate), 0o644); err != nil {
		return err
	}
	*changes = append(*changes, Change{Status: StatusApplied, Message: "Created: " + path})
	return nil
}

func ensureStateDir(dryRun bool, changes *[]Change) error {
	path := ".max-ops"
	if _, err := os.Stat(path); err == nil {
		*changes = append(*changes, Change{Status: StatusSkipped, Message: "State: already exists"})
		return nil
	}
	if dryRun {
		*changes = append(*changes, Change{Status: StatusSkipped, Message: "State: would create .max-ops/"})
		return nil
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	*changes = append(*changes, Change{Status: StatusApplied, Message: "Created: .max-ops/"})
	return nil
}

func missingLabels(existing []string, required []string) []string {
	lower := map[string]bool{}
	for _, label := range existing {
		lower[strings.ToLower(label)] = true
	}
	missing := []string{}
	for _, label := range required {
		if !lower[strings.ToLower(label)] {
			missing = append(missing, label)
		}
	}
	return missing
}

func configFieldMissing(path string, field string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return false
	}
	if len(node.Content) == 0 {
		return false
	}
	root := node.Content[0]
	if root.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i < len(root.Content); i += 2 {
		if root.Content[i].Value == field {
			return false
		}
	}
	return true
}

const devcontainerTemplate = `{
  "name": "max-ops-daemon",
  "image": "mcr.microsoft.com/devcontainers/go:1.24",
  "postStartCommand": "go build -o /tmp/max-ops . && /tmp/max-ops project daemon --max-per-hour 3",
  "features": {
    "ghcr.io/devcontainers/features/github-cli:1": {}
  },
  "customizations": {
    "codespaces": {
      "machine": "basicLinux32gb"
    }
  }
}
`

const sourceOfTruthTemplate = `# Source of Truth

## Mission

## Current Focus

## Success Metrics

## Risks & Blockers

## Next Up
`
