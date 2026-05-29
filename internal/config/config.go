package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	URL   string `toml:"url"`
	Token string `toml:"token"`
	Theme string `toml:"theme"`
	Font  string `toml:"font"`
	Tail  int    `toml:"tail"`
}

func Default() Config {
	return Config{
		URL:   "http://localhost:8089",
		Theme: "amber",
		Font:  "JetBrains Mono",
		Tail:  100,
	}
}

// Path returns the canonical config path: $XDG_CONFIG_HOME/logify/config.toml
// or ~/.config/logify/config.toml.
func Path() (string, error) {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "logify", "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "logify", "config.toml"), nil
}

func Load() (Config, string, error) {
	cfg := Default()
	p, err := Path()
	if err != nil {
		return cfg, "", err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, p, nil
		}
		return cfg, p, err
	}
	if _, err := toml.Decode(string(b), &cfg); err != nil {
		return cfg, p, fmt.Errorf("parse %s: %w", p, err)
	}
	return cfg, p, nil
}

func Save(c Config) (string, error) {
	p, err := Path()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return p, err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return p, err
	}
	defer f.Close()
	if err := toml.NewEncoder(f).Encode(c); err != nil {
		return p, err
	}
	return p, nil
}
