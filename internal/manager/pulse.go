package manager

import (
	"context"
	"fmt"
	"math"

	"github.com/maxbeizer/gh-helm/internal/github"
	"github.com/maxbeizer/gh-helm/internal/pillars"
)

type PulseEntry struct {
	Handle           string  `json:"handle"`
	MergedPerWeek    float64 `json:"merged_per_week"`
	AvgCycleDays     float64 `json:"avg_cycle_days"`
	ReviewsCompleted int     `json:"reviews_completed"`
	BlockedCount     int     `json:"blocked_count"`
	PillarCoverage   map[string]int `json:"pillar_coverage"`
}

type PulseResult struct {
	Since string       `json:"since"`
	Team  []PulseEntry `json:"team"`
}

func (m *Manager) Pulse(ctx context.Context, opts PulseOptions) (PulseResult, error) {
	ctx = ctxOrBackground(ctx)
	sinceText := opts.Since
	if sinceText == "" {
		sinceText = "30d"
	}
	start, err := parseSince(sinceText)
	if err != nil {
		return PulseResult{}, err
	}
	end := now()
	weeks := math.Max(1, end.Sub(start).Hours()/24/7)

	entries := make([]PulseEntry, 0, len(m.Config.Team))
	for _, member := range m.Config.Team {
		activity, err := fetchActivity(ctx, member.Handle, start)
		if err != nil {
			return PulseResult{}, err
		}

		avgCycle := averageCycleDays(activity.MergedPRs)
		mergedPerWeek := float64(len(activity.MergedPRs)) / weeks
		blocked, err := github.SearchIssues(ctx, fmt.Sprintf("label:blocked is:open author:%s", member.Handle))
		if err != nil {
			return PulseResult{}, err
		}

		coverage := map[string]int{}
		for _, item := range append(activity.MergedPRs, activity.IssuesClosed...) {
			matches := m.Config.Pillars
			for _, match := range m.mapItem(item, matches) {
				coverage[match.Pillar]++
			}
		}

		entries = append(entries, PulseEntry{
			Handle:           member.Handle,
			MergedPerWeek:    round(mergedPerWeek, 2),
			AvgCycleDays:     round(avgCycle, 2),
			ReviewsCompleted: len(activity.Reviews),
			BlockedCount:     len(blocked),
			PillarCoverage:   coverage,
		})
	}

	return PulseResult{Since: sinceText, Team: entries}, nil
}

func averageCycleDays(items []pillars.ActivityItem) float64 {
	var total float64
	var count int
	for _, item := range items {
		if item.CreatedAt.IsZero() || item.ClosedAt.IsZero() {
			continue
		}
		count++
		total += item.ClosedAt.Sub(item.CreatedAt).Hours() / 24
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func round(val float64, precision int) float64 {
	pow := math.Pow(10, float64(precision))
	return math.Round(val*pow) / pow
}
