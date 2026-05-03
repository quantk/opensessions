package index

import (
	"context"
	"fmt"
)

func (s *Store) initSchema(ctx context.Context) error {
	statements := []string{
		`PRAGMA foreign_keys = ON`,
		`CREATE TABLE IF NOT EXISTS projects (
  id TEXT PRIMARY KEY,
  worktree TEXT,
  vcs TEXT,
  created_at INTEGER,
  updated_at INTEGER,
  source_path TEXT
)`,
		`CREATE TABLE IF NOT EXISTS sessions (
  id TEXT PRIMARY KEY,
  project_id TEXT,
  project_path TEXT,
  directory TEXT,
  title TEXT,
  slug TEXT,
  version TEXT,
  model_provider TEXT,
  model_id TEXT,
  created_at INTEGER,
  updated_at INTEGER,
  message_count INTEGER NOT NULL DEFAULT 0,
  part_count INTEGER NOT NULL DEFAULT 0,
  heavy_part_count INTEGER NOT NULL DEFAULT 0,
  token_usage_available INTEGER NOT NULL DEFAULT 0,
  token_total INTEGER NOT NULL DEFAULT 0,
  token_input INTEGER NOT NULL DEFAULT 0,
  token_output INTEGER NOT NULL DEFAULT 0,
  token_reasoning INTEGER NOT NULL DEFAULT 0,
  token_cache_read INTEGER NOT NULL DEFAULT 0,
  token_cache_write INTEGER NOT NULL DEFAULT 0,
  source_path TEXT
)`,
		`CREATE TABLE IF NOT EXISTS messages (
  id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL,
  role TEXT,
  agent TEXT,
  summary_title TEXT,
  model_provider TEXT,
  model_id TEXT,
  created_at INTEGER,
  updated_at INTEGER,
  source_path TEXT,
  FOREIGN KEY(session_id) REFERENCES sessions(id) ON DELETE CASCADE
)`,
		`CREATE TABLE IF NOT EXISTS parts (
  id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL,
  message_id TEXT NOT NULL,
  type TEXT,
  kind TEXT,
  tool_name TEXT,
  status TEXT,
  title TEXT,
  file_path TEXT,
  mime TEXT,
  filename TEXT,
  preview TEXT,
  index_text TEXT,
  raw_json TEXT,
  source_path TEXT,
  size_bytes INTEGER,
  heavy INTEGER NOT NULL DEFAULT 0,
  binary INTEGER NOT NULL DEFAULT 0,
  skipped_raw INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER,
  updated_at INTEGER,
  FOREIGN KEY(session_id) REFERENCES sessions(id) ON DELETE CASCADE,
  FOREIGN KEY(message_id) REFERENCES messages(id) ON DELETE CASCADE
)`,
		`CREATE TABLE IF NOT EXISTS searchable_documents (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id TEXT NOT NULL,
  part_id TEXT NOT NULL,
  scope TEXT NOT NULL,
  content TEXT NOT NULL,
  UNIQUE(session_id, part_id, scope),
  FOREIGN KEY(session_id) REFERENCES sessions(id) ON DELETE CASCADE,
  FOREIGN KEY(part_id) REFERENCES parts(id) ON DELETE CASCADE
)`,
		`CREATE TABLE IF NOT EXISTS scan_metadata (
  path TEXT PRIMARY KEY,
  size_bytes INTEGER NOT NULL,
  mod_time INTEGER NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS tags (
  session_id TEXT NOT NULL,
  tag TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  PRIMARY KEY(session_id, tag),
  FOREIGN KEY(session_id) REFERENCES sessions(id) ON DELETE CASCADE
)`,
		`CREATE TABLE IF NOT EXISTS bookmarks (
  session_id TEXT PRIMARY KEY,
  created_at INTEGER NOT NULL,
  FOREIGN KEY(session_id) REFERENCES sessions(id) ON DELETE CASCADE
)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_project ON sessions(project_id, updated_at)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_parts_session ON parts(session_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_searchable_session ON searchable_documents(session_id)`,
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("initialize schema: %w", err)
		}
	}
	if err := s.ensureColumn(ctx, "parts", "raw_json", "TEXT"); err != nil {
		return err
	}
	for _, column := range []struct {
		name       string
		definition string
	}{
		{"token_usage_available", "INTEGER NOT NULL DEFAULT 0"},
		{"token_total", "INTEGER NOT NULL DEFAULT 0"},
		{"token_input", "INTEGER NOT NULL DEFAULT 0"},
		{"token_output", "INTEGER NOT NULL DEFAULT 0"},
		{"token_reasoning", "INTEGER NOT NULL DEFAULT 0"},
		{"token_cache_read", "INTEGER NOT NULL DEFAULT 0"},
		{"token_cache_write", "INTEGER NOT NULL DEFAULT 0"},
	} {
		if err := s.ensureColumn(ctx, "sessions", column.name, column.definition); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ensureColumn(ctx context.Context, table, column, definition string) error {
	rows, err := s.db.QueryContext(ctx, `PRAGMA table_info(`+table+`)`)
	if err != nil {
		return fmt.Errorf("inspect schema %s: %w", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull int
		var defaultValue any
		var primaryKey int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return fmt.Errorf("inspect schema %s: %w", table, err)
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("inspect schema %s: %w", table, err)
	}
	if _, err := s.db.ExecContext(ctx, `ALTER TABLE `+table+` ADD COLUMN `+column+` `+definition); err != nil {
		return fmt.Errorf("migrate schema %s.%s: %w", table, column, err)
	}
	return nil
}
