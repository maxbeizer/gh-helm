package manager

import (
	"strings"
	"testing"
	"time"

	"github.com/maxbeizer/gh-helm/internal/config"
	"github.com/maxbeizer/gh-helm/internal/github"
	"github.com/maxbeizer/gh-helm/internal/pillars"
)

func TestTitleCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"code-quality", "Code Quality"},
		{"velocity", "Velocity"},
		{"team-health-metrics", "Team Health Metrics"},
		{"", ""},
		{"a", "A"},
		{"already-Capitalized", "Already Capitalized"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := titleCase(tc.input)
			if got != tc.want {
				t.Errorf("titleCase(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestSortedKeys(t *testing.T) {
	tests := []struct {
		name string
		in   map[string][]pillars.PillarMatch
		want []string
	}{
		{
			name: "empty map",
			in:   map[string][]pillars.PillarMatch{},
			want: []string{},
		},
		{
			name: "sorted output",
			in: map[string][]pillars.PillarMatch{
				"zebra": {},
				"alpha": {},
				"mango": {},
			},
			want: []string{"alpha", "mango", "zebra"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sortedKeys(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("got %d keys, want %d", len(got), len(tc.want))
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("key[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestMissingPillars(t *testing.T) {
	tests := []struct {
		name     string
		assigned []string
		matches  map[string][]pillars.PillarMatch
		want     []string
	}{
		{
			name:     "no assigned pillars",
			assigned: []string{},
			matches:  map[string][]pillars.PillarMatch{},
			want:     []string{},
		},
		{
			name:     "all present",
			assigned: []string{"velocity", "quality"},
			matches: map[string][]pillars.PillarMatch{
				"velocity": {{Pillar: "velocity"}},
				"quality":  {{Pillar: "quality"}},
			},
			want: []string{},
		},
		{
			name:     "some missing",
			assigned: []string{"velocity", "quality", "impact"},
			matches: map[string][]pillars.PillarMatch{
				"velocity": {{Pillar: "velocity"}},
			},
			want: []string{"impact", "quality"},
		},
		{
			name:     "all missing",
			assigned: []string{"velocity", "quality"},
			matches:  map[string][]pillars.PillarMatch{},
			want:     []string{"quality", "velocity"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := missingPillars(tc.assigned, tc.matches)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestSummarizeSince(t *testing.T) {
	tests := []struct {
		name  string
		start time.Time
		end   time.Time
		want  string
	}{
		{
			name:  "7 days",
			start: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2024, 6, 8, 0, 0, 0, 0, time.UTC),
			want:  "7 days",
		},
		{
			name:  "zero or negative",
			start: time.Date(2024, 6, 8, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2024, 6, 8, 0, 0, 0, 0, time.UTC),
			want:  "the period",
		},
		{
			name:  "30 days",
			start: time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2024, 5, 31, 0, 0, 0, 0, time.UTC),
			want:  "30 days",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := summarizeSince(tc.start, tc.end)
			if got != tc.want {
				t.Errorf("summarizeSince() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseGitHubTime(t *testing.T) {
	tests := []struct {
		name  string
		input string
		isZero bool
		year  int
	}{
		{"empty", "", true, 0},
		{"invalid", "not-a-date", true, 0},
		{"valid rfc3339", "2024-06-15T10:30:00Z", false, 2024},
		{"valid with offset", "2024-01-01T00:00:00+05:00", false, 2024},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseGitHubTime(tc.input)
			if tc.isZero && !got.IsZero() {
				t.Errorf("expected zero time, got %v", got)
			}
			if !tc.isZero {
				if got.IsZero() {
					t.Fatal("expected non-zero time")
				}
				if got.Year() != tc.year {
					t.Errorf("year = %d, want %d", got.Year(), tc.year)
				}
			}
		})
	}
}

func TestBuildObservationBody(t *testing.T) {
	start := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 6, 8, 0, 0, 0, 0, time.UTC)
	member := config.TeamMember{Handle: "alice", Pillars: []string{"velocity", "quality"}}

	t.Run("empty activity", func(t *testing.T) {
		activity := activityData{}
		matches := map[string][]pillars.PillarMatch{}
		body := buildObservationBody(member, activity, matches, start, end, nil)

		if !strings.Contains(body, "0 PRs merged") {
			t.Error("expected '0 PRs merged'")
		}
		if !strings.Contains(body, "No PRs merged in this period") {
			t.Error("expected no PRs merged observation")
		}
		if !strings.Contains(body, "quality") {
			t.Error("expected missing pillar 'quality'")
		}
		if !strings.Contains(body, "velocity") {
			t.Error("expected missing pillar 'velocity'")
		}
	})

	t.Run("with activity and pillar matches", func(t *testing.T) {
		activity := activityData{
			MergedPRs: []pillars.ActivityItem{
				{Title: "Fix tests"},
				{Title: "Add feature"},
			},
			Reviews:      []github.SearchItem{{Title: "Review 1"}},
			IssuesOpened: []pillars.ActivityItem{{Title: "Bug report"}},
			IssuesClosed: []pillars.ActivityItem{{Title: "Closed bug"}},
		}
		matches := map[string][]pillars.PillarMatch{
			"velocity": {{Pillar: "velocity", Reason: "merged PR", Item: pillars.ActivityItem{Title: "Fix tests"}}},
			"quality":  {{Pillar: "quality", Reason: "test coverage", Item: pillars.ActivityItem{Title: "Add feature"}}},
		}
		body := buildObservationBody(member, activity, matches, start, end, nil)

		if !strings.Contains(body, "2 PRs merged") {
			t.Error("expected '2 PRs merged'")
		}
		if !strings.Contains(body, "1 reviews completed") {
			t.Error("expected '1 reviews completed'")
		}
		if !strings.Contains(body, "Pillar Impact: Velocity") {
			t.Error("expected velocity pillar section")
		}
		if !strings.Contains(body, "Balanced contribution") {
			t.Error("expected balanced observation when all pillars covered")
		}
		if !strings.Contains(body, "Acknowledge recent wins") {
			t.Error("expected suggested topic for wins")
		}
	})

	t.Run("pillar items limited to 3", func(t *testing.T) {
		items := make([]pillars.PillarMatch, 5)
		for i := range items {
			items[i] = pillars.PillarMatch{Pillar: "velocity", Reason: "reason", Item: pillars.ActivityItem{Title: "item"}}
		}
		matches := map[string][]pillars.PillarMatch{"velocity": items}
		activity := activityData{MergedPRs: []pillars.ActivityItem{{Title: "PR"}}}
		body := buildObservationBody(member, activity, matches, start, end, nil)

		// Count occurrences of "item — reason"
		count := strings.Count(body, "item — reason")
		if count != 3 {
			t.Errorf("expected 3 pillar items shown, got %d", count)
		}
	})
}

func TestMemberByHandle(t *testing.T) {
	m := &Manager{
		Config: config.ManagerConfig{
			Team: []config.TeamMember{
				{Handle: "alice"},
				{Handle: "Bob"},
			},
		},
	}

	tests := []struct {
		name    string
		handle  string
		want    string
		wantErr bool
	}{
		{"exact match", "alice", "alice", false},
		{"case insensitive", "ALICE", "alice", false},
		{"case insensitive mixed", "bob", "Bob", false},
		{"not found", "charlie", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := m.memberByHandle(tc.handle)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Handle != tc.want {
				t.Errorf("Handle = %q, want %q", got.Handle, tc.want)
			}
		})
	}
}

func TestFilterOpen(t *testing.T) {
	items := []pillars.ActivityItem{{Title: "a"}, {Title: "b"}}
	got := filterOpen(items)
	if len(got) != 2 {
		t.Errorf("filterOpen returned %d items, want 2", len(got))
	}
}
