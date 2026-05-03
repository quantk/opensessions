package index

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

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
	if !fixture.TokenUsage.Available || fixture.TokenUsage.Total != 321 || fixture.TokenUsage.Input != 100 || fixture.TokenUsage.Output != 70 || fixture.TokenUsage.Reasoning != 20 || fixture.TokenUsage.CacheRead != 30 || fixture.TokenUsage.CacheWrite != 10 {
		t.Fatalf("token usage = %#v", fixture.TokenUsage)
	}
	global := findSessionSummary(t, sessions, "ses_global")
	if global.TokenUsage.Available || global.TokenUsage.Total != 0 {
		t.Fatalf("global token usage = %#v, want unavailable", global.TokenUsage)
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

func TestStoreSubagentMetadataAndTopLevelQueries(t *testing.T) {
	ctx := context.Background()
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	snapshot := opencode.Snapshot{
		Projects: []opencode.Project{{ID: "proj", Worktree: "/tmp/project"}},
		Sessions: []opencode.Session{
			{
				ID:          "ses_parent",
				ProjectID:   "proj",
				ProjectPath: "/tmp/project",
				Title:       "Parent session",
				Messages: []opencode.Message{{
					ID:        "msg_parent",
					SessionID: "ses_parent",
					Role:      "assistant",
					Parts: []opencode.Part{{
						ID:              "prt_task",
						SessionID:       "ses_parent",
						MessageID:       "msg_parent",
						Kind:            opencode.PartKindTool,
						ToolName:        "task",
						Title:           "Run child session",
						SubagentName:    "explore",
						LinkedSessionID: "ses_child",
						IndexText:       "task child launcher",
					}},
				}},
			},
			{
				ID:          "ses_child",
				ProjectID:   "proj",
				ParentID:    "ses_parent",
				ProjectPath: "/tmp/project",
				Title:       "Child session",
				Messages: []opencode.Message{{
					ID:        "msg_child",
					SessionID: "ses_child",
					Role:      "assistant",
					Parts: []opencode.Part{{
						ID:        "prt_child_text",
						SessionID: "ses_child",
						MessageID: "msg_child",
						Kind:      opencode.PartKindText,
						Preview:   "child transcript needle",
						IndexText: "child transcript needle",
					}},
				}},
			},
		},
	}
	if err := store.UpsertSnapshot(ctx, snapshot); err != nil {
		t.Fatalf("UpsertSnapshot: %v", err)
	}

	sessions, err := store.ListSessions(ctx)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].ID != "ses_parent" {
		t.Fatalf("top-level sessions = %#v, want only parent", sessions)
	}
	childResults, err := store.SearchSessions(ctx, "child transcript needle")
	if err != nil {
		t.Fatalf("SearchSessions child content: %v", err)
	}
	if len(childResults) != 0 {
		t.Fatalf("child session search results should be hidden, got %#v", childResults)
	}
	childSummary, err := store.Session(ctx, "ses_child")
	if err != nil {
		t.Fatalf("Session child: %v", err)
	}
	if childSummary.ParentID != "ses_parent" {
		t.Fatalf("child summary parent id = %q, want ses_parent", childSummary.ParentID)
	}

	parentTimeline, err := store.SessionTimeline(ctx, "ses_parent")
	if err != nil {
		t.Fatalf("SessionTimeline parent: %v", err)
	}
	task := findTimelinePart(t, parentTimeline, "prt_task")
	if task.LinkedSessionID != "ses_child" {
		t.Fatalf("linked session id = %q, want ses_child", task.LinkedSessionID)
	}
	if task.SubagentName != "explore" {
		t.Fatalf("subagent name = %q, want explore", task.SubagentName)
	}
	childTimeline, err := store.SessionTimeline(ctx, "ses_child")
	if err != nil {
		t.Fatalf("SessionTimeline child: %v", err)
	}
	if len(childTimeline) != 1 || childTimeline[0].PartID != "prt_child_text" {
		t.Fatalf("child timeline = %#v, want child text", childTimeline)
	}
}

func TestStoreBatchScanMetadataAndIncrementalUpsert(t *testing.T) {
	ctx := context.Background()
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()
	root := filepath.Join(t.TempDir(), "storage")
	base := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	snapshot := incrementalSnapshot(root, base, "old text", "old text")
	if err := store.UpsertSnapshot(ctx, snapshot); err != nil {
		t.Fatalf("UpsertSnapshot first: %v", err)
	}
	partSource := snapshot.Sessions[0].Messages[0].Parts[0].Source
	metadata, err := store.ScanMetadataBatch(ctx, []string{partSource.Path, partSource.Path, filepath.Join(root, "missing.json")})
	if err != nil {
		t.Fatalf("ScanMetadataBatch: %v", err)
	}
	if len(metadata) != 1 || metadata[partSource.Path].SizeBytes != partSource.SizeBytes || !metadata[partSource.Path].ModTime.Equal(partSource.ModTime) {
		t.Fatalf("batch metadata = %#v", metadata)
	}

	mustExec(t, store.db, `CREATE TABLE update_count (table_name TEXT)`)
	mustExec(t, store.db, `CREATE TRIGGER count_session_update AFTER UPDATE ON sessions BEGIN INSERT INTO update_count (table_name) VALUES ('sessions'); END`)
	mustExec(t, store.db, `CREATE TRIGGER count_part_update AFTER UPDATE ON parts BEGIN INSERT INTO update_count (table_name) VALUES ('parts'); END`)
	if err := store.UpsertSnapshot(ctx, snapshot); err != nil {
		t.Fatalf("UpsertSnapshot unchanged: %v", err)
	}
	if got := updateCount(t, store); got != 0 {
		t.Fatalf("unchanged upsert rewrote rows, update count=%d", got)
	}

	changed := incrementalSnapshot(root, base, "new text", "new text")
	changed.Sessions[0].Messages[0].Parts[0].Source.ModTime = base.Add(time.Minute)
	changed.Sessions[0].Messages[0].Parts[0].Source.SizeBytes++
	if err := store.UpsertSnapshot(ctx, changed); err != nil {
		t.Fatalf("UpsertSnapshot changed: %v", err)
	}
	if got := updateCount(t, store); got == 0 {
		t.Fatalf("changed part did not refresh dependent rows")
	}
	timeline, err := store.SearchSession(ctx, "ses", "new text")
	if err != nil {
		t.Fatalf("SearchSession new text: %v", err)
	}
	if len(timeline) != 1 || timeline[0].PartID != "prt" || timeline[0].Preview != "new text" {
		t.Fatalf("changed part not searchable/refreshed: %#v", timeline)
	}
	old, err := store.SearchSession(ctx, "ses", "old text")
	if err != nil {
		t.Fatalf("SearchSession old text: %v", err)
	}
	if len(old) != 0 {
		t.Fatalf("stale searchable document remained: %#v", old)
	}

	removed := incrementalSnapshot(root, base, "", "")
	removed.Sessions[0].PartCount = 0
	removed.Sessions[0].Messages[0].Parts = nil
	if err := store.UpsertSnapshot(ctx, removed); err != nil {
		t.Fatalf("UpsertSnapshot removed part: %v", err)
	}
	timeline, err = store.SessionTimeline(ctx, "ses")
	if err != nil {
		t.Fatalf("SessionTimeline removed: %v", err)
	}
	if len(timeline) != 0 {
		t.Fatalf("stale part remained after reconciliation: %#v", timeline)
	}
	summary, err := store.Session(ctx, "ses")
	if err != nil {
		t.Fatalf("Session summary removed: %v", err)
	}
	if summary.PartCount != 0 {
		t.Fatalf("session summary was not refreshed after stale part deletion: %#v", summary)
	}
}

func TestOpenMigratesExistingDatabaseForSubagentColumns(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "opensession.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open old sqlite: %v", err)
	}
	mustExec(t, db, `CREATE TABLE sessions (id TEXT PRIMARY KEY, project_id TEXT, project_path TEXT, directory TEXT, title TEXT, slug TEXT, version TEXT, model_provider TEXT, model_id TEXT, created_at INTEGER, updated_at INTEGER, message_count INTEGER NOT NULL DEFAULT 0, part_count INTEGER NOT NULL DEFAULT 0, heavy_part_count INTEGER NOT NULL DEFAULT 0, source_path TEXT)`)
	mustExec(t, db, `CREATE TABLE messages (id TEXT PRIMARY KEY, session_id TEXT NOT NULL, role TEXT, agent TEXT, summary_title TEXT, model_provider TEXT, model_id TEXT, created_at INTEGER, updated_at INTEGER, source_path TEXT)`)
	mustExec(t, db, `CREATE TABLE parts (id TEXT PRIMARY KEY, session_id TEXT NOT NULL, message_id TEXT NOT NULL, type TEXT, kind TEXT, tool_name TEXT, status TEXT, title TEXT, file_path TEXT, mime TEXT, filename TEXT, preview TEXT, index_text TEXT, source_path TEXT, size_bytes INTEGER, heavy INTEGER NOT NULL DEFAULT 0, binary INTEGER NOT NULL DEFAULT 0, skipped_raw INTEGER NOT NULL DEFAULT 0, created_at INTEGER, updated_at INTEGER)`)
	if err := db.Close(); err != nil {
		t.Fatalf("close old sqlite: %v", err)
	}

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open migrated database: %v", err)
	}
	defer store.Close()
	if !columnExists(t, store, "sessions", "parent_id") || !columnExists(t, store, "parts", "linked_session_id") || !columnExists(t, store, "parts", "subagent_name") {
		t.Fatal("subagent migration columns missing")
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

func findTimelinePart(t *testing.T, parts []TimelinePart, id string) TimelinePart {
	t.Helper()
	for _, part := range parts {
		if part.PartID == id {
			return part
		}
	}
	t.Fatalf("timeline part %q not found", id)
	return TimelinePart{}
}

func incrementalSnapshot(root string, modTime time.Time, preview, indexText string) opencode.Snapshot {
	part := opencode.Part{
		ID:        "prt",
		SessionID: "ses",
		MessageID: "msg",
		Kind:      opencode.PartKindText,
		Preview:   preview,
		IndexText: indexText,
		Source:    opencode.FileRecord{Path: filepath.Join(root, "part", "msg", "prt.json"), SizeBytes: int64(len(indexText) + 1), ModTime: modTime},
	}
	return opencode.Snapshot{
		Root: root,
		Projects: []opencode.Project{{
			ID:       "proj",
			Worktree: "/tmp/project",
			Source:   opencode.FileRecord{Path: filepath.Join(root, "project", "proj.json"), SizeBytes: 10, ModTime: modTime},
		}},
		Sessions: []opencode.Session{{
			ID:           "ses",
			ProjectID:    "proj",
			ProjectPath:  "/tmp/project",
			Title:        "Session",
			MessageCount: 1,
			PartCount:    1,
			Source:       opencode.FileRecord{Path: filepath.Join(root, "session", "proj", "ses.json"), SizeBytes: 11, ModTime: modTime},
			Messages: []opencode.Message{{
				ID:        "msg",
				SessionID: "ses",
				Role:      "assistant",
				Source:    opencode.FileRecord{Path: filepath.Join(root, "message", "ses", "msg.json"), SizeBytes: 12, ModTime: modTime},
				Parts:     []opencode.Part{part},
			}},
		}},
	}
}

func updateCount(t *testing.T, store *Store) int {
	t.Helper()
	var count int
	if err := store.db.QueryRow(`SELECT count(*) FROM update_count`).Scan(&count); err != nil {
		t.Fatalf("update count: %v", err)
	}
	if _, err := store.db.Exec(`DELETE FROM update_count`); err != nil {
		t.Fatalf("clear update count: %v", err)
	}
	return count
}

func columnExists(t *testing.T, store *Store, table, column string) bool {
	t.Helper()
	rows, err := store.db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		t.Fatalf("table info %s: %v", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull int
		var defaultValue any
		var primaryKey int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatalf("scan table info %s: %v", table, err)
		}
		if name == column {
			return true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("table info rows %s: %v", table, err)
	}
	return false
}

func mustExec(t *testing.T, db *sql.DB, query string) {
	t.Helper()
	if _, err := db.Exec(query); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}
