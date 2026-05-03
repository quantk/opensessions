package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/quantick/opensession/internal/config"
	"github.com/quantick/opensession/internal/index"
	"github.com/quantick/opensession/internal/opencode"
	"github.com/quantick/opensession/internal/tui"
)

const version = "0.1.0"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("opensession", flag.ContinueOnError)
	storageRoot := flags.String("storage-root", "", "OpenCode storage root override")
	dbPath := flags.String("db", "", "opensession SQLite database path override")
	noScan := flags.Bool("no-scan", false, "skip scanning OpenCode storage before opening the TUI")
	showVersion := flags.Bool("version", false, "print version and exit")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *showVersion {
		fmt.Println(version)
		return nil
	}

	cfg, err := config.Resolve(*storageRoot, *dbPath, *noScan)
	if err != nil {
		return err
	}

	store, err := index.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer store.Close()

	ctx := context.Background()
	if !cfg.NoScan {
		paths, err := opencode.DiscoverSourcePaths(cfg.StorageRoot)
		if err != nil {
			return err
		}
		metadata, err := store.ScanMetadataBatch(ctx, paths)
		if err != nil {
			return err
		}
		existing, err := store.Snapshot(ctx)
		if err != nil {
			return err
		}
		snapshot, err := opencode.ScanWithMetadata(cfg.StorageRoot, opencodeMetadata(metadata), existing)
		if err != nil {
			return err
		}
		if err := store.UpsertSnapshot(ctx, snapshot); err != nil {
			return err
		}
	}

	sessions, err := store.ListSessions(ctx)
	if err != nil {
		return err
	}
	program := tea.NewProgram(tui.NewModel(store, sessions), tea.WithAltScreen())
	_, err = program.Run()
	return err
}

func opencodeMetadata(metadata map[string]index.ScanMetadata) map[string]opencode.FileRecord {
	out := make(map[string]opencode.FileRecord, len(metadata))
	for path, record := range metadata {
		out[path] = opencode.FileRecord{Path: record.Path, SizeBytes: record.SizeBytes, ModTime: record.ModTime}
	}
	return out
}
