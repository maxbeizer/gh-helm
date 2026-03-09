package github

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

func TestListIssues(t *testing.T) {
	withMockGh(t, func(_ context.Context, args ...string) ([]byte, error) {
		// Verify state flag is passed
		hasState := false
		for _, a := range args {
			if a == "--state" {
				hasState = true
			}
		}
		if !hasState {
			return nil, fmt.Errorf("expected --state flag")
		}
		resp := []IssueListItem{
			{Number: 1, Title: "Open issue", State: "OPEN", URL: "https://github.com/org/repo/issues/1"},
			{Number: 2, Title: "Closed issue", State: "CLOSED", URL: "https://github.com/org/repo/issues/2"},
		}
		return json.Marshal(resp)
	})

	items, err := ListIssues(context.Background(), "org/repo", "all")
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("got %d items, want 2", len(items))
	}
	if items[0].Number != 1 {
		t.Errorf("first item number = %d, want 1", items[0].Number)
	}
}

func TestListIssues_DefaultState(t *testing.T) {
	withMockGh(t, func(_ context.Context, args ...string) ([]byte, error) {
		for i, a := range args {
			if a == "--state" && i+1 < len(args) && args[i+1] == "all" {
				return json.Marshal([]IssueListItem{})
			}
		}
		return nil, fmt.Errorf("expected --state all")
	})

	_, err := ListIssues(context.Background(), "org/repo", "")
	if err != nil {
		t.Fatalf("ListIssues with empty state: %v", err)
	}
}

func TestFetchPR(t *testing.T) {
	withMockGh(t, func(_ context.Context, _ ...string) ([]byte, error) {
		resp := PR{
			Number: 10,
			Title:  "Add auth feature",
			Body:   "Implements auth\n\nCloses #5",
			State:  "OPEN",
			URL:    "https://github.com/org/repo/pull/10",
		}
		return json.Marshal(resp)
	})

	pr, err := FetchPR(context.Background(), "org/repo", 10)
	if err != nil {
		t.Fatalf("FetchPR: %v", err)
	}
	if pr.Number != 10 {
		t.Errorf("Number = %d, want 10", pr.Number)
	}
	if pr.Title != "Add auth feature" {
		t.Errorf("Title = %q, want %q", pr.Title, "Add auth feature")
	}
}

func TestFetchPRDiff(t *testing.T) {
	expectedDiff := "diff --git a/foo.go b/foo.go\n+new line"
	withMockGh(t, func(_ context.Context, _ ...string) ([]byte, error) {
		return []byte(expectedDiff + "\n"), nil
	})

	diff, err := FetchPRDiff(context.Background(), "org/repo", 10)
	if err != nil {
		t.Fatalf("FetchPRDiff: %v", err)
	}
	if diff != expectedDiff {
		t.Errorf("diff = %q, want %q", diff, expectedDiff)
	}
}

func TestFetchPRClosingIssues(t *testing.T) {
	withMockGh(t, func(_ context.Context, _ ...string) ([]byte, error) {
		return []byte("5\n12\n"), nil
	})

	numbers, err := FetchPRClosingIssues(context.Background(), "org/repo", 10)
	if err != nil {
		t.Fatalf("FetchPRClosingIssues: %v", err)
	}
	if len(numbers) != 2 {
		t.Fatalf("got %d numbers, want 2", len(numbers))
	}
	if numbers[0] != 5 || numbers[1] != 12 {
		t.Errorf("numbers = %v, want [5 12]", numbers)
	}
}

func TestFetchPRClosingIssues_Empty(t *testing.T) {
	withMockGh(t, func(_ context.Context, _ ...string) ([]byte, error) {
		return []byte(""), nil
	})

	numbers, err := FetchPRClosingIssues(context.Background(), "org/repo", 10)
	if err != nil {
		t.Fatalf("FetchPRClosingIssues: %v", err)
	}
	if numbers != nil {
		t.Errorf("expected nil, got %v", numbers)
	}
}
