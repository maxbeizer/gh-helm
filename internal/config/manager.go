package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

const CurrentManagerConfigVersion = 1

type ManagerConfig struct {
	Version       int                     `toml:"version"`
	Manager       ManagerSettings         `toml:"manager"`
	Projects      []ManagerProject        `toml:"projects"`
	Team          []TeamMember            `toml:"team"`
	Pillars       map[string]PillarConfig `toml:"pillars"`
	Notifications NotificationsConfig     `toml:"notifications"`
	Schedule      ManagerSchedule         `toml:"schedule"`
}

type ManagerSettings struct {
	User string `toml:"user"`
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
	if cfg.Version == 0 {
		return ManagerConfig{}, fmt.Errorf("missing 'version' field in %s, expected version = %d", path, CurrentManagerConfigVersion)
	}
	if cfg.Version != CurrentManagerConfigVersion {
		return ManagerConfig{}, fmt.Errorf("unsupported config version %d in %s (expected %d), run 'gh helm upgrade' to migrate", cfg.Version, path, CurrentManagerConfigVersion)
	}
	if err := cfg.Validate(); err != nil {
		return ManagerConfig{}, fmt.Errorf("invalid config in %s: %w", path, err)
	}
	return cfg, nil
}

func (c *ManagerConfig) Validate() error {
	if c.Manager.User == "" {
		return fmt.Errorf("manager.user must be non-empty")
	}
	if len(c.Team) == 0 {
		return fmt.Errorf("team must have at least one member")
	}
	for i, member := range c.Team {
		if member.OneOneRepo == "" {
			return fmt.Errorf("team[%d].one-one-repo must be non-empty (handle: %q)", i, member.Handle)
		}
	}
	return nil
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
