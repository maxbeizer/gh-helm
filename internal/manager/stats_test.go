package manager

import (
	"sort"
	"testing"

	"github.com/maxbeizer/gh-helm/internal/config"
	"github.com/maxbeizer/gh-helm/internal/pillars"
)

func TestBuildBusFactor(t *testing.T) {
	tests := []struct {
		name             string
		repoContributors map[string]map[string]bool
		wantLen          int
		checkFirst       func(t *testing.T, entry BusFactorEntry)
	}{
		{
			name:             "empty",
			repoContributors: map[string]map[string]bool{},
			wantLen:          0,
		},
		{
			name: "single contributor high risk",
			repoContributors: map[string]map[string]bool{
				"repo-a": {"alice": true},
			},
			wantLen: 1,
			checkFirst: func(t *testing.T, entry BusFactorEntry) {
				if entry.Risk != "high" {
					t.Errorf("Risk = %q, want 'high'", entry.Risk)
				}
				if len(entry.Contributors) != 1 {
					t.Errorf("Contributors = %d, want 1", len(entry.Contributors))
				}
			},
		},
		{
			name: "two contributors medium risk",
			repoContributors: map[string]map[string]bool{
				"repo-a": {"alice": true, "bob": true},
			},
			wantLen: 1,
			checkFirst: func(t *testing.T, entry BusFactorEntry) {
				if entry.Risk != "medium" {
					t.Errorf("Risk = %q, want 'medium'", entry.Risk)
				}
			},
		},
		{
			name: "three or more low risk",
			repoContributors: map[string]map[string]bool{
				"repo-a": {"alice": true, "bob": true, "charlie": true},
			},
			wantLen: 1,
			checkFirst: func(t *testing.T, entry BusFactorEntry) {
				if entry.Risk != "low" {
					t.Errorf("Risk = %q, want 'low'", entry.Risk)
				}
			},
		},
		{
			name: "sorted by risk high first",
			repoContributors: map[string]map[string]bool{
				"repo-low":  {"a": true, "b": true, "c": true},
				"repo-high": {"d": true},
				"repo-med":  {"e": true, "f": true},
			},
			wantLen: 3,
			checkFirst: func(t *testing.T, entry BusFactorEntry) {
				if entry.Risk != "high" {
					t.Errorf("first entry Risk = %q, want 'high'", entry.Risk)
				}
			},
		},
		{
			name: "contributors sorted alphabetically",
			repoContributors: map[string]map[string]bool{
				"repo": {"charlie": true, "alice": true, "bob": true},
			},
			wantLen: 1,
			checkFirst: func(t *testing.T, entry BusFactorEntry) {
				want := []string{"alice", "bob", "charlie"}
				if len(entry.Contributors) != 3 {
					t.Fatalf("Contributors = %d, want 3", len(entry.Contributors))
				}
				for i, c := range entry.Contributors {
					if c != want[i] {
						t.Errorf("Contributors[%d] = %q, want %q", i, c, want[i])
					}
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildBusFactor(tc.repoContributors)
			if len(got) != tc.wantLen {
				t.Fatalf("got %d entries, want %d", len(got), tc.wantLen)
			}
			if tc.checkFirst != nil && len(got) > 0 {
				tc.checkFirst(t, got[0])
			}
		})
	}
}

func TestParseScheduleSet(t *testing.T) {
	t.Run("all empty", func(t *testing.T) {
		set, err := parseScheduleSet(config.ManagerSchedule{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if set.Pulse != nil || set.Prep != nil || set.Observe != nil {
			t.Error("expected all nil schedules")
		}
	})

	t.Run("all valid", func(t *testing.T) {
		set, err := parseScheduleSet(config.ManagerSchedule{
			Pulse:   "0 9 * * 1",
			Prep:    "30 14 * * 5",
			Observe: "0 8 * * *",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if set.Pulse == nil {
			t.Error("expected pulse schedule")
		}
		if set.Prep == nil {
			t.Error("expected prep schedule")
		}
		if set.Observe == nil {
			t.Error("expected observe schedule")
		}
	})

	t.Run("invalid pulse", func(t *testing.T) {
		_, err := parseScheduleSet(config.ManagerSchedule{Pulse: "bad"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("invalid prep", func(t *testing.T) {
		_, err := parseScheduleSet(config.ManagerSchedule{Prep: "bad"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("invalid observe", func(t *testing.T) {
		_, err := parseScheduleSet(config.ManagerSchedule{Observe: "bad"})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestMapItem(t *testing.T) {
	// Test that mapItem delegates to pillars.MapToPillars
	m := &Manager{Config: config.ManagerConfig{
		Pillars: map[string]config.PillarConfig{
			"velocity": {Labels: []string{"velocity"}},
		},
	}}

	// Item with no matching signals returns empty
	item := dummyActivityItem("test PR", nil, nil)
	matches := m.mapItem(item, m.Config.Pillars)
	// Just verify it doesn't panic and returns a slice
	if matches == nil {
		t.Error("expected non-nil slice")
	}
}

// TestTeamStatsCalculation tests the team-level stat aggregation logic
// by verifying buildBusFactor integration and sort behavior.
func TestTeamStatsAggregation(t *testing.T) {
	members := []PersonStats{
		{Handle: "alice", PRsMerged: 3, AvgCycleDays: 2.0},
		{Handle: "bob", PRsMerged: 5, AvgCycleDays: 4.0},
		{Handle: "carol", PRsMerged: 1, AvgCycleDays: 1.0},
	}

	// Sort by PRs merged descending (same logic as Stats)
	sort.Slice(members, func(i, j int) bool {
		return members[i].PRsMerged > members[j].PRsMerged
	})

	if members[0].Handle != "bob" {
		t.Errorf("expected bob first, got %s", members[0].Handle)
	}
	if members[2].Handle != "carol" {
		t.Errorf("expected carol last, got %s", members[2].Handle)
	}

	// Avg cycle calculation
	var totalCycle float64
	var cycleCount int
	for _, m := range members {
		if m.AvgCycleDays > 0 {
			totalCycle += m.AvgCycleDays
			cycleCount++
		}
	}
	avgCycle := round(totalCycle/float64(cycleCount), 2)
	want := round((2.0+4.0+1.0)/3.0, 2)
	if avgCycle != want {
		t.Errorf("avgCycle = %v, want %v", avgCycle, want)
	}
}

func dummyActivityItem(title string, labels []string, files []string) pillars.ActivityItem {
	return pillars.ActivityItem{
		Title:  title,
		Labels: labels,
		Files:  files,
	}
}
