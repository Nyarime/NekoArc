package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	LastDir     string   `json:"lastDir"`
	RecentFiles []string `json:"recentFiles"`
}

func configPath() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".nekoarc")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "config.json")
}

func loadConfig() Config {
	var cfg Config
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}
	json.Unmarshal(data, &cfg)
	return cfg
}

func saveConfig(cfg Config) {
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(configPath(), data, 0644)
}

func (c *Config) AddRecentFile(path string) {
	// Remove duplicates
	var filtered []string
	for _, f := range c.RecentFiles {
		if f != path {
			filtered = append(filtered, f)
		}
	}
	c.RecentFiles = append([]string{path}, filtered...)
	if len(c.RecentFiles) > 10 {
		c.RecentFiles = c.RecentFiles[:10]
	}
}
