package oneone

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	gh "github.com/maxbeizer/gh-helm/internal/github"
)

type ObservationIssue struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
	URL       string `json:"url"`
}

func PostObservation(ctx context.Context, handle, repo, title, body string) error {
	out, err := gh.RunWith(ctx, "issue", "create", "--repo", repo, "--title", title, "--body", body)
	if err != nil {
		return fmt.Errorf("gh issue create: %w (%s)", err, string(out))
	}
	return nil
}

func FetchRecentObservations(ctx context.Context, repo string, limit int) ([]ObservationIssue, error) {
	if limit <= 0 {
		limit = 5
	}
	out, err := gh.RunWith(ctx, "issue", "list", "--repo", repo, "--limit", strconv.Itoa(limit), "--json", "number,title,createdAt,url", "--state", "all")
	if err != nil {
		return nil, fmt.Errorf("list observations from %s: %w", repo, err)
	}
	var issues []ObservationIssue
	if err := json.Unmarshal(out, &issues); err != nil {
		return nil, fmt.Errorf("parse observations from %s: %w", repo, err)
	}
	return issues, nil
}
