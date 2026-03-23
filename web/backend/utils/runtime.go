package utils

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// GetDefaultConfigPath returns the default path to the moonhub config file.
func GetDefaultConfigPath() string {
	if configPath := os.Getenv("MOONHUB_CONFIG"); configPath != "" {
		return configPath
	}
	if moonhubHome := os.Getenv("MOONHUB_HOME"); moonhubHome != "" {
		return filepath.Join(moonhubHome, "config.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.json"
	}
	return filepath.Join(home, ".moonhub", "config.json")
}

// FindMoonHubBinary locates the moonhub executable.
// Search order:
//  1. MOONHUB_BINARY environment variable (explicit override)
//  2. Same directory as the current executable
//  3. Falls back to "moonhub" and relies on $PATH
func FindMoonHubBinary() string {
	binaryName := "moonhub"
	if runtime.GOOS == "windows" {
		binaryName = "moonhub.exe"
	}

	if p := os.Getenv("MOONHUB_BINARY"); p != "" {
		if info, _ := os.Stat(p); info != nil && !info.IsDir() {
			return p
		}
	}

	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), binaryName)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}

	return "moonhub"
}

// GetLocalIP returns the local IP address of the machine.
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String()
		}
	}
	return ""
}

// OpenBrowser automatically opens the given URL in the default browser.
func OpenBrowser(url string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return fmt.Errorf("unsupported platform")
	}
}
