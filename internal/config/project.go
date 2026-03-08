package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Project        ProjectConfig        `yaml:"project"`
	Agent          AgentConfig          `yaml:"agent"`
	Notifications  NotificationsConfig  `yaml:"notifications"`
	SourceOfTruth  string               `yaml:"source-of-truth"`
	Filters        FiltersConfig        `yaml:"filters"`
}

type ProjectConfig struct {
	Board int    `yaml:"board"`
	Owner string `yaml:"owner"`
}

type AgentConfig struct {
	Hubber    string `yaml:"hubber"`
	Model     string `yaml:"model"`
	MaxPerHour int   `yaml:"max-per-hour"`
}

type NotificationsConfig struct {
	Channel    string `yaml:"channel"`
	OpsChannel string `yaml:"ops-channel"`
	WebhookURL string `yaml:"webhook-url"`
}

type FiltersConfig struct {
	Status string   `yaml:"status"`
	Labels []string `yaml:"labels"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.SourceOfTruth == "" {
		cfg.SourceOfTruth = "docs/SOURCE_OF_TRUTH.md"
	}
	return cfg, nil
}

func Write(path string, cfg Config) error {
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}
