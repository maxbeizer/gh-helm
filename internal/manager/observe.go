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

type ObservationResult struct {
	Handle string `json:"handle"`
	Repo   string `json:"repo"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

type activityData struct {
	MergedPRs   []pillars.ActivityItem
	OpenPRs     []pillars.ActivityItem
	IssuesClosed []pillars.ActivityItem
	IssuesOpened []pillars.ActivityItem
	Reviews     []github.SearchItem
}

func (m *Manager) Observe(ctx context.Context, opts ObserveOptions) ([]ObservationResult, error) {
	ctx = ctxOrBackground(ctx)
	sinceText := opts.Since
	if sinceText == "" {
		sinceText = "7d"
	}
	start, err := parseSince(sinceText)
	if err != nil {
		return nil, err
	}
	end := now()
	cfg := m.Config

	members := cfg.Team
	if opts.Handle != "" {
		filtered := make([]config.TeamMember, 0, 1)
		for _, member := range members {
			if strings.EqualFold(member.Handle, opts.Handle) {
				filtered = append(filtered, member)
			}
		}
		members = filtered
	}

	results := make([]ObservationResult, 0, len(members))
	for _, member := range members {
		activity, err := fetchActivity(ctx, member.Handle, start)
		if err != nil {
			return nil, err
		}

		matches := map[string][]pillars.PillarMatch{}
		for _, item := range append(activity.MergedPRs, activity.IssuesClosed...) {
			itemMatches := pillars.MapToPillars(item, cfg.Pillars)
			for _, match := range itemMatches {
				matches[match.Pillar] = append(matches[match.Pillar], match)
			}
		}

		body := buildObservationBody(member, activity, matches, start, end, cfg.Pillars)
		title := fmt.Sprintf("📊 Week of %s", formatRange(start, end))
		result := ObservationResult{Handle: member.Handle, Repo: member.OneOneRepo, Title: title, Body: body}

		if !opts.DryRun && member.OneOneRepo != "" {
			if err := oneone.PostObservation(ctx, member.Handle, member.OneOneRepo, title, body); err != nil {
				return nil, err
			}
		}
		results = append(results, result)
	}

	return results, nil
}

func fetchActivity(ctx context.Context, handle string, since time.Time) (activityData, error) {
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

	mergedItems, err := toActivityItems(ctx, mergedPRs, true)
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

func toActivityItems(ctx context.Context, items []github.SearchItem, fetchFiles bool) ([]pillars.ActivityItem, error) {
	results := make([]pillars.ActivityItem, 0, len(items))
	for _, item := range items {
		labels := make([]string, 0, len(item.Labels))
		for _, label := range item.Labels {
			labels = append(labels, label.Name)
		}
		repo := github.RepoFromURL(item.RepositoryURL)
		activity := pillars.ActivityItem{
			Title:     item.Title,
			Body:      item.Body,
			Labels:    labels,
			Repo:      repo,
			URL:       item.HTMLURL,
			Number:    item.Number,
			CreatedAt: parseGitHubTime(item.CreatedAt),
			ClosedAt:  parseGitHubTime(item.ClosedAt),
		}
		if fetchFiles && repo != "" {
			files, err := github.PullFiles(ctx, repo, item.Number)
			if err != nil {
				return nil, err
			}
			activity.Files = files
		}
		results = append(results, activity)
	}
	return results, nil
}

func buildObservationBody(member config.TeamMember, activity activityData, matches map[string][]pillars.PillarMatch, start, end time.Time, pillarConfig map[string]config.PillarConfig) string {
	var b strings.Builder
	b.WriteString("## 📊 Week of ")
	b.WriteString(formatRange(start, end))
	b.WriteString("\n\n### Activity\n")
	b.WriteString(fmt.Sprintf("- %d PRs merged\n", len(activity.MergedPRs)))
	b.WriteString(fmt.Sprintf("- %d reviews completed\n", len(activity.Reviews)))
	b.WriteString(fmt.Sprintf("- %d issues opened, %d closed\n", len(activity.IssuesOpened), len(activity.IssuesClosed)))

	pillarOrder := sortedKeys(matches)
	for _, pillar := range pillarOrder {
		items := matches[pillar]
		if len(items) == 0 {
			continue
		}
		b.WriteString("\n### Pillar Impact: ")
		b.WriteString(titleCase(pillar))
		b.WriteString("\n")
		for i, item := range items {
			if i >= 3 {
				break
			}
			b.WriteString(fmt.Sprintf("- %s — %s\n", item.Item.Title, item.Reason))
		}
	}

	assigned := member.Pillars
	if len(assigned) == 0 {
		for key := range pillarConfig {
			assigned = append(assigned, key)
		}
	}
	missing := missingPillars(assigned, matches)
	b.WriteString("\n### Observations\n")
	if len(activity.MergedPRs) == 0 {
		b.WriteString("- No PRs merged in this period\n")
	}
	for _, pillar := range missing {
		b.WriteString(fmt.Sprintf("- No %s-related work in %s\n", pillar, summarizeSince(start, end)))
	}
	if len(missing) == 0 {
		b.WriteString("- Balanced contribution across assigned pillars\n")
	}

	b.WriteString("\n### Suggested 1-1 Topics\n")
	if len(activity.MergedPRs) > 0 {
		b.WriteString("- Acknowledge recent wins\n")
	}
	for _, pillar := range missing {
		b.WriteString(fmt.Sprintf("- Check in on %s focus\n", pillar))
	}
	if len(activity.MergedPRs) == 0 && len(missing) == 0 {
		b.WriteString("- Review current priorities\n")
	}

	return b.String()
}

func sortedKeys(matches map[string][]pillars.PillarMatch) []string {
	keys := make([]string, 0, len(matches))
	for key := range matches {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func titleCase(input string) string {
	parts := strings.Split(input, "-")
	for i, part := range parts {
		if len(part) == 0 {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func missingPillars(assigned []string, matches map[string][]pillars.PillarMatch) []string {
	missing := []string{}
	for _, pillar := range assigned {
		if len(matches[pillar]) == 0 {
			missing = append(missing, pillar)
		}
	}
	sort.Strings(missing)
	return missing
}

func summarizeSince(start, end time.Time) string {
	days := int(end.Sub(start).Hours() / 24)
	if days <= 0 {
		return "the period"
	}
	return fmt.Sprintf("%d days", days)
}

func parseGitHubTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return t
}
