package index

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/quantick/opensession/internal/opencode"
)

func TestDefaultPath(t *testing.T) {
	override := filepath.Join(t.TempDir(), "custom.sqlite")
	got, err := DefaultPath(override)
	if err != nil {
		t.Fatalf("DefaultPath override: %v", err)
	}
	if got != override {
		t.Fatalf("override path = %q, want %q", got, override)
	}

	state := filepath.Join(t.TempDir(), "state")
	t.Setenv("XDG_STATE_HOME", state)
	got, err = DefaultPath("")
	if err != nil {
		t.Fatalf("DefaultPath xdg: %v", err)
	}
	want := filepath.Join(state, "opensession", "opensession.sqlite")
	if got != want {
		t.Fatalf("default path = %q, want %q", got, want)
	}
}

func TestStoreUpsertsSearchTagsBookmarksAndScanMetadata(t *testing.T) {
	ctx := context.Background()
	snapshot := scanFixture(t)

	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	if err := store.UpsertSnapshot(ctx, snapshot); err != nil {
		t.Fatalf("UpsertSnapshot first: %v", err)
	}
	if err := store.UpsertSnapshot(ctx, snapshot); err != nil {
		t.Fatalf("UpsertSnapshot second: %v", err)
	}

	sessions, err := store.ListSessions(ctx)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("sessions = %d, want 2", len(sessions))
	}
	fixture := findSessionSummary(t, sessions, "ses_fixture")
	if fixture.MessageCount != 2 || fixture.PartCount != 7 || fixture.HeavyPartCount != 1 {
		t.Fatalf("counts = messages:%d parts:%d heavy:%d", fixture.MessageCount, fixture.PartCount, fixture.HeavyPartCount)
	}

	assertSessionSearch(t, store, "Find sessions", "ses_fixture")
	assertSessionSearch(t, store, "go test", "ses_fixture")
	assertSessionSearch(t, store, "planner note", "ses_fixture")

	heavyResults, err := store.SearchSessions(ctx, "AAECAwQFBgc")
	if err != nil {
		t.Fatalf("SearchSessions heavy raw: %v", err)
	}
	if len(heavyResults) != 0 {
		t.Fatalf("heavy raw payload should not be searchable, got %#v", heavyResults)
	}

	timeline, err := store.SearchSession(ctx, "ses_fixture", "README.md")
	if err != nil {
		t.Fatalf("SearchSession: %v", err)
	}
	if len(timeline) != 1 || timeline[0].PartID != "prt_file" {
		t.Fatalf("session search = %#v, want prt_file", timeline)
	}

	if err := store.SetTag(ctx, "ses_fixture", "favorite"); err != nil {
		t.Fatalf("SetTag: %v", err)
	}
	assertSessionSearch(t, store, "favorite", "ses_fixture")
	tags, err := store.Tags(ctx, "ses_fixture")
	if err != nil {
		t.Fatalf("Tags: %v", err)
	}
	if len(tags) != 1 || tags[0] != "favorite" {
		t.Fatalf("tags = %#v", tags)
	}

	if err := store.SetBookmark(ctx, "ses_fixture", true); err != nil {
		t.Fatalf("SetBookmark true: %v", err)
	}
	bookmarked, err := store.IsBookmarked(ctx, "ses_fixture")
	if err != nil {
		t.Fatalf("IsBookmarked: %v", err)
	}
	if !bookmarked {
		t.Fatal("bookmark not persisted")
	}
	if err := store.SetBookmark(ctx, "ses_fixture", false); err != nil {
		t.Fatalf("SetBookmark false: %v", err)
	}
	bookmarked, err = store.IsBookmarked(ctx, "ses_fixture")
	if err != nil {
		t.Fatalf("IsBookmarked after unset: %v", err)
	}
	if bookmarked {
		t.Fatal("bookmark should be removed")
	}

	heavyPart, err := store.RawPart(ctx, "prt_heavy")
	if err != nil {
		t.Fatalf("RawPart: %v", err)
	}
	if !heavyPart.Heavy || heavyPart.SourcePath == "" {
		t.Fatalf("raw part metadata = %#v", heavyPart)
	}

	session := findSnapshotSession(t, snapshot, "ses_fixture")
	if err := store.UpsertScanMetadata(ctx, session.Source.Path, session.Source.SizeBytes, session.Source.ModTime); err != nil {
		t.Fatalf("UpsertScanMetadata: %v", err)
	}
	metadata, ok, err := store.ScanMetadata(ctx, session.Source.Path)
	if err != nil {
		t.Fatalf("ScanMetadata: %v", err)
	}
	if !ok || metadata.SizeBytes != session.Source.SizeBytes || !metadata.ModTime.Equal(session.Source.ModTime) {
		t.Fatalf("metadata = %#v ok=%v", metadata, ok)
	}
}

func assertSessionSearch(t *testing.T, store *Store, query, wantID string) {
	t.Helper()
	results, err := store.SearchSessions(context.Background(), query)
	if err != nil {
		t.Fatalf("SearchSessions(%q): %v", query, err)
	}
	for _, result := range results {
		if result.ID == wantID {
			return
		}
	}
	t.Fatalf("SearchSessions(%q) = %#v, want %s", query, results, wantID)
}

func scanFixture(t *testing.T) opencode.Snapshot {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "testdata", "opencode", "storage"))
	if err != nil {
		t.Fatalf("abs fixture: %v", err)
	}
	snapshot, err := opencode.Scan(root)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	return snapshot
}

func findSessionSummary(t *testing.T, sessions []SessionSummary, id string) SessionSummary {
	t.Helper()
	for _, session := range sessions {
		if session.ID == id {
			return session
		}
	}
	t.Fatalf("session summary %q not found", id)
	return SessionSummary{}
}

func findSnapshotSession(t *testing.T, snapshot opencode.Snapshot, id string) opencode.Session {
	t.Helper()
	for _, session := range snapshot.Sessions {
		if session.ID == id {
			return session
		}
	}
	t.Fatalf("snapshot session %q not found", id)
	return opencode.Session{}
}
