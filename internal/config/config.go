package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type RegistrationPolicy string

const (
	AdminOnly RegistrationPolicy = "admin_only"
	Open      RegistrationPolicy = "open"
)

type StationConfig struct {
	Username         string   `yaml:"username"`
	Name             string   `yaml:"name"`
	Summary          string   `yaml:"summary"`
	LicenseTerritory []string `yaml:"license_territory"`
	RelayPolicy      string   `yaml:"relay_policy"` // "open" | "allowlist" | "closed"
}

type Config struct {
	Domain       string             `yaml:"domain"`
	Scheme       string             `yaml:"scheme"`       // "http" | "https"
	Registration RegistrationPolicy `yaml:"registration"` // "admin_only" | "open"
	Stations     []StationConfig    `yaml:"stations"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.Domain == "" {
		return nil, fmt.Errorf("config: domain must not be empty")
	}
	if cfg.Scheme == "" {
		cfg.Scheme = "https"
	}
	if cfg.Registration == "" {
		cfg.Registration = AdminOnly
	}

	return &cfg, nil
}

func (c *Config) Registry() map[string]StationConfig {
	m := make(map[string]StationConfig, len(c.Stations))
	for _, s := range c.Stations {
		m[s.Username] = s
	}
	return m
}

func (c *Config) BaseURL() string {
	return fmt.Sprintf("%s://%s", c.Scheme, c.Domain)
}
