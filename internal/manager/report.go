package manager

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/maxbeizer/max-ops/internal/github"
	"github.com/maxbeizer/max-ops/internal/pillars"
)

type TimelineEntry struct {
	Date  string `json:"date"`
	Title string `json:"title"`
	URL   string `json:"url"`
	Kind  string `json:"kind"`
}

type WeekVelocity struct {
	Week  string `json:"week"`
	Count int    `json:"count"`
}

type VelocityTrends struct {
	Trend string         `json:"trend"`
	Weeks []WeekVelocity `json:"weeks"`
}

type ReviewQuality struct {
	Reviews             int     `json:"reviews"`
	AvgTurnaroundHours  float64 `json:"avg_turnaround_hours"`
	AvgCommentsPerReview float64 `json:"avg_comments_per_review"`
}

type NotableContribution struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Kind  string `json:"kind"`
}

type GrowthTracking struct {
	Available bool           `json:"available"`
	Current   map[string]int `json:"current,omitempty"`
	Previous  map[string]int `json:"previous,omitempty"`
	Delta     map[string]int `json:"delta,omitempty"`
}

type ReportResult struct {
	Handle            string                 `json:"handle"`
	Since             string                 `json:"since"`
	Activity          map[string]int         `json:"activity"`
	PillarStats       map[string]int         `json:"pillar_stats"`
	Highlights        []string               `json:"highlights"`
	Timeline          []TimelineEntry        `json:"timeline"`
	ActivityTimeline  []TimelineEntry        `json:"activity_timeline"`
	VelocityTrends    VelocityTrends         `json:"velocity_trends"`
	PillarHeatmap     map[string]string      `json:"pillar_heatmap"`
	ReviewQuality     ReviewQuality          `json:"review_quality"`
	NotableContribs   []NotableContribution  `json:"notable_contributions"`
	GrowthTracking    GrowthTracking         `json:"growth_tracking"`
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
		"prs_merged":    len(activity.MergedPRs),
		"reviews":       len(activity.Reviews),
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
	activityTimeline := buildActivityTimeline(activity)
	velocity := buildVelocityTrends(activity.MergedPRs, start, now())
	heatmap := buildPillarHeatmap(summary.Counts)
	reviewQuality := buildReviewQuality(activity)
	notables := buildNotableContributions(activity)
	growth := buildGrowthTracking(ctx, member.Handle, start, now(), activityCounts)

	return ReportResult{
		Handle:           member.Handle,
		Since:            sinceText,
		Activity:         activityCounts,
		PillarStats:      summary.Counts,
		Highlights:       highlights,
		Timeline:         timeline,
		ActivityTimeline: activityTimeline,
		VelocityTrends:   velocity,
		PillarHeatmap:    heatmap,
		ReviewQuality:    reviewQuality,
		NotableContribs:  notables,
		GrowthTracking:   growth,
	}, nil
}

func buildTimeline(items []pillars.ActivityItem) []TimelineEntry {
	entries := []TimelineEntry{}
	for _, item := range items {
		if item.ClosedAt.IsZero() {
			continue
		}
		entries = append(entries, TimelineEntry{
			Date:  item.ClosedAt.Format("2006-01-02"),
			Title: item.Title,
			URL:   item.URL,
			Kind:  "pr",
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Date < entries[j].Date
	})
	return entries
}

func buildActivityTimeline(activity activityData) []TimelineEntry {
	entries := []TimelineEntry{}
	for _, item := range activity.MergedPRs {
		if item.ClosedAt.IsZero() {
			continue
		}
		entries = append(entries, TimelineEntry{Date: item.ClosedAt.Format("2006-01-02"), Title: item.Title, URL: item.URL, Kind: "pr"})
	}
	for _, item := range activity.IssuesClosed {
		if item.ClosedAt.IsZero() {
			continue
		}
		entries = append(entries, TimelineEntry{Date: item.ClosedAt.Format("2006-01-02"), Title: item.Title, URL: item.URL, Kind: "issue"})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Date == entries[j].Date {
			return entries[i].Title < entries[j].Title
		}
		return entries[i].Date < entries[j].Date
	})
	return entries
}

func buildVelocityTrends(items []pillars.ActivityItem, start, end time.Time) VelocityTrends {
	counts := map[string]int{}
	for _, item := range items {
		if item.ClosedAt.IsZero() {
			continue
		}
		if item.ClosedAt.Before(start) || item.ClosedAt.After(end) {
			continue
		}
		key := weekKey(item.ClosedAt)
		counts[key]++
	}

	weeks := []WeekVelocity{}
	cursor := truncateToWeek(start)
	limit := truncateToWeek(end)
	for !cursor.After(limit) {
		key := weekKey(cursor)
		weeks = append(weeks, WeekVelocity{Week: key, Count: counts[key]})
		cursor = cursor.AddDate(0, 0, 7)
	}

	trend := "→"
	if len(weeks) >= 2 {
		prev := weeks[len(weeks)-2].Count
		last := weeks[len(weeks)-1].Count
		delta := last - prev
		switch {
		case delta >= 2:
			trend = "↑"
		case delta == 1:
			trend = "↗"
		case delta == 0:
			trend = "→"
		case delta == -1:
			trend = "↘"
		case delta <= -2:
			trend = "↓"
		}
	}

	return VelocityTrends{Trend: trend, Weeks: weeks}
}

func buildPillarHeatmap(counts map[string]int) map[string]string {
	max := 0
	for _, count := range counts {
		if count > max {
			max = count
		}
	}
	result := map[string]string{}
	for pillar, count := range counts {
		result[pillar] = densityBar(count, max)
	}
	return result
}

func densityBar(count int, max int) string {
	if max == 0 {
		return "·"
	}
	level := int(float64(count) / float64(max) * 5.0)
	if level == 0 && count > 0 {
		level = 1
	}
	if level > 5 {
		level = 5
	}
	return strings.Repeat("█", level)
}

func buildReviewQuality(activity activityData) ReviewQuality {
	quality := ReviewQuality{Reviews: len(activity.Reviews)}
	if len(activity.Reviews) == 0 {
		return quality
	}
	var totalHours float64
	var totalComments int
	for _, review := range activity.Reviews {
		created := parseGitHubTime(review.CreatedAt)
		updated := parseGitHubTime(review.UpdatedAt)
		if !created.IsZero() && !updated.IsZero() {
			totalHours += updated.Sub(created).Hours()
		}
		totalComments += review.Comments
	}
	quality.AvgTurnaroundHours = totalHours / float64(len(activity.Reviews))
	quality.AvgCommentsPerReview = float64(totalComments) / float64(len(activity.Reviews))
	return quality
}

func buildNotableContributions(activity activityData) []NotableContribution {
	items := []struct {
		item  pillars.ActivityItem
		kind  string
		score int
	}{}
	for _, item := range activity.MergedPRs {
		items = append(items, struct {
			item  pillars.ActivityItem
			kind  string
			score int
		}{item: item, kind: "pr", score: len(item.Files) + len(item.Title)/40})
	}
	for _, item := range activity.IssuesClosed {
		items = append(items, struct {
			item  pillars.ActivityItem
			kind  string
			score int
		}{item: item, kind: "issue", score: 1 + len(item.Title)/40})
	}
	if len(items) == 0 {
		return nil
	}
	cache := items
	sort.Slice(cache, func(i, j int) bool {
		if cache[i].score == cache[j].score {
			return cache[i].item.Title < cache[j].item.Title
		}
		return cache[i].score > cache[j].score
	})
	limit := 3
	if len(cache) < limit {
		limit = len(cache)
	}
	result := make([]NotableContribution, 0, limit)
	for i := 0; i < limit; i++ {
		result = append(result, NotableContribution{Title: cache[i].item.Title, URL: cache[i].item.URL, Kind: cache[i].kind})
	}
	return result
}

func buildGrowthTracking(ctx context.Context, handle string, start, end time.Time, current map[string]int) GrowthTracking {
	duration := end.Sub(start)
	if duration <= 0 {
		return GrowthTracking{Available: false}
	}
	prevStart := start.Add(-duration)
	activity, err := fetchActivitySummary(ctx, handle, prevStart)
	if err != nil {
		return GrowthTracking{Available: false}
	}
	filtered := filterActivityBetween(activity, prevStart, start)
	previous := map[string]int{
		"prs_merged":    len(filtered.MergedPRs),
		"reviews":       len(filtered.Reviews),
		"issues_closed": len(filtered.IssuesClosed),
		"issues_opened": len(filtered.IssuesOpened),
	}
	delta := map[string]int{}
	for key, value := range current {
		delta[key] = value - previous[key]
	}
	return GrowthTracking{Available: true, Current: current, Previous: previous, Delta: delta}
}

func fetchActivitySummary(ctx context.Context, handle string, since time.Time) (activityData, error) {
	queryDate := since.Format("2006-01-02")
	mergedPRs, err := github.SearchIssues(ctx, fmt.Sprintf("is:pr author:%s merged:>=%s", handle, queryDate))
	if err != nil {
		return activityData{}, err
	}
	openedPRs, err := github.SearchIssues(ctx, fmt.Sprintf("is:pr author:%s created:>=%s", handle, queryDate))
	if err != nil {
		return activityData{}, err
	}
	issuesClosed, err := github.SearchIssues(ctx, fmt.Sprintf("is:issue author:%s closed:>=%s", handle, queryDate))
	if err != nil {
		return activityData{}, err
	}
	issuesOpened, err := github.SearchIssues(ctx, fmt.Sprintf("is:issue author:%s created:>=%s", handle, queryDate))
	if err != nil {
		return activityData{}, err
	}
	reviews, err := github.SearchIssues(ctx, fmt.Sprintf("is:pr reviewed-by:%s updated:>=%s", handle, queryDate))
	if err != nil {
		return activityData{}, err
	}

	mergedItems, err := toActivityItems(ctx, mergedPRs, false)
	if err != nil {
		return activityData{}, err
	}
	openItems, err := toActivityItems(ctx, openedPRs, false)
	if err != nil {
		return activityData{}, err
	}
	closedIssues, err := toActivityItems(ctx, issuesClosed, false)
	if err != nil {
		return activityData{}, err
	}
	openedIssues, err := toActivityItems(ctx, issuesOpened, false)
	if err != nil {
		return activityData{}, err
	}

	return activityData{
		MergedPRs:   mergedItems,
		OpenPRs:     openItems,
		IssuesClosed: closedIssues,
		IssuesOpened: openedIssues,
		Reviews:     reviews,
	}, nil
}

func filterActivityBetween(activity activityData, start, end time.Time) activityData {
	return activityData{
		MergedPRs:    filterItemsBetween(activity.MergedPRs, start, end, func(item pillars.ActivityItem) time.Time { return item.ClosedAt }),
		OpenPRs:      filterItemsBetween(activity.OpenPRs, start, end, func(item pillars.ActivityItem) time.Time { return item.CreatedAt }),
		IssuesClosed: filterItemsBetween(activity.IssuesClosed, start, end, func(item pillars.ActivityItem) time.Time { return item.ClosedAt }),
		IssuesOpened: filterItemsBetween(activity.IssuesOpened, start, end, func(item pillars.ActivityItem) time.Time { return item.CreatedAt }),
		Reviews:      filterReviewsBetween(activity.Reviews, start, end),
	}
}

func filterItemsBetween(items []pillars.ActivityItem, start, end time.Time, pick func(pillars.ActivityItem) time.Time) []pillars.ActivityItem {
	filtered := []pillars.ActivityItem{}
	for _, item := range items {
		timeValue := pick(item)
		if timeValue.IsZero() {
			continue
		}
		if !timeValue.Before(start) && timeValue.Before(end) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterReviewsBetween(items []github.SearchItem, start, end time.Time) []github.SearchItem {
	filtered := []github.SearchItem{}
	for _, item := range items {
		timeValue := parseGitHubTime(item.UpdatedAt)
		if timeValue.IsZero() {
			continue
		}
		if !timeValue.Before(start) && timeValue.Before(end) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func weekKey(t time.Time) string {
	year, week := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}

func truncateToWeek(t time.Time) time.Time {
	weekday := (int(t.Weekday()) + 6) % 7
	return time.Date(t.Year(), t.Month(), t.Day()-weekday, 0, 0, 0, 0, t.Location())
}
