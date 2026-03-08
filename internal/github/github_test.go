package github

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

func withMockGh(t *testing.T, fn func(ctx context.Context, args ...string) ([]byte, error)) {
	t.Helper()
	orig := RunGhFunc
	RunGhFunc = fn
	t.Cleanup(func() { RunGhFunc = orig })
}

func TestFetchIssue(t *testing.T) {
	withMockGh(t, func(_ context.Context, _ ...string) ([]byte, error) {
		resp := Issue{
			Number: 42,
			Title:  "Fix auth bug",
			Body:   "Auth tokens expire too quickly",
			NodeID: "I_abc123",
			URL:    "https://github.com/org/repo/issues/42",
			Labels: []Label{{Name: "bug"}},
		}
		return json.Marshal(resp)
	})

	issue, err := FetchIssue(context.Background(), "org/repo", 42)
	if err != nil {
		t.Fatalf("FetchIssue: %v", err)
	}
	if issue.Number != 42 {
		t.Errorf("Number = %d, want 42", issue.Number)
	}
	if issue.Title != "Fix auth bug" {
		t.Errorf("Title = %q, want %q", issue.Title, "Fix auth bug")
	}
}

func TestSearchIssues(t *testing.T) {
	withMockGh(t, func(_ context.Context, _ ...string) ([]byte, error) {
		resp := searchResponse{
			Items: []SearchItem{
				{Title: "PR 1", Number: 10, State: "closed"},
				{Title: "PR 2", Number: 11, State: "open"},
			},
		}
		return json.Marshal(resp)
	})

	items, err := SearchIssues(context.Background(), "author:test")
	if err != nil {
		t.Fatalf("SearchIssues: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("got %d items, want 2", len(items))
	}
	if items[0].Number != 10 {
		t.Errorf("first item number = %d, want 10", items[0].Number)
	}
}

func TestPullFiles(t *testing.T) {
	withMockGh(t, func(_ context.Context, _ ...string) ([]byte, error) {
		resp := []struct {
			Filename string `json:"filename"`
		}{
			{Filename: "internal/auth/login.go"},
			{Filename: "internal/auth/login_test.go"},
			{Filename: "docs/auth.md"},
		}
		return json.Marshal(resp)
	})

	files, err := PullFiles(context.Background(), "org/repo", 42)
	if err != nil {
		t.Fatalf("PullFiles: %v", err)
	}
	if len(files) != 3 {
		t.Errorf("got %d files, want 3", len(files))
	}
}

func TestCurrentUser(t *testing.T) {
	withMockGh(t, func(_ context.Context, _ ...string) ([]byte, error) {
		return []byte("testuser\n"), nil
	})

	user, err := CurrentUser(context.Background())
	if err != nil {
		t.Fatalf("CurrentUser: %v", err)
	}
	if user != "testuser" {
		t.Errorf("user = %q, want %q", user, "testuser")
	}
}

func TestCurrentRepo(t *testing.T) {
	withMockGh(t, func(_ context.Context, _ ...string) ([]byte, error) {
		return []byte("org/my-repo\n"), nil
	})

	repo, err := CurrentRepo(context.Background())
	if err != nil {
		t.Fatalf("CurrentRepo: %v", err)
	}
	if repo != "org/my-repo" {
		t.Errorf("repo = %q, want %q", repo, "org/my-repo")
	}
}

func TestRepoFromURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://api.github.com/repos/org/repo", "org/repo"},
		{"org/repo", "org/repo"},
	}
	for _, tt := range tests {
		got := RepoFromURL(tt.input)
		if got != tt.want {
			t.Errorf("RepoFromURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFetchIssueError(t *testing.T) {
	withMockGh(t, func(_ context.Context, _ ...string) ([]byte, error) {
		return nil, fmt.Errorf("gh: not found (HTTP 404)")
	})

	_, err := FetchIssue(context.Background(), "org/repo", 999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
