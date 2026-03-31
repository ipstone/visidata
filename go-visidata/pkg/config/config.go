package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const (
	DefaultConfigName = ".vdgorc"
	EnvConfigPath     = "VDGO_CONFIG"
)

type Config struct {
	Filetype    *string `toml:"filetype"`
	Header      *int    `toml:"header"`
	PreviewRows *int    `toml:"preview_rows"`
	Table       *string `toml:"table"`
	Theme       *string `toml:"theme"`
	ShowHidden  *bool   `toml:"show_hidden"`
}

func Load(path string) (Config, error) {
	var cfg Config
	if path == "" {
		return cfg, nil
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func Discover(explicitPath string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	home, err := os.UserHomeDir()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	return ResolvePath(explicitPath, os.Getenv(EnvConfigPath), cwd, home)
}

func ResolvePath(explicitPath, envPath, cwd, home string) (string, error) {
	if explicitPath != "" {
		return explicitPath, nil
	}
	if envPath != "" {
		return envPath, nil
	}

	candidates := []string{}
	if cwd != "" {
		candidates = append(candidates, filepath.Join(cwd, DefaultConfigName))
	}
	if home != "" {
		homeConfig := filepath.Join(home, DefaultConfigName)
		if homeConfig != filepath.Join(cwd, DefaultConfigName) {
			candidates = append(candidates, homeConfig)
		}
	}

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, nil
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
	}
	return "", nil
}
