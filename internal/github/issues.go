package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
)

type Issue struct {
	Number     int      `json:"number"`
	Title      string   `json:"title"`
	Body       string   `json:"body"`
	Comments   []Comment `json:"comments"`
	Labels     []Label  `json:"labels"`
	Assignees  []User   `json:"assignees"`
	URL        string   `json:"url"`
	NodeID     string   `json:"id"`
}

type Comment struct {
	Body string `json:"body"`
}

type Label struct {
	Name string `json:"name"`
}

type User struct {
	Login string `json:"login"`
}

func FetchIssue(ctx context.Context, repo string, number int) (Issue, error) {
	slog.Debug("fetching issue", "repo", repo, "number", number)
	args := []string{"issue", "view", strconv.Itoa(number), "--json", "number,title,body,comments,labels,assignees,url,id"}
	if repo != "" {
		args = append(args, "--repo", repo)
	}
	out, err := runGh(ctx, args...)
	if err != nil {
		return Issue{}, fmt.Errorf("fetch issue #%d: %w", number, err)
	}
	var issue Issue
	if err := json.Unmarshal(out, &issue); err != nil {
		return Issue{}, fmt.Errorf("parse issue #%d response: %w", number, err)
	}
	slog.Debug("fetched issue", "number", issue.Number, "title", issue.Title, "nodeID", issue.NodeID, "labels", len(issue.Labels))
	return issue, nil
}

// IssueListItem is a lightweight issue representation returned by ListIssues.
type IssueListItem struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	URL    string `json:"url"`
}

// ListIssues returns issues for a repo filtered by state ("open", "closed", or "all").
func ListIssues(ctx context.Context, repo, state string) ([]IssueListItem, error) {
	slog.Debug("listing issues", "repo", repo, "state", state)
	if state == "" {
		state = "all"
	}
	args := []string{"issue", "list", "--state", state, "--json", "number,title,state,url", "--limit", "200"}
	if repo != "" {
		args = append(args, "--repo", repo)
	}
	out, err := runGh(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("list issues: %w", err)
	}
	var items []IssueListItem
	if err := json.Unmarshal(out, &items); err != nil {
		return nil, fmt.Errorf("parse issue list: %w", err)
	}
	return items, nil
}

func CommentIssue(ctx context.Context, repo string, number int, body string) error {
	args := []string{"issue", "comment", strconv.Itoa(number), "--body", body}
	if repo != "" {
		args = append(args, "--repo", repo)
	}
	_, err := runGh(ctx, args...)
	if err != nil {
		return fmt.Errorf("comment on issue #%d: %w", number, err)
	}
	return nil
}
