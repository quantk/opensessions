package opencode

import (
	"errors"
	"os"
	"path/filepath"
)

const (
	EnvStorageRoot    = "OPENSESSION_STORAGE_ROOT"
	EnvOpenCodeRoot   = "OPENCODE_STORAGE_ROOT"
	defaultStorageSub = "opencode/storage"
)

func DiscoverStorageRoot(override string) (string, error) {
	if override != "" {
		return filepath.Clean(override), nil
	}
	if value := os.Getenv(EnvStorageRoot); value != "" {
		return filepath.Clean(value), nil
	}
	if value := os.Getenv(EnvOpenCodeRoot); value != "" {
		return filepath.Clean(value), nil
	}
	root, err := DefaultStorageRoot()
	if err != nil {
		return "", err
	}
	return root, nil
}

func DefaultStorageRoot() (string, error) {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, defaultStorageSub), nil
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", errors.New("cannot determine home directory for default OpenCode storage root")
	}
	return filepath.Join(home, ".local", "share", defaultStorageSub), nil
}
