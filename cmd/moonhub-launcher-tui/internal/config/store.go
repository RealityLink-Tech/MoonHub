package configstore

import (
	"errors"
	"os"
	"path/filepath"

	moonhubconfig "github.com/RealityLink-Tech/MoonHub/pkg/config"
)

const (
	configDirName  = ".moonhub"
	configFileName = "config.json"
)

func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configDirName), nil
}

func Load() (*moonhubconfig.Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	return moonhubconfig.LoadConfig(path)
}

func Save(cfg *moonhubconfig.Config) error {
	if cfg == nil {
		return errors.New("config is nil")
	}
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	return moonhubconfig.SaveConfig(path, cfg)
}
