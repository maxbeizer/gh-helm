package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/maxbeizer/gh-helm/internal/github"
	"github.com/maxbeizer/gh-helm/internal/guardrails"
)

// withMockGh overrides github.RunGhFunc for the duration of a test.
func withMockGh(t *testing.T, fn func(ctx context.Context, args ...string) ([]byte, error)) {
	t.Helper()
	orig := github.RunGhFunc
	github.RunGhFunc = fn
	t.Cleanup(func() { github.RunGhFunc = orig })
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"normal title", "Fix the auth bug", "fix-the-auth-bug"},
		{"uppercase", "ADD NEW FEATURE", "add-new-feature"},
		{"special chars", "feat: add oauth2 (v2) support!", "feat-add-oauth2-v2-support"},
		{"consecutive specials", "hello---world", "hello-world"},
		{"long title truncated", "this-is-a-really-long-branch-name-that-should-be-truncated-at-forty-characters", "this-is-a-really-long-branch-name-that-s"},
		{"empty string", "", "issue"},
		{"only special chars", "!@#$%^&*()", "issue"},
		{"numbers", "issue 123 fix", "issue-123-fix"},
		{"leading trailing spaces", " hello world ", "hello-world"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := slugify(tc.input)
			if got != tc.want {
				t.Errorf("slugify(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestApplyChanges(t *testing.T) {
	tests := []struct {
		name  string
		files []github.FileChange
	}{
		{
			"empty file list",
			nil,
		},
		{
			"single file",
			[]github.FileChange{
				{Path: "hello.txt", Content: "hello world"},
			},
		},
		{
			"nested paths",
			[]github.FileChange{
				{Path: "a/b/c/deep.go", Content: "package deep"},
				{Path: "x/y.txt", Content: "data"},
			},
		},
		{
			"empty path skipped",
			[]github.FileChange{
				{Path: "", Content: "should be skipped"},
				{Path: "valid.txt", Content: "ok"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()

			// Rewrite paths relative to temp dir
			adjusted := make([]github.FileChange, len(tc.files))
			for i, f := range tc.files {
				adjusted[i] = f
				if f.Path != "" {
					adjusted[i].Path = filepath.Join(tmp, f.Path)
				}
			}

			if err := applyChanges(adjusted); err != nil {
				t.Fatalf("applyChanges() error: %v", err)
			}

			for _, f := range tc.files {
				if f.Path == "" {
					continue
				}
				full := filepath.Join(tmp, f.Path)
				data, err := os.ReadFile(full)
				if err != nil {
					t.Fatalf("expected file %s to exist: %v", f.Path, err)
				}
				if string(data) != f.Content {
					t.Errorf("file %s content = %q, want %q", f.Path, string(data), f.Content)
				}
			}
		})
	}
}

func TestContainsLabel(t *testing.T) {
	tests := []struct {
		name   string
		labels []string
		target string
		want   bool
	}{
		{"exact match", []string{"bug", "feature"}, "bug", true},
		{"case insensitive", []string{"Bug", "Feature"}, "bug", true},
		{"no match", []string{"bug"}, "feature", false},
		{"empty labels", nil, "bug", false},
		{"empty target", []string{"bug"}, "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := containsLabel(tc.labels, tc.target)
			if got != tc.want {
				t.Errorf("containsLabel(%v, %q) = %v, want %v", tc.labels, tc.target, got, tc.want)
			}
		})
	}
}

func TestDefaultIfEmpty(t *testing.T) {
	tests := []struct {
		name     string
		val      string
		fallback string
		want     string
	}{
		{"non-empty returns val", "hello", "default", "hello"},
		{"empty returns fallback", "", "default", "default"},
		{"both empty", "", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := defaultIfEmpty(tc.val, tc.fallback)
			if got != tc.want {
				t.Errorf("defaultIfEmpty(%q, %q) = %q, want %q", tc.val, tc.fallback, got, tc.want)
			}
		})
	}
}

func TestNewProjectAgent(t *testing.T) {
	agent := NewProjectAgent()
	if agent == nil {
		t.Fatal("NewProjectAgent() returned nil")
	}
}

func TestRepoDisplay(t *testing.T) {
	tests := []struct {
		name string
		repo string
		want string
	}{
		{"with repo", "owner/repo", "owner/repo"},
		{"empty repo", "", "current repo"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := repoDisplay(tc.repo)
			if got != tc.want {
				t.Errorf("repoDisplay(%q) = %q, want %q", tc.repo, got, tc.want)
			}
		})
	}
}

func TestReadStatus(t *testing.T) {
	// Save and restore working directory to avoid polluting the repo.
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	t.Run("missing file returns default state", func(t *testing.T) {
		tmp := t.TempDir()
		os.Chdir(tmp)

		state, err := ReadStatus()
		if err != nil {
			t.Fatalf("ReadStatus() error: %v", err)
		}
		if state.Session != "none" {
			t.Errorf("Session = %q, want %q", state.Session, "none")
		}
		if state.IssuesWorked == nil {
			t.Error("IssuesWorked should be non-nil empty slice")
		}
		if state.PullsCreated == nil {
			t.Error("PullsCreated should be non-nil empty slice")
		}
	})

	t.Run("valid state file", func(t *testing.T) {
		tmp := t.TempDir()
		os.Chdir(tmp)

		s := State{
			Session:      "12345",
			IssuesWorked: []IssueInfo{{Number: 1, Title: "Test issue"}},
			PullsCreated: []PullInfo{{Number: 10, URL: "https://github.com/o/r/pull/10"}},
		}
		data, _ := json.Marshal(s)
		os.MkdirAll(".helm", 0o755)
		os.WriteFile(".helm/state.json", data, 0o644)

		state, err := ReadStatus()
		if err != nil {
			t.Fatalf("ReadStatus() error: %v", err)
		}
		if state.Session != "12345" {
			t.Errorf("Session = %q, want %q", state.Session, "12345")
		}
		if len(state.IssuesWorked) != 1 || state.IssuesWorked[0].Number != 1 {
			t.Errorf("IssuesWorked = %v, want [{1 Test issue}]", state.IssuesWorked)
		}
		if len(state.PullsCreated) != 1 || state.PullsCreated[0].Number != 10 {
			t.Errorf("PullsCreated = %v, want [{10 ...}]", state.PullsCreated)
		}
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		tmp := t.TempDir()
		os.Chdir(tmp)

		os.MkdirAll(".helm", 0o755)
		os.WriteFile(".helm/state.json", []byte("not json"), 0o644)

		_, err := ReadStatus()
		if err == nil {
			t.Fatal("ReadStatus() expected error for invalid JSON")
		}
	})
}

func TestProjectItemNodeStatus(t *testing.T) {
	tests := []struct {
		name string
		node github.ProjectItemNode
		want string
	}{
		{
			"has status field",
			github.ProjectItemNode{
				FieldValues: struct {
					Nodes []struct {
						Name  string `json:"name"`
						Field struct {
							Name string `json:"name"`
						} `json:"field"`
					} `json:"nodes"`
				}{
					Nodes: []struct {
						Name  string `json:"name"`
						Field struct {
							Name string `json:"name"`
						} `json:"field"`
					}{
						{Name: "Ready", Field: struct {
							Name string `json:"name"`
						}{Name: "Status"}},
					},
				},
			},
			"Ready",
		},
		{
			"no status field",
			github.ProjectItemNode{},
			"",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.node.Status()
			if got != tc.want {
				t.Errorf("Status() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestProjectItemNodeLabelNames(t *testing.T) {
	node := github.ProjectItemNode{}
	node.Content.Labels.Nodes = []struct {
		Name string `json:"name"`
	}{
		{Name: "bug"},
		{Name: "priority"},
	}

	labels := node.LabelNames()
	if len(labels) != 2 {
		t.Fatalf("LabelNames() returned %d items, want 2", len(labels))
	}
	if labels[0] != "bug" || labels[1] != "priority" {
		t.Errorf("LabelNames() = %v, want [bug priority]", labels)
	}
}

func TestFetchQueueItems(t *testing.T) {
	t.Run("missing owner or project returns error", func(t *testing.T) {
		_, err := fetchQueueItems(context.Background(), "", 0, "Ready", "")
		if err == nil {
			t.Fatal("expected error for empty owner/project")
		}
	})

	t.Run("parses org project response", func(t *testing.T) {
		resp := buildGraphQLResponse("organization", "Ready", "owner/repo", 42, "Fix bug", []string{"bug", "agent"})

		withMockGh(t, func(_ context.Context, _ ...string) ([]byte, error) {
			return json.Marshal(resp)
		})

		items, err := fetchQueueItems(context.Background(), "myorg", 1, "Ready", "")
		if err != nil {
			t.Fatalf("fetchQueueItems() error: %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("got %d items, want 1", len(items))
		}
		assertQueueItem(t, items[0], 42, "Fix bug", "owner/repo", []string{"bug", "agent"})
	})

	t.Run("parses user project response", func(t *testing.T) {
		resp := buildGraphQLResponse("user", "Todo", "user/repo", 7, "Add tests", []string{"tests"})

		withMockGh(t, func(_ context.Context, _ ...string) ([]byte, error) {
			return json.Marshal(resp)
		})

		items, err := fetchQueueItems(context.Background(), "myuser", 2, "Todo", "")
		if err != nil {
			t.Fatalf("fetchQueueItems() error: %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("got %d items, want 1", len(items))
		}
		assertQueueItem(t, items[0], 7, "Add tests", "user/repo", []string{"tests"})
	})

	t.Run("filters by status", func(t *testing.T) {
		resp := buildGraphQLResponse("organization", "Done", "owner/repo", 10, "Old issue", nil)

		withMockGh(t, func(_ context.Context, _ ...string) ([]byte, error) {
			return json.Marshal(resp)
		})

		items, err := fetchQueueItems(context.Background(), "myorg", 1, "Ready", "")
		if err != nil {
			t.Fatalf("fetchQueueItems() error: %v", err)
		}
		if len(items) != 0 {
			t.Errorf("expected 0 items after status filter, got %d", len(items))
		}
	})

	t.Run("filters by label", func(t *testing.T) {
		resp := buildGraphQLResponse("organization", "Ready", "owner/repo", 10, "No label match", []string{"bug"})

		withMockGh(t, func(_ context.Context, _ ...string) ([]byte, error) {
			return json.Marshal(resp)
		})

		items, err := fetchQueueItems(context.Background(), "myorg", 1, "Ready", "agent")
		if err != nil {
			t.Fatalf("fetchQueueItems() error: %v", err)
		}
		if len(items) != 0 {
			t.Errorf("expected 0 items after label filter, got %d", len(items))
		}
	})

	t.Run("gh API error propagated", func(t *testing.T) {
		withMockGh(t, func(_ context.Context, _ ...string) ([]byte, error) {
			return nil, fmt.Errorf("network error")
		})

		_, err := fetchQueueItems(context.Background(), "myorg", 1, "Ready", "")
		if err == nil {
			t.Fatal("expected error from RunGhFunc failure")
		}
	})
}

// --- helpers ---

func buildGraphQLResponse(ownerType, status, repo string, number int, title string, labels []string) map[string]any {
	labelNodes := make([]map[string]string, len(labels))
	for i, l := range labels {
		labelNodes[i] = map[string]string{"name": l}
	}

	item := map[string]any{
		"id": fmt.Sprintf("item-%d", number),
		"content": map[string]any{
			"number":     number,
			"title":      title,
			"body":       "issue body",
			"url":        fmt.Sprintf("https://github.com/%s/issues/%d", repo, number),
			"id":         fmt.Sprintf("I_%d", number),
			"repository": map[string]string{"nameWithOwner": repo},
			"labels":     map[string]any{"nodes": labelNodes},
		},
		"fieldValues": map[string]any{
			"nodes": []map[string]any{
				{
					"name":  status,
					"field": map[string]string{"name": "Status"},
				},
			},
		},
	}

	project := map[string]any{
		"projectV2": map[string]any{
			"items": map[string]any{
				"nodes": []any{item},
			},
		},
	}

	resp := map[string]any{}
	if ownerType == "organization" {
		resp["organization"] = project
	} else {
		resp["user"] = project
	}
	return resp
}

func assertQueueItem(t *testing.T, item guardrails.QueueItem, number int, title, repo string, labels []string) {
	t.Helper()
	if item.Number != number {
		t.Errorf("Number = %d, want %d", item.Number, number)
	}
	if item.Title != title {
		t.Errorf("Title = %q, want %q", item.Title, title)
	}
	if item.Repo != repo {
		t.Errorf("Repo = %q, want %q", item.Repo, repo)
	}
	if len(item.Labels) != len(labels) {
		t.Errorf("Labels = %v, want %v", item.Labels, labels)
	}
}
