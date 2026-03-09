package sot

import (
	"context"
	"fmt"
	"strings"

	gh "github.com/maxbeizer/gh-helm/internal/github"
)

// ProposeFromPR generates a SOT update proposal based on a PR's metadata and diff.
func ProposeFromPR(ctx context.Context, path, repo string, prNumber int, session string) (string, error) {
	pr, err := gh.FetchPR(ctx, repo, prNumber)
	if err != nil {
		return "", fmt.Errorf("fetch PR: %w", err)
	}

	diff, err := gh.FetchPRDiff(ctx, repo, prNumber)
	if err != nil {
		return "", fmt.Errorf("fetch PR diff: %w", err)
	}

	closingIssues, err := gh.FetchPRClosingIssues(ctx, repo, prNumber)
	if err != nil {
		// Non-fatal — we can still propose without closing issue info
		closingIssues = nil
	}

	decision := buildProposal(pr, diff, closingIssues)

	if err := Propose(path, decision, session, fmt.Sprintf("#%d", prNumber)); err != nil {
		return "", err
	}

	return decision, nil
}

func buildProposal(pr gh.PR, diff string, closingIssues []int) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("PR #%d: %s", pr.Number, pr.Title))

	if len(closingIssues) > 0 {
		var refs []string
		for _, n := range closingIssues {
			refs = append(refs, fmt.Sprintf("#%d", n))
		}
		parts = append(parts, fmt.Sprintf("Closes: %s", strings.Join(refs, ", ")))
	}

	changes := summarizeDiff(diff)
	if len(changes) > 0 {
		parts = append(parts, "Changes: "+strings.Join(changes, "; "))
	}

	return strings.Join(parts, ". ")
}

// summarizeDiff extracts a high-level summary of what files/packages changed.
func summarizeDiff(diff string) []string {
	seen := make(map[string]bool)
	var changes []string

	for _, line := range strings.Split(diff, "\n") {
		if !strings.HasPrefix(line, "diff --git") {
			continue
		}
		// Extract b/path from "diff --git a/foo b/foo"
		parts := strings.Fields(line)
		if len(parts) < 4 {
			continue
		}
		file := strings.TrimPrefix(parts[3], "b/")
		dir := categorizeFile(file)
		if !seen[dir] {
			seen[dir] = true
			changes = append(changes, dir)
		}
	}

	return changes
}

func categorizeFile(path string) string {
	parts := strings.Split(path, "/")
	switch {
	case strings.HasPrefix(path, "cmd/"):
		return "CLI commands"
	case strings.HasPrefix(path, "internal/sot/"):
		return "SOT logic"
	case strings.HasPrefix(path, "internal/github/"):
		return "GitHub API layer"
	case strings.HasPrefix(path, "internal/agent/"):
		return "project agent"
	case strings.HasPrefix(path, "internal/manager/"):
		return "manager agent"
	case strings.HasPrefix(path, "internal/config/"):
		return "configuration"
	case strings.HasPrefix(path, "docs/"):
		return "documentation"
	case strings.HasSuffix(path, "_test.go"):
		return "tests"
	case len(parts) >= 2 && parts[0] == "internal":
		return parts[1] + " package"
	default:
		return path
	}
}
