package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &Config{
		Auth: Auth{
			APIKey: "bilt_live_abc123def456",
		},
		Projects: map[string]ProjectConfig{
			"my-app": {
				TeamID:    "ABCDE12345",
				Scheme:    "MyApp",
				Workspace: "MyApp.xcworkspace",
				LastBuild: time.Date(2026, 3, 25, 14, 30, 0, 0, time.UTC),
			},
		},
	}

	// Marshal and write
	data, err := yaml.Marshal(cfg)
	assert.NoError(t, err)
	assert.NoError(t, os.WriteFile(path, data, 0644))

	// Read and unmarshal
	data2, err := os.ReadFile(path)
	assert.NoError(t, err)

	var cfg2 Config
	assert.NoError(t, yaml.Unmarshal(data2, &cfg2))

	assert.Equal(t, cfg.Auth.APIKey, cfg2.Auth.APIKey)
	assert.Equal(t, cfg.Projects["my-app"].TeamID, cfg2.Projects["my-app"].TeamID)
	assert.Equal(t, cfg.Projects["my-app"].Scheme, cfg2.Projects["my-app"].Scheme)
	assert.Equal(t, cfg.Projects["my-app"].Workspace, cfg2.Projects["my-app"].Workspace)
}

func TestConfigEmptyProjects(t *testing.T) {
	cfg := &Config{}
	data, err := yaml.Marshal(cfg)
	assert.NoError(t, err)

	var cfg2 Config
	assert.NoError(t, yaml.Unmarshal(data, &cfg2))
	assert.Nil(t, cfg2.Projects)
}
