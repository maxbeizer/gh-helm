package guardrails

import (
	"strings"
	"testing"
)

func TestRateLimiterAllow(t *testing.T) {
	t.Run("allows up to maxPerHour calls", func(t *testing.T) {
		rl := NewRateLimiter(3)
		for i := 0; i < 3; i++ {
			if !rl.Allow() {
				t.Fatalf("call %d should be allowed", i+1)
			}
		}
	})

	t.Run("rejects after maxPerHour calls", func(t *testing.T) {
		rl := NewRateLimiter(2)
		rl.Allow()
		rl.Allow()
		if rl.Allow() {
			t.Fatal("third call should be rejected")
		}
	})

	t.Run("maxPerHour zero allows unlimited", func(t *testing.T) {
		rl := NewRateLimiter(0)
		for i := 0; i < 100; i++ {
			if !rl.Allow() {
				t.Fatalf("call %d should be allowed with maxPerHour=0", i+1)
			}
		}
	})

	t.Run("negative maxPerHour allows unlimited", func(t *testing.T) {
		rl := NewRateLimiter(-1)
		for i := 0; i < 10; i++ {
			if !rl.Allow() {
				t.Fatalf("call %d should be allowed with maxPerHour=-1", i+1)
			}
		}
	})
}

func TestSafetyChecksValidateItem(t *testing.T) {
	sc := &SafetyChecks{}

	tests := []struct {
		name    string
		item    QueueItem
		wantErr string
	}{
		{
			name:    "empty body",
			item:    QueueItem{Title: "Test", Body: ""},
			wantErr: "no body",
		},
		{
			name:    "whitespace-only body",
			item:    QueueItem{Title: "Test", Body: "   \n\t  "},
			wantErr: "no body",
		},
		{
			name:    "body over 10k chars",
			item:    QueueItem{Title: "Test", Body: strings.Repeat("x", 10001)},
			wantErr: "exceeds 10k",
		},
		{
			name:    "do-not-automate label",
			item:    QueueItem{Title: "Test", Body: "valid body", Labels: []string{"do-not-automate"}},
			wantErr: "do-not-automate",
		},
		{
			name:    "do-not-automate label case insensitive",
			item:    QueueItem{Title: "Test", Body: "valid body", Labels: []string{"Do-Not-Automate"}},
			wantErr: "do-not-automate",
		},
		{
			name: "valid item",
			item: QueueItem{Title: "Test", Body: "A valid issue body", Labels: []string{"bug"}},
		},
		{
			name:    "body exactly 10k is valid",
			item:    QueueItem{Title: "Test", Body: strings.Repeat("x", 10000)},
			wantErr: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := sc.ValidateItem(tc.item)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %q, want substring %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
