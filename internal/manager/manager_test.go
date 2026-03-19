package manager

import (
	"context"
	"testing"
	"time"
)

func TestParseSince(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, got time.Time)
	}{
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid string",
			input:   "notaduration",
			wantErr: true,
		},
		{
			name:  "go duration hours",
			input: "24h",
			check: func(t *testing.T, got time.Time) {
				expected := time.Now().Add(-24 * time.Hour)
				if got.Sub(expected).Abs() > 2*time.Second {
					t.Errorf("expected ~%v, got %v", expected, got)
				}
			},
		},
		{
			name:  "go duration minutes",
			input: "30m",
			check: func(t *testing.T, got time.Time) {
				expected := time.Now().Add(-30 * time.Minute)
				if got.Sub(expected).Abs() > 2*time.Second {
					t.Errorf("expected ~%v, got %v", expected, got)
				}
			},
		},
		{
			name:  "days format",
			input: "7d",
			check: func(t *testing.T, got time.Time) {
				expected := time.Now().Add(-7 * 24 * time.Hour)
				if got.Sub(expected).Abs() > 2*time.Second {
					t.Errorf("expected ~%v, got %v", expected, got)
				}
			},
		},
		{
			name:  "days format 30d",
			input: "30d",
			check: func(t *testing.T, got time.Time) {
				expected := time.Now().Add(-30 * 24 * time.Hour)
				if got.Sub(expected).Abs() > 2*time.Second {
					t.Errorf("expected ~%v, got %v", expected, got)
				}
			},
		},
		{
			name:    "zero days",
			input:   "0d",
			wantErr: true,
		},
		{
			name:    "negative days format",
			input:   "-5d",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseSince(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.check != nil {
				tc.check(t, got)
			}
		})
	}
}

func TestFormatRange(t *testing.T) {
	tests := []struct {
		name  string
		start time.Time
		end   time.Time
		want  string
	}{
		{
			name: "zero start",
			want: "",
		},
		{
			name:  "zero end",
			start: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			want:  "",
		},
		{
			name:  "same month same year",
			start: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			want:  "Jun 1–15, 2024",
		},
		{
			name:  "different month same year",
			start: time.Date(2024, 5, 28, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2024, 6, 4, 0, 0, 0, 0, time.UTC),
			want:  "May 28–Jun 4, 2024",
		},
		{
			name:  "different year",
			start: time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
			want:  "Dec 25, 2023–Jan 5, 2024",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatRange(tc.start, tc.end)
			if got != tc.want {
				t.Errorf("formatRange() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCtxOrBackground(t *testing.T) {
	t.Run("nil returns background", func(t *testing.T) {
		got := ctxOrBackground(nil)
		if got == nil {
			t.Fatal("expected non-nil context")
		}
	})

	t.Run("non-nil returns same context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "key", "value")
		got := ctxOrBackground(ctx)
		if got != ctx {
			t.Error("expected same context back")
		}
	})
}
