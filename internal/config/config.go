package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/longnguyen/ws-cli/internal/model"
)

// Path returns the config file path, honoring $WS_CONFIG and $XDG_CONFIG_HOME.
func Path() (string, error) {
	if p := os.Getenv("WS_CONFIG"); p != "" {
		return p, nil
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "ws", "config.toml"), nil
}

// Load reads the config file. Missing file returns default empty config.
func Load() (model.Config, string, error) {
	path, err := Path()
	if err != nil {
		return model.Config{}, "", err
	}
	cfg := model.DefaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, path, nil
		}
		return cfg, path, err
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, path, fmt.Errorf("parse %s: %w", path, err)
	}
	if cfg.Schema == 0 {
		cfg.Schema = 1
	}
	if cfg.Scan.Depth == 0 {
		cfg.Scan = model.DefaultConfig().Scan
	}
	return cfg, path, nil
}

// Save writes atomically (temp file + rename).
func Save(cfg model.Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".config-*.toml")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	enc := toml.NewEncoder(tmp)
	enc.Indent = "  "
	if err := enc.Encode(cfg); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
