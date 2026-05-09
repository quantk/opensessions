package indexer

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/quantick/opensession/internal/index"
	"github.com/quantick/opensession/internal/opencode"
	"github.com/quantick/opensession/internal/pi"
	"github.com/quantick/opensession/internal/source"
)

type EventKind string

const (
	EventStarted  EventKind = "started"
	EventPhase    EventKind = "phase"
	EventSessions EventKind = "sessions"
	EventComplete EventKind = "complete"
	EventFailed   EventKind = "failed"
)

type Event struct {
	Kind     EventKind
	Source   source.Kind
	Phase    string
	Current  int
	Total    int
	Sessions []index.SessionSummary
	Err      error
}

type Options struct {
	OpenCodeRoot   string
	PiSessionsRoot string
	Sources        []source.Kind
}

type Store interface {
	ScanMetadataBatch(context.Context, []string) (map[string]index.ScanMetadata, error)
	Snapshot(context.Context) (opencode.Snapshot, error)
	UpsertSnapshot(context.Context, opencode.Snapshot) error
	ListSessions(context.Context) ([]index.SessionSummary, error)
}

func Run(ctx context.Context, store Store, options Options, emit func(Event)) error {
	emitEvent(emit, Event{Kind: EventStarted, Phase: "starting"})
	for _, kind := range options.Sources {
		if err := ctx.Err(); err != nil {
			emitEvent(emit, Event{Kind: EventFailed, Source: kind, Phase: "cancelled", Err: err})
			return err
		}
		switch source.NormalizeKind(string(kind)) {
		case source.KindOpenCode:
			if err := refreshOpenCode(ctx, store, options.OpenCodeRoot, emit); err != nil {
				emitEvent(emit, Event{Kind: EventFailed, Source: source.KindOpenCode, Phase: "failed", Err: err})
				return err
			}
		case source.KindPi:
			if err := refreshPi(ctx, store, options.PiSessionsRoot, emit); err != nil {
				emitEvent(emit, Event{Kind: EventFailed, Source: source.KindPi, Phase: "failed", Err: err})
				return err
			}
		}
	}
	emitEvent(emit, Event{Kind: EventPhase, Phase: "refreshing sessions"})
	sessions, err := store.ListSessions(ctx)
	if err != nil {
		emitEvent(emit, Event{Kind: EventFailed, Phase: "refreshing sessions", Err: err})
		return err
	}
	emitEvent(emit, Event{Kind: EventSessions, Phase: "sessions refreshed", Sessions: sessions, Current: len(sessions), Total: len(sessions)})
	emitEvent(emit, Event{Kind: EventComplete, Phase: "complete", Sessions: sessions})
	return nil
}

func refreshOpenCode(ctx context.Context, store Store, root string, emit func(Event)) error {
	emitEvent(emit, Event{Kind: EventPhase, Source: source.KindOpenCode, Phase: "discovering sources"})
	paths, err := opencode.DiscoverSourcePaths(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			emitEvent(emit, Event{Kind: EventPhase, Source: source.KindOpenCode, Phase: "source missing"})
			return nil
		}
		return err
	}
	return refreshSnapshot(ctx, store, source.KindOpenCode, paths, emit, func(metadata map[string]opencode.FileRecord, existing opencode.Snapshot) (opencode.Snapshot, error) {
		return opencode.ScanWithMetadata(root, metadata, existing)
	})
}

func refreshPi(ctx context.Context, store Store, root string, emit func(Event)) error {
	emitEvent(emit, Event{Kind: EventPhase, Source: source.KindPi, Phase: "discovering sources"})
	paths, err := pi.DiscoverSourcePaths(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			emitEvent(emit, Event{Kind: EventPhase, Source: source.KindPi, Phase: "source missing"})
			return nil
		}
		return err
	}
	return refreshSnapshot(ctx, store, source.KindPi, paths, emit, func(metadata map[string]opencode.FileRecord, existing opencode.Snapshot) (opencode.Snapshot, error) {
		return pi.ScanWithMetadata(root, metadata, existing)
	})
}

func refreshSnapshot(ctx context.Context, store Store, kind source.Kind, paths []string, emit func(Event), scan func(map[string]opencode.FileRecord, opencode.Snapshot) (opencode.Snapshot, error)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	emitEvent(emit, Event{Kind: EventPhase, Source: kind, Phase: "loading scan metadata", Current: 0, Total: len(paths)})
	metadata, err := store.ScanMetadataBatch(ctx, paths)
	if err != nil {
		return fmt.Errorf("load %s scan metadata: %w", kind, err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	emitEvent(emit, Event{Kind: EventPhase, Source: kind, Phase: "loading cached snapshot", Current: len(metadata), Total: len(paths)})
	existing, err := store.Snapshot(ctx)
	if err != nil {
		return fmt.Errorf("load cached snapshot for %s: %w", kind, err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	emitEvent(emit, Event{Kind: EventPhase, Source: kind, Phase: "scanning sources", Current: len(metadata), Total: len(paths)})
	snapshot, err := scan(opencodeMetadata(metadata), existing)
	if err != nil {
		return fmt.Errorf("scan %s sources: %w", kind, err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	emitEvent(emit, Event{Kind: EventPhase, Source: kind, Phase: "writing index", Current: len(paths), Total: len(paths)})
	if err := store.UpsertSnapshot(ctx, snapshot); err != nil {
		return fmt.Errorf("write %s index: %w", kind, err)
	}
	emitEvent(emit, Event{Kind: EventPhase, Source: kind, Phase: "indexed", Current: len(paths), Total: len(paths)})
	return nil
}

func opencodeMetadata(metadata map[string]index.ScanMetadata) map[string]opencode.FileRecord {
	out := make(map[string]opencode.FileRecord, len(metadata))
	for path, record := range metadata {
		out[path] = opencode.FileRecord{Path: record.Path, SizeBytes: record.SizeBytes, ModTime: record.ModTime}
	}
	return out
}

func emitEvent(emit func(Event), event Event) {
	if emit != nil {
		emit(event)
	}
}
