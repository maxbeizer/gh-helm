package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

// PR holds pull request metadata.
type PR struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"`
	URL    string `json:"url"`
}

// FetchPR retrieves pull request metadata.
func FetchPR(ctx context.Context, repo string, number int) (PR, error) {
	slog.Debug("fetching PR", "repo", repo, "number", number)
	args := []string{"pr", "view", fmt.Sprint(number), "--json", "number,title,body,state,url"}
	if repo != "" {
		args = append(args, "--repo", repo)
	}
	out, err := runGh(ctx, args...)
	if err != nil {
		return PR{}, fmt.Errorf("fetch PR #%d: %w", number, err)
	}
	var pr PR
	if err := json.Unmarshal(out, &pr); err != nil {
		return PR{}, fmt.Errorf("parse PR #%d response: %w", number, err)
	}
	return pr, nil
}

// FetchPRDiff retrieves the diff for a pull request.
func FetchPRDiff(ctx context.Context, repo string, number int) (string, error) {
	slog.Debug("fetching PR diff", "repo", repo, "number", number)
	args := []string{"pr", "diff", fmt.Sprint(number)}
	if repo != "" {
		args = append(args, "--repo", repo)
	}
	out, err := runGh(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("fetch PR #%d diff: %w", number, err)
	}
	return string(bytes.TrimSpace(out)), nil
}

// FetchPRClosingIssues returns issue numbers that a PR will close on merge.
func FetchPRClosingIssues(ctx context.Context, repo string, number int) ([]int, error) {
	slog.Debug("fetching PR closing issues", "repo", repo, "number", number)
	args := []string{"pr", "view", fmt.Sprint(number), "--json", "closingIssuesReferences", "--jq", ".[\"closingIssuesReferences\"].[].number"}
	if repo != "" {
		args = append(args, "--repo", repo)
	}
	out, err := runGh(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("fetch PR #%d closing issues: %w", number, err)
	}
	trimmed := bytes.TrimSpace(out)
	if len(trimmed) == 0 {
		return nil, nil
	}

	var numbers []int
	for _, line := range bytes.Split(trimmed, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(string(line), "%d", &n); err == nil {
			numbers = append(numbers, n)
		}
	}
	return numbers, nil
}
