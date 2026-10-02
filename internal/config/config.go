package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"proxy-tray/internal/core"
)

type Config struct {
	Filters          core.Filters `json:"filters"`
	StartWithWindows bool         `json:"start_with_windows"`
}

func Default() Config {
	return Config{Filters: core.DefaultFilters(), StartWithWindows: false}
}

type Store struct{ Root string }

func DefaultStore() Store {
	root := filepath.Join(os.Getenv("APPDATA"), "proxytray")
	if os.Getenv("APPDATA") == "" {
		home, _ := os.UserHomeDir()
		root = filepath.Join(home, ".proxytray")
	}
	return Store{Root: root}
}

func (s Store) ensure() error { return os.MkdirAll(s.Root, 0o755) }

func (s Store) Load() (Config, error) {
	b, err := os.ReadFile(filepath.Join(s.Root, "config.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return Config{}, err
	}
	return c, nil
}

func (s Store) Save(c Config) error {
	if err := s.ensure(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.Root, "config.json"), b, 0o644)
}

func (s Store) LoadCache() ([]core.Proxy, error) {
	b, err := os.ReadFile(filepath.Join(s.Root, "cache.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var p []core.Proxy
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s Store) SaveCache(p []core.Proxy) error {
	if err := s.ensure(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.Root, "cache.json"), b, 0o644)
}

func (s Store) LogPath() string { return filepath.Join(s.Root, "proxytray.log") }

func (s Store) CachePath() string { return filepath.Join(s.Root, "cache.json") }
