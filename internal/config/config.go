package config

import (
	"github.com/quantick/opensession/internal/index"
	"github.com/quantick/opensession/internal/opencode"
)

type Config struct {
	StorageRoot string
	DBPath      string
	NoScan      bool
}

func Resolve(storageRootOverride, dbPathOverride string, noScan bool) (Config, error) {
	storageRoot, err := opencode.DiscoverStorageRoot(storageRootOverride)
	if err != nil {
		return Config{}, err
	}
	dbPath, err := index.DefaultPath(dbPathOverride)
	if err != nil {
		return Config{}, err
	}
	return Config{StorageRoot: storageRoot, DBPath: dbPath, NoScan: noScan}, nil
}
