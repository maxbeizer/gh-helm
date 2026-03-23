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
	Action  string `json:"action"`
	Detail  string `json:"detail"`
	Command string `json:"command,omitempty"`
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
			if step.Command != "" {
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
				Action:  "review",
				Detail:  fmt.Sprintf("Review and merge PR #%d: %s", pr.Number, pr.Title),
				Command: fmt.Sprintf("gh pr merge %d --squash", pr.Number),
			})
		}
	}

	// Check project board for in-progress and ready items.
	if cfg.Project.Board != 0 && cfg.Project.Owner != "" {
		inProgress, ready := boardState(ctx, cfg)

		for _, item := range inProgress {
			// Only flag if there's no open PR for it already.
			hasPR := false
			for _, step := range steps {
				if step.Action == "review" {
					hasPR = true
					break
				}
			}
			if !hasPR {
				steps = append(steps, nextAction{
					Action: "check",
					Detail: fmt.Sprintf("Issue #%d is in progress but has no PR: %s", item.Number, item.Title),
				})
			}
		}

		if len(steps) == 0 && len(ready) > 0 {
			item := ready[0]
			steps = append(steps, nextAction{
				Action:  "start",
				Detail:  fmt.Sprintf("Start the next issue: #%d %s", item.Number, item.Title),
				Command: fmt.Sprintf("gh helm project start --issue %d", item.Number),
			})
			if len(ready) > 1 {
				steps = append(steps, nextAction{
					Action: "info",
					Detail: fmt.Sprintf("%d more issues ready on the board", len(ready)-1),
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
	for _, status := range []string{"In Progress", "Ready"} {
		out, err := github.RunWith(ctx, "project", "item-list", fmt.Sprintf("%d", cfg.Project.Board),
			"--owner", cfg.Project.Owner, "--format", "json")
		if err != nil {
			continue
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
			continue
		}

		for _, item := range resp.Items {
			if item.Content.Type != "Issue" {
				continue
			}
			if item.Status == status {
				bi := boardItem{Number: item.Content.Number, Title: item.Title}
				if status == "In Progress" {
					inProgress = append(inProgress, bi)
				} else {
					ready = append(ready, bi)
				}
			}
		}
		break // Only need one API call — we check both statuses from the same response.
	}
	return
}

func init() {
	rootCmd.AddCommand(nextCmd)
}
