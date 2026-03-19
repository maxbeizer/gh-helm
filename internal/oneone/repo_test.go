package oneone

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	gh "github.com/maxbeizer/gh-helm/internal/github"
)

func withMockGh(t *testing.T, fn func(ctx context.Context, args ...string) ([]byte, error)) {
	t.Helper()
	orig := gh.RunGhFunc
	gh.RunGhFunc = fn
	t.Cleanup(func() { gh.RunGhFunc = orig })
}

func TestPostObservation(t *testing.T) {
	tests := []struct {
		name    string
		repo    string
		title   string
		body    string
		mockFn  func(ctx context.Context, args ...string) ([]byte, error)
		wantErr string
	}{
		{
			name:  "successful issue creation",
			repo:  "org/one-on-one",
			title: "Weekly observation",
			body:  "Great work on the API refactor",
			mockFn: func(_ context.Context, args ...string) ([]byte, error) {
				return []byte("https://github.com/org/one-on-one/issues/1\n"), nil
			},
		},
		{
			name:  "verifies correct gh args",
			repo:  "org/one-on-one",
			title: "My Title",
			body:  "My Body",
			mockFn: func(_ context.Context, args ...string) ([]byte, error) {
				want := []string{"issue", "create", "--repo", "org/one-on-one", "--title", "My Title", "--body", "My Body"}
				if len(args) != len(want) {
					return nil, fmt.Errorf("args = %v, want %v", args, want)
				}
				for i, a := range args {
					if a != want[i] {
						return nil, fmt.Errorf("args[%d] = %q, want %q", i, a, want[i])
					}
				}
				return []byte("ok"), nil
			},
		},
		{
			name:  "CLI failure propagates error",
			repo:  "org/one-on-one",
			title: "title",
			body:  "body",
			mockFn: func(_ context.Context, _ ...string) ([]byte, error) {
				return []byte("permission denied"), fmt.Errorf("exit status 1")
			},
			wantErr: "gh issue create",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			withMockGh(t, tc.mockFn)
			err := PostObservation(context.Background(), "user", tc.repo, tc.title, tc.body)
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

func TestFetchRecentObservations(t *testing.T) {
	tests := []struct {
		name      string
		repo      string
		limit     int
		mockFn    func(ctx context.Context, args ...string) ([]byte, error)
		wantCount int
		wantErr   string
	}{
		{
			name:  "successful fetch with parsed results",
			repo:  "org/one-on-one",
			limit: 3,
			mockFn: func(_ context.Context, _ ...string) ([]byte, error) {
				issues := []ObservationIssue{
					{Number: 1, Title: "Obs 1", CreatedAt: "2024-01-01T00:00:00Z", URL: "https://github.com/org/one-on-one/issues/1"},
					{Number: 2, Title: "Obs 2", CreatedAt: "2024-01-02T00:00:00Z", URL: "https://github.com/org/one-on-one/issues/2"},
				}
				return json.Marshal(issues)
			},
			wantCount: 2,
		},
		{
			name:  "default limit when zero",
			repo:  "org/one-on-one",
			limit: 0,
			mockFn: func(_ context.Context, args ...string) ([]byte, error) {
				// Verify default limit of 5 is applied
				for i, a := range args {
					if a == "--limit" && i+1 < len(args) {
						if args[i+1] != "5" {
							return nil, fmt.Errorf("limit = %q, want %q", args[i+1], "5")
						}
					}
				}
				return []byte("[]"), nil
			},
			wantCount: 0,
		},
		{
			name:  "default limit when negative",
			repo:  "org/one-on-one",
			limit: -1,
			mockFn: func(_ context.Context, args ...string) ([]byte, error) {
				for i, a := range args {
					if a == "--limit" && i+1 < len(args) {
						if args[i+1] != "5" {
							return nil, fmt.Errorf("limit = %q, want %q", args[i+1], "5")
						}
					}
				}
				return []byte("[]"), nil
			},
			wantCount: 0,
		},
		{
			name:  "empty response returns empty slice",
			repo:  "org/one-on-one",
			limit: 5,
			mockFn: func(_ context.Context, _ ...string) ([]byte, error) {
				return []byte("[]"), nil
			},
			wantCount: 0,
		},
		{
			name:  "CLI failure returns error",
			repo:  "org/one-on-one",
			limit: 5,
			mockFn: func(_ context.Context, _ ...string) ([]byte, error) {
				return nil, fmt.Errorf("gh: network error")
			},
			wantErr: "network error",
		},
		{
			name:  "malformed JSON returns error",
			repo:  "org/one-on-one",
			limit: 5,
			mockFn: func(_ context.Context, _ ...string) ([]byte, error) {
				return []byte("{not valid json"), nil
			},
			wantErr: "invalid character",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			withMockGh(t, tc.mockFn)
			issues, err := FetchRecentObservations(context.Background(), tc.repo, tc.limit)
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
			if len(issues) != tc.wantCount {
				t.Fatalf("got %d issues, want %d", len(issues), tc.wantCount)
			}
		})
	}
}
