package indexer

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/quantick/opensession/internal/index"
	"github.com/quantick/opensession/internal/source"
)

func TestRunRefreshesOpenCodeFixtureAndEmitsEvents(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "opencode", "storage")
	watchPath := filepath.Join(root, "part", "msg_user", "prt_text.json")
	before := mustStat(t, watchPath)
	store, err := index.Open(filepath.Join(t.TempDir(), "opensession.sqlite"))
	if err != nil {
		t.Fatalf("open index: %v", err)
	}
	defer store.Close()

	var events []Event
	err = Run(context.Background(), store, Options{OpenCodeRoot: root, Sources: []source.Kind{source.KindOpenCode}}, func(event Event) {
		events = append(events, event)
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	after := mustStat(t, watchPath)
	if before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		t.Fatalf("refresh modified source fixture: before size=%d mod=%s after size=%d mod=%s", before.Size(), before.ModTime(), after.Size(), after.ModTime())
	}
	if !hasEventKind(events, EventStarted) || !hasEventKind(events, EventSessions) || !hasEventKind(events, EventComplete) {
		t.Fatalf("events missing started/sessions/complete: %#v", events)
	}
	if !hasSourcePhase(events, source.KindOpenCode, "writing index") {
		t.Fatalf("events missing OpenCode writing phase: %#v", events)
	}
	sessions, err := store.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) == 0 {
		t.Fatalf("no sessions indexed")
	}
}

func TestRunRefreshesPiFixtureWithoutModifyingSource(t *testing.T) {
	root := t.TempDir()
	sessionPath := filepath.Join(root, "project", "session.jsonl")
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0o755); err != nil {
		t.Fatalf("mkdir pi fixture: %v", err)
	}
	content := strings.Join([]string{
		`{"type":"session","id":"pi-refresh","timestamp":"2026-05-09T10:00:00Z","cwd":"/tmp/pi-refresh"}`,
		`{"type":"message","id":"u1","parentId":null,"timestamp":"2026-05-09T10:00:01Z","message":{"role":"user","content":"hello"}}`,
	}, "\n") + "\n"
	if err := os.WriteFile(sessionPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write pi fixture: %v", err)
	}
	before := mustStat(t, sessionPath)
	store, err := index.Open(filepath.Join(t.TempDir(), "opensession.sqlite"))
	if err != nil {
		t.Fatalf("open index: %v", err)
	}
	defer store.Close()
	if err := Run(context.Background(), store, Options{PiSessionsRoot: root, Sources: []source.Kind{source.KindPi}}, nil); err != nil {
		t.Fatalf("Run Pi: %v", err)
	}
	after := mustStat(t, sessionPath)
	if before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		t.Fatalf("refresh modified Pi fixture: before size=%d mod=%s after size=%d mod=%s", before.Size(), before.ModTime(), after.Size(), after.ModTime())
	}
	sessions, err := store.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].SourceKind != string(source.KindPi) {
		t.Fatalf("Pi sessions = %#v", sessions)
	}
}

func TestRunIgnoresMissingSourceRoot(t *testing.T) {
	store, err := index.Open(filepath.Join(t.TempDir(), "opensession.sqlite"))
	if err != nil {
		t.Fatalf("open index: %v", err)
	}
	defer store.Close()
	var events []Event
	err = Run(context.Background(), store, Options{OpenCodeRoot: filepath.Join(t.TempDir(), "missing"), Sources: []source.Kind{source.KindOpenCode}}, func(event Event) {
		events = append(events, event)
	})
	if err != nil {
		t.Fatalf("Run missing source: %v", err)
	}
	if !hasSourcePhase(events, source.KindOpenCode, "source missing") || !hasEventKind(events, EventComplete) {
		t.Fatalf("missing source events = %#v", events)
	}
}

func hasEventKind(events []Event, kind EventKind) bool {
	for _, event := range events {
		if event.Kind == kind {
			return true
		}
	}
	return false
}

func hasSourcePhase(events []Event, kind source.Kind, phase string) bool {
	for _, event := range events {
		if event.Source == kind && event.Phase == phase {
			return true
		}
	}
	return false
}

func mustStat(t *testing.T, path string) os.FileInfo {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	return info
}
