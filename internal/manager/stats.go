package manager

import (
	"context"
	"fmt"
	"sort"

	"github.com/maxbeizer/gh-helm/internal/config"
)

type StatsOptions struct {
	Since  string
	Handle string // if empty, show team-wide stats
}

type PersonStats struct {
	Handle              string         `json:"handle"`
	PRsMerged           int            `json:"prs_merged"`
	PRsOpened           int            `json:"prs_opened"`
	IssuesClosed        int            `json:"issues_closed"`
	IssuesOpened        int            `json:"issues_opened"`
	ReviewsCompleted    int            `json:"reviews_completed"`
	AvgCycleDays        float64        `json:"avg_cycle_days"`
	AvgReviewTurnaround float64        `json:"avg_review_turnaround_hours"`
	PillarCoverage      map[string]int `json:"pillar_coverage"`
}

type TeamStats struct {
	Since             string           `json:"since"`
	TotalPRsMerged    int              `json:"total_prs_merged"`
	TotalIssuesClosed int              `json:"total_issues_closed"`
	TotalReviews      int              `json:"total_reviews"`
	AvgCycleDays      float64          `json:"avg_cycle_days"`
	PillarCoverage    map[string]int   `json:"pillar_coverage"`
	Members           []PersonStats    `json:"members"`
	BusFactor         []BusFactorEntry `json:"bus_factor"`
}

type BusFactorEntry struct {
	Area         string   `json:"area"`
	Contributors []string `json:"contributors"`
	Risk         string   `json:"risk"` // "high" (1 person), "medium" (2), "low" (3+)
}

func (m *Manager) Stats(ctx context.Context, opts StatsOptions) (TeamStats, error) {
	ctx = ctxOrBackground(ctx)
	sinceText := opts.Since
	if sinceText == "" {
		sinceText = "30d"
	}
	start, err := parseSince(sinceText)
	if err != nil {
		return TeamStats{}, err
	}

	members := m.Config.Team
	if opts.Handle != "" {
		member, err := m.memberByHandle(opts.Handle)
		if err != nil {
			return TeamStats{}, err
		}
		members = []config.TeamMember{member}
	}

	result := TeamStats{
		Since:          sinceText,
		PillarCoverage: map[string]int{},
		Members:        make([]PersonStats, 0, len(members)),
	}

	// Track which repos each person touches for bus factor
	repoContributors := map[string]map[string]bool{}

	for _, member := range members {
		activity, err := fetchActivity(ctx, member.Handle, start)
		if err != nil {
			return TeamStats{}, fmt.Errorf("fetch activity for %s: %w", member.Handle, err)
		}

		avgCycle := averageCycleDays(activity.MergedPRs)
		reviewQuality := buildReviewQuality(activity)

		coverage := map[string]int{}
		for _, item := range append(activity.MergedPRs, activity.IssuesClosed...) {
			for _, match := range m.mapItem(item, m.Config.Pillars) {
				coverage[match.Pillar]++
				result.PillarCoverage[match.Pillar]++
			}
			if item.Repo != "" {
				if repoContributors[item.Repo] == nil {
					repoContributors[item.Repo] = map[string]bool{}
				}
				repoContributors[item.Repo][member.Handle] = true
			}
		}

		ps := PersonStats{
			Handle:              member.Handle,
			PRsMerged:           len(activity.MergedPRs),
			PRsOpened:           len(activity.OpenPRs),
			IssuesClosed:        len(activity.IssuesClosed),
			IssuesOpened:        len(activity.IssuesOpened),
			ReviewsCompleted:    len(activity.Reviews),
			AvgCycleDays:        round(avgCycle, 2),
			AvgReviewTurnaround: round(reviewQuality.AvgTurnaroundHours, 2),
			PillarCoverage:      coverage,
		}

		result.TotalPRsMerged += ps.PRsMerged
		result.TotalIssuesClosed += ps.IssuesClosed
		result.TotalReviews += ps.ReviewsCompleted
		result.Members = append(result.Members, ps)
	}

	// Calculate team average cycle time
	var totalCycle float64
	var cycleCount int
	for _, m := range result.Members {
		if m.AvgCycleDays > 0 {
			totalCycle += m.AvgCycleDays
			cycleCount++
		}
	}
	if cycleCount > 0 {
		result.AvgCycleDays = round(totalCycle/float64(cycleCount), 2)
	}

	result.BusFactor = buildBusFactor(repoContributors)

	// Sort members by PRs merged descending
	sort.Slice(result.Members, func(i, j int) bool {
		return result.Members[i].PRsMerged > result.Members[j].PRsMerged
	})

	return result, nil
}

func buildBusFactor(repoContributors map[string]map[string]bool) []BusFactorEntry {
	entries := []BusFactorEntry{}
	for repo, contributors := range repoContributors {
		names := make([]string, 0, len(contributors))
		for name := range contributors {
			names = append(names, name)
		}
		sort.Strings(names)

		risk := "low"
		if len(names) == 1 {
			risk = "high"
		} else if len(names) == 2 {
			risk = "medium"
		}

		entries = append(entries, BusFactorEntry{
			Area:         repo,
			Contributors: names,
			Risk:         risk,
		})
	}
	// Sort: high risk first
	sort.Slice(entries, func(i, j int) bool {
		riskOrder := map[string]int{"high": 0, "medium": 1, "low": 2}
		return riskOrder[entries[i].Risk] < riskOrder[entries[j].Risk]
	})
	return entries
}
