package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maxbeizer/gh-helm/internal/config"
	"github.com/maxbeizer/gh-helm/internal/github"
)

func withMockRun(t *testing.T, fn func(ctx context.Context, args ...string) error) {
	t.Helper()
	orig := github.RunGhFunc
	github.RunGhFunc = func(ctx context.Context, args ...string) ([]byte, error) {
		return nil, fn(ctx, args...)
	}
	t.Cleanup(func() { github.RunGhFunc = orig })
}

// --- Factory tests ---

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		channel     string
		webhookURL  string
		wantNil     bool
		wantType    string
	}{
		{
			name:       "slack with webhook",
			channel:    "slack",
			webhookURL: "https://hooks.slack.com/test",
			wantType:   "*notifications.SlackNotifier",
		},
		{
			name:    "slack without webhook",
			channel: "slack",
			wantNil: true,
		},
		{
			name:     "github",
			channel:  "github",
			wantType: "*notifications.GitHubNotifier",
		},
		{
			name:    "empty channel",
			channel: "",
			wantNil: true,
		},
		{
			name:    "unknown channel",
			channel: "email",
			wantNil: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Config{
				Notifications: config.NotificationsConfig{
					Channel:    tc.channel,
					WebhookURL: tc.webhookURL,
				},
			}
			n := New(cfg, "org/repo", 42)
			if tc.wantNil {
				if n != nil {
					t.Fatalf("expected nil, got %T", n)
				}
				return
			}
			if n == nil {
				t.Fatal("expected non-nil notifier, got nil")
			}
			got := fmt.Sprintf("%T", n)
			if got != tc.wantType {
				t.Errorf("type = %s, want %s", got, tc.wantType)
			}
		})
	}
}

func TestNew_SlackWebhookURL(t *testing.T) {
	cfg := config.Config{
		Notifications: config.NotificationsConfig{
			Channel:    "slack",
			WebhookURL: "https://hooks.slack.com/services/T00/B00/xxx",
		},
	}
	n := New(cfg, "org/repo", 1)
	sn, ok := n.(*SlackNotifier)
	if !ok {
		t.Fatalf("expected *SlackNotifier, got %T", n)
	}
	if sn.WebhookURL != cfg.Notifications.WebhookURL {
		t.Errorf("WebhookURL = %q, want %q", sn.WebhookURL, cfg.Notifications.WebhookURL)
	}
}

func TestNew_GitHubFields(t *testing.T) {
	cfg := config.Config{
		Notifications: config.NotificationsConfig{Channel: "github"},
	}
	n := New(cfg, "org/repo", 99)
	gn, ok := n.(*GitHubNotifier)
	if !ok {
		t.Fatalf("expected *GitHubNotifier, got %T", n)
	}
	if gn.Repo != "org/repo" {
		t.Errorf("Repo = %q, want %q", gn.Repo, "org/repo")
	}
	if gn.IssueNumber != 99 {
		t.Errorf("IssueNumber = %d, want 99", gn.IssueNumber)
	}
}

// --- SlackNotifier tests ---

func TestSlackNotifier_Notify_Success(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := &SlackNotifier{WebhookURL: srv.URL}
	msg := Message{Title: "Deploy", Body: "v1.2.3 shipped"}
	if err := s.Notify(context.Background(), msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload map[string]string
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	wantText := "Deploy\nv1.2.3 shipped"
	if payload["text"] != wantText {
		t.Errorf("payload text = %q, want %q", payload["text"], wantText)
	}
}

func TestSlackNotifier_Notify_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	s := &SlackNotifier{WebhookURL: srv.URL}
	err := s.Notify(context.Background(), Message{Title: "t", Body: "b"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error = %q, want substring %q", err.Error(), "500")
	}
}

func TestSlackNotifier_Notify_ContentType(t *testing.T) {
	var gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := &SlackNotifier{WebhookURL: srv.URL}
	if err := s.Notify(context.Background(), Message{Title: "t", Body: "b"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", gotContentType, "application/json")
	}
}

func TestSlackNotifier_Notify_ContextCanceled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := &SlackNotifier{WebhookURL: srv.URL}
	err := s.Notify(ctx, Message{Title: "t", Body: "b"})
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
}

// --- GitHubNotifier tests ---

func TestGitHubNotifier_Notify_Success(t *testing.T) {
	var gotArgs []string
	withMockRun(t, func(_ context.Context, args ...string) error {
		gotArgs = args
		return nil
	})

	g := &GitHubNotifier{Repo: "org/repo", IssueNumber: 42}
	msg := Message{Title: "Build passed", Body: "All checks green"}
	if err := g.Notify(context.Background(), msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify args passed to gh CLI
	wantArgs := []string{"issue", "comment", "42", "--body", "Build passed\nAll checks green", "--repo", "org/repo"}
	if len(gotArgs) != len(wantArgs) {
		t.Fatalf("args len = %d, want %d: %v", len(gotArgs), len(wantArgs), gotArgs)
	}
	for i, got := range gotArgs {
		if got != wantArgs[i] {
			t.Errorf("args[%d] = %q, want %q", i, got, wantArgs[i])
		}
	}
}

func TestGitHubNotifier_Notify_NoRepo(t *testing.T) {
	var gotArgs []string
	withMockRun(t, func(_ context.Context, args ...string) error {
		gotArgs = args
		return nil
	})

	g := &GitHubNotifier{Repo: "", IssueNumber: 7}
	if err := g.Notify(context.Background(), Message{Title: "t", Body: "b"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not contain --repo flag
	for _, a := range gotArgs {
		if a == "--repo" {
			t.Error("expected no --repo flag when Repo is empty")
		}
	}
}

func TestGitHubNotifier_Notify_Error(t *testing.T) {
	withMockRun(t, func(_ context.Context, args ...string) error {
		return fmt.Errorf("gh: command failed")
	})

	g := &GitHubNotifier{Repo: "org/repo", IssueNumber: 1}
	err := g.Notify(context.Background(), Message{Title: "t", Body: "b"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "command failed") {
		t.Errorf("error = %q, want substring %q", err.Error(), "command failed")
	}
}
