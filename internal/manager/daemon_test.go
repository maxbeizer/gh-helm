package manager

import (
	"testing"
	"time"
)

func TestParseSchedule(t *testing.T) {
	tests := []struct {
		name       string
		spec       string
		wantErr    bool
		wantMinute int
		wantHour   int
		wantDow    *time.Weekday
	}{
		{
			name:       "monday 9am",
			spec:       "0 9 * * 1",
			wantMinute: 0,
			wantHour:   9,
			wantDow:    weekdayPtr(time.Monday),
		},
		{
			name:       "daily 2:30pm",
			spec:       "30 14 * * *",
			wantMinute: 30,
			wantHour:   14,
			wantDow:    nil,
		},
		{
			name:       "sunday midnight",
			spec:       "0 0 * * 0",
			wantMinute: 0,
			wantHour:   0,
			wantDow:    weekdayPtr(time.Sunday),
		},
		{
			name:       "all wildcards",
			spec:       "* * * * *",
			wantMinute: -1,
			wantHour:   -1,
			wantDow:    nil,
		},
		{
			name:    "invalid string",
			spec:    "invalid",
			wantErr: true,
		},
		{
			name:    "too few parts",
			spec:    "0 9 *",
			wantErr: true,
		},
		{
			name:    "invalid minute",
			spec:    "abc 9 * * *",
			wantErr: true,
		},
		{
			name:    "invalid hour",
			spec:    "0 abc * * *",
			wantErr: true,
		},
		{
			name:    "invalid dow",
			spec:    "0 9 * * abc",
			wantErr: true,
		},
		{
			name:    "dow out of range",
			spec:    "0 9 * * 7",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			entry, err := parseSchedule(tc.spec)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if entry.Minute != tc.wantMinute {
				t.Errorf("Minute = %d, want %d", entry.Minute, tc.wantMinute)
			}
			if entry.Hour != tc.wantHour {
				t.Errorf("Hour = %d, want %d", entry.Hour, tc.wantHour)
			}
			if tc.wantDow == nil {
				if entry.Dow != nil {
					t.Errorf("Dow = %v, want nil", *entry.Dow)
				}
			} else {
				if entry.Dow == nil {
					t.Fatalf("Dow = nil, want %v", *tc.wantDow)
				}
				if *entry.Dow != *tc.wantDow {
					t.Errorf("Dow = %v, want %v", *entry.Dow, *tc.wantDow)
				}
			}
		})
	}
}

func TestScheduleEntryDue(t *testing.T) {
	monday9am := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC) // Monday

	tests := []struct {
		name  string
		entry scheduleEntry
		now   time.Time
		want  bool
	}{
		{
			name:  "exact match monday 9:00",
			entry: scheduleEntry{Minute: 0, Hour: 9, Dow: weekdayPtr(time.Monday)},
			now:   monday9am,
			want:  true,
		},
		{
			name:  "wrong minute",
			entry: scheduleEntry{Minute: 0, Hour: 9, Dow: weekdayPtr(time.Monday)},
			now:   monday9am.Add(time.Minute),
			want:  false,
		},
		{
			name:  "wrong day",
			entry: scheduleEntry{Minute: 0, Hour: 9, Dow: weekdayPtr(time.Monday)},
			now:   monday9am.AddDate(0, 0, 1), // Tuesday
			want:  false,
		},
		{
			name:  "all wildcards always true",
			entry: scheduleEntry{Minute: -1, Hour: -1, Dow: nil},
			now:   monday9am,
			want:  true,
		},
		{
			name:  "wildcard dow matches any day",
			entry: scheduleEntry{Minute: 0, Hour: 9, Dow: nil},
			now:   monday9am.AddDate(0, 0, 2), // Wednesday 9:00
			want:  true,
		},
		{
			name:  "wildcard minute and hour",
			entry: scheduleEntry{Minute: -1, Hour: -1, Dow: weekdayPtr(time.Monday)},
			now:   monday9am.Add(3*time.Hour + 45*time.Minute),
			want:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.entry.Due(tc.now)
			if got != tc.want {
				t.Errorf("Due(%v) = %v, want %v", tc.now, got, tc.want)
			}
		})
	}
}

func TestAlreadyRan(t *testing.T) {
	now := time.Date(2024, 6, 15, 10, 30, 45, 0, time.UTC)

	tests := []struct {
		name    string
		lastRun map[string]time.Time
		key     string
		now     time.Time
		want    bool
	}{
		{
			name:    "same minute returns true",
			lastRun: map[string]time.Time{"pulse": now.Add(-10 * time.Second)},
			key:     "pulse",
			now:     now,
			want:    true,
		},
		{
			name:    "different minute returns false",
			lastRun: map[string]time.Time{"pulse": now.Add(-61 * time.Second)},
			key:     "pulse",
			now:     now,
			want:    false,
		},
		{
			name:    "key not in map returns false",
			lastRun: map[string]time.Time{},
			key:     "pulse",
			now:     now,
			want:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := alreadyRan(tc.lastRun, tc.key, tc.now)
			if got != tc.want {
				t.Errorf("alreadyRan = %v, want %v", got, tc.want)
			}
		})
	}
}

func weekdayPtr(d time.Weekday) *time.Weekday {
	return &d
}
