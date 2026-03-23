package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/maxbeizer/gh-helm/internal/config"
	"github.com/maxbeizer/gh-helm/internal/github"
	"github.com/maxbeizer/gh-helm/internal/output"
	"github.com/spf13/cobra"
)

type nextAction struct {
	Action   string   `json:"action"`
	Detail   string   `json:"detail"`
	Command  string   `json:"command,omitempty"`
	Commands []string `json:"commands,omitempty"`
}

type nextResult struct {
	Steps []nextAction `json:"steps"`
}

var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "Show what to do next",
	RunE: func(cmd *cobra.Command, args []string) error {
		result := figureOutNext(cmd.Context())

		out := output.New(cmd)
		if out.WantsJSON() {
			return out.Print(result)
		}

		fmt.Fprintln(os.Stdout, "🧭 gh-helm — what's next?")
		fmt.Fprintln(os.Stdout)
		if len(result.Steps) == 0 {
			fmt.Fprintln(os.Stdout, "  All caught up! 🎉")
			return nil
		}
		for i, step := range result.Steps {
			fmt.Fprintf(os.Stdout, "  %d. %s\n", i+1, step.Detail)
			if len(step.Commands) > 0 {
				for _, c := range step.Commands {
					fmt.Fprintf(os.Stdout, "     → %s\n", c)
				}
			} else if step.Command != "" {
				fmt.Fprintf(os.Stdout, "     → %s\n", step.Command)
			}
		}
		fmt.Fprintln(os.Stdout)
		return nil
	},
}

func figureOutNext(ctx context.Context) nextResult {
	var steps []nextAction

	// Check if helm.toml exists.
	cfg, err := config.Load("helm.toml")
	if err != nil {
		steps = append(steps, nextAction{
			Action:  "init",
			Detail:  "Set up gh-helm in this repo",
			Command: "gh helm project init",
		})
		return nextResult{Steps: steps}
	}

	repo := currentRepo(ctx)

	// Check for open draft PRs from helm.
	if repo != "" {
		prs := findHelmPRs(ctx, repo)
		for _, pr := range prs {
			steps = append(steps, nextAction{
				Action: "review",
				Detail: fmt.Sprintf("Review and merge PR #%d: %s", pr.Number, pr.Title),
				Commands: []string{
					fmt.Sprintf("gh pr view -w %d", pr.Number),
					fmt.Sprintf("gh pr merge %d --squash", pr.Number),
				},
			})
		}
	}

	// Check project board for actionable items.
	if cfg.Project.Board != 0 && cfg.Project.Owner != "" {
		inProgress, ready := boardState(ctx, cfg)

		// Flag stale in-progress items (issue already closed but board not updated).
		for _, item := range inProgress {
			if repo != "" && isIssueClosed(ctx, repo, item.Number) {
				steps = append(steps, nextAction{
					Action: "cleanup",
					Detail: fmt.Sprintf("Issue #%d is closed but still 'In Progress' on the board: %s", item.Number, item.Title),
				})
			}
		}

		if len(ready) > 0 {
			item := ready[0]
			steps = append(steps, nextAction{
				Action:  "start",
				Detail:  fmt.Sprintf("Start the next issue: #%d %s", item.Number, item.Title),
				Command: fmt.Sprintf("gh helm project start --issue %d", item.Number),
			})
			if len(ready) > 1 {
				remaining := len(ready) - 1
				noun := "issues"
				if remaining == 1 {
					noun = "issue"
				}
				steps = append(steps, nextAction{
					Action: "info",
					Detail: fmt.Sprintf("%d more %s ready on the board", remaining, noun),
				})
			}
		}
	}

	// If no steps yet, check for open issues not on the board.
	if len(steps) == 0 && repo != "" {
		untracked := findUntrackedIssues(ctx, repo)
		if len(untracked) > 0 {
			item := untracked[0]
			steps = append(steps, nextAction{
				Action:  "start",
				Detail:  fmt.Sprintf("Start open issue: #%d %s", item.Number, item.Title),
				Command: fmt.Sprintf("gh helm project start --issue %d", item.Number),
			})
			if len(untracked) > 1 {
				remaining := len(untracked) - 1
				noun := "issue"
				if remaining > 1 {
					noun = "issues"
				}
				steps = append(steps, nextAction{
					Action: "info",
					Detail: fmt.Sprintf("%d more open %s in the repo", remaining, noun),
				})
			}
		}
	}

	return nextResult{Steps: steps}
}

func currentRepo(ctx context.Context) string {
	out, err := github.RunWith(ctx, "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

type helmPR struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
}

func findHelmPRs(ctx context.Context, repo string) []helmPR {
	out, err := github.RunWith(ctx, "pr", "list", "--repo", repo,
		"--state", "open", "--json", "number,title,url,headRefName")
	if err != nil {
		return nil
	}

	var prs []struct {
		Number      int    `json:"number"`
		Title       string `json:"title"`
		URL         string `json:"url"`
		HeadRefName string `json:"headRefName"`
	}
	if err := json.Unmarshal(out, &prs); err != nil {
		return nil
	}

	var helmPRs []helmPR
	for _, pr := range prs {
		if strings.HasPrefix(pr.HeadRefName, "gh-helm/") {
			helmPRs = append(helmPRs, helmPR{Number: pr.Number, Title: pr.Title, URL: pr.URL})
		}
	}
	return helmPRs
}

type boardItem struct {
	Number int
	Title  string
}

func boardState(ctx context.Context, cfg config.Config) (inProgress, ready []boardItem) {
	out, err := github.RunWith(ctx, "project", "item-list", fmt.Sprintf("%d", cfg.Project.Board),
		"--owner", cfg.Project.Owner, "--format", "json")
	if err != nil {
		return
	}

	var resp struct {
		Items []struct {
			Title  string `json:"title"`
			Status string `json:"status"`
			Content struct {
				Number int    `json:"number"`
				Type   string `json:"type"`
			} `json:"content"`
		} `json:"items"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return
	}

	for _, item := range resp.Items {
		if item.Content.Type != "Issue" {
			continue
		}
		bi := boardItem{Number: item.Content.Number, Title: item.Title}
		switch item.Status {
		case "In Progress":
			inProgress = append(inProgress, bi)
		case "Ready", "Todo", "":
			ready = append(ready, bi)
		}
	}
	return
}

func isIssueClosed(ctx context.Context, repo string, number int) bool {
	out, err := github.RunWith(ctx, "issue", "view", fmt.Sprintf("%d", number),
		"--repo", repo, "--json", "state", "--jq", ".state")
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "CLOSED"
}

type untrackedIssue struct {
	Number int
	Title  string
}

func findUntrackedIssues(ctx context.Context, repo string) []untrackedIssue {
	out, err := github.RunWith(ctx, "issue", "list",
		"--repo", repo, "--state", "open",
		"--json", "number,title",
		"--limit", "10")
	if err != nil {
		return nil
	}
	var issues []untrackedIssue
	if err := json.Unmarshal(out, &issues); err != nil {
		return nil
	}
	return issues
}

func init() {
	rootCmd.AddCommand(nextCmd)
}
