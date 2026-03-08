package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type ManagerConfig struct {
	Manager       ManagerSettings         `toml:"manager"`
	Projects      []ManagerProject        `toml:"projects"`
	Team          []TeamMember            `toml:"team"`
	Pillars       map[string]PillarConfig `toml:"pillars"`
	Notifications NotificationsConfig     `toml:"notifications"`
	Schedule      ManagerSchedule         `toml:"schedule"`
}

type ManagerSettings struct {
	Hubber string `toml:"hubber"`
}

type ManagerProject struct {
	Owner string `toml:"owner"`
	Board int    `toml:"board"`
	Name  string `toml:"name"`
}

type TeamMember struct {
	Handle     string   `toml:"handle"`
	OneOneRepo string   `toml:"one-one-repo"`
	Pillars    []string `toml:"pillars"`
}

type PillarConfig struct {
	Description string   `toml:"description"`
	Signals     []string `toml:"signals"`
	Repos       []string `toml:"repos"`
	Labels      []string `toml:"labels"`
	Paths       []string `toml:"paths"`
}

type ManagerSchedule struct {
	Pulse   string `toml:"pulse"`
	Prep    string `toml:"prep"`
	Observe string `toml:"observe"`
}

func LoadManager(path string) (ManagerConfig, error) {
	var cfg ManagerConfig
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return ManagerConfig{}, err
	}
	return cfg, nil
}

func WriteManager(path string, cfg ManagerConfig) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create manager config file: %w", err)
	}
	defer f.Close()
	enc := toml.NewEncoder(f)
	return enc.Encode(cfg)
}
