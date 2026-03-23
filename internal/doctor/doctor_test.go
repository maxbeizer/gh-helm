package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maxbeizer/gh-helm/internal/github"
)

func withMockGh(t *testing.T, fn func(ctx context.Context, args ...string) ([]byte, error)) {
	t.Helper()
	orig := github.RunGhFunc
	github.RunGhFunc = fn
	t.Cleanup(func() { github.RunGhFunc = orig })
}

// --------------- summarize ---------------

func TestSummarize(t *testing.T) {
	tests := []struct {
		name   string
		checks []CheckResult
		want   Summary
	}{
		{
			name:   "empty list",
			checks: nil,
			want:   Summary{},
		},
		{
			name: "all pass",
			checks: []CheckResult{
				{Status: StatusPass},
				{Status: StatusPass},
			},
			want: Summary{Passed: 2},
		},
		{
			name: "mixed statuses",
			checks: []CheckResult{
				{Status: StatusPass},
				{Status: StatusWarn},
				{Status: StatusFail},
				{Status: StatusInfo},
				{Status: StatusPass},
				{Status: StatusFail},
			},
			want: Summary{Passed: 2, Warnings: 1, Failures: 2, Info: 1},
		},
		{
			name: "all info",
			checks: []CheckResult{
				{Status: StatusInfo},
				{Status: StatusInfo},
				{Status: StatusInfo},
			},
			want: Summary{Info: 3},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := summarize(tc.checks)
			if got != tc.want {
				t.Errorf("summarize() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// --------------- missingLabels ---------------

func TestMissingLabels(t *testing.T) {
	tests := []struct {
		name     string
		existing []string
		required []string
		want     []string
	}{
		{
			name:     "all present exact case",
			existing: []string{"agent-ready", "agent-in-progress", "bug"},
			required: []string{"agent-ready", "agent-in-progress"},
			want:     []string{},
		},
		{
			name:     "all present mixed case",
			existing: []string{"Agent-Ready", "AGENT-IN-PROGRESS"},
			required: []string{"agent-ready", "agent-in-progress"},
			want:     []string{},
		},
		{
			name:     "some missing",
			existing: []string{"agent-ready"},
			required: []string{"agent-ready", "agent-done", "needs-attention"},
			want:     []string{"'agent-done'", "'needs-attention'"},
		},
		{
			name:     "all missing",
			existing: []string{},
			required: []string{"agent-ready"},
			want:     []string{"'agent-ready'"},
		},
		{
			name:     "empty required",
			existing: []string{"bug", "feature"},
			required: []string{},
			want:     []string{},
		},
		{
			name:     "both empty",
			existing: nil,
			required: nil,
			want:     []string{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := missingLabels(tc.existing, tc.required)
			if len(got) != len(tc.want) {
				t.Fatalf("missingLabels() returned %d items %v, want %d items %v", len(got), got, len(tc.want), tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("missingLabels()[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// --------------- formatProjectBoardMessage ---------------

func TestFormatProjectBoardMessage(t *testing.T) {
	tests := []struct {
		number int
		count  int
		want   string
	}{
		{42, 10, "#42 accessible (10 items)"},
		{1, 0, "#1 accessible (0 items)"},
	}
	for _, tc := range tests {
		got := formatProjectBoardMessage(tc.number, tc.count)
		if got != tc.want {
			t.Errorf("formatProjectBoardMessage(%d, %d) = %q, want %q", tc.number, tc.count, got, tc.want)
		}
	}
}

// --------------- checkAuth ---------------

func TestCheckAuth(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		err        error
		wantStatus Status
		wantSubstr string // substring expected in Message
	}{
		{
			name:       "auth failure",
			output:     "",
			err:        fmt.Errorf("exit status 1"),
			wantStatus: StatusFail,
			wantSubstr: "not configured",
		},
		{
			name:       "no scopes line",
			output:     "Logged in to github.com\nAccount: user\n",
			err:        nil,
			wantStatus: StatusWarn,
			wantSubstr: "scopes not found",
		},
		{
			name:       "all scopes present",
			output:     "Logged in\nToken scopes: repo, read:org, admin:public_key\n",
			err:        nil,
			wantStatus: StatusPass,
			wantSubstr: "required scopes",
		},
		{
			name:       "missing read:org",
			output:     "Token scopes: repo, admin:public_key\n",
			err:        nil,
			wantStatus: StatusFail,
			wantSubstr: "read:org",
		},
		{
			name:       "missing repo",
			output:     "Token scopes: read:org\n",
			err:        nil,
			wantStatus: StatusFail,
			wantSubstr: "repo",
		},
		{
			name:       "missing both scopes",
			output:     "Token scopes: admin:public_key\n",
			err:        nil,
			wantStatus: StatusFail,
			wantSubstr: "repo",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			withMockGh(t, func(_ context.Context, args ...string) ([]byte, error) {
				return []byte(tc.output), tc.err
			})
			// Test checkAuth with mocked gh CLI output.
			got := checkAuth(context.Background())
			if got.Status != tc.wantStatus {
				t.Errorf("status = %q, want %q", got.Status, tc.wantStatus)
			}
			if !strings.Contains(got.Message, tc.wantSubstr) {
				t.Errorf("message = %q, want substring %q", got.Message, tc.wantSubstr)
			}
			if got.Key != "auth" {
				t.Errorf("key = %q, want %q", got.Key, "auth")
			}
		})
	}
}

// --------------- checkStateDir ---------------

func TestCheckStateDir(t *testing.T) {
	origDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origDir) })

	t.Run("dir exists", func(t *testing.T) {
		dir := t.TempDir()
		os.Mkdir(filepath.Join(dir, ".helm"), 0755)
		os.Chdir(dir)

		got := checkStateDir()
		if got.Status != StatusInfo {
			t.Errorf("status = %q, want %q", got.Status, StatusInfo)
		}
		if !strings.Contains(got.Message, "present") {
			t.Errorf("message = %q, want substring %q", got.Message, "present")
		}
	})

	t.Run("dir missing", func(t *testing.T) {
		dir := t.TempDir()
		os.Chdir(dir)

		got := checkStateDir()
		if got.Status != StatusInfo {
			t.Errorf("status = %q, want %q", got.Status, StatusInfo)
		}
		if !strings.Contains(got.Message, "not found") {
			t.Errorf("message = %q, want substring %q", got.Message, "not found")
		}
	})
}

// --------------- hasDevContainer ---------------

func TestHasDevContainer(t *testing.T) {
	origDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origDir) })

	t.Run("file exists", func(t *testing.T) {
		dir := t.TempDir()
		dcDir := filepath.Join(dir, ".devcontainer")
		os.Mkdir(dcDir, 0755)
		os.WriteFile(filepath.Join(dcDir, "devcontainer.json"), []byte("{}"), 0644)
		os.Chdir(dir)

		if !hasDevContainer() {
			t.Error("hasDevContainer() = false, want true")
		}
	})

	t.Run("file missing", func(t *testing.T) {
		dir := t.TempDir()
		os.Chdir(dir)

		if hasDevContainer() {
			t.Error("hasDevContainer() = true, want false")
		}
	})
}

// --------------- Run (integration) ---------------

func TestRun_ConfigMissing(t *testing.T) {
	origDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origDir) })

	dir := t.TempDir()
	os.Chdir(dir)

	// Mock gh calls so auth/repo checks don't hit real CLI
	withMockGh(t, func(_ context.Context, args ...string) ([]byte, error) {
		if len(args) > 0 && args[0] == "auth" {
			return []byte("Token scopes: repo, read:org\n"), nil
		}
		return nil, fmt.Errorf("mock: unexpected call %v", args)
	})

	result, err := Run(context.Background(), Options{})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	// First check must be config fail
	if len(result.Checks) == 0 {
		t.Fatal("expected at least one check")
	}
	if result.Checks[0].Key != "config" || result.Checks[0].Status != StatusFail {
		t.Errorf("first check = {Key:%q Status:%q}, want config/fail", result.Checks[0].Key, result.Checks[0].Status)
	}

	// The 4 config-dependent checks should be skipped (warn)
	skipped := 0
	for _, c := range result.Checks {
		if strings.Contains(c.Message, "skipped") {
			skipped++
		}
	}
	if skipped != 4 {
		t.Errorf("expected 4 skipped checks, got %d", skipped)
	}

	if result.Summary.Failures < 1 {
		t.Error("expected at least 1 failure in summary")
	}
}

func TestRun_ConfigPresent(t *testing.T) {
	origDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origDir) })

	dir := t.TempDir()
	os.Chdir(dir)

	// Write minimal valid helm.toml
	toml := `version = 1
source-of-truth = "docs/SOT.md"

[project]
owner = "testorg"
board = 99

[notifications]
webhook-url = "https://hooks.example.com/test"
`
	os.WriteFile(filepath.Join(dir, "helm.toml"), []byte(toml), 0644)

	// Create source of truth file
	os.MkdirAll(filepath.Join(dir, "docs"), 0755)
	os.WriteFile(filepath.Join(dir, "docs", "SOT.md"), []byte("# SOT"), 0644)

	// Create .helm dir
	os.Mkdir(filepath.Join(dir, ".helm"), 0755)

	// Create devcontainer
	os.MkdirAll(filepath.Join(dir, ".devcontainer"), 0755)
	os.WriteFile(filepath.Join(dir, ".devcontainer", "devcontainer.json"), []byte("{}"), 0644)

	type labelInfo struct {
		Name string `json:"name"`
	}

	withMockGh(t, func(_ context.Context, args ...string) ([]byte, error) {
		if len(args) == 0 {
			return nil, fmt.Errorf("no args")
		}
		switch args[0] {
		case "auth":
			return []byte("Token scopes: repo, read:org\n"), nil
		case "repo":
			return []byte("testorg/testrepo"), nil
		case "label":
			labels := []labelInfo{
				{Name: "agent-ready"},
				{Name: "agent-in-progress"},
				{Name: "agent-done"},
				{Name: "needs-attention"},
			}
			return json.Marshal(labels)
		default:
			// project board fetch uses "api" command
			if args[0] == "api" {
				resp := map[string]interface{}{
					"data": map[string]interface{}{
						"organization": map[string]interface{}{
							"projectV2": map[string]interface{}{
								"id": "PVT_123",
								"items": map[string]interface{}{
									"totalCount": 5,
								},
							},
						},
					},
				}
				return json.Marshal(resp)
			}
			return nil, fmt.Errorf("mock: unexpected %v", args)
		}
	})

	result, err := Run(context.Background(), Options{})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	// Verify key checks by key name
	found := map[string]Status{}
	for _, c := range result.Checks {
		found[c.Key] = c.Status
	}

	// Config should pass
	if found["config"] != StatusPass {
		t.Errorf("config status = %q, want pass", found["config"])
	}
	// Source of truth should pass
	if found["source_of_truth"] != StatusPass {
		t.Errorf("source_of_truth status = %q, want pass", found["source_of_truth"])
	}
	// Notifications should pass (webhook set)
	if found["notifications"] != StatusPass {
		t.Errorf("notifications status = %q, want pass", found["notifications"])
	}
	// Auth should pass
	if found["auth"] != StatusPass {
		t.Errorf("auth status = %q, want pass", found["auth"])
	}
	// Devcontainer should pass
	if found["devcontainer"] != StatusPass {
		t.Errorf("devcontainer status = %q, want pass", found["devcontainer"])
	}
	// State dir should be info present
	if found["state"] != StatusInfo {
		t.Errorf("state status = %q, want info", found["state"])
	}

	// Summary should have no failures
	if result.Summary.Failures != 0 {
		t.Errorf("summary.Failures = %d, want 0", result.Summary.Failures)
	}
}

func TestRun_MissingLabelsAndNoWebhook(t *testing.T) {
	origDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origDir) })

	dir := t.TempDir()
	os.Chdir(dir)

	toml := `version = 1
[project]
owner = "testorg"
board = 1
`
	os.WriteFile(filepath.Join(dir, "helm.toml"), []byte(toml), 0644)

	type labelInfo struct {
		Name string `json:"name"`
	}

	withMockGh(t, func(_ context.Context, args ...string) ([]byte, error) {
		if len(args) == 0 {
			return nil, fmt.Errorf("no args")
		}
		switch args[0] {
		case "auth":
			return []byte("Token scopes: repo, read:org\n"), nil
		case "repo":
			return []byte("testorg/testrepo"), nil
		case "label":
			// Only one of the required labels
			labels := []labelInfo{{Name: "agent-ready"}}
			return json.Marshal(labels)
		case "api":
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"organization": map[string]interface{}{
						"projectV2": map[string]interface{}{
							"id": "PVT_1",
							"items": map[string]interface{}{
								"totalCount": 0,
							},
						},
					},
				},
			}
			return json.Marshal(resp)
		default:
			return nil, fmt.Errorf("mock: unexpected %v", args)
		}
	})

	result, err := Run(context.Background(), Options{})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	found := map[string]CheckResult{}
	for _, c := range result.Checks {
		found[c.Key] = c
	}

	// Labels should warn about missing labels
	if found["labels"].Status != StatusWarn {
		t.Errorf("labels status = %q, want warn", found["labels"].Status)
	}
	if !strings.Contains(found["labels"].Message, "agent-in-progress") {
		t.Errorf("labels message = %q, want mention of missing label", found["labels"].Message)
	}

	// Notifications should warn (no webhook)
	if found["notifications"].Status != StatusWarn {
		t.Errorf("notifications status = %q, want warn", found["notifications"].Status)
	}

	// Source of truth should fail (default path missing)
	if found["source_of_truth"].Status != StatusFail {
		t.Errorf("source_of_truth status = %q, want fail", found["source_of_truth"].Status)
	}

	if result.Summary.Warnings < 2 {
		t.Errorf("summary.Warnings = %d, want >= 2", result.Summary.Warnings)
	}
}
