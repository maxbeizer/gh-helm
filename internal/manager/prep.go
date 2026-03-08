package manager

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/maxbeizer/gh-helm/internal/config"
	"github.com/maxbeizer/gh-helm/internal/github"
	"github.com/maxbeizer/gh-helm/internal/oneone"
	"github.com/maxbeizer/gh-helm/internal/pillars"
)

type PrepResult struct {
	Handle string `json:"handle"`
	Body   string `json:"body"`
}

func (m *Manager) Prep(ctx context.Context, opts PrepOptions) (PrepResult, error) {
	ctx = ctxOrBackground(ctx)
	sinceText := opts.Since
	if sinceText == "" {
		sinceText = "14d"
	}
	start, err := parseSince(sinceText)
	if err != nil {
		return PrepResult{}, err
	}
	end := now()
	member, err := m.memberByHandle(opts.Handle)
	if err != nil {
		return PrepResult{}, err
	}
	activity, err := fetchActivity(ctx, member.Handle, start)
	if err != nil {
		return PrepResult{}, err
	}

	openPRs := filterOpen(activity.OpenPRs)
	blocked, err := fetchBlocked(ctx, member.Handle)
	if err != nil {
		return PrepResult{}, err
	}

	matches := map[string][]pillars.PillarMatch{}
	for _, item := range append(activity.MergedPRs, activity.IssuesClosed...) {
		itemMatches := pillars.MapToPillars(item, m.Config.Pillars)
		for _, match := range itemMatches {
			matches[match.Pillar] = append(matches[match.Pillar], match)
		}
	}

	prevObservations := []oneone.ObservationIssue{}
	if member.OneOneRepo != "" {
		prevObservations, _ = oneone.FetchRecentObservations(ctx, member.OneOneRepo, 3)
	}

	body := buildPrepBody(member, activity, openPRs, blocked, matches, start, end, prevObservations, m.Config.Pillars)
	return PrepResult{Handle: member.Handle, Body: body}, nil
}

func (m *Manager) memberByHandle(handle string) (config.TeamMember, error) {
	for _, member := range m.Config.Team {
		if strings.EqualFold(member.Handle, handle) {
			return member, nil
		}
	}
	return config.TeamMember{}, fmt.Errorf("unknown handle: %s", handle)
}

func filterOpen(items []pillars.ActivityItem) []pillars.ActivityItem {
	return items
}

func fetchBlocked(ctx context.Context, handle string) ([]github.SearchItem, error) {
	return github.SearchIssues(ctx, fmt.Sprintf("label:blocked is:open author:%s", handle))
}

func buildPrepBody(member config.TeamMember, activity activityData, openPRs []pillars.ActivityItem, blocked []github.SearchItem, matches map[string][]pillars.PillarMatch, start, end time.Time, prev []oneone.ObservationIssue, pillarConfig map[string]config.PillarConfig) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("📋 1-1 Prep: @%s — %s\n\n", member.Handle, end.Format("Mon Jan 2, 2006")))

	b.WriteString("🏆 Wins\n")
	if len(activity.MergedPRs) == 0 {
		b.WriteString("  • No PRs merged yet\n")
	} else {
		for i, pr := range activity.MergedPRs {
			if i >= 2 {
				break
			}
			b.WriteString(fmt.Sprintf("  • %s\n", pr.Title))
		}
	}

	b.WriteString("\n🔵 Current Work\n")
	if len(openPRs) == 0 {
		b.WriteString("  • No open PRs\n")
	} else {
		for i, pr := range openPRs {
			if i >= 3 {
				break
			}
			b.WriteString(fmt.Sprintf("  • %s\n", pr.Title))
		}
	}

	b.WriteString("\n🚫 Blockers\n")
	if len(blocked) == 0 {
		b.WriteString("  • None reported\n")
	} else {
		for i, item := range blocked {
			if i >= 3 {
				break
			}
			b.WriteString(fmt.Sprintf("  • %s\n", item.Title))
		}
	}

	summary := buildPillarSummary(matches, member.Pillars, pillarConfig)
	b.WriteString("\n📊 Pillar Summary (")
	b.WriteString(fmt.Sprintf("%d days", int(end.Sub(start).Hours()/24)))
	b.WriteString(")\n")
	for _, line := range summary {
		b.WriteString("  • ")
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n🔄 From Previous Observations\n")
	if len(prev) == 0 {
		b.WriteString("  • No recent notes\n")
	} else {
		for i, item := range prev {
			if i >= 3 {
				break
			}
			b.WriteString(fmt.Sprintf("  • %s\n", item.Title))
		}
	}

	b.WriteString("\n💬 Suggested Topics\n")
	if len(activity.MergedPRs) > 0 {
		b.WriteString("  • Acknowledge recent wins\n")
	}
	missing := missingPillars(member.Pillars, matches)
	for _, pillar := range missing {
		b.WriteString(fmt.Sprintf("  • Check in on %s focus\n", pillar))
	}
	if len(missing) == 0 && len(activity.MergedPRs) == 0 {
		b.WriteString("  • Review current priorities\n")
	}

	return b.String()
}

func buildPillarSummary(matches map[string][]pillars.PillarMatch, assigned []string, pillarConfig map[string]config.PillarConfig) []string {
	lines := []string{}
	if len(assigned) == 0 {
		for key := range pillarConfig {
			assigned = append(assigned, key)
		}
	}
	for _, pillar := range assigned {
		count := len(matches[pillar])
		if count == 0 {
			lines = append(lines, fmt.Sprintf("%s: no activity ⚠️", titleCase(pillar)))
		} else {
			lines = append(lines, fmt.Sprintf("%s: %d contributions", titleCase(pillar), count))
		}
	}
	sort.Strings(lines)
	return lines
}
