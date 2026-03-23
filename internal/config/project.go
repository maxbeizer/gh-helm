package config

import (
"fmt"
"os"

"github.com/BurntSushi/toml"
)

const CurrentConfigVersion = 1

type Config struct {
Version       int                 `toml:"version"`
Project       ProjectConfig       `toml:"project"`
Agent         AgentConfig         `toml:"agent"`
Notifications NotificationsConfig `toml:"notifications"`
SourceOfTruth string              `toml:"source-of-truth"`
Filters       FiltersConfig       `toml:"filters"`
Codespace     CodespaceConfig     `toml:"codespace"`
}

type ProjectConfig struct {
Board int    `toml:"board"`
Owner string `toml:"owner"`
}

type AgentConfig struct {
User       string `toml:"user"`
Model      string `toml:"model"`
MaxPerHour int    `toml:"max-per-hour"`
}

type NotificationsConfig struct {
Channel    string `toml:"channel"`
OpsChannel string `toml:"ops-channel"`
WebhookURL string `toml:"webhook-url"`
}

type FiltersConfig struct {
Status string   `toml:"status"`
Labels []string `toml:"labels"`
}

type CodespaceConfig struct {
Enabled     bool   `toml:"enabled"`
Machine     string `toml:"machine"`
IdleTimeout string `toml:"idle-timeout"`
}

func Load(path string) (Config, error) {
var cfg Config
if _, err := toml.DecodeFile(path, &cfg); err != nil {
return Config{}, err
}
if cfg.Version == 0 {
return Config{}, fmt.Errorf("missing 'version' field in %s, expected version = %d", path, CurrentConfigVersion)
}
if cfg.Version != CurrentConfigVersion {
return Config{}, fmt.Errorf("unsupported config version %d in %s (expected %d), run 'gh helm upgrade' to migrate", cfg.Version, path, CurrentConfigVersion)
}
if cfg.SourceOfTruth == "" {
cfg.SourceOfTruth = "docs/SOURCE_OF_TRUTH.md"
}
if err := cfg.Validate(); err != nil {
return Config{}, fmt.Errorf("invalid config in %s: %w", path, err)
}
return cfg, nil
}

func (c *Config) Validate() error {
// Board and owner are optional — board integration is skipped when absent.
if c.Project.Board < 0 {
return fmt.Errorf("project.board must be >= 0")
}
if c.Project.Board > 0 && c.Project.Owner == "" {
return fmt.Errorf("project.owner must be non-empty when project.board is set")
}
if c.Agent.MaxPerHour < 0 {
return fmt.Errorf("agent.max-per-hour must be >= 0")
}
switch c.Notifications.Channel {
case "", "slack", "github":
default:
return fmt.Errorf("notifications.channel must be one of: \"\", \"slack\", \"github\"")
}
if c.Notifications.Channel == "slack" && c.Notifications.WebhookURL == "" {
return fmt.Errorf("notifications.webhook-url is required when channel is \"slack\"")
}
return nil
}

func WriteDefault(path string) error {
cfg := Config{Version: CurrentConfigVersion}
return Write(path, cfg)
}

func Write(path string, cfg Config) error {
f, err := os.Create(path)
if err != nil {
return fmt.Errorf("create config file: %w", err)
}
defer f.Close()
enc := toml.NewEncoder(f)
return enc.Encode(cfg)
}
