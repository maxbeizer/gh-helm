package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ManagerConfig struct {
	Manager       ManagerSettings              `yaml:"manager"`
	Projects      []ManagerProject             `yaml:"projects"`
	Team          []TeamMember                 `yaml:"team"`
	Pillars       map[string]PillarConfig      `yaml:"pillars"`
	Notifications NotificationsConfig          `yaml:"notifications"`
	Schedule      ManagerSchedule              `yaml:"schedule"`
}

type ManagerSettings struct {
	Hubber string `yaml:"hubber"`
}

type ManagerProject struct {
	Owner string `yaml:"owner"`
	Board int    `yaml:"board"`
	Name  string `yaml:"name"`
}

type TeamMember struct {
	Handle    string   `yaml:"handle"`
	OneOneRepo string  `yaml:"1-1-repo"`
	Pillars   []string `yaml:"pillars"`
}

type PillarConfig struct {
	Description string   `yaml:"description"`
	Signals     []string `yaml:"signals"`
	Repos       []string `yaml:"repos"`
	Labels      []string `yaml:"labels"`
	Paths       []string `yaml:"paths"`
}

type ManagerSchedule struct {
	Pulse   string `yaml:"pulse"`
	Prep    string `yaml:"prep"`
	Observe string `yaml:"observe"`
}

func LoadManager(path string) (ManagerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ManagerConfig{}, err
	}
	var cfg ManagerConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return ManagerConfig{}, err
	}
	return cfg, nil
}

func WriteManager(path string, cfg ManagerConfig) error {
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshal manager config: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}
