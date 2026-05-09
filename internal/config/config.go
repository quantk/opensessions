package config

import (
	"fmt"
	"strings"

	"github.com/quantick/opensession/internal/index"
	"github.com/quantick/opensession/internal/opencode"
	"github.com/quantick/opensession/internal/pi"
	"github.com/quantick/opensession/internal/source"
)

type Config struct {
	StorageRoot    string
	OpenCodeRoot   string
	PiSessionsRoot string
	DBPath         string
	NoScan         bool
	Sources        []source.Kind
}

type ResolveOptions struct {
	StorageRootOverride     string
	PiSessionsRootOverride  string
	DBPathOverride          string
	SourceSelectionOverride string
	NoScan                  bool
}

func Resolve(storageRootOverride, dbPathOverride string, noScan bool) (Config, error) {
	return ResolveWithOptions(ResolveOptions{StorageRootOverride: storageRootOverride, DBPathOverride: dbPathOverride, NoScan: noScan})
}

func ResolveWithOptions(options ResolveOptions) (Config, error) {
	sources, err := parseSources(options.SourceSelectionOverride)
	if err != nil {
		return Config{}, err
	}
	openCodeRoot, err := opencode.DiscoverStorageRoot(options.StorageRootOverride)
	if err != nil && sourceEnabled(sources, source.KindOpenCode) {
		return Config{}, err
	}
	piRoot, err := pi.DiscoverSessionsRoot(options.PiSessionsRootOverride)
	if err != nil && sourceEnabled(sources, source.KindPi) {
		return Config{}, err
	}
	dbPath, err := index.DefaultPath(options.DBPathOverride)
	if err != nil {
		return Config{}, err
	}
	return Config{StorageRoot: openCodeRoot, OpenCodeRoot: openCodeRoot, PiSessionsRoot: piRoot, DBPath: dbPath, NoScan: options.NoScan, Sources: sources}, nil
}

func parseSources(selection string) ([]source.Kind, error) {
	selection = strings.TrimSpace(selection)
	if selection == "" || strings.EqualFold(selection, "all") {
		return []source.Kind{source.KindOpenCode, source.KindPi}, nil
	}
	parts := strings.Split(selection, ",")
	out := make([]source.Kind, 0, len(parts))
	seen := map[source.Kind]bool{}
	for _, part := range parts {
		kind := source.NormalizeKind(part)
		switch kind {
		case source.KindOpenCode, source.KindPi:
			if !seen[kind] {
				out = append(out, kind)
				seen[kind] = true
			}
		default:
			return nil, fmt.Errorf("unsupported source %q", strings.TrimSpace(part))
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no sources selected")
	}
	return out, nil
}

func sourceEnabled(sources []source.Kind, kind source.Kind) bool {
	for _, sourceKind := range sources {
		if sourceKind == kind {
			return true
		}
	}
	return false
}
