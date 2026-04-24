package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server     ServerConfig              `yaml:"server"`
	Accounts   map[string][]AccountEntry `yaml:"accounts"` // "yandex", "vk"
	OpenRouter OpenRouterConfig          `yaml:"openrouter"`
}

type ServerConfig struct {
	Port        int    `yaml:"port"`
	BearerToken string `yaml:"bearer_token"`
	DataDir     string `yaml:"data_dir"`    // directory for SQLite databases (default: /app/data)
	PreviewDir  string `yaml:"preview_dir"` // directory for generated image previews (default: docs/campaign_previews)
}

type AccountEntry struct {
	Name    string `yaml:"name"`
	Token   string `yaml:"token"`
	Default bool   `yaml:"default"`
}

// OpenRouterConfig holds credentials for the OpenRouter image generation API.
// Used by the imagegen package for the RSYA skill branch (R6.5).
type OpenRouterConfig struct {
	APIKey string `yaml:"api_key"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}

	// Env var fallback for OpenRouter key — keeps the token out of committed files
	// for users who prefer env-based secrets.
	if cfg.OpenRouter.APIKey == "" {
		cfg.OpenRouter.APIKey = os.Getenv("OPENROUTER_API_KEY")
	}

	return cfg, nil
}
