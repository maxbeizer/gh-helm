package pillars

import (
	"testing"

	"github.com/maxbeizer/gh-helm/internal/config"
)

func TestMapToPillars(t *testing.T) {
	pillars := map[string]config.PillarConfig{
		"reliability": {
			Labels:  []string{"bug", "testing"},
			Repos:   []string{"myorg/monitoring"},
			Signals: []string{"bug fixes", "test coverage"},
		},
		"velocity": {
			Labels:  []string{"feature"},
			Signals: []string{"PRs merged"},
		},
	}

	tests := []struct {
		name       string
		item       ActivityItem
		wantCount  int
		wantPillar string
		wantConf   string
		check      func(t *testing.T, matches []PillarMatch)
	}{
		{
			name: "label match high confidence",
			item: ActivityItem{
				Title:  "Fix crash",
				Labels: []string{"bug"},
			},
			wantCount:  1,
			wantPillar: "reliability",
			wantConf:   "high",
		},
		{
			name: "repo match high confidence",
			item: ActivityItem{
				Title: "Update alerts",
				Repo:  "myorg/monitoring",
			},
			wantCount:  1,
			wantPillar: "reliability",
			wantConf:   "high",
		},
		{
			name: "path match medium confidence via default patterns",
			item: ActivityItem{
				Title: "Add auth tests",
				Files: []string{"tests/auth_test.go"},
			},
			wantCount:  1,
			wantPillar: "reliability",
			wantConf:   "medium",
		},
		{
			name: "keyword substring match low confidence",
			item: ActivityItem{
				Title: "Improve test coverage metrics",
				Body:  "We need better test coverage",
			},
			check: func(t *testing.T, matches []PillarMatch) {
				t.Helper()
				found := false
				for _, m := range matches {
					if m.Pillar == "reliability" && m.Confidence == "low" {
						found = true
					}
				}
				if !found {
					t.Error("expected reliability match at low confidence from keyword")
				}
			},
		},
		{
			name: "keyword no match when substring not present",
			item: ActivityItem{
				Title: "fix bug in auth",
				Body:  "something unrelated",
			},
			check: func(t *testing.T, matches []PillarMatch) {
				t.Helper()
				// "bug fixes" is the signal — "fix bug" does NOT contain "bug fixes"
				// but "bug" is a label signal too, so we should not have a label match
				// (no labels set). The keyword "bug fixes" should NOT match "fix bug".
				for _, m := range matches {
					if m.Confidence == "low" && m.Reason == "keyword: bug fixes" {
						t.Error("should not match 'bug fixes' in 'fix bug in auth'")
					}
				}
			},
		},
		{
			name: "no match returns empty",
			item: ActivityItem{
				Title: "unrelated work",
				Body:  "nothing here",
			},
			wantCount: 0,
		},
		{
			name: "multiple pillars matched",
			item: ActivityItem{
				Title:  "New feature with tests",
				Labels: []string{"feature", "bug"},
			},
			check: func(t *testing.T, matches []PillarMatch) {
				t.Helper()
				pillarsFound := map[string]bool{}
				for _, m := range matches {
					pillarsFound[m.Pillar] = true
				}
				if !pillarsFound["reliability"] {
					t.Error("expected reliability match")
				}
				if !pillarsFound["velocity"] {
					t.Error("expected velocity match")
				}
			},
		},
		{
			name: "dedup same pillar from label and repo",
			item: ActivityItem{
				Title:  "Fix monitoring bug",
				Labels: []string{"bug"},
				Repo:   "myorg/monitoring",
			},
			check: func(t *testing.T, matches []PillarMatch) {
				t.Helper()
				count := 0
				for _, m := range matches {
					if m.Pillar == "reliability" {
						count++
					}
				}
				if count != 1 {
					t.Errorf("reliability matched %d times, want 1 (dedup)", count)
				}
			},
		},
		{
			name: "case insensitive label matching",
			item: ActivityItem{
				Title:  "Uppercase label",
				Labels: []string{"BUG"},
			},
			wantCount:  1,
			wantPillar: "reliability",
			wantConf:   "high",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			matches := MapToPillars(tc.item, pillars)

			if tc.check != nil {
				tc.check(t, matches)
				return
			}

			if len(matches) != tc.wantCount {
				t.Fatalf("got %d matches, want %d: %+v", len(matches), tc.wantCount, matches)
			}
			if tc.wantCount > 0 {
				if matches[0].Pillar != tc.wantPillar {
					t.Errorf("Pillar = %q, want %q", matches[0].Pillar, tc.wantPillar)
				}
				if matches[0].Confidence != tc.wantConf {
					t.Errorf("Confidence = %q, want %q", matches[0].Confidence, tc.wantConf)
				}
			}
		})
	}
}
