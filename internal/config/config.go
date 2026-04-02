package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Domain string     `toml:"domain"`
	Names  []string   `toml:"names"`
	Sops   SopsConfig `toml:"sops"`
}

type SopsConfig struct {
	Enabled bool `toml:"enabled"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if cfg.Domain == "" {
		return nil, fmt.Errorf("config: 'domain' is required")
	}
	if len(cfg.Names) == 0 {
		return nil, fmt.Errorf("config: 'names' list is required and must not be empty")
	}

	return &cfg, nil
}

// UsedNames returns the set of host names that already have a directory under hostsDir.
func UsedNames(hostsDir string) (map[string]bool, error) {
	used := make(map[string]bool)

	entries, err := os.ReadDir(hostsDir)
	if os.IsNotExist(err) {
		return used, nil
	}
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.IsDir() {
			used[e.Name()] = true
		}
	}
	return used, nil
}

// PickName returns the first unused name from the config's name list.
func PickName(cfg *Config, hostsDir string) (string, error) {
	used, err := UsedNames(hostsDir)
	if err != nil {
		return "", fmt.Errorf("scanning hosts dir: %w", err)
	}

	for _, name := range cfg.Names {
		if !used[name] {
			return name, nil
		}
	}

	return "", fmt.Errorf("all %d names are in use, add more names to nixmgr.toml", len(cfg.Names))
}
