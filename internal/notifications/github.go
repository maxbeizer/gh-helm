package notifications

import (
	"context"
	"fmt"

	"github.com/maxbeizer/gh-helm/internal/github"
)

type GitHubNotifier struct {
	Repo        string
	IssueNumber int
}

func (g *GitHubNotifier) Notify(ctx context.Context, msg Message) error {
	body := fmt.Sprintf("%s\n%s", msg.Title, msg.Body)
	args := []string{"issue", "comment", fmt.Sprint(g.IssueNumber), "--body", body}
	if g.Repo != "" {
		args = append(args, "--repo", g.Repo)
	}
	_, err := github.RunWith(ctx, args...)
	return err
}
