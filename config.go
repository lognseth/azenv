package main

import (
	"errors"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	Version        int       `yaml:"version"`
	DefaultContext string    `yaml:"default_context,omitempty"`
	CreatedAt      time.Time `yaml:"created_at"`
	UpdatedAt      time.Time `yaml:"updated_at"`
}

func loadConfig() (*AppConfig, error) {
	b, err := os.ReadFile(configPath())
	if err != nil {
		return nil, err
	}
	var cfg AppConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	return &cfg, nil
}

func saveConfig(cfg *AppConfig) error {
	now := time.Now().UTC()
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = now
	}
	cfg.UpdatedAt = now
	if err := os.MkdirAll(azenvHome(), 0700); err != nil {
		return err
	}
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), b, 0600)
}

func ensureConfig() (*AppConfig, error) {
	cfg, err := loadConfig()
	if err == nil {
		return cfg, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	cfg = &AppConfig{}
	return cfg, saveConfig(cfg)
}
