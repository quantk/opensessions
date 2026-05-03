package opencode

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverStorageRoot(t *testing.T) {
	t.Setenv("OPENSESSION_STORAGE_ROOT", "")
	t.Setenv("OPENCODE_STORAGE_ROOT", "")

	override := filepath.Join(t.TempDir(), "custom-storage")
	got, err := DiscoverStorageRoot(override)
	if err != nil {
		t.Fatalf("DiscoverStorageRoot override: %v", err)
	}
	if got != override {
		t.Fatalf("override root = %q, want %q", got, override)
	}

	envRoot := filepath.Join(t.TempDir(), "env-storage")
	t.Setenv("OPENSESSION_STORAGE_ROOT", envRoot)
	got, err = DiscoverStorageRoot("")
	if err != nil {
		t.Fatalf("DiscoverStorageRoot env: %v", err)
	}
	if got != envRoot {
		t.Fatalf("env root = %q, want %q", got, envRoot)
	}
}

func TestScanAssemblesSessionsAndClassifiesParts(t *testing.T) {
	root := fixtureRoot(t)
	snapshot, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(snapshot.Projects) != 2 {
		t.Fatalf("projects = %d, want 2", len(snapshot.Projects))
	}
	if len(snapshot.Sessions) != 2 {
		t.Fatalf("sessions = %d, want 2", len(snapshot.Sessions))
	}

	session := findSession(t, snapshot, "ses_fixture")
	if session.ProjectID != "proj1" {
		t.Fatalf("project ID = %q, want proj1", session.ProjectID)
	}
	if session.ProjectPath != "/tmp/fixture-project" {
		t.Fatalf("project path = %q", session.ProjectPath)
	}
	if session.MessageCount != 2 || session.PartCount != 7 || session.HeavyPartCount != 1 {
		t.Fatalf("counts = messages:%d parts:%d heavy:%d", session.MessageCount, session.PartCount, session.HeavyPartCount)
	}
	if !session.TokenUsage.Available || session.TokenUsage.Total != 321 || session.TokenUsage.Input != 100 || session.TokenUsage.Output != 70 || session.TokenUsage.Reasoning != 20 || session.TokenUsage.CacheRead != 30 || session.TokenUsage.CacheWrite != 10 {
		t.Fatalf("token usage = %#v", session.TokenUsage)
	}
	if session.ModelProvider != "openai" || session.ModelID != "gpt-test" {
		t.Fatalf("model = %q/%q", session.ModelProvider, session.ModelID)
	}

	if len(session.Messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(session.Messages))
	}
	if session.Messages[0].ID != "msg_user" || session.Messages[1].ID != "msg_assistant" {
		t.Fatalf("messages not chronological: %#v", []string{session.Messages[0].ID, session.Messages[1].ID})
	}
	if session.Messages[0].TokenUsage.Available || !session.Messages[1].TokenUsage.Available {
		t.Fatalf("message token availability = %#v / %#v", session.Messages[0].TokenUsage, session.Messages[1].TokenUsage)
	}

	global := findSession(t, snapshot, "ses_global")
	if global.TokenUsage.Available || global.TokenUsage.Total != 0 {
		t.Fatalf("global token usage = %#v, want unavailable", global.TokenUsage)
	}

	text := findPart(t, session, "prt_text")
	if text.Kind != PartKindText || !strings.Contains(text.IndexText, "find sessions") {
		t.Fatalf("text part kind/index = %q/%q", text.Kind, text.IndexText)
	}

	reasoning := findPart(t, session, "prt_reasoning")
	if reasoning.Kind != PartKindReasoning || !strings.Contains(reasoning.IndexText, "planner note") {
		t.Fatalf("reasoning part kind/index = %q/%q", reasoning.Kind, reasoning.IndexText)
	}

	tool := findPart(t, session, "prt_tool")
	if tool.Kind != PartKindTool || tool.ToolName != "bash" || tool.Status != "completed" {
		t.Fatalf("tool part = kind:%q tool:%q status:%q", tool.Kind, tool.ToolName, tool.Status)
	}
	if !strings.Contains(tool.IndexText, "List Go files") || !strings.Contains(tool.IndexText, "go test") {
		t.Fatalf("tool index text missing safe summary fields: %q", tool.IndexText)
	}

	file := findPart(t, session, "prt_file")
	if file.Kind != PartKindFile || file.FilePath != "README.md" {
		t.Fatalf("file part = kind:%q path:%q", file.Kind, file.FilePath)
	}
	if strings.Contains(file.IndexText, "base64") || !strings.Contains(file.IndexText, "README.md") {
		t.Fatalf("file index text should keep path and skip data URL: %q", file.IndexText)
	}

	heavy := findPart(t, session, "prt_heavy")
	if heavy.Kind != PartKindTool || !heavy.Heavy || !heavy.SkippedRaw {
		t.Fatalf("heavy tool flags = kind:%q heavy:%v skipped:%v", heavy.Kind, heavy.Heavy, heavy.SkippedRaw)
	}
	if strings.Contains(heavy.IndexText, "AAECAwQFBgc") || strings.Contains(heavy.Preview, "AAECAwQFBgc") {
		t.Fatalf("heavy raw payload leaked into index/preview: index=%q preview=%q", heavy.IndexText, heavy.Preview)
	}
}

func TestScanIncludesSQLiteDatabaseSessions(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "opencode", "storage")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir storage root: %v", err)
	}
	dbPath := filepath.Join(base, "opencode", "opencode.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite fixture: %v", err)
	}
	defer db.Close()
	mustExec(t, db, `CREATE TABLE project (id text PRIMARY KEY, worktree text NOT NULL, vcs text, time_created integer NOT NULL, time_updated integer NOT NULL)`)
	mustExec(t, db, `CREATE TABLE session (id text PRIMARY KEY, project_id text NOT NULL, slug text NOT NULL, directory text NOT NULL, title text NOT NULL, version text NOT NULL, time_created integer NOT NULL, time_updated integer NOT NULL)`)
	mustExec(t, db, `CREATE TABLE message (id text PRIMARY KEY, session_id text NOT NULL, time_created integer NOT NULL, time_updated integer NOT NULL, data text NOT NULL)`)
	mustExec(t, db, `CREATE TABLE part (id text PRIMARY KEY, message_id text NOT NULL, session_id text NOT NULL, time_created integer NOT NULL, time_updated integer NOT NULL, data text NOT NULL)`)
	mustExec(t, db, `INSERT INTO project (id, worktree, vcs, time_created, time_updated) VALUES ('proj-db', '/tmp/db-project', 'git', 1777800000000, 1777800000000)`)
	mustExec(t, db, `INSERT INTO session (id, project_id, slug, directory, title, version, time_created, time_updated) VALUES ('ses_db', 'proj-db', 'fresh', '/tmp/db-project', 'Fresh database session', '1.2.3', 1777800000000, 1777800100000)`)
	mustExec(t, db, `INSERT INTO message (id, session_id, time_created, time_updated, data) VALUES ('msg_db_user', 'ses_db', 1777800001000, 1777800001000, '{"role":"user","agent":"build","model":{"providerID":"openai","modelID":"gpt-5.5"},"summary":{"title":"Fresh question"}}')`)
	mustExec(t, db, `INSERT INTO message (id, session_id, time_created, time_updated, data) VALUES ('msg_db_assistant', 'ses_db', 1777800002000, 1777800002000, '{"role":"assistant","agent":"build","model":{"providerID":"openai","modelID":"gpt-5.5"},"tokens":{"input":100,"output":25,"reasoning":5,"cache":{"read":10,"write":3}},"cost":9.99}')`)
	mustExec(t, db, `INSERT INTO part (id, message_id, session_id, time_created, time_updated, data) VALUES ('prt_db_text', 'msg_db_user', 'ses_db', 1777800001000, 1777800001000, '{"type":"text","text":"fresh transcript from sqlite"}')`)

	snapshot, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	session := findSession(t, snapshot, "ses_db")
	if session.Title != "Fresh database session" || session.ProjectPath != "/tmp/db-project" {
		t.Fatalf("database session = %#v", session)
	}
	if session.MessageCount != 2 || session.PartCount != 1 {
		t.Fatalf("database counts = messages:%d parts:%d", session.MessageCount, session.PartCount)
	}
	if !session.TokenUsage.Available || session.TokenUsage.Total != 143 || session.TokenUsage.Input != 100 || session.TokenUsage.Output != 25 || session.TokenUsage.Reasoning != 5 || session.TokenUsage.CacheRead != 10 || session.TokenUsage.CacheWrite != 3 {
		t.Fatalf("database token usage = %#v", session.TokenUsage)
	}
	part := findPart(t, session, "prt_db_text")
	if !strings.Contains(part.IndexText, "fresh transcript") || !strings.Contains(part.RawJSON, "fresh transcript") {
		t.Fatalf("database part text/raw = index:%q raw:%q", part.IndexText, part.RawJSON)
	}
}

func TestScanDoesNotModifyStorage(t *testing.T) {
	root := fixtureRoot(t)
	path := filepath.Join(root, "part", "msg_assistant", "prt_tool.json")
	before, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat before: %v", err)
	}

	if _, err := Scan(root); err != nil {
		t.Fatalf("Scan: %v", err)
	}

	after, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat after: %v", err)
	}
	if before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		t.Fatalf("scanner modified storage file: before size=%d mod=%s after size=%d mod=%s", before.Size(), before.ModTime(), after.Size(), after.ModTime())
	}
}

func mustExec(t *testing.T, db *sql.DB, query string) {
	t.Helper()
	if _, err := db.Exec(query); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

func fixtureRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join("..", "..", "testdata", "opencode", "storage")
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatalf("abs fixture root: %v", err)
	}
	return abs
}

func findSession(t *testing.T, snapshot Snapshot, id string) Session {
	t.Helper()
	for _, session := range snapshot.Sessions {
		if session.ID == id {
			return session
		}
	}
	t.Fatalf("session %q not found", id)
	return Session{}
}

func findPart(t *testing.T, session Session, id string) Part {
	t.Helper()
	for _, message := range session.Messages {
		for _, part := range message.Parts {
			if part.ID == id {
				return part
			}
		}
	}
	t.Fatalf("part %q not found", id)
	return Part{}
}
