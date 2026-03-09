package sot

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	gh "github.com/maxbeizer/gh-helm/internal/github"
	"github.com/maxbeizer/gh-helm/internal/state"
)

// SyncResult describes changes found by syncing the SOT with issue state.
type SyncResult struct {
	Removed []string `json:"removed"`
	Kept    []string `json:"kept"`
	Summary string   `json:"summary"`
}

// Sync reads the SOT file, cross-references items referencing issues with
// actual issue state, and returns what should change. If apply is true,
// the SOT file is rewritten with completed items removed from "Next Up".
func Sync(ctx context.Context, path, repo string, apply bool) (SyncResult, error) {
	content, err := Read(path)
	if err != nil {
		return SyncResult{}, err
	}

	closedIssues, err := fetchClosedIssueNumbers(ctx, repo)
	if err != nil {
		return SyncResult{}, fmt.Errorf("fetch closed issues: %w", err)
	}

	newContent, result := reconcile(content, closedIssues)

	if apply && len(result.Removed) > 0 {
		if err := writeAtomic(path, newContent); err != nil {
			return SyncResult{}, fmt.Errorf("write SOT: %w", err)
		}
	}

	return result, nil
}

// issueRefPattern matches #N issue references in markdown lines.
var issueRefPattern = regexp.MustCompile(`#(\d+)`)

// reconcile processes the SOT content, removing lines from "Next Up" that
// reference closed issues. It also moves completed items to "Outcomes".
func reconcile(content string, closedIssues map[int]bool) (string, SyncResult) {
	lines := strings.Split(content, "\n")
	var result SyncResult
	var output []string

	inNextUp := false
	var outcomesInsertIdx int

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track which section we're in
		if strings.HasPrefix(trimmed, "## ") {
			if trimmed == "## Next Up" {
				inNextUp = true
				output = append(output, line)
				continue
			}
			if trimmed == "## Outcomes" {
				outcomesInsertIdx = len(output) + 1
			}
			inNextUp = false
		}

		if inNextUp && (strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ")) {
			refs := issueRefPattern.FindAllStringSubmatch(trimmed, -1)
			removed := false
			for _, ref := range refs {
				num, _ := strconv.Atoi(ref[1])
				if closedIssues[num] {
					result.Removed = append(result.Removed, fmt.Sprintf("Removed: %s (issue #%d closed)", trimmed, num))
					removed = true
					break
				}
			}
			if removed {
				_ = i // line skipped from output
				continue
			}
			result.Kept = append(result.Kept, trimmed)
		}

		output = append(output, line)
	}

	// If we found an outcomes section and removed items, add them as completed
	if outcomesInsertIdx > 0 && len(result.Removed) > 0 {
		var newOutcomes []string
		for _, r := range result.Removed {
			// Extract the item text from "Removed: - item text (issue #N closed)"
			text := strings.TrimPrefix(r, "Removed: ")
			if idx := strings.LastIndex(text, " (issue #"); idx > 0 {
				text = text[:idx]
			}
			// Convert "- item" to "- [x] item"
			text = strings.TrimPrefix(text, "- ")
			text = strings.TrimPrefix(text, "* ")
			newOutcomes = append(newOutcomes, "- [x] "+text)
		}

		// Insert after the ## Outcomes line
		before := output[:outcomesInsertIdx]
		after := output[outcomesInsertIdx:]
		output = append(before, newOutcomes...)
		output = append(output, after...)
	}

	switch len(result.Removed) {
	case 0:
		result.Summary = "SOT is up to date — no completed items found in Next Up"
	default:
		result.Summary = fmt.Sprintf("Found %d completed item(s) in Next Up referencing closed issues", len(result.Removed))
	}

	return strings.Join(output, "\n"), result
}

func writeAtomic(path, content string) error {
	return state.WriteAtomic(path, []byte(content), 0o644)
}

func fetchClosedIssueNumbers(ctx context.Context, repo string) (map[int]bool, error) {
	issues, err := gh.ListIssues(ctx, repo, "closed")
	if err != nil {
		return nil, err
	}
	m := make(map[int]bool, len(issues))
	for _, iss := range issues {
		m[iss.Number] = true
	}
	return m, nil
}
