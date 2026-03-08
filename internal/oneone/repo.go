package oneone

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

type ObservationIssue struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
	URL       string `json:"url"`
}

func PostObservation(ctx context.Context, handle, repo, title, body string) error {
	cmd := exec.CommandContext(ctx, "gh", "issue", "create", "--repo", repo, "--title", title, "--body", body)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh issue create: %w (%s)", err, string(out))
	}
	return nil
}

func FetchRecentObservations(ctx context.Context, repo string, limit int) ([]ObservationIssue, error) {
	if limit <= 0 {
		limit = 5
	}
	cmd := exec.CommandContext(ctx, "gh", "issue", "list", "--repo", repo, "--limit", fmt.Sprintf("%d", limit), "--json", "number,title,createdAt,url", "--state", "all")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var issues []ObservationIssue
	if err := json.Unmarshal(out, &issues); err != nil {
		return nil, err
	}
	return issues, nil
}
