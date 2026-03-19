package notifications

import (
	"context"
	"fmt"
	"os/exec"
)

// RunFunc executes a gh CLI command. Override in tests to mock.
var RunFunc = defaultRun

func defaultRun(ctx context.Context, args ...string) error {
	return exec.CommandContext(ctx, "gh", args...).Run()
}

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
	return RunFunc(ctx, args...)
}
