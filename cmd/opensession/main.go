package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/quantick/opensession/internal/config"
	"github.com/quantick/opensession/internal/index"
	"github.com/quantick/opensession/internal/indexer"
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
	piSessionsRoot := flags.String("pi-sessions-root", "", "Pi sessions root override")
	sourceSelection := flags.String("source", "", "comma-separated sources to scan/display: all, opencode, pi")
	dbPath := flags.String("db", "", "opensession SQLite database path override")
	noScan := flags.Bool("no-scan", false, "skip scanning source storage before opening the TUI")
	showVersion := flags.Bool("version", false, "print version and exit")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *showVersion {
		fmt.Println(version)
		return nil
	}

	cfg, err := config.ResolveWithOptions(config.ResolveOptions{
		StorageRootOverride:     *storageRoot,
		PiSessionsRootOverride:  *piSessionsRoot,
		DBPathOverride:          *dbPath,
		SourceSelectionOverride: *sourceSelection,
		NoScan:                  *noScan,
	})
	if err != nil {
		return err
	}

	store, err := index.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sessions, err := store.ListSessions(ctx)
	if err != nil {
		return err
	}

	var events <-chan indexer.Event
	var refreshDone <-chan struct{}
	if !cfg.NoScan {
		eventCh := make(chan indexer.Event, 16)
		doneCh := make(chan struct{})
		events = eventCh
		refreshDone = doneCh
		go func() {
			defer close(doneCh)
			defer close(eventCh)
			_ = indexer.Run(ctx, store, indexer.Options{OpenCodeRoot: cfg.OpenCodeRoot, PiSessionsRoot: cfg.PiSessionsRoot, Sources: cfg.Sources}, func(event indexer.Event) {
				select {
				case eventCh <- event:
				case <-ctx.Done():
				}
			})
		}()
	}

	model := tui.NewModelWithIndexEvents(store, sessions, events, !cfg.NoScan)
	if cfg.NoScan {
		model = tui.NewModelWithIndexingDisabled(store, sessions)
	}
	program := tea.NewProgram(model, tea.WithAltScreen())
	_, err = program.Run()
	cancel()
	if refreshDone != nil {
		<-refreshDone
	}
	return err
}
