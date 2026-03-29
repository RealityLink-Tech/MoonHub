package internal

import (
	"os"
	"path/filepath"

	"github.com/RealityLink-Tech/MoonHub/pkg/config"
)

const Logo = "🦞"

// GetMoonHubHome returns the moonhub home directory.
// Priority: $MOONHUB_HOME > ~/.moonhub
func GetMoonHubHome() string {
	if home := os.Getenv("MOONHUB_HOME"); home != "" {
		return home
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".moonhub")
}

func GetConfigPath() string {
	if configPath := os.Getenv("MOONHUB_CONFIG"); configPath != "" {
		return configPath
	}
	return filepath.Join(GetMoonHubHome(), "config.json")
}

func LoadConfig() (*config.Config, error) {
	return config.LoadConfig(GetConfigPath())
}
