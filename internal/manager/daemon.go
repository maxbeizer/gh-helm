package manager

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

type Logger interface {
	Printf(format string, args ...any)
}

type scheduleEntry struct {
	Minute int
	Hour   int
	Dow    *time.Weekday
	Raw    string
}

type scheduleSet struct {
	Pulse   *scheduleEntry
	Prep    *scheduleEntry
	Observe *scheduleEntry
}

func RunManagerDaemon(ctx context.Context, cfgConfigPath string, logger Logger) error {
	mgr, err := Load(cfgConfigPath)
	if err != nil {
		return err
	}

	sched, err := parseScheduleSet(mgr.Config.Schedule)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	logf := log.Printf
	if logger != nil {
		logf = logger.Printf
	}

	lastRun := map[string]time.Time{}
	logf("manager daemon started")

	for {
		now := time.Now()
		if sched.Pulse != nil && sched.Pulse.Due(now) && !alreadyRan(lastRun, "pulse", now) {
			if _, err := mgr.Pulse(ctx, PulseOptions{Since: "30d"}); err != nil {
				logf("pulse error: %v", err)
			}
			lastRun["pulse"] = now
		}
		if sched.Observe != nil && sched.Observe.Due(now) && !alreadyRan(lastRun, "observe", now) {
			if _, err := mgr.Observe(ctx, ObserveOptions{Since: "7d"}); err != nil {
				logf("observe error: %v", err)
			}
			lastRun["observe"] = now
		}
		if sched.Prep != nil && sched.Prep.Due(now) && !alreadyRan(lastRun, "prep", now) {
			for _, member := range mgr.Config.Team {
				if member.Handle == "" {
					continue
				}
				if _, err := mgr.Prep(ctx, PrepOptions{Since: "14d", Handle: member.Handle}); err != nil {
					logf("prep error for %s: %v", member.Handle, err)
				}
			}
			lastRun["prep"] = now
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func parseScheduleSet(cfg ManagerSchedule) (scheduleSet, error) {
	set := scheduleSet{}
	if cfg.Pulse != "" {
		entry, err := parseSchedule(cfg.Pulse)
		if err != nil {
			return set, fmt.Errorf("pulse schedule: %w", err)
		}
		set.Pulse = entry
	}
	if cfg.Prep != "" {
		entry, err := parseSchedule(cfg.Prep)
		if err != nil {
			return set, fmt.Errorf("prep schedule: %w", err)
		}
		set.Prep = entry
	}
	if cfg.Observe != "" {
		entry, err := parseSchedule(cfg.Observe)
		if err != nil {
			return set, fmt.Errorf("observe schedule: %w", err)
		}
		set.Observe = entry
	}
	return set, nil
}

func parseSchedule(spec string) (*scheduleEntry, error) {
	parts := strings.Fields(spec)
	if len(parts) < 5 {
		return nil, fmt.Errorf("invalid schedule: %s", spec)
	}
	minute, err := parsePart(parts[0])
	if err != nil {
		return nil, err
	}
	hour, err := parsePart(parts[1])
	if err != nil {
		return nil, err
	}
	dow, err := parseDow(parts[4])
	if err != nil {
		return nil, err
	}
	return &scheduleEntry{Minute: minute, Hour: hour, Dow: dow, Raw: spec}, nil
}

func parsePart(value string) (int, error) {
	if value == "*" {
		return -1, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid schedule part: %s", value)
	}
	return parsed, nil
}

func parseDow(value string) (*time.Weekday, error) {
	if value == "*" {
		return nil, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return nil, fmt.Errorf("invalid weekday: %s", value)
	}
	if parsed < 0 || parsed > 6 {
		return nil, fmt.Errorf("weekday out of range: %d", parsed)
	}
	dow := time.Weekday(parsed)
	return &dow, nil
}

func (s scheduleEntry) Due(now time.Time) bool {
	if s.Minute >= 0 && now.Minute() != s.Minute {
		return false
	}
	if s.Hour >= 0 && now.Hour() != s.Hour {
		return false
	}
	if s.Dow != nil && now.Weekday() != *s.Dow {
		return false
	}
	return true
}

func alreadyRan(lastRun map[string]time.Time, key string, now time.Time) bool {
	if prev, ok := lastRun[key]; ok {
		if prev.Truncate(time.Minute).Equal(now.Truncate(time.Minute)) {
			return true
		}
	}
	return false
}
