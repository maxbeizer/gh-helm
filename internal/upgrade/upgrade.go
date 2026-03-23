package upgrade

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/maxbeizer/gh-helm/internal/config"
	"github.com/maxbeizer/gh-helm/internal/github"
	"github.com/BurntSushi/toml"
	"log/slog"
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
				desc := "gh-helm required label"
				if err := github.CreateLabel(ctx, repo, label, color, desc); err != nil {
					return Result{}, fmt.Errorf("create label %q: %w", label, err)
				}
				changes = append(changes, Change{Status: StatusApplied, Message: "Created label: " + label})
			}
		} else {
			changes = append(changes, Change{Status: StatusSkipped, Message: "Labels: unable to list"})
		}
	} else {
		changes = append(changes, Change{Status: StatusSkipped, Message: "Labels: repo not detected"})
	}

	cfg, cfgErr := config.Load("helm.toml")
	var cfgPtr *config.Config
	if cfgErr == nil {
		cfgPtr = &cfg
		missingSOT := configFieldMissing("helm.toml", "source-of-truth")
		updated, changed := mergeDefaults(cfg, missingSOT)
		if changed {
			if opts.DryRun {
				changes = append(changes, Change{Status: StatusSkipped, Message: "Config: would update helm.toml"})
			} else {
				if err := config.Write("helm.toml", updated); err != nil {
					return Result{}, fmt.Errorf("write config: %w", err)
				}
				changes = append(changes, Change{Status: StatusApplied, Message: "Config: updated helm.toml"})
			}
		} else {
			changes = append(changes, Change{Status: StatusSkipped, Message: "Config: helm.toml already up to date"})
		}
	} else {
		// Scaffold a new helm.toml with sensible defaults.
		scaffolded := config.Config{Version: config.CurrentConfigVersion}
		scaffolded, _ = mergeDefaults(scaffolded, true)
		if opts.DryRun {
			changes = append(changes, Change{Status: StatusSkipped, Message: "Config: would create helm.toml"})
		} else {
			if err := config.Write("helm.toml", scaffolded); err != nil {
				return Result{}, fmt.Errorf("scaffold config: %w", err)
			}
			changes = append(changes, Change{Status: StatusApplied, Message: "Config: created helm.toml with defaults"})
		}
		cfgPtr = &scaffolded
	}

	if err := ensureDevcontainer(opts.DryRun, &changes); err != nil {
		return Result{}, fmt.Errorf("ensure devcontainer: %w", err)
	}

	if err := ensureSourceOfTruth(cfgPtr, opts.DryRun, &changes); err != nil {
		return Result{}, fmt.Errorf("ensure source of truth: %w", err)
	}

	if err := ensureBoardStatuses(ctx, cfgPtr, opts.DryRun, &changes); err != nil {
		return Result{}, fmt.Errorf("ensure board statuses: %w", err)
	}

	if err := ensureStateDir(opts.DryRun, &changes); err != nil {
		return Result{}, fmt.Errorf("ensure state directory: %w", err)
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
		cfg.Agent.Model = "gpt-4.1"
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
		return fmt.Errorf("create devcontainer directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(devcontainerTemplate), 0o644); err != nil {
		return fmt.Errorf("write devcontainer config: %w", err)
	}
	*changes = append(*changes, Change{Status: StatusApplied, Message: "Created: .devcontainer/devcontainer.json"})
	return nil
}

func ensureSourceOfTruth(cfg *config.Config, dryRun bool, changes *[]Change) error {
	if cfg == nil {
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
		return fmt.Errorf("create source of truth directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(sourceOfTruthTemplate), 0o644); err != nil {
		return fmt.Errorf("write source of truth: %w", err)
	}
	*changes = append(*changes, Change{Status: StatusApplied, Message: "Created: " + path})
	return nil
}

func ensureStateDir(dryRun bool, changes *[]Change) error {
	path := ".helm"
	if _, err := os.Stat(path); err == nil {
		*changes = append(*changes, Change{Status: StatusSkipped, Message: "State: already exists"})
		return nil
	}
	if dryRun {
		*changes = append(*changes, Change{Status: StatusSkipped, Message: "State: would create .helm/"})
		return nil
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	*changes = append(*changes, Change{Status: StatusApplied, Message: "Created: .helm/"})
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
var raw map[string]interface{}
if err := toml.Unmarshal(data, &raw); err != nil {
return false
}
_, ok := raw[field]
return !ok
}

const devcontainerTemplate = `{
  "name": "gh-helm-daemon",
  "image": "mcr.microsoft.com/devcontainers/go:1.24",
  "postStartCommand": "go build -o /tmp/gh-helm . && /tmp/gh-helm project daemon --max-per-hour 3",
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

var requiredStatuses = []string{"Ready", "In Progress", "In Review", "Done"}

func ensureBoardStatuses(ctx context.Context, cfg *config.Config, dryRun bool, changes *[]Change) error {
	if cfg == nil || cfg.Project.Board == 0 || cfg.Project.Owner == "" {
		*changes = append(*changes, Change{Status: StatusSkipped, Message: "Board statuses: no project board configured"})
		return nil
	}

	existing, projectID, fieldID, err := github.FetchBoardStatuses(ctx, cfg.Project.Owner, cfg.Project.Board)
	if err != nil {
		slog.Warn("could not fetch board statuses", "error", err)
		*changes = append(*changes, Change{Status: StatusSkipped, Message: "Board statuses: could not fetch board"})
		return nil
	}

	existingSet := map[string]bool{}
	for _, s := range existing {
		existingSet[s] = true
	}

	var missing []string
	for _, s := range requiredStatuses {
		if !existingSet[s] {
			missing = append(missing, s)
		}
	}

	if len(missing) == 0 {
		*changes = append(*changes, Change{Status: StatusSkipped, Message: "Board statuses: all required statuses present"})
		return nil
	}

	if dryRun {
		*changes = append(*changes, Change{Status: StatusSkipped, Message: "Board statuses: would add " + strings.Join(missing, ", ")})
		return nil
	}

	if err := github.AddBoardStatuses(ctx, projectID, fieldID, existing, missing); err != nil {
		slog.Warn("could not add board statuses", "error", err)
		*changes = append(*changes, Change{Status: StatusSkipped, Message: "Board statuses: failed to add — " + err.Error()})
		return nil
	}

	*changes = append(*changes, Change{Status: StatusApplied, Message: "Board statuses: added " + strings.Join(missing, ", ")})
	return nil
}
