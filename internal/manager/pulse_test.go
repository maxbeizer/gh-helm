package manager

import (
	"testing"
	"time"

	"github.com/maxbeizer/gh-helm/internal/pillars"
)

func TestAverageCycleDays(t *testing.T) {
	tests := []struct {
		name  string
		items []pillars.ActivityItem
		want  float64
	}{
		{
			name:  "empty",
			items: nil,
			want:  0,
		},
		{
			name: "all zero times",
			items: []pillars.ActivityItem{
				{Title: "no dates"},
			},
			want: 0,
		},
		{
			name: "one item 2 days",
			items: []pillars.ActivityItem{
				{
					CreatedAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
					ClosedAt:  time.Date(2024, 6, 3, 0, 0, 0, 0, time.UTC),
				},
			},
			want: 2.0,
		},
		{
			name: "multiple items average",
			items: []pillars.ActivityItem{
				{
					CreatedAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
					ClosedAt:  time.Date(2024, 6, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					CreatedAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
					ClosedAt:  time.Date(2024, 6, 4, 0, 0, 0, 0, time.UTC),
				},
			},
			want: 2.0, // (1 + 3) / 2
		},
		{
			name: "skips items with zero created",
			items: []pillars.ActivityItem{
				{
					CreatedAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
					ClosedAt:  time.Date(2024, 6, 3, 0, 0, 0, 0, time.UTC),
				},
				{
					ClosedAt: time.Date(2024, 6, 5, 0, 0, 0, 0, time.UTC),
				},
			},
			want: 2.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := averageCycleDays(tc.items)
			if got != tc.want {
				t.Errorf("averageCycleDays() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRound(t *testing.T) {
	tests := []struct {
		name      string
		val       float64
		precision int
		want      float64
	}{
		{"zero", 0, 2, 0},
		{"round down", 1.234, 2, 1.23},
		{"round up", 1.235, 2, 1.24},
		{"no decimals", 5.0, 0, 5},
		{"negative", -1.555, 2, -1.56},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := round(tc.val, tc.precision)
			if got != tc.want {
				t.Errorf("round(%v, %d) = %v, want %v", tc.val, tc.precision, got, tc.want)
			}
		})
	}
}
