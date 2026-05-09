package index

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/quantick/opensession/internal/opencode"
	"github.com/quantick/opensession/internal/pi"
	"github.com/quantick/opensession/internal/source"
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

func TestStoreMigratesPreSourceDatabaseAsOpenCode(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "opensession.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open fixture db: %v", err)
	}
	mustExec(t, db, `CREATE TABLE projects (id TEXT PRIMARY KEY, worktree TEXT, vcs TEXT, created_at INTEGER, updated_at INTEGER, source_path TEXT)`)
	mustExec(t, db, `CREATE TABLE sessions (id TEXT PRIMARY KEY, project_id TEXT, parent_id TEXT, project_path TEXT, directory TEXT, title TEXT, slug TEXT, version TEXT, model_provider TEXT, model_id TEXT, created_at INTEGER, updated_at INTEGER, message_count INTEGER NOT NULL DEFAULT 0, part_count INTEGER NOT NULL DEFAULT 0, heavy_part_count INTEGER NOT NULL DEFAULT 0, token_usage_available INTEGER NOT NULL DEFAULT 0, token_total INTEGER NOT NULL DEFAULT 0, token_input INTEGER NOT NULL DEFAULT 0, token_output INTEGER NOT NULL DEFAULT 0, token_reasoning INTEGER NOT NULL DEFAULT 0, token_cache_read INTEGER NOT NULL DEFAULT 0, token_cache_write INTEGER NOT NULL DEFAULT 0, source_path TEXT)`)
	mustExec(t, db, `CREATE TABLE messages (id TEXT PRIMARY KEY, session_id TEXT NOT NULL, role TEXT, agent TEXT, summary_title TEXT, model_provider TEXT, model_id TEXT, token_usage_available INTEGER NOT NULL DEFAULT 0, token_total INTEGER NOT NULL DEFAULT 0, token_input INTEGER NOT NULL DEFAULT 0, token_output INTEGER NOT NULL DEFAULT 0, token_reasoning INTEGER NOT NULL DEFAULT 0, token_cache_read INTEGER NOT NULL DEFAULT 0, token_cache_write INTEGER NOT NULL DEFAULT 0, created_at INTEGER, updated_at INTEGER, source_path TEXT)`)
	mustExec(t, db, `CREATE TABLE parts (id TEXT PRIMARY KEY, session_id TEXT NOT NULL, message_id TEXT NOT NULL, type TEXT, kind TEXT, tool_name TEXT, status TEXT, title TEXT, subagent_name TEXT, linked_session_id TEXT, file_path TEXT, mime TEXT, filename TEXT, preview TEXT, index_text TEXT, raw_json TEXT, source_path TEXT, size_bytes INTEGER, heavy INTEGER NOT NULL DEFAULT 0, binary INTEGER NOT NULL DEFAULT 0, skipped_raw INTEGER NOT NULL DEFAULT 0, created_at INTEGER, updated_at INTEGER)`)
	mustExec(t, db, `CREATE TABLE searchable_documents (id INTEGER PRIMARY KEY AUTOINCREMENT, session_id TEXT NOT NULL, part_id TEXT NOT NULL, scope TEXT NOT NULL, content TEXT NOT NULL, UNIQUE(session_id, part_id, scope))`)
	mustExec(t, db, `CREATE TABLE scan_metadata (path TEXT PRIMARY KEY, size_bytes INTEGER NOT NULL, mod_time INTEGER NOT NULL)`)
	mustExec(t, db, `CREATE TABLE tags (session_id TEXT NOT NULL, tag TEXT NOT NULL, created_at INTEGER NOT NULL, PRIMARY KEY(session_id, tag))`)
	mustExec(t, db, `CREATE TABLE bookmarks (session_id TEXT PRIMARY KEY, created_at INTEGER NOT NULL)`)
	mustExec(t, db, `INSERT INTO projects (id, worktree) VALUES ('proj_old', '/tmp/old')`)
	mustExec(t, db, `INSERT INTO sessions (id, project_id, project_path, directory, title, slug, version, model_provider, model_id, created_at, updated_at, message_count, part_count) VALUES ('ses_old', 'proj_old', '/tmp/old', '/tmp/old', 'Old session', '', '', '', '', 1777800000000, 1777800000000, 1, 1)`)
	mustExec(t, db, `INSERT INTO messages (id, session_id, role, created_at) VALUES ('msg_old', 'ses_old', 'assistant', 1777800000000)`)
	mustExec(t, db, `INSERT INTO parts (id, session_id, message_id, kind, preview, index_text, created_at) VALUES ('prt_old', 'ses_old', 'msg_old', 'text', 'old preview', 'old searchable', 1777800000000)`)
	if err := db.Close(); err != nil {
		t.Fatalf("close fixture db: %v", err)
	}

	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open migrated db: %v", err)
	}
	defer store.Close()

	sessions, err := store.ListSessions(ctx)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].ID != "ses_old" || sessions[0].SourceKind != "opencode" {
		t.Fatalf("migrated sessions = %#v", sessions)
	}
	if err := store.SetTag(ctx, "ses_old", "legacy"); err != nil {
		t.Fatalf("SetTag: %v", err)
	}
	if err := store.SetBookmark(ctx, "ses_old", true); err != nil {
		t.Fatalf("SetBookmark: %v", err)
	}
	tags, err := store.Tags(ctx, "ses_old")
	if err != nil || len(tags) != 1 || tags[0] != "legacy" {
		t.Fatalf("Tags = %#v err=%v", tags, err)
	}
	bookmarked, err := store.IsBookmarked(ctx, "ses_old")
	if err != nil || !bookmarked {
		t.Fatalf("IsBookmarked = %v err=%v", bookmarked, err)
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

func TestStoreIndexesPiBranchesAndRefreshesChangedFiles(t *testing.T) {
	ctx := context.Background()
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()
	root := t.TempDir()
	path := filepath.Join(root, "project", "session.jsonl")
	writePiJSONL(t, path, []string{
		`{"type":"session","id":"same-id","timestamp":"2026-05-09T10:00:00Z","cwd":"/tmp/pi"}`,
		`{"type":"message","id":"u1","parentId":null,"timestamp":"2026-05-09T10:00:01Z","message":{"role":"user","content":"root prompt"}}`,
		`{"type":"message","id":"a1","parentId":"u1","timestamp":"2026-05-09T10:00:02Z","message":{"role":"assistant","content":[{"type":"text","text":"shared answer"}]}}`,
		`{"type":"message","id":"branch-a","parentId":"a1","timestamp":"2026-05-09T10:00:03Z","message":{"role":"user","content":"branch A only"}}`,
		`{"type":"message","id":"branch-b","parentId":"a1","timestamp":"2026-05-09T10:00:04Z","message":{"role":"user","content":"branch B latest"}}`,
		`{"type":"label","id":"label-b","parentId":"branch-b","timestamp":"2026-05-09T10:00:05Z","targetId":"branch-b","label":"chosen"}`,
	})
	snapshot, err := pi.Scan(root)
	if err != nil {
		t.Fatalf("pi.Scan: %v", err)
	}
	if err := store.UpsertSnapshot(ctx, snapshot); err != nil {
		t.Fatalf("UpsertSnapshot pi: %v", err)
	}
	sessions, err := store.ListSessions(ctx)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].SourceKind != string(source.KindPi) || sessions[0].ID != "pi:same-id" {
		t.Fatalf("pi sessions = %#v", sessions)
	}
	timeline, err := store.SessionTimeline(ctx, "pi:same-id")
	if err != nil {
		t.Fatalf("SessionTimeline pi: %v", err)
	}
	if containsTimelineText(timeline, "branch A only") || !containsTimelineText(timeline, "branch B latest") {
		t.Fatalf("latest branch timeline = %#v", timeline)
	}
	allBranchSearch, err := store.SearchSessions(ctx, "branch A only")
	if err != nil || len(allBranchSearch) != 1 {
		t.Fatalf("SearchSessions all branches = %#v err=%v", allBranchSearch, err)
	}
	currentBranchSearch, err := store.SearchSession(ctx, "pi:same-id", "branch A only")
	if err != nil {
		t.Fatalf("SearchSession branch: %v", err)
	}
	if len(currentBranchSearch) != 0 {
		t.Fatalf("timeline search should use latest branch, got %#v", currentBranchSearch)
	}
	tree, err := store.SessionTree(ctx, "pi:same-id")
	if err != nil {
		t.Fatalf("SessionTree: %v", err)
	}
	if label := treeLabel(tree, "pi:same-id:branch-b"); label != "chosen" {
		t.Fatalf("branch label = %q", label)
	}
	branchA, err := store.SessionTimelineForEntry(ctx, "pi:same-id", "pi:same-id:branch-a")
	if err != nil {
		t.Fatalf("SessionTimelineForEntry: %v", err)
	}
	if !containsTimelineText(branchA, "branch A only") || containsTimelineText(branchA, "branch B latest") {
		t.Fatalf("branch A timeline = %#v", branchA)
	}

	changedTime := time.Date(2026, 5, 9, 11, 0, 0, 0, time.UTC)
	writePiJSONL(t, path, []string{
		`{"type":"session","id":"same-id","timestamp":"2026-05-09T11:00:00Z","cwd":"/tmp/pi"}`,
		`{"type":"message","id":"u1","parentId":null,"timestamp":"2026-05-09T11:00:01Z","message":{"role":"user","content":"fresh only"}}`,
	})
	if err := os.Chtimes(path, changedTime, changedTime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	changed, err := pi.Scan(root)
	if err != nil {
		t.Fatalf("pi.Scan changed: %v", err)
	}
	if err := store.UpsertSnapshot(ctx, changed); err != nil {
		t.Fatalf("UpsertSnapshot changed pi: %v", err)
	}
	stale, err := store.SearchSessions(ctx, "branch B latest")
	if err != nil {
		t.Fatalf("SearchSessions stale: %v", err)
	}
	if len(stale) != 0 {
		t.Fatalf("stale Pi content remained: %#v", stale)
	}
	fresh, err := store.SearchSessions(ctx, "fresh only")
	if err != nil || len(fresh) != 1 {
		t.Fatalf("fresh Pi content missing: %#v err=%v", fresh, err)
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

func writePiJSONL(t *testing.T, path string, lines []string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir Pi JSONL: %v", err)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write Pi JSONL: %v", err)
	}
}

func containsTimelineText(parts []TimelinePart, text string) bool {
	for _, part := range parts {
		if strings.Contains(part.Preview, text) || strings.Contains(part.IndexText, text) {
			return true
		}
	}
	return false
}

func treeLabel(entries []SessionTreeEntry, id string) string {
	for _, entry := range entries {
		if entry.ID == id {
			return entry.Label
		}
	}
	return ""
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
