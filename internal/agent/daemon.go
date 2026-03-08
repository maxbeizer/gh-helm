package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/maxbeizer/gh-helm/internal/config"
	"github.com/maxbeizer/gh-helm/internal/state"
	"github.com/maxbeizer/gh-helm/internal/github"
	"github.com/maxbeizer/gh-helm/internal/guardrails"
	"github.com/maxbeizer/gh-helm/internal/notifications"
)

type Logger interface {
	Printf(format string, args ...any)
}

type DaemonOpts struct {
	Interval             time.Duration
	MaxPerHour           int
	Status               string
	Label                string
	Codespace            bool
	CodespaceMachine     string
	CodespaceIdleTimeout string
	DryRun               bool
	ProjectOwner         string
	ProjectNumber        int
	Logger               Logger
}

type failureEntry struct {
	Time   string `json:"time"`
	Repo   string `json:"repo"`
	Issue  int    `json:"issue"`
	Error  string `json:"error"`
}

func RunDaemon(ctx context.Context, cfg config.ProjectConfig, opts DaemonOpts) error {
	interval := opts.Interval
	if interval == 0 {
		interval = 30 * time.Second
	}
	maxPerHour := opts.MaxPerHour
	if maxPerHour == 0 {
		maxPerHour = 3
	}
	status := opts.Status
	if status == "" {
		status = "Ready"
	}
	owner := opts.ProjectOwner
	if owner == "" {
		owner = cfg.Owner
	}
	project := opts.ProjectNumber
	if project == 0 {
		project = cfg.Board
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	limiter := guardrails.NewRateLimiter(maxPerHour)
	seen := make(map[string]bool)

	logf := func(format string, args ...any) { slog.Info(fmt.Sprintf(format, args...)) }
	if opts.Logger != nil {
		logf = opts.Logger.Printf
	}
	logf("project daemon started (interval: %s)", interval)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			items, err := fetchQueueItems(ctx, owner, project, status, opts.Label)
			if err != nil {
				logf("poll error: %v", err)
				continue
			}
			for _, item := range items {
				if seen[item.ID] || !limiter.Allow() {
					continue
				}
				seen[item.ID] = true
				if err := processItem(ctx, item, opts); err != nil {
					commentFailure(ctx, item, err)
					moveToStatus(ctx, owner, project, item, "Needs Attention")
					logFailure(item, err)
					continue
				}
			}
		}
	}
}

func processItem(ctx context.Context, item guardrails.QueueItem, opts DaemonOpts) error {
	checks := guardrails.SafetyChecks{}
	if err := checks.ValidateItem(item); err != nil {
		return err
	}
	if opts.DryRun {
		if opts.Logger != nil {
			opts.Logger.Printf("dry run: would process %s#%d", item.Repo, item.Number)
		} else {
			slog.Info("dry run: would process", "repo", item.Repo, "issue", item.Number)
		}
		return nil
	}

	agentRunner := NewProjectAgent()
	result, err := agentRunner.Start(ctx, StartOptions{
		IssueNumber: item.Number,
		Repo:        item.Repo,
		DryRun:      opts.DryRun,
	})
	if err != nil {
		return err
	}

	if opts.Codespace {
		name, url, err := CreateCodespace(ctx, CodespaceOpts{
			Repo:        item.Repo,
			Branch:      result.Branch,
			Machine:     defaultIfEmpty(opts.CodespaceMachine, "basicLinux32gb"),
			IdleTimeout: defaultIfEmpty(opts.CodespaceIdleTimeout, "30m"),
		})
		if err != nil {
			return err
		}
		defer func() {
			// Clean up codespace after work is done
			if delErr := DeleteCodespace(context.Background(), name); delErr != nil {
				slog.Warn("codespace cleanup failed", "error", delErr)
			}
		}()
		if err := WaitForReady(ctx, name, 20*time.Minute); err != nil {
			return err
		}

		cfg, err := config.Load("helm.toml")
		if err != nil {
			return err
		}
		notifier := notifications.New(cfg, item.Repo, item.Number)
		if notifier != nil {
			if err := notifier.Notify(ctx, notifications.Message{
				Title:   "💻 Codespace ready",
				Body:    fmt.Sprintf("Codespace ready for %s#%d\nBranch: %s", item.Repo, item.Number, result.Branch),
				Channel: cfg.Notifications.OpsChannel,
				URL:     url,
			}); err != nil {
				slog.Warn("notification failed", "error", err)
			}
		}
	}

	return nil
}

func fetchQueueItems(ctx context.Context, owner string, projectNumber int, status string, label string) ([]guardrails.QueueItem, error) {
	if owner == "" || projectNumber == 0 {
		return nil, fmt.Errorf("project owner and number required")
	}
	query := `
query($owner: String!, $number: Int!) {
  organization(login: $owner) {
    projectV2(number: $number) {
      items(first: 50) {
        nodes {
          id
          content {
            ... on Issue {
              number
              title
              body
              url
              id
              repository { nameWithOwner }
              labels(first: 20) { nodes { name } }
            }
          }
          fieldValues(first: 20) {
            nodes {
              ... on ProjectV2ItemFieldSingleSelectValue {
                name
                field { ... on ProjectV2SingleSelectField { name } }
              }
            }
          }
        }
      }
    }
  }
  user(login: $owner) {
    projectV2(number: $number) {
      items(first: 50) {
        nodes {
          id
          content {
            ... on Issue {
              number
              title
              body
              url
              id
              repository { nameWithOwner }
              labels(first: 20) { nodes { name } }
            }
          }
          fieldValues(first: 20) {
            nodes {
              ... on ProjectV2ItemFieldSingleSelectValue {
                name
                field { ... on ProjectV2SingleSelectField { name } }
              }
            }
          }
        }
      }
    }
  }
}`

	out, err := github.RunWith(ctx, "api", "graphql", "-f", "query="+query, "-F", "owner="+owner, "-F", fmt.Sprintf("number=%d", projectNumber))
	if err != nil {
		return nil, err
	}

	var resp projectItemsResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, err
	}
	project := resp.Organization
	if project == nil || project.Project == nil {
		project = resp.User
	}
	if project == nil || project.Project == nil {
		return nil, fmt.Errorf("project not found")
	}

	items := []guardrails.QueueItem{}
	for _, node := range project.Project.Items.Nodes {
		if node.Content.Number == 0 {
			continue
		}
		itemStatus := node.Status()
		if status != "" && itemStatus != status {
			continue
		}
		labels := node.LabelNames()
		if label != "" && !containsLabel(labels, label) {
			continue
		}
		items = append(items, guardrails.QueueItem{
			ID:     node.ID,
			NodeID: node.Content.ID,
			Number: node.Content.Number,
			Title:  node.Content.Title,
			Body:   node.Content.Body,
			Repo:   node.Content.Repository.NameWithOwner,
			URL:    node.Content.URL,
			Labels: labels,
		})
	}
	return items, nil
}

type projectItemsResponse struct {
	Organization *struct {
		Project *projectItems `json:"projectV2"`
	} `json:"organization"`
	User *struct {
		Project *projectItems `json:"projectV2"`
	} `json:"user"`
}

type projectItems struct {
	Items struct {
		Nodes []projectItemNode `json:"nodes"`
	} `json:"items"`
}

type projectItemNode struct {
	ID      string `json:"id"`
	Content struct {
		Number     int    `json:"number"`
		Title      string `json:"title"`
		Body       string `json:"body"`
		URL        string `json:"url"`
		ID         string `json:"id"`
		Repository struct {
			NameWithOwner string `json:"nameWithOwner"`
		} `json:"repository"`
		Labels struct {
			Nodes []struct {
				Name string `json:"name"`
			} `json:"nodes"`
		} `json:"labels"`
	} `json:"content"`
	FieldValues struct {
		Nodes []struct {
			Name  string `json:"name"`
			Field struct {
				Name string `json:"name"`
			} `json:"field"`
		} `json:"nodes"`
	} `json:"fieldValues"`
}

func (n projectItemNode) Status() string {
	for _, field := range n.FieldValues.Nodes {
		if field.Field.Name == "Status" {
			return field.Name
		}
	}
	return ""
}

func (n projectItemNode) LabelNames() []string {
	labels := make([]string, 0, len(n.Content.Labels.Nodes))
	for _, node := range n.Content.Labels.Nodes {
		labels = append(labels, node.Name)
	}
	return labels
}

func containsLabel(labels []string, target string) bool {
	for _, label := range labels {
		if strings.EqualFold(label, target) {
			return true
		}
	}
	return false
}

func commentFailure(ctx context.Context, item guardrails.QueueItem, err error) {
	if err := github.CommentIssue(ctx, item.Repo, item.Number, fmt.Sprintf("🤖 gh-helm agent encountered an issue: `%s`", err.Error())); err != nil {
		slog.Warn("comment failed", "error", err)
	}
}

func moveToStatus(ctx context.Context, owner string, project int, item guardrails.QueueItem, status string) {
	if owner == "" || project == 0 {
		return
	}
	if err := github.MoveIssueToStatus(ctx, owner, project, item.NodeID, status); err != nil {
		slog.Warn("move status failed", "error", err)
	}
}

func logFailure(item guardrails.QueueItem, err error) {
	entry := failureEntry{
		Time:  time.Now().Format(time.RFC3339),
		Repo:  item.Repo,
		Issue: item.Number,
		Error: err.Error(),
	}
	path := filepath.Join(".helm", "failures.json")
	var entries []failureEntry
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, &entries); err != nil {
			slog.Warn("unmarshal failures log", "error", err)
		}
	}
	entries = append(entries, entry)
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return
	}
	if err := state.WriteAtomic(path, data, 0o644); err != nil {
		slog.Warn("write failures log", "error", err)
	}
}

func defaultIfEmpty(val string, fallback string) string {
	if val == "" {
		return fallback
	}
	return val
}
