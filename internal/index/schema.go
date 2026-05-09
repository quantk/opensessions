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
  source_kind TEXT NOT NULL DEFAULT 'opencode',
  worktree TEXT,
  vcs TEXT,
  created_at INTEGER,
  updated_at INTEGER,
  source_path TEXT
)`,
		`CREATE TABLE IF NOT EXISTS sessions (
  id TEXT PRIMARY KEY,
  source_kind TEXT NOT NULL DEFAULT 'opencode',
  project_id TEXT,
  parent_id TEXT,
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
  source_kind TEXT NOT NULL DEFAULT 'opencode',
  session_id TEXT NOT NULL,
  parent_id TEXT,
  entry_type TEXT,
  append_order INTEGER NOT NULL DEFAULT 0,
  label TEXT,
  role TEXT,
  agent TEXT,
  summary_title TEXT,
  model_provider TEXT,
  model_id TEXT,
  token_usage_available INTEGER NOT NULL DEFAULT 0,
  token_total INTEGER NOT NULL DEFAULT 0,
  token_input INTEGER NOT NULL DEFAULT 0,
  token_output INTEGER NOT NULL DEFAULT 0,
  token_reasoning INTEGER NOT NULL DEFAULT 0,
  token_cache_read INTEGER NOT NULL DEFAULT 0,
  token_cache_write INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER,
  updated_at INTEGER,
  source_path TEXT,
  FOREIGN KEY(session_id) REFERENCES sessions(id) ON DELETE CASCADE
)`,
		`CREATE TABLE IF NOT EXISTS parts (
  id TEXT PRIMARY KEY,
  source_kind TEXT NOT NULL DEFAULT 'opencode',
  session_id TEXT NOT NULL,
  message_id TEXT NOT NULL,
  type TEXT,
  kind TEXT,
  tool_name TEXT,
  status TEXT,
  title TEXT,
  subagent_name TEXT,
  linked_session_id TEXT,
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
  source_kind TEXT NOT NULL DEFAULT 'opencode',
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
  source_kind TEXT NOT NULL DEFAULT 'opencode',
  size_bytes INTEGER NOT NULL,
  mod_time INTEGER NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS tags (
  session_id TEXT NOT NULL,
  source_kind TEXT NOT NULL DEFAULT 'opencode',
  tag TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  PRIMARY KEY(session_id, tag),
  FOREIGN KEY(session_id) REFERENCES sessions(id) ON DELETE CASCADE
)`,
		`CREATE TABLE IF NOT EXISTS bookmarks (
  session_id TEXT PRIMARY KEY,
  source_kind TEXT NOT NULL DEFAULT 'opencode',
  created_at INTEGER NOT NULL,
  FOREIGN KEY(session_id) REFERENCES sessions(id) ON DELETE CASCADE
)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_project ON sessions(project_id, updated_at)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_parts_session ON parts(session_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_searchable_session ON searchable_documents(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_searchable_session_part ON searchable_documents(session_id, part_id)`,
		`CREATE INDEX IF NOT EXISTS idx_searchable_part ON searchable_documents(part_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tags_tag_session ON tags(tag, session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_projects_source ON projects(source_path)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_source ON sessions(source_path)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_source ON messages(source_path)`,
		`CREATE INDEX IF NOT EXISTS idx_parts_source ON parts(source_path)`,
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("initialize schema: %w", err)
		}
	}
	if err := s.ensureColumn(ctx, "parts", "raw_json", "TEXT"); err != nil {
		return err
	}
	if err := s.ensureColumn(ctx, "sessions", "parent_id", "TEXT"); err != nil {
		return err
	}
	if err := s.ensureColumn(ctx, "parts", "linked_session_id", "TEXT"); err != nil {
		return err
	}
	if err := s.ensureColumn(ctx, "parts", "subagent_name", "TEXT"); err != nil {
		return err
	}
	for _, table := range []string{"projects", "sessions", "messages", "parts", "searchable_documents", "scan_metadata", "tags", "bookmarks"} {
		if err := s.ensureColumn(ctx, table, "source_kind", "TEXT NOT NULL DEFAULT 'opencode'"); err != nil {
			return err
		}
	}
	for _, column := range []struct {
		name       string
		definition string
	}{
		{"parent_id", "TEXT"},
		{"entry_type", "TEXT"},
		{"append_order", "INTEGER NOT NULL DEFAULT 0"},
		{"label", "TEXT"},
	} {
		if err := s.ensureColumn(ctx, "messages", column.name, column.definition); err != nil {
			return err
		}
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
		if err := s.ensureColumn(ctx, "messages", column.name, column.definition); err != nil {
			return err
		}
	}
	for _, statement := range []string{
		`CREATE INDEX IF NOT EXISTS idx_sessions_parent ON sessions(parent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_source_kind ON sessions(source_kind, updated_at)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_source_kind ON messages(source_kind, session_id, append_order)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_parent ON messages(parent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_parts_source_kind ON parts(source_kind, session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_searchable_source_kind ON searchable_documents(source_kind, session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_parts_linked_session ON parts(linked_session_id)`,
	} {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("initialize schema: %w", err)
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
