package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// BiltDir returns the path to ~/.bilt/
func BiltDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".bilt")
	}
	return filepath.Join(home, ".bilt")
}

func configPath() string {
	return filepath.Join(BiltDir(), "config.yaml")
}

// Auth holds authentication state.
type Auth struct {
	APIKey string `yaml:"api_key,omitempty"` // bilt_live_... API key
}

// Defaults holds user defaults.
type Defaults struct {
	DeviceUDID string `yaml:"device_udid,omitempty"`
}

// ProjectConfig holds per-project cached settings.
type ProjectConfig struct {
	LastBuild time.Time `yaml:"last_build,omitempty"`
	TeamID    string    `yaml:"team_id,omitempty"`
	Scheme    string    `yaml:"scheme,omitempty"`
	Workspace string    `yaml:"workspace,omitempty"`
}

// Config is the top-level config structure stored in ~/.bilt/config.yaml.
type Config struct {
	Auth     Auth                     `yaml:"auth,omitempty"`
	Defaults Defaults                 `yaml:"defaults,omitempty"`
	Projects map[string]ProjectConfig `yaml:"projects,omitempty"`

	mu sync.Mutex `yaml:"-"`
}

// Load reads the config from disk, creating defaults if the file doesn't exist.
func Load() (*Config, error) {
	if err := os.MkdirAll(BiltDir(), 0755); err != nil {
		return nil, fmt.Errorf("creating bilt directory: %w", err)
	}

	data, err := os.ReadFile(configPath())
	if err != nil {
		if os.IsNotExist(err) {
			cfg := &Config{
				Projects: make(map[string]ProjectConfig),
			}
			return cfg, cfg.Save()
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	if cfg.Projects == nil {
		cfg.Projects = make(map[string]ProjectConfig)
	}
	return &cfg, nil
}

// Save writes the config to disk.
func (c *Config) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	return os.WriteFile(configPath(), data, 0644)
}

// SetAPIKey stores the API key and saves.
func (c *Config) SetAPIKey(apiKey string) error {
	c.Auth.APIKey = apiKey
	return c.Save()
}

// ClearAuth removes auth and saves.
func (c *Config) ClearAuth() error {
	c.Auth = Auth{}
	return c.Save()
}

// SetProject updates a project's cached config and saves.
func (c *Config) SetProject(slug string, pc ProjectConfig) error {
	c.Projects[slug] = pc
	return c.Save()
}

// GetProject returns the cached config for a project.
func (c *Config) GetProject(slug string) ProjectConfig {
	return c.Projects[slug]
}
