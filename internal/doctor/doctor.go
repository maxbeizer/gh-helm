package doctor

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/maxbeizer/max-ops/internal/config"
	"github.com/maxbeizer/max-ops/internal/github"
	"github.com/maxbeizer/max-ops/internal/upgrade"
)

type Status string

const (
	StatusPass Status = "pass"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
	StatusInfo Status = "info"
)

type CheckResult struct {
	Key     string `json:"key"`
	Status  Status `json:"status"`
	Message string `json:"message"`
}

type Summary struct {
	Passed   int `json:"passed"`
	Warnings int `json:"warnings"`
	Failures int `json:"failures"`
	Info     int `json:"info"`
}

type Result struct {
	Checks  []CheckResult   `json:"checks"`
	Summary Summary         `json:"summary"`
	Upgrade *upgrade.Result `json:"upgrade,omitempty"`
}

type Options struct {
	Fix bool
}

var requiredLabels = []string{"agent-ready", "agent-in-progress", "agent-done", "needs-attention"}

func Run(ctx context.Context, opts Options) (Result, error) {
	checks := []CheckResult{}
	cfg, cfgErr := config.Load("max-ops.yaml")
	if cfgErr != nil {
		checks = append(checks, CheckResult{Key: "config", Status: StatusFail, Message: "max-ops.yaml missing or invalid"})
	} else {
		checks = append(checks, CheckResult{Key: "config", Status: StatusPass, Message: "max-ops.yaml found and valid"})
	}

	if cfgErr != nil {
		checks = append(checks, CheckResult{Key: "source_of_truth", Status: StatusWarn, Message: "skipped (config missing)"})
		checks = append(checks, CheckResult{Key: "project_board", Status: StatusWarn, Message: "skipped (config missing)"})
		checks = append(checks, CheckResult{Key: "labels", Status: StatusWarn, Message: "skipped (config missing)"})
		checks = append(checks, CheckResult{Key: "notifications", Status: StatusWarn, Message: "skipped (config missing)"})
	} else {
		sotPath := cfg.SourceOfTruth
		if sotPath == "" {
			sotPath = "docs/SOURCE_OF_TRUTH.md"
		}
		if _, err := os.Stat(sotPath); err == nil {
			checks = append(checks, CheckResult{Key: "source_of_truth", Status: StatusPass, Message: sotPath + " exists"})
		} else {
			checks = append(checks, CheckResult{Key: "source_of_truth", Status: StatusFail, Message: sotPath + " missing"})
		}

		if cfg.Project.Owner != "" && cfg.Project.Board != 0 {
			if info, err := github.FetchProjectInfo(ctx, cfg.Project.Owner, cfg.Project.Board); err == nil {
				checks = append(checks, CheckResult{Key: "project_board", Status: StatusPass, Message: formatProjectBoardMessage(cfg.Project.Board, info.ItemCount)})
			} else {
				checks = append(checks, CheckResult{Key: "project_board", Status: StatusFail, Message: "project board not accessible"})
			}
		} else {
			checks = append(checks, CheckResult{Key: "project_board", Status: StatusWarn, Message: "project board not configured"})
		}

		repo, repoErr := github.CurrentRepo(ctx)
		if repoErr != nil {
			checks = append(checks, CheckResult{Key: "labels", Status: StatusWarn, Message: "unable to resolve repo"})
		} else {
			labels, err := github.ListLabels(ctx, repo)
			if err != nil {
				checks = append(checks, CheckResult{Key: "labels", Status: StatusWarn, Message: "unable to list labels"})
			} else {
				missing := missingLabels(labels, requiredLabels)
				if len(missing) == 0 {
					checks = append(checks, CheckResult{Key: "labels", Status: StatusPass, Message: "all required labels present"})
				} else {
					checks = append(checks, CheckResult{Key: "labels", Status: StatusWarn, Message: "missing " + strings.Join(missing, ", ")})
				}
			}
		}

		if cfg.Notifications.WebhookURL != "" {
			checks = append(checks, CheckResult{Key: "notifications", Status: StatusPass, Message: "webhook configured"})
		} else {
			checks = append(checks, CheckResult{Key: "notifications", Status: StatusWarn, Message: "webhook-url not configured"})
		}
	}

	if hasDevContainer() {
		checks = append(checks, CheckResult{Key: "devcontainer", Status: StatusPass, Message: ".devcontainer/devcontainer.json configured"})
	} else {
		checks = append(checks, CheckResult{Key: "devcontainer", Status: StatusInfo, Message: "devcontainer not configured"})
	}

	authStatus := checkAuth(ctx)
	checks = append(checks, authStatus)

	stateStatus := checkStateDir()
	checks = append(checks, stateStatus)

	result := Result{Checks: checks, Summary: summarize(checks)}

	if opts.Fix {
		upgradeResult, err := upgrade.Run(ctx, upgrade.Options{DryRun: false})
		if err != nil {
			return result, err
		}
		result.Upgrade = &upgradeResult
		refreshed, err := Run(ctx, Options{})
		if err != nil {
			return result, err
		}
		result.Checks = refreshed.Checks
		result.Summary = refreshed.Summary
	}

	return result, nil
}

func summarize(checks []CheckResult) Summary {
	summary := Summary{}
	for _, check := range checks {
		switch check.Status {
		case StatusPass:
			summary.Passed++
		case StatusWarn:
			summary.Warnings++
		case StatusFail:
			summary.Failures++
		case StatusInfo:
			summary.Info++
		}
	}
	return summary
}

func formatProjectBoardMessage(number int, count int) string {
	return "#" + itoa(number) + " accessible (" + itoa(count) + " items)"
}

func missingLabels(existing []string, required []string) []string {
	lower := map[string]bool{}
	for _, label := range existing {
		lower[strings.ToLower(label)] = true
	}
	missing := []string{}
	for _, label := range required {
		if !lower[strings.ToLower(label)] {
			missing = append(missing, "'"+label+"'")
		}
	}
	return missing
}

func hasDevContainer() bool {
	path := filepath.Join(".devcontainer", "devcontainer.json")
	_, err := os.Stat(path)
	return err == nil
}

func checkAuth(ctx context.Context) CheckResult {
	time.Sleep(3 * time.Second)
	out, err := github.RunWith(ctx, "auth", "status", "-h", "github.com")
	if err != nil {
		return CheckResult{Key: "auth", Status: StatusFail, Message: "gh auth not configured"}
	}
	text := string(out)
	line := "Token scopes:"
	idx := strings.Index(text, line)
	if idx == -1 {
		return CheckResult{Key: "auth", Status: StatusWarn, Message: "token scopes not found"}
	}
	chunk := text[idx+len(line):]
	scopes := strings.Split(strings.TrimSpace(strings.Split(chunk, "\n")[0]), ",")
	required := []string{"repo", "read:org"}
	missing := []string{}
	for _, req := range required {
		found := false
		for _, scope := range scopes {
			if strings.TrimSpace(scope) == req {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, req)
		}
	}
	if len(missing) > 0 {
		return CheckResult{Key: "auth", Status: StatusFail, Message: "missing scopes: " + strings.Join(missing, ", ")}
	}
	return CheckResult{Key: "auth", Status: StatusPass, Message: "token has required scopes"}
}

func checkStateDir() CheckResult {
	path := ".max-ops"
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return CheckResult{Key: "state", Status: StatusInfo, Message: ".max-ops/ present"}
	}
	return CheckResult{Key: "state", Status: StatusInfo, Message: ".max-ops/ not found (first run?)"}
}

func itoa(value int) string {
	return strconv.Itoa(value)
}
