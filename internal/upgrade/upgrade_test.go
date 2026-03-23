package upgrade

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maxbeizer/gh-helm/internal/config"
	"github.com/maxbeizer/gh-helm/internal/github"
)

func TestMergeDefaults(t *testing.T) {
	tests := []struct {
		name               string
		cfg                config.Config
		sourceOfTruthMissing bool
		wantChanged        bool
		check              func(t *testing.T, cfg config.Config)
	}{
		{
			name:               "empty config gets all defaults",
			cfg:                config.Config{},
			sourceOfTruthMissing: true,
			wantChanged:        true,
			check: func(t *testing.T, cfg config.Config) {
				t.Helper()
				if cfg.Agent.Model != "gpt-4.1" {
					t.Errorf("Model = %q, want %q", cfg.Agent.Model, "gpt-4.1")
				}
				if cfg.Agent.MaxPerHour != 3 {
					t.Errorf("MaxPerHour = %d, want 3", cfg.Agent.MaxPerHour)
				}
				if cfg.Notifications.Channel != "slack" {
					t.Errorf("Channel = %q, want %q", cfg.Notifications.Channel, "slack")
				}
				if cfg.Notifications.OpsChannel != "#project-alpha-ops" {
					t.Errorf("OpsChannel = %q, want %q", cfg.Notifications.OpsChannel, "#project-alpha-ops")
				}
				if cfg.SourceOfTruth != "docs/SOURCE_OF_TRUTH.md" {
					t.Errorf("SourceOfTruth = %q, want %q", cfg.SourceOfTruth, "docs/SOURCE_OF_TRUTH.md")
				}
				if cfg.Filters.Status != "Ready" {
					t.Errorf("Filters.Status = %q, want %q", cfg.Filters.Status, "Ready")
				}
				if len(cfg.Filters.Labels) != 1 || cfg.Filters.Labels[0] != "agent-ready" {
					t.Errorf("Filters.Labels = %v, want [agent-ready]", cfg.Filters.Labels)
				}
			},
		},
		{
			name: "partially filled config only fills missing",
			cfg: config.Config{
				Agent: config.AgentConfig{
					Model:      "claude-sonnet",
					MaxPerHour: 10,
				},
				Notifications: config.NotificationsConfig{
					Channel: "teams",
				},
				SourceOfTruth: "my/SOT.md",
				Filters: config.FiltersConfig{
					Status: "Open",
				},
			},
			sourceOfTruthMissing: false,
			wantChanged:        true,
			check: func(t *testing.T, cfg config.Config) {
				t.Helper()
				if cfg.Agent.Model != "claude-sonnet" {
					t.Errorf("Model should be preserved, got %q", cfg.Agent.Model)
				}
				if cfg.Agent.MaxPerHour != 10 {
					t.Errorf("MaxPerHour should be preserved, got %d", cfg.Agent.MaxPerHour)
				}
				if cfg.Notifications.Channel != "teams" {
					t.Errorf("Channel should be preserved, got %q", cfg.Notifications.Channel)
				}
				// OpsChannel was missing, should be filled
				if cfg.Notifications.OpsChannel != "#project-alpha-ops" {
					t.Errorf("OpsChannel = %q, want %q", cfg.Notifications.OpsChannel, "#project-alpha-ops")
				}
				if cfg.SourceOfTruth != "my/SOT.md" {
					t.Errorf("SourceOfTruth should be preserved, got %q", cfg.SourceOfTruth)
				}
				if cfg.Filters.Status != "Open" {
					t.Errorf("Filters.Status should be preserved, got %q", cfg.Filters.Status)
				}
				// Labels was missing, should be filled
				if len(cfg.Filters.Labels) != 1 || cfg.Filters.Labels[0] != "agent-ready" {
					t.Errorf("Filters.Labels = %v, want [agent-ready]", cfg.Filters.Labels)
				}
			},
		},
		{
			name: "fully filled config unchanged",
			cfg: config.Config{
				Agent: config.AgentConfig{
					Model:      "gpt-4",
					MaxPerHour: 5,
				},
				Notifications: config.NotificationsConfig{
					Channel:    "slack",
					OpsChannel: "#ops",
				},
				SourceOfTruth: "docs/SOT.md",
				Filters: config.FiltersConfig{
					Status: "Todo",
					Labels: []string{"ready"},
				},
			},
			sourceOfTruthMissing: false,
			wantChanged:        false,
			check: func(t *testing.T, cfg config.Config) {
				t.Helper()
				if cfg.Agent.Model != "gpt-4" {
					t.Errorf("Model = %q, want gpt-4", cfg.Agent.Model)
				}
			},
		},
		{
			name: "source-of-truth set but missing from TOML",
			cfg: config.Config{
				Agent: config.AgentConfig{
					Model:      "gpt-4",
					MaxPerHour: 5,
				},
				Notifications: config.NotificationsConfig{
					Channel:    "slack",
					OpsChannel: "#ops",
				},
				SourceOfTruth: "docs/SOT.md",
				Filters: config.FiltersConfig{
					Status: "Todo",
					Labels: []string{"ready"},
				},
			},
			sourceOfTruthMissing: true,
			wantChanged:        true,
			check: func(t *testing.T, cfg config.Config) {
				t.Helper()
				// When sourceOfTruthMissing is true, it gets overwritten to default
				if cfg.SourceOfTruth != "docs/SOURCE_OF_TRUTH.md" {
					t.Errorf("SourceOfTruth = %q, want default", cfg.SourceOfTruth)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed := mergeDefaults(tt.cfg, tt.sourceOfTruthMissing)
			if changed != tt.wantChanged {
				t.Errorf("changed = %v, want %v", changed, tt.wantChanged)
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestMissingLabels(t *testing.T) {
	tests := []struct {
		name     string
		existing []string
		required []string
		want     []string
	}{
		{
			name:     "all labels present",
			existing: []string{"agent-ready", "agent-in-progress", "agent-done", "needs-attention"},
			required: []string{"agent-ready", "agent-in-progress", "agent-done", "needs-attention"},
			want:     []string{},
		},
		{
			name:     "some missing",
			existing: []string{"agent-ready", "bug"},
			required: []string{"agent-ready", "agent-in-progress", "needs-attention"},
			want:     []string{"agent-in-progress", "needs-attention"},
		},
		{
			name:     "case insensitive comparison",
			existing: []string{"Agent-Ready", "AGENT-IN-PROGRESS"},
			required: []string{"agent-ready", "agent-in-progress"},
			want:     []string{},
		},
		{
			name:     "all missing",
			existing: []string{},
			required: []string{"agent-ready", "needs-attention"},
			want:     []string{"agent-ready", "needs-attention"},
		},
		{
			name:     "empty required",
			existing: []string{"agent-ready"},
			required: []string{},
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := missingLabels(tt.existing, tt.required)
			if len(got) != len(tt.want) {
				t.Fatalf("missingLabels() returned %v, want %v", got, tt.want)
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("missingLabels()[%d] = %q, want %q", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestConfigFieldMissing(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name    string
		content string
		field   string
		want    bool
	}{
		{
			name:    "field present",
			content: "version = 1\nsource-of-truth = \"docs/SOT.md\"\n",
			field:   "source-of-truth",
			want:    false,
		},
		{
			name:    "field missing",
			content: "version = 1\n",
			field:   "source-of-truth",
			want:    true,
		},
		{
			name:    "different field present",
			content: "version = 1\n[agent]\nmodel = \"gpt-4\"\n",
			field:   "agent",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(dir, tt.name+".toml")
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}
			got := configFieldMissing(path, tt.field)
			if got != tt.want {
				t.Errorf("configFieldMissing(%q) = %v, want %v", tt.field, got, tt.want)
			}
		})
	}

	t.Run("nonexistent file returns false", func(t *testing.T) {
		got := configFieldMissing(filepath.Join(dir, "nope.toml"), "field")
		if got != false {
			t.Error("expected false for nonexistent file")
		}
	})

	t.Run("invalid TOML returns false", func(t *testing.T) {
		path := filepath.Join(dir, "bad.toml")
		if err := os.WriteFile(path, []byte("{{invalid"), 0o644); err != nil {
			t.Fatal(err)
		}
		got := configFieldMissing(path, "field")
		if got != false {
			t.Error("expected false for invalid TOML")
		}
	})
}

func TestEnsureStateDir(t *testing.T) {
	t.Run("creates .helm directory", func(t *testing.T) {
		dir := t.TempDir()
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		os.Chdir(dir)

		var changes []Change
		if err := ensureStateDir(false, &changes); err != nil {
			t.Fatalf("ensureStateDir: %v", err)
		}

		if _, err := os.Stat(filepath.Join(dir, ".helm")); os.IsNotExist(err) {
			t.Error(".helm directory was not created")
		}
		if len(changes) != 1 || changes[0].Status != StatusApplied {
			t.Errorf("expected 1 applied change, got %v", changes)
		}
	})

	t.Run("skips when already exists", func(t *testing.T) {
		dir := t.TempDir()
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		os.Chdir(dir)

		os.MkdirAll(filepath.Join(dir, ".helm"), 0o755)

		var changes []Change
		if err := ensureStateDir(false, &changes); err != nil {
			t.Fatalf("ensureStateDir: %v", err)
		}

		if len(changes) != 1 || changes[0].Status != StatusSkipped {
			t.Errorf("expected 1 skipped change, got %v", changes)
		}
	})

	t.Run("dry run does not create", func(t *testing.T) {
		dir := t.TempDir()
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		os.Chdir(dir)

		var changes []Change
		if err := ensureStateDir(true, &changes); err != nil {
			t.Fatalf("ensureStateDir: %v", err)
		}

		if _, err := os.Stat(filepath.Join(dir, ".helm")); !os.IsNotExist(err) {
			t.Error(".helm directory should not be created in dry run")
		}
		if len(changes) != 1 || changes[0].Status != StatusSkipped {
			t.Errorf("expected 1 skipped change, got %v", changes)
		}
	})
}

func TestEnsureDevcontainer(t *testing.T) {
	t.Run("creates devcontainer.json", func(t *testing.T) {
		dir := t.TempDir()
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		os.Chdir(dir)

		var changes []Change
		if err := ensureDevcontainer(false, &changes); err != nil {
			t.Fatalf("ensureDevcontainer: %v", err)
		}

		path := filepath.Join(dir, ".devcontainer", "devcontainer.json")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("devcontainer.json not created: %v", err)
		}
		if !strings.Contains(string(data), "gh-helm-daemon") {
			t.Error("devcontainer.json missing expected content")
		}
		if len(changes) != 1 || changes[0].Status != StatusApplied {
			t.Errorf("expected 1 applied change, got %v", changes)
		}
	})

	t.Run("skips when already exists", func(t *testing.T) {
		dir := t.TempDir()
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		os.Chdir(dir)

		dcDir := filepath.Join(dir, ".devcontainer")
		os.MkdirAll(dcDir, 0o755)
		os.WriteFile(filepath.Join(dcDir, "devcontainer.json"), []byte("{}"), 0o644)

		var changes []Change
		if err := ensureDevcontainer(false, &changes); err != nil {
			t.Fatalf("ensureDevcontainer: %v", err)
		}

		if len(changes) != 1 || changes[0].Status != StatusSkipped {
			t.Errorf("expected 1 skipped change, got %v", changes)
		}
	})

	t.Run("dry run does not create", func(t *testing.T) {
		dir := t.TempDir()
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		os.Chdir(dir)

		var changes []Change
		if err := ensureDevcontainer(true, &changes); err != nil {
			t.Fatalf("ensureDevcontainer: %v", err)
		}

		path := filepath.Join(dir, ".devcontainer", "devcontainer.json")
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Error("devcontainer.json should not be created in dry run")
		}
		if len(changes) != 1 || changes[0].Status != StatusSkipped {
			t.Errorf("expected 1 skipped change, got %v", changes)
		}
	})
}

func TestEnsureSourceOfTruth(t *testing.T) {
	t.Run("creates source of truth file", func(t *testing.T) {
		dir := t.TempDir()
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		os.Chdir(dir)

		// Write a helm.toml so ensureSourceOfTruth can load config
		cfgContent := "version = 1\nsource-of-truth = \"docs/SOURCE_OF_TRUTH.md\"\n"
		os.WriteFile(filepath.Join(dir, "helm.toml"), []byte(cfgContent), 0o644)

		cfg := config.Config{SourceOfTruth: "docs/SOURCE_OF_TRUTH.md"}
		var changes []Change
		if err := ensureSourceOfTruth(&cfg, false, &changes); err != nil {
			t.Fatalf("ensureSourceOfTruth: %v", err)
		}

		path := filepath.Join(dir, "docs", "SOURCE_OF_TRUTH.md")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("SOURCE_OF_TRUTH.md not created: %v", err)
		}
		if !strings.Contains(string(data), "# Source of Truth") {
			t.Error("SOURCE_OF_TRUTH.md missing expected content")
		}
		if len(changes) != 1 || changes[0].Status != StatusApplied {
			t.Errorf("expected 1 applied change, got %v", changes)
		}
	})

	t.Run("skips when file already exists", func(t *testing.T) {
		dir := t.TempDir()
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		os.Chdir(dir)

		cfgContent := "version = 1\nsource-of-truth = \"docs/SOT.md\"\n"
		os.WriteFile(filepath.Join(dir, "helm.toml"), []byte(cfgContent), 0o644)
		os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
		os.WriteFile(filepath.Join(dir, "docs", "SOT.md"), []byte("existing"), 0o644)

		cfg := config.Config{SourceOfTruth: "docs/SOT.md"}
		var changes []Change
		if err := ensureSourceOfTruth(&cfg, false, &changes); err != nil {
			t.Fatalf("ensureSourceOfTruth: %v", err)
		}

		if len(changes) != 1 || changes[0].Status != StatusSkipped {
			t.Errorf("expected 1 skipped change, got %v", changes)
		}
	})

	t.Run("skips when no config", func(t *testing.T) {
		dir := t.TempDir()
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		os.Chdir(dir)

		var changes []Change
		if err := ensureSourceOfTruth(nil, false, &changes); err != nil {
			t.Fatalf("ensureSourceOfTruth: %v", err)
		}

		if len(changes) != 1 || changes[0].Status != StatusSkipped {
			t.Errorf("expected 1 skipped change, got %v", changes)
		}
		if !strings.Contains(changes[0].Message, "config missing") {
			t.Errorf("expected config missing message, got %q", changes[0].Message)
		}
	})

	t.Run("dry run does not create", func(t *testing.T) {
		dir := t.TempDir()
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		os.Chdir(dir)

		cfgContent := "version = 1\nsource-of-truth = \"docs/SOURCE_OF_TRUTH.md\"\n"
		os.WriteFile(filepath.Join(dir, "helm.toml"), []byte(cfgContent), 0o644)

		cfg := config.Config{SourceOfTruth: "docs/SOURCE_OF_TRUTH.md"}
		var changes []Change
		if err := ensureSourceOfTruth(&cfg, true, &changes); err != nil {
			t.Fatalf("ensureSourceOfTruth: %v", err)
		}

		path := filepath.Join(dir, "docs", "SOURCE_OF_TRUTH.md")
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Error("SOURCE_OF_TRUTH.md should not be created in dry run")
		}
		if len(changes) != 1 || changes[0].Status != StatusSkipped {
			t.Errorf("expected 1 skipped change, got %v", changes)
		}
	})
}

func TestRun_DryRun(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(dir)

	// Mock gh CLI calls
	origRunGh := github.RunGhFunc
	defer func() { github.RunGhFunc = origRunGh }()

	github.RunGhFunc = func(ctx context.Context, args ...string) ([]byte, error) {
		if len(args) >= 2 && args[0] == "repo" && args[1] == "view" {
			return []byte("owner/repo"), nil
		}
		if len(args) >= 2 && args[0] == "label" && args[1] == "list" {
			return []byte(`[{"name":"bug"}]`), nil
		}
		return nil, nil
	}

	// Write config so config loading works
	cfgContent := "version = 1\n"
	os.WriteFile(filepath.Join(dir, "helm.toml"), []byte(cfgContent), 0o644)

	result, err := Run(context.Background(), Options{DryRun: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.Applied != 0 {
		t.Errorf("DryRun should apply 0 changes, got %d", result.Applied)
	}

	// Should have skipped changes for labels, config, devcontainer, SOT, state
	if len(result.Changes) == 0 {
		t.Error("expected changes to be reported")
	}

	// Verify no directories/files were created
	if _, err := os.Stat(filepath.Join(dir, ".helm")); !os.IsNotExist(err) {
		t.Error(".helm should not be created in dry run")
	}
	if _, err := os.Stat(filepath.Join(dir, ".devcontainer")); !os.IsNotExist(err) {
		t.Error(".devcontainer should not be created in dry run")
	}
}

func TestRun_AppliesChanges(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(dir)

	// Track gh CLI calls
	var createdLabels []string
	origRunGh := github.RunGhFunc
	defer func() { github.RunGhFunc = origRunGh }()

	github.RunGhFunc = func(ctx context.Context, args ...string) ([]byte, error) {
		if len(args) >= 2 && args[0] == "repo" && args[1] == "view" {
			return []byte("owner/repo"), nil
		}
		if len(args) >= 2 && args[0] == "label" && args[1] == "list" {
			// Return only one of the required labels
			return []byte(`[{"name":"agent-ready"}]`), nil
		}
		if len(args) >= 2 && args[0] == "label" && args[1] == "create" {
			createdLabels = append(createdLabels, args[2])
			return nil, nil
		}
		return nil, nil
	}

	// Write config with all required fields
	cfgContent := `version = 1
source-of-truth = "docs/SOURCE_OF_TRUTH.md"

[project]
board = 1
owner = "test-owner"
`
	os.WriteFile(filepath.Join(dir, "helm.toml"), []byte(cfgContent), 0o644)

	result, err := Run(context.Background(), Options{DryRun: false})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Should have created missing labels
	expectedLabels := []string{"agent-in-progress", "agent-done", "needs-attention"}
	if len(createdLabels) != len(expectedLabels) {
		t.Fatalf("created %d labels, want %d: %v", len(createdLabels), len(expectedLabels), createdLabels)
	}
	for i, label := range createdLabels {
		if label != expectedLabels[i] {
			t.Errorf("created label %d = %q, want %q", i, label, expectedLabels[i])
		}
	}

	// Should have created .helm directory
	if _, err := os.Stat(filepath.Join(dir, ".helm")); os.IsNotExist(err) {
		t.Error(".helm directory should be created")
	}

	// Should have created devcontainer
	if _, err := os.Stat(filepath.Join(dir, ".devcontainer", "devcontainer.json")); os.IsNotExist(err) {
		t.Error("devcontainer.json should be created")
	}

	// Should have created SOURCE_OF_TRUTH.md
	if _, err := os.Stat(filepath.Join(dir, "docs", "SOURCE_OF_TRUTH.md")); os.IsNotExist(err) {
		t.Error("SOURCE_OF_TRUTH.md should be created")
	}

	if result.Applied == 0 {
		t.Error("expected some applied changes")
	}
}

func TestRun_NoRepo(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(dir)

	origRunGh := github.RunGhFunc
	defer func() { github.RunGhFunc = origRunGh }()

	github.RunGhFunc = func(ctx context.Context, args ...string) ([]byte, error) {
		if len(args) >= 2 && args[0] == "repo" && args[1] == "view" {
			return nil, &os.PathError{Op: "exec", Err: os.ErrNotExist}
		}
		return nil, nil
	}

	// No helm.toml → config skipped too
	result, err := Run(context.Background(), Options{DryRun: false})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Labels should be skipped with "repo not detected"
	foundRepoSkip := false
	for _, c := range result.Changes {
		if strings.Contains(c.Message, "repo not detected") {
			foundRepoSkip = true
			break
		}
	}
	if !foundRepoSkip {
		t.Error("expected 'repo not detected' skip message")
	}
}

func TestResultCounts(t *testing.T) {
	// Verify that Result.Applied and Result.Skipped are counted correctly
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(dir)

	origRunGh := github.RunGhFunc
	defer func() { github.RunGhFunc = origRunGh }()

	github.RunGhFunc = func(ctx context.Context, args ...string) ([]byte, error) {
		if len(args) >= 2 && args[0] == "repo" && args[1] == "view" {
			return []byte("owner/repo"), nil
		}
		if len(args) >= 2 && args[0] == "label" && args[1] == "list" {
			return []byte(`[{"name":"agent-ready"},{"name":"agent-in-progress"},{"name":"agent-done"},{"name":"needs-attention"}]`), nil
		}
		return nil, nil
	}

	// No helm.toml
	result, err := Run(context.Background(), Options{DryRun: false})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	total := result.Applied + result.Skipped
	if total != len(result.Changes) {
		t.Errorf("Applied(%d) + Skipped(%d) = %d, want %d (len(Changes))",
			result.Applied, result.Skipped, total, len(result.Changes))
	}
}
