package manager

import (
	"strings"
	"testing"
	"time"

	"github.com/maxbeizer/gh-helm/internal/config"
	"github.com/maxbeizer/gh-helm/internal/github"
	"github.com/maxbeizer/gh-helm/internal/oneone"
	"github.com/maxbeizer/gh-helm/internal/pillars"
)

func TestBuildPrepBody(t *testing.T) {
	start := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	member := config.TeamMember{Handle: "alice", Pillars: []string{"velocity", "quality"}}

	t.Run("empty activity", func(t *testing.T) {
		activity := activityData{}
		body := buildPrepBody(member, activity, nil, nil, nil, start, end, nil, nil)

		if !strings.Contains(body, "@alice") {
			t.Error("expected handle in header")
		}
		if !strings.Contains(body, "No PRs merged yet") {
			t.Error("expected no PRs win message")
		}
		if !strings.Contains(body, "No open PRs") {
			t.Error("expected no open PRs")
		}
		if !strings.Contains(body, "None reported") {
			t.Error("expected no blockers")
		}
		if !strings.Contains(body, "No recent notes") {
			t.Error("expected no recent observations")
		}
		// With assigned pillars but no matches, should suggest checking missing pillars
		if !strings.Contains(body, "Check in on") {
			t.Error("expected check-in suggestion for missing pillars")
		}
	})

	t.Run("with activity", func(t *testing.T) {
		activity := activityData{
			MergedPRs: []pillars.ActivityItem{
				{Title: "First PR"},
				{Title: "Second PR"},
				{Title: "Third PR"},
			},
		}
		openPRs := []pillars.ActivityItem{{Title: "Open PR 1"}}
		blocked := []github.SearchItem{{Title: "Blocked issue"}}
		matches := map[string][]pillars.PillarMatch{
			"velocity": {{Pillar: "velocity"}},
		}
		prev := []oneone.ObservationIssue{{Title: "Past observation"}}

		body := buildPrepBody(member, activity, openPRs, blocked, matches, start, end, prev, nil)

		if !strings.Contains(body, "First PR") {
			t.Error("expected first PR in wins")
		}
		if !strings.Contains(body, "Second PR") {
			t.Error("expected second PR in wins")
		}
		// Third PR should be truncated (limit 2)
		if strings.Contains(body, "Third PR") {
			t.Error("expected third PR to be truncated")
		}
		if !strings.Contains(body, "Open PR 1") {
			t.Error("expected open PR")
		}
		if !strings.Contains(body, "Blocked issue") {
			t.Error("expected blocked issue")
		}
		if !strings.Contains(body, "Past observation") {
			t.Error("expected past observation")
		}
		if !strings.Contains(body, "Acknowledge recent wins") {
			t.Error("expected wins suggestion")
		}
		if !strings.Contains(body, "Check in on quality focus") {
			t.Error("expected missing pillar suggestion")
		}
	})
}

func TestBuildPillarSummary(t *testing.T) {
	t.Run("with assigned pillars", func(t *testing.T) {
		matches := map[string][]pillars.PillarMatch{
			"velocity": {{Pillar: "velocity"}, {Pillar: "velocity"}},
		}
		assigned := []string{"velocity", "quality"}
		lines := buildPillarSummary(matches, assigned, nil)

		if len(lines) != 2 {
			t.Fatalf("got %d lines, want 2", len(lines))
		}
		// Lines are sorted
		hasQualityWarning := false
		hasVelocityCount := false
		for _, line := range lines {
			if strings.Contains(line, "Quality: no activity") {
				hasQualityWarning = true
			}
			if strings.Contains(line, "Velocity: 2 contributions") {
				hasVelocityCount = true
			}
		}
		if !hasQualityWarning {
			t.Error("expected quality warning")
		}
		if !hasVelocityCount {
			t.Error("expected velocity count")
		}
	})

	t.Run("falls back to pillar config keys when no assigned", func(t *testing.T) {
		pillarConfig := map[string]config.PillarConfig{
			"impact":  {},
			"quality": {},
		}
		lines := buildPillarSummary(nil, nil, pillarConfig)

		if len(lines) != 2 {
			t.Fatalf("got %d lines, want 2", len(lines))
		}
	})
}
