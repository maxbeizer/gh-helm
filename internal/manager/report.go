package manager

import (
	"context"
	"fmt"
	"sort"

	"github.com/maxbeizer/max-ops/internal/pillars"
)

type ReportResult struct {
	Handle      string                 `json:"handle"`
	Since       string                 `json:"since"`
	Activity    map[string]int         `json:"activity"`
	PillarStats map[string]int         `json:"pillar_stats"`
	Highlights  []string               `json:"highlights"`
	Timeline    []string               `json:"timeline"`
}

func (m *Manager) Report(ctx context.Context, opts ReportOptions) (ReportResult, error) {
	ctx = ctxOrBackground(ctx)
	sinceText := opts.Since
	if sinceText == "" {
		sinceText = "90d"
	}
	start, err := parseSince(sinceText)
	if err != nil {
		return ReportResult{}, err
	}
	member, err := m.memberByHandle(opts.Handle)
	if err != nil {
		return ReportResult{}, err
	}
	activity, err := fetchActivity(ctx, member.Handle, start)
	if err != nil {
		return ReportResult{}, err
	}

	matches := []pillars.PillarMatch{}
	for _, item := range append(activity.MergedPRs, activity.IssuesClosed...) {
		matches = append(matches, m.mapItem(item, m.Config.Pillars)...)
	}
	summary := pillars.BuildSummary(matches)

	activityCounts := map[string]int{
		"prs_merged":   len(activity.MergedPRs),
		"reviews":      len(activity.Reviews),
		"issues_closed": len(activity.IssuesClosed),
		"issues_opened": len(activity.IssuesOpened),
	}

	highlights := []string{}
	if len(activity.MergedPRs) > 0 {
		highlights = append(highlights, fmt.Sprintf("%d PRs merged", len(activity.MergedPRs)))
	}
	if len(summary.Counts) > 0 {
		for pillar, count := range summary.Counts {
			highlights = append(highlights, fmt.Sprintf("%s: %d contributions", titleCase(pillar), count))
		}
	}
	sort.Strings(highlights)

	timeline := buildTimeline(activity.MergedPRs)

	return ReportResult{
		Handle:      member.Handle,
		Since:       sinceText,
		Activity:    activityCounts,
		PillarStats: summary.Counts,
		Highlights:  highlights,
		Timeline:    timeline,
	}, nil
}

func buildTimeline(items []pillars.ActivityItem) []string {
	entries := []string{}
	for _, item := range items {
		if item.ClosedAt.IsZero() {
			continue
		}
		entries = append(entries, fmt.Sprintf("%s — %s", item.ClosedAt.Format("2006-01-02"), item.Title))
	}
	sort.Strings(entries)
	return entries
}
