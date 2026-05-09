package pi

import (
	"errors"
	"os"
	"path/filepath"
)

const (
	EnvSessionsRoot    = "OPENSESSION_PI_SESSIONS_ROOT"
	EnvPiSessionsRoot  = "PI_SESSIONS_ROOT"
	defaultSessionsSub = ".pi/agent/sessions"
)

func DiscoverSessionsRoot(override string) (string, error) {
	if override != "" {
		return filepath.Clean(override), nil
	}
	if value := os.Getenv(EnvSessionsRoot); value != "" {
		return filepath.Clean(value), nil
	}
	if value := os.Getenv(EnvPiSessionsRoot); value != "" {
		return filepath.Clean(value), nil
	}
	return DefaultSessionsRoot()
}

func DefaultSessionsRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", errors.New("cannot determine home directory for default Pi sessions root")
	}
	return filepath.Join(home, defaultSessionsSub), nil
}
