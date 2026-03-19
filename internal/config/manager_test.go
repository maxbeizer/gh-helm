package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadManager(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
		check   func(t *testing.T, cfg ManagerConfig)
	}{
		{
			name: "valid full config",
			content: `version = 1

[manager]
user = "lead"

[[projects]]
owner = "orgA"
board = 10
name = "ProjectA"

[[projects]]
owner = "orgB"
board = 20
name = "ProjectB"

[[team]]
handle = "alice"
one-one-repo = "org/one-on-one-alice"
pillars = ["reliability", "velocity"]

[[team]]
handle = "bob"
one-one-repo = "org/one-on-one-bob"
pillars = ["developer-experience"]

[pillars.reliability]
description = "Keep things running"
signals = ["bug fixes"]
repos = ["org/monitoring"]
labels = ["bug"]
paths = ["tests/**"]

[pillars.velocity]
description = "Ship faster"
signals = ["PRs merged"]
labels = ["feature"]

[notifications]
channel = "#mgr"
ops-channel = "#mgr-ops"
webhook-url = "https://mgr.hook"

[schedule]
pulse = "0 9 * * 1"
prep = "0 8 * * 5"
observe = "30 14 * * *"
`,
			check: func(t *testing.T, cfg ManagerConfig) {
				t.Helper()
				if cfg.Version != 1 {
					t.Errorf("Version = %d, want 1", cfg.Version)
				}
				if cfg.Manager.User != "lead" {
					t.Errorf("Manager.User = %q, want %q", cfg.Manager.User, "lead")
				}
				if len(cfg.Projects) != 2 {
					t.Fatalf("len(Projects) = %d, want 2", len(cfg.Projects))
				}
				if cfg.Projects[0].Owner != "orgA" || cfg.Projects[0].Board != 10 {
					t.Errorf("Projects[0] = %+v", cfg.Projects[0])
				}
				if len(cfg.Team) != 2 {
					t.Fatalf("len(Team) = %d, want 2", len(cfg.Team))
				}
				if cfg.Team[0].Handle != "alice" {
					t.Errorf("Team[0].Handle = %q, want %q", cfg.Team[0].Handle, "alice")
				}
				if len(cfg.Pillars) != 2 {
					t.Fatalf("len(Pillars) = %d, want 2", len(cfg.Pillars))
				}
				rel, ok := cfg.Pillars["reliability"]
				if !ok {
					t.Fatal("missing reliability pillar")
				}
				if rel.Description != "Keep things running" {
					t.Errorf("reliability.Description = %q", rel.Description)
				}
				if cfg.Schedule.Pulse != "0 9 * * 1" {
					t.Errorf("Schedule.Pulse = %q", cfg.Schedule.Pulse)
				}
			},
		},
		{
			name:    "missing version",
			content: "[manager]\nuser = \"x\"\n",
			wantErr: "missing 'version' field",
		},
		{
			name:    "wrong version with migration message",
			content: "version = 99\n[manager]\nuser = \"x\"\n",
			wantErr: "gh helm upgrade",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "helm-manager.toml")
			if err := os.WriteFile(path, []byte(tc.content), 0644); err != nil {
				t.Fatalf("write temp file: %v", err)
			}

			cfg, err := LoadManager(path)
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

func TestLoadManager_MissingFile(t *testing.T) {
	_, err := LoadManager("/nonexistent/path/helm-manager.toml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestWriteManagerLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "helm-manager.toml")

	original := ManagerConfig{
		Version: CurrentManagerConfigVersion,
		Manager: ManagerSettings{User: "roundtrip-lead"},
		Projects: []ManagerProject{
			{Owner: "org1", Board: 100, Name: "P1"},
		},
		Team: []TeamMember{
			{Handle: "dev1", OneOneRepo: "org/1on1-dev1", Pillars: []string{"reliability"}},
		},
		Pillars: map[string]PillarConfig{
			"reliability": {
				Description: "Uptime",
				Signals:     []string{"incidents"},
				Repos:       []string{"org/infra"},
				Labels:      []string{"sev1"},
			},
		},
		Notifications: NotificationsConfig{
			Channel: "#rt-channel",
		},
		Schedule: ManagerSchedule{
			Pulse: "0 9 * * 1",
		},
	}

	if err := WriteManager(path, original); err != nil {
		t.Fatalf("WriteManager: %v", err)
	}

	loaded, err := LoadManager(path)
	if err != nil {
		t.Fatalf("LoadManager: %v", err)
	}

	if loaded.Version != original.Version {
		t.Errorf("Version = %d, want %d", loaded.Version, original.Version)
	}
	if loaded.Manager.User != original.Manager.User {
		t.Errorf("Manager.User = %q, want %q", loaded.Manager.User, original.Manager.User)
	}
	if len(loaded.Projects) != 1 || loaded.Projects[0].Owner != "org1" {
		t.Errorf("Projects = %+v", loaded.Projects)
	}
	if len(loaded.Team) != 1 || loaded.Team[0].Handle != "dev1" {
		t.Errorf("Team = %+v", loaded.Team)
	}
	rel, ok := loaded.Pillars["reliability"]
	if !ok {
		t.Fatal("missing reliability pillar after roundtrip")
	}
	if rel.Description != "Uptime" {
		t.Errorf("reliability.Description = %q, want %q", rel.Description, "Uptime")
	}
}
