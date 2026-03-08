package guardrails

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type QueueItem struct {
	ID      string
	NodeID  string
	Number  int
	Title   string
	Body    string
	Repo    string
	URL     string
	Labels  []string
}

type RateLimiter struct {
	maxPerHour int
	timestamps []time.Time
	mu         sync.Mutex
}

func NewRateLimiter(maxPerHour int) *RateLimiter {
	return &RateLimiter{maxPerHour: maxPerHour}
}

func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.maxPerHour <= 0 {
		return true
	}
	cutoff := time.Now().Add(-1 * time.Hour)
	filtered := r.timestamps[:0]
	for _, ts := range r.timestamps {
		if ts.After(cutoff) {
			filtered = append(filtered, ts)
		}
	}
	r.timestamps = filtered
	if len(r.timestamps) >= r.maxPerHour {
		return false
	}
	r.timestamps = append(r.timestamps, time.Now())
	return true
}

type SafetyChecks struct{}

func (s *SafetyChecks) ValidateItem(item QueueItem) error {
	if strings.TrimSpace(item.Body) == "" {
		return fmt.Errorf("issue has no body")
	}
	if len(item.Body) > 10000 {
		return fmt.Errorf("issue body exceeds 10k characters")
	}
	for _, label := range item.Labels {
		if strings.EqualFold(label, "do-not-automate") {
			return fmt.Errorf("issue labeled do-not-automate")
		}
	}
	return nil
}
