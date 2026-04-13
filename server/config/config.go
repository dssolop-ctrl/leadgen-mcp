package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig              `yaml:"server"`
	Accounts map[string][]AccountEntry `yaml:"accounts"` // "yandex", "vk"
}

type ServerConfig struct {
	Port        int    `yaml:"port"`
	BearerToken string `yaml:"bearer_token"`
}

type AccountEntry struct {
	Name    string `yaml:"name"`
	Token   string `yaml:"token"`
	Default bool   `yaml:"default"`
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

	return cfg, nil
}
