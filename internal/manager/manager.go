package manager

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/maxbeizer/max-ops/internal/config"
	"github.com/maxbeizer/max-ops/internal/pillars"
)

type Manager struct {
	Config config.ManagerConfig
}

type ObserveOptions struct {
	Since  string
	Handle string
	DryRun bool
}

type PrepOptions struct {
	Since  string
	Handle string
}

type PulseOptions struct {
	Since string
}

type ReportOptions struct {
	Since  string
	Handle string
}

func Load(path string) (*Manager, error) {
	cfg, err := config.LoadManager(path)
	if err != nil {
		return nil, err
	}
	return &Manager{Config: cfg}, nil
}

func parseSince(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, errors.New("missing since")
	}
	if dur, err := time.ParseDuration(s); err == nil {
		return time.Now().Add(-dur), nil
	}
	if len(s) > 1 && s[len(s)-1] == 'd' {
		var days int
		_, err := fmt.Sscanf(s, "%dd", &days)
		if err == nil && days > 0 {
			return time.Now().Add(-time.Duration(days) * 24 * time.Hour), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid duration: %s", s)
}

func formatRange(start, end time.Time) string {
	if start.IsZero() || end.IsZero() {
		return ""
	}
	if start.Year() == end.Year() {
		if start.Month() == end.Month() {
			return fmt.Sprintf("%s–%d, %d", start.Format("Jan 2"), end.Day(), end.Year())
		}
		return fmt.Sprintf("%s–%s, %d", start.Format("Jan 2"), end.Format("Jan 2"), end.Year())
	}
	return fmt.Sprintf("%s–%s", start.Format("Jan 2, 2006"), end.Format("Jan 2, 2006"))
}

func now() time.Time {
	return time.Now()
}

func ctxOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func (m *Manager) mapItem(item pillars.ActivityItem, configs map[string]config.PillarConfig) []pillars.PillarMatch {
	return pillars.MapToPillars(item, configs)
}
