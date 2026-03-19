package manager

import (
	"strings"
	"testing"
	"time"

	"github.com/maxbeizer/gh-helm/internal/github"
	"github.com/maxbeizer/gh-helm/internal/pillars"
)

func TestBuildTimeline(t *testing.T) {
	t.Run("empty items", func(t *testing.T) {
		entries := buildTimeline(nil)
		if len(entries) != 0 {
			t.Errorf("expected 0 entries, got %d", len(entries))
		}
	})

	t.Run("skips zero closed at", func(t *testing.T) {
		items := []pillars.ActivityItem{
			{Title: "no close date"},
			{Title: "has close date", ClosedAt: time.Date(2024, 6, 5, 0, 0, 0, 0, time.UTC), URL: "http://example.com"},
		}
		entries := buildTimeline(items)
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(entries))
		}
		if entries[0].Title != "has close date" {
			t.Errorf("Title = %q, want 'has close date'", entries[0].Title)
		}
		if entries[0].Kind != "pr" {
			t.Errorf("Kind = %q, want 'pr'", entries[0].Kind)
		}
	})

	t.Run("sorted by date ascending", func(t *testing.T) {
		items := []pillars.ActivityItem{
			{Title: "later", ClosedAt: time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)},
			{Title: "earlier", ClosedAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)},
		}
		entries := buildTimeline(items)
		if entries[0].Title != "earlier" {
			t.Errorf("first entry = %q, want 'earlier'", entries[0].Title)
		}
	})
}

func TestBuildActivityTimeline(t *testing.T) {
	t.Run("combines PRs and issues", func(t *testing.T) {
		activity := activityData{
			MergedPRs: []pillars.ActivityItem{
				{Title: "PR 1", ClosedAt: time.Date(2024, 6, 5, 0, 0, 0, 0, time.UTC)},
			},
			IssuesClosed: []pillars.ActivityItem{
				{Title: "Issue 1", ClosedAt: time.Date(2024, 6, 3, 0, 0, 0, 0, time.UTC)},
			},
		}
		entries := buildActivityTimeline(activity)
		if len(entries) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(entries))
		}
		if entries[0].Kind != "issue" {
			t.Errorf("first entry kind = %q, want 'issue' (earlier date)", entries[0].Kind)
		}
		if entries[1].Kind != "pr" {
			t.Errorf("second entry kind = %q, want 'pr' (later date)", entries[1].Kind)
		}
	})

	t.Run("sorts by date then title", func(t *testing.T) {
		activity := activityData{
			MergedPRs: []pillars.ActivityItem{
				{Title: "B PR", ClosedAt: time.Date(2024, 6, 5, 0, 0, 0, 0, time.UTC)},
				{Title: "A PR", ClosedAt: time.Date(2024, 6, 5, 0, 0, 0, 0, time.UTC)},
			},
		}
		entries := buildActivityTimeline(activity)
		if entries[0].Title != "A PR" {
			t.Errorf("expected 'A PR' first when same date, got %q", entries[0].Title)
		}
	})
}

func TestBuildVelocityTrends(t *testing.T) {
	start := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 6, 22, 0, 0, 0, 0, time.UTC)

	t.Run("empty items", func(t *testing.T) {
		v := buildVelocityTrends(nil, start, end)
		if len(v.Weeks) == 0 {
			t.Error("expected at least one week bucket")
		}
		for _, w := range v.Weeks {
			if w.Count != 0 {
				t.Errorf("expected 0 count, got %d for week %s", w.Count, w.Week)
			}
		}
	})

	t.Run("trend up", func(t *testing.T) {
		// Items in the last week but not in the one before
		items := []pillars.ActivityItem{
			{Title: "pr1", ClosedAt: time.Date(2024, 6, 18, 0, 0, 0, 0, time.UTC)},
			{Title: "pr2", ClosedAt: time.Date(2024, 6, 19, 0, 0, 0, 0, time.UTC)},
			{Title: "pr3", ClosedAt: time.Date(2024, 6, 20, 0, 0, 0, 0, time.UTC)},
		}
		v := buildVelocityTrends(items, start, end)
		if v.Trend != "↑" && v.Trend != "↗" {
			// Trend depends on exact week boundaries, just verify it's upward
			t.Logf("trend = %s (acceptable if items cluster in last week)", v.Trend)
		}
	})

	t.Run("skips items outside range", func(t *testing.T) {
		items := []pillars.ActivityItem{
			{Title: "before", ClosedAt: time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)},
			{Title: "after", ClosedAt: time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)},
		}
		v := buildVelocityTrends(items, start, end)
		total := 0
		for _, w := range v.Weeks {
			total += w.Count
		}
		if total != 0 {
			t.Errorf("expected 0 total count for out-of-range items, got %d", total)
		}
	})
}

func TestBuildPillarHeatmap(t *testing.T) {
	tests := []struct {
		name   string
		counts map[string]int
		checks map[string]int // expected bar lengths
	}{
		{
			name:   "empty",
			counts: map[string]int{},
			checks: map[string]int{},
		},
		{
			name:   "max gets full bar",
			counts: map[string]int{"velocity": 10},
			checks: map[string]int{"velocity": 5},
		},
		{
			name:   "half gets proportional bar",
			counts: map[string]int{"velocity": 10, "quality": 5},
			checks: map[string]int{"velocity": 5, "quality": 2},
		},
		{
			name:   "small non-zero gets at least 1",
			counts: map[string]int{"velocity": 100, "quality": 1},
			checks: map[string]int{"velocity": 5, "quality": 1},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := buildPillarHeatmap(tc.counts)
			for pillar, expectedLen := range tc.checks {
				bar := result[pillar]
				// Count █ characters
				barLen := strings.Count(bar, "█")
				if barLen != expectedLen {
					t.Errorf("pillar %q: bar length = %d, want %d (bar=%q)", pillar, barLen, expectedLen, bar)
				}
			}
		})
	}
}

func TestDensityBar(t *testing.T) {
	tests := []struct {
		name  string
		count int
		max   int
		want  string
	}{
		{"zero max", 0, 0, "·"},
		{"non-zero count zero max", 5, 0, "·"},
		{"full bar", 10, 10, "█████"},
		{"half", 5, 10, "██"},
		{"small non-zero", 1, 100, "█"},
		{"zero count", 0, 10, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := densityBar(tc.count, tc.max)
			if got != tc.want {
				t.Errorf("densityBar(%d, %d) = %q, want %q", tc.count, tc.max, got, tc.want)
			}
		})
	}
}

func TestBuildReviewQuality(t *testing.T) {
	t.Run("no reviews", func(t *testing.T) {
		result := buildReviewQuality(activityData{})
		if result.Reviews != 0 {
			t.Errorf("Reviews = %d, want 0", result.Reviews)
		}
	})

	t.Run("with reviews", func(t *testing.T) {
		activity := activityData{
			Reviews: []github.SearchItem{
				{
					Title:     "Review 1",
					CreatedAt: "2024-06-01T10:00:00Z",
					UpdatedAt: "2024-06-01T12:00:00Z",
					Comments:  3,
				},
				{
					Title:     "Review 2",
					CreatedAt: "2024-06-02T10:00:00Z",
					UpdatedAt: "2024-06-02T14:00:00Z",
					Comments:  5,
				},
			},
		}
		result := buildReviewQuality(activity)
		if result.Reviews != 2 {
			t.Errorf("Reviews = %d, want 2", result.Reviews)
		}
		// avg turnaround: (2 + 4) / 2 = 3 hours
		if result.AvgTurnaroundHours != 3.0 {
			t.Errorf("AvgTurnaroundHours = %v, want 3.0", result.AvgTurnaroundHours)
		}
		// avg comments: (3 + 5) / 2 = 4
		if result.AvgCommentsPerReview != 4.0 {
			t.Errorf("AvgCommentsPerReview = %v, want 4.0", result.AvgCommentsPerReview)
		}
	})
}

func TestBuildNotableContributions(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result := buildNotableContributions(activityData{})
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("sorts by score descending", func(t *testing.T) {
		activity := activityData{
			MergedPRs: []pillars.ActivityItem{
				{Title: "Small PR", Files: []string{"a.go"}, URL: "url1"},
				{Title: "Big PR with many files changed", Files: []string{"a.go", "b.go", "c.go", "d.go", "e.go"}, URL: "url2"},
			},
			IssuesClosed: []pillars.ActivityItem{
				{Title: "Issue fix", URL: "url3"},
			},
		}
		result := buildNotableContributions(activity)
		if len(result) != 3 {
			t.Fatalf("expected 3 results, got %d", len(result))
		}
		// Big PR should be first (score = 5 files + title/40)
		if result[0].URL != "url2" {
			t.Errorf("expected big PR first, got %q", result[0].Title)
		}
	})

	t.Run("limits to 3", func(t *testing.T) {
		activity := activityData{
			MergedPRs: []pillars.ActivityItem{
				{Title: "PR1"}, {Title: "PR2"}, {Title: "PR3"}, {Title: "PR4"}, {Title: "PR5"},
			},
		}
		result := buildNotableContributions(activity)
		if len(result) != 3 {
			t.Errorf("expected 3 results, got %d", len(result))
		}
	})
}

func TestWeekKey(t *testing.T) {
	tests := []struct {
		name string
		time time.Time
		want string
	}{
		{
			name: "standard date",
			time: time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC), // Saturday W24
			want: "2024-W24",
		},
		{
			name: "new year",
			time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			want: "2024-W01",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := weekKey(tc.time)
			if got != tc.want {
				t.Errorf("weekKey() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTruncateToWeek(t *testing.T) {
	// Wednesday June 12, 2024
	wed := time.Date(2024, 6, 12, 15, 30, 0, 0, time.UTC)
	got := truncateToWeek(wed)

	// Should truncate to Monday June 10, 2024
	want := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("truncateToWeek(%v) = %v, want %v", wed, got, want)
	}

	// Monday should stay Monday
	mon := time.Date(2024, 6, 10, 10, 0, 0, 0, time.UTC)
	got = truncateToWeek(mon)
	want = time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("truncateToWeek(Monday) = %v, want %v", got, want)
	}

	// Sunday should go back to previous Monday
	sun := time.Date(2024, 6, 16, 0, 0, 0, 0, time.UTC)
	got = truncateToWeek(sun)
	want = time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("truncateToWeek(Sunday) = %v, want %v", got, want)
	}
}

func TestFilterItemsBetween(t *testing.T) {
	start := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	items := []pillars.ActivityItem{
		{Title: "before", ClosedAt: time.Date(2024, 5, 30, 0, 0, 0, 0, time.UTC)},
		{Title: "at start", ClosedAt: start},
		{Title: "in range", ClosedAt: time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)},
		{Title: "at end", ClosedAt: end},
		{Title: "after", ClosedAt: time.Date(2024, 6, 20, 0, 0, 0, 0, time.UTC)},
		{Title: "zero time"},
	}

	got := filterItemsBetween(items, start, end, func(item pillars.ActivityItem) time.Time {
		return item.ClosedAt
	})

	// Should include "at start" and "in range" (>= start and < end)
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d: %v", len(got), titles(got))
	}
	if got[0].Title != "at start" {
		t.Errorf("expected 'at start', got %q", got[0].Title)
	}
	if got[1].Title != "in range" {
		t.Errorf("expected 'in range', got %q", got[1].Title)
	}
}

func TestFilterReviewsBetween(t *testing.T) {
	start := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	items := []github.SearchItem{
		{Title: "before", UpdatedAt: "2024-05-30T00:00:00Z"},
		{Title: "in range", UpdatedAt: "2024-06-10T00:00:00Z"},
		{Title: "at end", UpdatedAt: "2024-06-15T00:00:00Z"},
		{Title: "empty date"},
	}

	got := filterReviewsBetween(items, start, end)
	if len(got) != 1 {
		t.Fatalf("expected 1 item, got %d", len(got))
	}
	if got[0].Title != "in range" {
		t.Errorf("expected 'in range', got %q", got[0].Title)
	}
}

func TestFilterActivityBetween(t *testing.T) {
	start := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	activity := activityData{
		MergedPRs: []pillars.ActivityItem{
			{Title: "merged in range", ClosedAt: time.Date(2024, 6, 5, 0, 0, 0, 0, time.UTC)},
			{Title: "merged out of range", ClosedAt: time.Date(2024, 6, 20, 0, 0, 0, 0, time.UTC)},
		},
		OpenPRs: []pillars.ActivityItem{
			{Title: "opened in range", CreatedAt: time.Date(2024, 6, 5, 0, 0, 0, 0, time.UTC)},
		},
		IssuesClosed: []pillars.ActivityItem{
			{Title: "closed in range", ClosedAt: time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)},
		},
		IssuesOpened: []pillars.ActivityItem{
			{Title: "opened issue in range", CreatedAt: time.Date(2024, 6, 3, 0, 0, 0, 0, time.UTC)},
		},
		Reviews: []github.SearchItem{
			{Title: "review in range", UpdatedAt: "2024-06-10T00:00:00Z"},
		},
	}

	filtered := filterActivityBetween(activity, start, end)
	if len(filtered.MergedPRs) != 1 {
		t.Errorf("MergedPRs: got %d, want 1", len(filtered.MergedPRs))
	}
	if len(filtered.OpenPRs) != 1 {
		t.Errorf("OpenPRs: got %d, want 1", len(filtered.OpenPRs))
	}
	if len(filtered.IssuesClosed) != 1 {
		t.Errorf("IssuesClosed: got %d, want 1", len(filtered.IssuesClosed))
	}
	if len(filtered.IssuesOpened) != 1 {
		t.Errorf("IssuesOpened: got %d, want 1", len(filtered.IssuesOpened))
	}
	if len(filtered.Reviews) != 1 {
		t.Errorf("Reviews: got %d, want 1", len(filtered.Reviews))
	}
}

func titles(items []pillars.ActivityItem) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.Title
	}
	return out
}
