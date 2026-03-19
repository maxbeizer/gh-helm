package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
		check   func(t *testing.T, cfg Config)
	}{
		{
			name: "valid config with all fields",
			content: `version = 1
source-of-truth = "docs/SOT.md"

[project]
board = 25
owner = "myorg"

[agent]
user = "testuser"
model = "gpt-4"
max-per-hour = 10

[notifications]
channel = "slack"
ops-channel = "#ops"
webhook-url = "https://example.com/hook"

[filters]
status = "open"
labels = ["bug", "feature"]
`,
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.Version != 1 {
					t.Errorf("Version = %d, want 1", cfg.Version)
				}
				if cfg.Project.Board != 25 {
					t.Errorf("Board = %d, want 25", cfg.Project.Board)
				}
				if cfg.Project.Owner != "myorg" {
					t.Errorf("Owner = %q, want %q", cfg.Project.Owner, "myorg")
				}
				if cfg.Agent.User != "testuser" {
					t.Errorf("User = %q, want %q", cfg.Agent.User, "testuser")
				}
				if cfg.Agent.Model != "gpt-4" {
					t.Errorf("Model = %q, want %q", cfg.Agent.Model, "gpt-4")
				}
				if cfg.Agent.MaxPerHour != 10 {
					t.Errorf("MaxPerHour = %d, want 10", cfg.Agent.MaxPerHour)
				}
				if cfg.Notifications.Channel != "slack" {
					t.Errorf("Channel = %q, want %q", cfg.Notifications.Channel, "slack")
				}
				if cfg.Notifications.OpsChannel != "#ops" {
					t.Errorf("OpsChannel = %q, want %q", cfg.Notifications.OpsChannel, "#ops")
				}
				if cfg.Notifications.WebhookURL != "https://example.com/hook" {
					t.Errorf("WebhookURL = %q, want %q", cfg.Notifications.WebhookURL, "https://example.com/hook")
				}
				if cfg.SourceOfTruth != "docs/SOT.md" {
					t.Errorf("SourceOfTruth = %q, want %q", cfg.SourceOfTruth, "docs/SOT.md")
				}
				if cfg.Filters.Status != "open" {
					t.Errorf("Filters.Status = %q, want %q", cfg.Filters.Status, "open")
				}
				if len(cfg.Filters.Labels) != 2 || cfg.Filters.Labels[0] != "bug" || cfg.Filters.Labels[1] != "feature" {
					t.Errorf("Filters.Labels = %v, want [bug feature]", cfg.Filters.Labels)
				}
			},
		},
		{
			name:    "missing version field",
			content: "[project]\nboard = 1\nowner = \"x\"\n",
			wantErr: "missing 'version' field",
		},
		{
			name:    "wrong version",
			content: "version = 99\n[project]\nboard = 1\nowner = \"x\"\n",
			wantErr: "gh helm upgrade",
		},
		{
			name:    "empty source-of-truth defaults",
			content: "version = 1\n[project]\nboard = 1\nowner = \"x\"\n",
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.SourceOfTruth != "docs/SOURCE_OF_TRUTH.md" {
					t.Errorf("SourceOfTruth = %q, want default %q", cfg.SourceOfTruth, "docs/SOURCE_OF_TRUTH.md")
				}
			},
		},
		{
			name: "valid minimal config",
			content: `version = 1
[project]
board = 42
owner = "minimal-org"
`,
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.Version != 1 {
					t.Errorf("Version = %d, want 1", cfg.Version)
				}
				if cfg.Project.Board != 42 {
					t.Errorf("Board = %d, want 42", cfg.Project.Board)
				}
				if cfg.Project.Owner != "minimal-org" {
					t.Errorf("Owner = %q, want %q", cfg.Project.Owner, "minimal-org")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "helm.toml")
			if err := os.WriteFile(path, []byte(tc.content), 0644); err != nil {
				t.Fatalf("write temp file: %v", err)
			}

			cfg, err := Load(path)
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
			if tc.check != nil {
				tc.check(t, cfg)
			}
		})
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/helm.toml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestWriteLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "helm.toml")

	original := Config{
		Version: CurrentConfigVersion,
		Project: ProjectConfig{
			Board: 99,
			Owner: "roundtrip-org",
		},
		Agent: AgentConfig{
			User:       "alice",
			Model:      "gpt-4",
			MaxPerHour: 5,
		},
		Notifications: NotificationsConfig{
			Channel:    "slack",
			OpsChannel: "#test-ops",
			WebhookURL: "https://hook.example.com",
		},
		SourceOfTruth: "docs/CUSTOM_SOT.md",
		Filters: FiltersConfig{
			Status: "closed",
			Labels: []string{"p1", "p2"},
		},
	}

	if err := Write(path, original); err != nil {
		t.Fatalf("Write: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Version != original.Version {
		t.Errorf("Version = %d, want %d", loaded.Version, original.Version)
	}
	if loaded.Project.Board != original.Project.Board {
		t.Errorf("Board = %d, want %d", loaded.Project.Board, original.Project.Board)
	}
	if loaded.Project.Owner != original.Project.Owner {
		t.Errorf("Owner = %q, want %q", loaded.Project.Owner, original.Project.Owner)
	}
	if loaded.Agent.User != original.Agent.User {
		t.Errorf("User = %q, want %q", loaded.Agent.User, original.Agent.User)
	}
	if loaded.Agent.Model != original.Agent.Model {
		t.Errorf("Model = %q, want %q", loaded.Agent.Model, original.Agent.Model)
	}
	if loaded.Agent.MaxPerHour != original.Agent.MaxPerHour {
		t.Errorf("MaxPerHour = %d, want %d", loaded.Agent.MaxPerHour, original.Agent.MaxPerHour)
	}
	if loaded.SourceOfTruth != original.SourceOfTruth {
		t.Errorf("SourceOfTruth = %q, want %q", loaded.SourceOfTruth, original.SourceOfTruth)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name: "valid minimal",
			cfg: Config{
				Version: 1,
				Project: ProjectConfig{Board: 1, Owner: "org"},
			},
		},
		{
			name: "board zero",
			cfg: Config{
				Version: 1,
				Project: ProjectConfig{Board: 0, Owner: "org"},
			},
			wantErr: "project.board must be greater than 0",
		},
		{
			name: "board negative",
			cfg: Config{
				Version: 1,
				Project: ProjectConfig{Board: -1, Owner: "org"},
			},
			wantErr: "project.board must be greater than 0",
		},
		{
			name: "owner empty",
			cfg: Config{
				Version: 1,
				Project: ProjectConfig{Board: 1, Owner: ""},
			},
			wantErr: "project.owner must be non-empty",
		},
		{
			name: "negative max-per-hour",
			cfg: Config{
				Version: 1,
				Project: ProjectConfig{Board: 1, Owner: "org"},
				Agent:   AgentConfig{MaxPerHour: -1},
			},
			wantErr: "agent.max-per-hour must be >= 0",
		},
		{
			name: "zero max-per-hour is valid",
			cfg: Config{
				Version: 1,
				Project: ProjectConfig{Board: 1, Owner: "org"},
				Agent:   AgentConfig{MaxPerHour: 0},
			},
		},
		{
			name: "invalid channel",
			cfg: Config{
				Version:       1,
				Project:       ProjectConfig{Board: 1, Owner: "org"},
				Notifications: NotificationsConfig{Channel: "email"},
			},
			wantErr: "notifications.channel must be one of",
		},
		{
			name: "slack channel without webhook",
			cfg: Config{
				Version:       1,
				Project:       ProjectConfig{Board: 1, Owner: "org"},
				Notifications: NotificationsConfig{Channel: "slack", WebhookURL: ""},
			},
			wantErr: "notifications.webhook-url is required",
		},
		{
			name: "slack channel with webhook",
			cfg: Config{
				Version:       1,
				Project:       ProjectConfig{Board: 1, Owner: "org"},
				Notifications: NotificationsConfig{Channel: "slack", WebhookURL: "https://hooks.slack.com/xxx"},
			},
		},
		{
			name: "github channel valid",
			cfg: Config{
				Version:       1,
				Project:       ProjectConfig{Board: 1, Owner: "org"},
				Notifications: NotificationsConfig{Channel: "github"},
			},
		},
		{
			name: "empty channel valid",
			cfg: Config{
				Version: 1,
				Project: ProjectConfig{Board: 1, Owner: "org"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
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
