package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/quantick/opensession/internal/source"
)

func TestResolveWithOptionsDefaultsAndOverrides(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	state := filepath.Join(t.TempDir(), "state")
	t.Setenv("XDG_STATE_HOME", state)
	t.Setenv("OPENSESSION_STORAGE_ROOT", "")
	t.Setenv("OPENCODE_STORAGE_ROOT", "")
	t.Setenv("OPENSESSION_PI_SESSIONS_ROOT", "")
	t.Setenv("PI_SESSIONS_ROOT", "")

	cfg, err := ResolveWithOptions(ResolveOptions{NoScan: true})
	if err != nil {
		t.Fatalf("ResolveWithOptions defaults: %v", err)
	}
	if cfg.OpenCodeRoot != filepath.Join(home, ".local", "share", "opencode", "storage") {
		t.Fatalf("OpenCodeRoot = %q", cfg.OpenCodeRoot)
	}
	if cfg.PiSessionsRoot != filepath.Join(home, ".pi", "agent", "sessions") {
		t.Fatalf("PiSessionsRoot = %q", cfg.PiSessionsRoot)
	}
	if cfg.DBPath != filepath.Join(state, "opensession", "opensession.sqlite") {
		t.Fatalf("DBPath = %q", cfg.DBPath)
	}
	if !reflect.DeepEqual(cfg.Sources, []source.Kind{source.KindOpenCode, source.KindPi}) {
		t.Fatalf("Sources = %#v", cfg.Sources)
	}

	cfg, err = ResolveWithOptions(ResolveOptions{
		StorageRootOverride:     "/tmp/opencode-storage",
		PiSessionsRootOverride:  "/tmp/pi-sessions",
		DBPathOverride:          "/tmp/opensession.sqlite",
		SourceSelectionOverride: "pi",
		NoScan:                  true,
	})
	if err != nil {
		t.Fatalf("ResolveWithOptions overrides: %v", err)
	}
	if cfg.OpenCodeRoot != "/tmp/opencode-storage" || cfg.StorageRoot != "/tmp/opencode-storage" {
		t.Fatalf("OpenCode overrides = %#v", cfg)
	}
	if cfg.PiSessionsRoot != "/tmp/pi-sessions" || cfg.DBPath != "/tmp/opensession.sqlite" {
		t.Fatalf("Pi/DB overrides = %#v", cfg)
	}
	if !reflect.DeepEqual(cfg.Sources, []source.Kind{source.KindPi}) {
		t.Fatalf("Sources override = %#v", cfg.Sources)
	}
}

func TestResolveWithOptionsSourceSelection(t *testing.T) {
	cfg, err := ResolveWithOptions(ResolveOptions{SourceSelectionOverride: "opencode,pi", NoScan: true})
	if err != nil {
		t.Fatalf("ResolveWithOptions source list: %v", err)
	}
	if !reflect.DeepEqual(cfg.Sources, []source.Kind{source.KindOpenCode, source.KindPi}) {
		t.Fatalf("Sources = %#v", cfg.Sources)
	}
	if _, err := ResolveWithOptions(ResolveOptions{SourceSelectionOverride: "unknown", NoScan: true}); err == nil {
		t.Fatal("unsupported source should fail")
	}
}

func TestResolveWithOptionsMissingOptionalRootsAreJustPaths(t *testing.T) {
	home := t.TempDir()
	missingPi := filepath.Join(home, "missing", "pi")
	cfg, err := ResolveWithOptions(ResolveOptions{PiSessionsRootOverride: missingPi, SourceSelectionOverride: "all", NoScan: true})
	if err != nil {
		t.Fatalf("ResolveWithOptions missing pi root path: %v", err)
	}
	if cfg.PiSessionsRoot != missingPi {
		t.Fatalf("PiSessionsRoot = %q, want %q", cfg.PiSessionsRoot, missingPi)
	}
}
