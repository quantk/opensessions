package index

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/quantick/opensession/internal/opencode"
	_ "modernc.org/sqlite"
)

const defaultDBName = "opensession.sqlite"

type Store struct {
	db *sql.DB
}

type SessionSummary struct {
	ID             string
	ProjectID      string
	ParentID       string
	ProjectPath    string
	Directory      string
	Title          string
	ModelProvider  string
	ModelID        string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	MessageCount   int
	PartCount      int
	HeavyPartCount int
	TokenUsage     opencode.TokenUsage
	Tags           []string
	Bookmarked     bool
}

type TimelinePart struct {
	PartID          string
	SessionID       string
	MessageID       string
	Role            string
	Type            string
	Kind            opencode.PartKind
	ToolName        string
	Status          string
	Title           string
	SubagentName    string
	LinkedSessionID string
	FilePath        string
	Preview         string
	IndexText       string
	RawJSON         string
	SourcePath      string
	SizeBytes       int64
	Heavy           bool
	Binary          bool
	SkippedRaw      bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type RawPart struct {
	PartID     string
	SessionID  string
	MessageID  string
	Role       string
	Type       string
	Kind       opencode.PartKind
	ToolName   string
	Status     string
	Title      string
	FilePath   string
	SourcePath string
	SizeBytes  int64
	Heavy      bool
	Binary     bool
	SkippedRaw bool
	Preview    string
	IndexText  string
	RawJSON    string
}

type ScanMetadata struct {
	Path      string
	SizeBytes int64
	ModTime   time.Time
}

func DefaultPath(override string) (string, error) {
	if override != "" {
		return filepath.Clean(override), nil
	}
	if env := os.Getenv("OPENSESSION_DB"); env != "" {
		return filepath.Clean(env), nil
	}
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, "opensession", defaultDBName), nil
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", errors.New("cannot determine home directory for default database path")
	}
	return filepath.Join(home, ".local", "state", "opensession", defaultDBName), nil
}

func Open(path string) (*Store, error) {
	if path == "" {
		var err error
		path, err = DefaultPath("")
		if err != nil {
			return nil, err
		}
	}
	if shouldCreateParent(path) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, fmt.Errorf("create database directory: %w", err)
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	store := &Store{db: db}
	if err := store.initSchema(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) UpsertSnapshot(ctx context.Context, snapshot opencode.Snapshot) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)

	for _, project := range snapshot.Projects {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO projects (id, worktree, vcs, created_at, updated_at, source_path)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  worktree = excluded.worktree,
  vcs = excluded.vcs,
  created_at = excluded.created_at,
  updated_at = excluded.updated_at,
  source_path = excluded.source_path`, project.ID, project.Worktree, project.VCS, millis(project.CreatedAt), millis(project.UpdatedAt), project.Source.Path); err != nil {
			return fmt.Errorf("upsert project %s: %w", project.ID, err)
		}
		if err := upsertScanMetadataTx(ctx, tx, project.Source.Path, project.Source.SizeBytes, project.Source.ModTime); err != nil {
			return err
		}
	}

	for _, session := range snapshot.Sessions {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO sessions (id, project_id, parent_id, project_path, directory, title, slug, version, model_provider, model_id, created_at, updated_at, message_count, part_count, heavy_part_count, token_usage_available, token_total, token_input, token_output, token_reasoning, token_cache_read, token_cache_write, source_path)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  project_id = excluded.project_id,
  parent_id = excluded.parent_id,
  project_path = excluded.project_path,
  directory = excluded.directory,
  title = excluded.title,
  slug = excluded.slug,
  version = excluded.version,
  model_provider = excluded.model_provider,
  model_id = excluded.model_id,
  created_at = excluded.created_at,
  updated_at = excluded.updated_at,
  message_count = excluded.message_count,
  part_count = excluded.part_count,
  heavy_part_count = excluded.heavy_part_count,
  token_usage_available = excluded.token_usage_available,
  token_total = excluded.token_total,
  token_input = excluded.token_input,
  token_output = excluded.token_output,
  token_reasoning = excluded.token_reasoning,
  token_cache_read = excluded.token_cache_read,
  token_cache_write = excluded.token_cache_write,
  source_path = excluded.source_path`, session.ID, session.ProjectID, session.ParentID, session.ProjectPath, session.Directory, session.Title, session.Slug, session.Version, session.ModelProvider, session.ModelID, millis(session.CreatedAt), millis(session.UpdatedAt), session.MessageCount, session.PartCount, session.HeavyPartCount, boolInt(session.TokenUsage.Available), session.TokenUsage.Total, session.TokenUsage.Input, session.TokenUsage.Output, session.TokenUsage.Reasoning, session.TokenUsage.CacheRead, session.TokenUsage.CacheWrite, session.Source.Path); err != nil {
			return fmt.Errorf("upsert session %s: %w", session.ID, err)
		}
		if err := upsertScanMetadataTx(ctx, tx, session.Source.Path, session.Source.SizeBytes, session.Source.ModTime); err != nil {
			return err
		}

		for _, message := range session.Messages {
			if _, err := tx.ExecContext(ctx, `
INSERT INTO messages (id, session_id, role, agent, summary_title, model_provider, model_id, created_at, updated_at, source_path)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  session_id = excluded.session_id,
  role = excluded.role,
  agent = excluded.agent,
  summary_title = excluded.summary_title,
  model_provider = excluded.model_provider,
  model_id = excluded.model_id,
  created_at = excluded.created_at,
  updated_at = excluded.updated_at,
  source_path = excluded.source_path`, message.ID, message.SessionID, message.Role, message.Agent, message.SummaryTitle, message.ModelProvider, message.ModelID, millis(message.CreatedAt), millis(message.UpdatedAt), message.Source.Path); err != nil {
				return fmt.Errorf("upsert message %s: %w", message.ID, err)
			}
			if err := upsertScanMetadataTx(ctx, tx, message.Source.Path, message.Source.SizeBytes, message.Source.ModTime); err != nil {
				return err
			}

			for _, part := range message.Parts {
				if _, err := tx.ExecContext(ctx, `
INSERT INTO parts (id, session_id, message_id, type, kind, tool_name, status, title, subagent_name, linked_session_id, file_path, mime, filename, preview, index_text, raw_json, source_path, size_bytes, heavy, binary, skipped_raw, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  session_id = excluded.session_id,
  message_id = excluded.message_id,
  type = excluded.type,
  kind = excluded.kind,
  tool_name = excluded.tool_name,
  status = excluded.status,
  title = excluded.title,
  subagent_name = excluded.subagent_name,
  linked_session_id = excluded.linked_session_id,
  file_path = excluded.file_path,
  mime = excluded.mime,
  filename = excluded.filename,
  preview = excluded.preview,
  index_text = excluded.index_text,
  raw_json = excluded.raw_json,
  source_path = excluded.source_path,
  size_bytes = excluded.size_bytes,
  heavy = excluded.heavy,
  binary = excluded.binary,
  skipped_raw = excluded.skipped_raw,
				  created_at = excluded.created_at,
				  updated_at = excluded.updated_at`, part.ID, part.SessionID, part.MessageID, part.Type, string(part.Kind), part.ToolName, part.Status, part.Title, part.SubagentName, part.LinkedSessionID, part.FilePath, part.MIME, part.Filename, part.Preview, part.IndexText, part.RawJSON, part.Source.Path, part.SizeBytes, boolInt(part.Heavy), boolInt(part.Binary), boolInt(part.SkippedRaw), millis(part.CreatedAt), millis(part.UpdatedAt)); err != nil {
					return fmt.Errorf("upsert part %s: %w", part.ID, err)
				}
				if err := upsertScanMetadataTx(ctx, tx, part.Source.Path, part.Source.SizeBytes, part.Source.ModTime); err != nil {
					return err
				}
				if part.IndexText != "" {
					if _, err := tx.ExecContext(ctx, `
INSERT INTO searchable_documents (session_id, part_id, scope, content)
VALUES (?, ?, 'part', ?)
ON CONFLICT(session_id, part_id, scope) DO UPDATE SET content = excluded.content`, part.SessionID, part.ID, part.IndexText); err != nil {
						return fmt.Errorf("upsert searchable document %s: %w", part.ID, err)
					}
				}
			}
		}
	}

	return tx.Commit()
}

func (s *Store) ListSessions(ctx context.Context) ([]SessionSummary, error) {
	return s.querySessions(ctx, `SELECT id, project_id, coalesce(parent_id, ''), project_path, directory, title, model_provider, model_id, created_at, updated_at, message_count, part_count, heavy_part_count, token_usage_available, token_total, token_input, token_output, token_reasoning, token_cache_read, token_cache_write FROM sessions WHERE coalesce(parent_id, '') = '' ORDER BY updated_at DESC, id`, nil)
}

func (s *Store) Session(ctx context.Context, sessionID string) (SessionSummary, error) {
	sessions, err := s.querySessions(ctx, `SELECT id, project_id, coalesce(parent_id, ''), project_path, directory, title, model_provider, model_id, created_at, updated_at, message_count, part_count, heavy_part_count, token_usage_available, token_total, token_input, token_output, token_reasoning, token_cache_read, token_cache_write FROM sessions WHERE id = ?`, []any{sessionID})
	if err != nil {
		return SessionSummary{}, err
	}
	if len(sessions) == 0 {
		return SessionSummary{}, sql.ErrNoRows
	}
	return sessions[0], nil
}

func (s *Store) SearchSessions(ctx context.Context, query string) ([]SessionSummary, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return s.ListSessions(ctx)
	}
	like := "%" + strings.ToLower(query) + "%"
	return s.querySessions(ctx, `
SELECT DISTINCT s.id, s.project_id, coalesce(s.parent_id, ''), s.project_path, s.directory, s.title, s.model_provider, s.model_id, s.created_at, s.updated_at, s.message_count, s.part_count, s.heavy_part_count, s.token_usage_available, s.token_total, s.token_input, s.token_output, s.token_reasoning, s.token_cache_read, s.token_cache_write
FROM sessions s
LEFT JOIN searchable_documents d ON d.session_id = s.id
LEFT JOIN tags t ON t.session_id = s.id
WHERE coalesce(s.parent_id, '') = ''
  AND (lower(coalesce(s.title, '')) LIKE ?
   OR lower(coalesce(s.project_path, '')) LIKE ?
   OR lower(coalesce(s.directory, '')) LIKE ?
   OR lower(coalesce(s.model_provider, '')) LIKE ?
   OR lower(coalesce(s.model_id, '')) LIKE ?
   OR lower(coalesce(d.content, '')) LIKE ?
   OR lower(coalesce(t.tag, '')) LIKE ?)
ORDER BY s.updated_at DESC, s.id`, []any{like, like, like, like, like, like, like})
}

func (s *Store) SessionTimeline(ctx context.Context, sessionID string) ([]TimelinePart, error) {
	return s.queryTimeline(ctx, `
SELECT p.id, p.session_id, p.message_id, m.role, p.type, p.kind, p.tool_name, p.status, p.title, coalesce(p.subagent_name, ''), coalesce(p.linked_session_id, ''), p.file_path, p.preview, p.index_text, coalesce(p.raw_json, ''), p.source_path, p.size_bytes, p.heavy, p.binary, p.skipped_raw, p.created_at, p.updated_at
FROM parts p
JOIN messages m ON m.id = p.message_id
WHERE p.session_id = ?
ORDER BY m.created_at, p.created_at, p.id`, sessionID)
}

func (s *Store) SearchSession(ctx context.Context, sessionID, query string) ([]TimelinePart, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return s.SessionTimeline(ctx, sessionID)
	}
	like := "%" + strings.ToLower(query) + "%"
	return s.queryTimeline(ctx, `
SELECT DISTINCT p.id, p.session_id, p.message_id, m.role, p.type, p.kind, p.tool_name, p.status, p.title, coalesce(p.subagent_name, ''), coalesce(p.linked_session_id, ''), p.file_path, p.preview, p.index_text, coalesce(p.raw_json, ''), p.source_path, p.size_bytes, p.heavy, p.binary, p.skipped_raw, p.created_at, p.updated_at
FROM parts p
JOIN messages m ON m.id = p.message_id
LEFT JOIN searchable_documents d ON d.part_id = p.id
WHERE p.session_id = ?
  AND (lower(coalesce(p.preview, '')) LIKE ? OR lower(coalesce(p.index_text, '')) LIKE ? OR lower(coalesce(d.content, '')) LIKE ?)
ORDER BY m.created_at, p.created_at, p.id`, sessionID, like, like, like)
}

func (s *Store) SetTag(ctx context.Context, sessionID, tag string) error {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO tags (session_id, tag, created_at) VALUES (?, ?, ?)`, sessionID, tag, millis(time.Now()))
	return err
}

func (s *Store) Tags(ctx context.Context, sessionID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT tag FROM tags WHERE session_id = ? ORDER BY tag`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func (s *Store) SetBookmark(ctx context.Context, sessionID string, bookmarked bool) error {
	if !bookmarked {
		_, err := s.db.ExecContext(ctx, `DELETE FROM bookmarks WHERE session_id = ?`, sessionID)
		return err
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO bookmarks (session_id, created_at) VALUES (?, ?) ON CONFLICT(session_id) DO UPDATE SET created_at = excluded.created_at`, sessionID, millis(time.Now()))
	return err
}

func (s *Store) IsBookmarked(ctx context.Context, sessionID string) (bool, error) {
	var value int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM bookmarks WHERE session_id = ?`, sessionID).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return value == 1, nil
}

func (s *Store) RawPart(ctx context.Context, partID string) (RawPart, error) {
	var part RawPart
	var kind string
	var heavy, binary, skipped int
	err := s.db.QueryRowContext(ctx, `
SELECT p.id, p.session_id, p.message_id, m.role, p.type, p.kind, p.tool_name, p.status, p.title, p.file_path, p.source_path, p.size_bytes, p.heavy, p.binary, p.skipped_raw, p.preview, p.index_text, coalesce(p.raw_json, '')
FROM parts p
JOIN messages m ON m.id = p.message_id
WHERE p.id = ?`, partID).Scan(&part.PartID, &part.SessionID, &part.MessageID, &part.Role, &part.Type, &kind, &part.ToolName, &part.Status, &part.Title, &part.FilePath, &part.SourcePath, &part.SizeBytes, &heavy, &binary, &skipped, &part.Preview, &part.IndexText, &part.RawJSON)
	if err != nil {
		return RawPart{}, err
	}
	part.Kind = opencode.PartKind(kind)
	part.Heavy = heavy == 1
	part.Binary = binary == 1
	part.SkippedRaw = skipped == 1
	return part, nil
}

func (s *Store) UpsertScanMetadata(ctx context.Context, path string, sizeBytes int64, modTime time.Time) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO scan_metadata (path, size_bytes, mod_time)
VALUES (?, ?, ?)
ON CONFLICT(path) DO UPDATE SET size_bytes = excluded.size_bytes, mod_time = excluded.mod_time`, path, sizeBytes, nanos(modTime))
	return err
}

func (s *Store) ScanMetadata(ctx context.Context, path string) (ScanMetadata, bool, error) {
	var metadata ScanMetadata
	var mod int64
	err := s.db.QueryRowContext(ctx, `SELECT path, size_bytes, mod_time FROM scan_metadata WHERE path = ?`, path).Scan(&metadata.Path, &metadata.SizeBytes, &mod)
	if errors.Is(err, sql.ErrNoRows) {
		return ScanMetadata{}, false, nil
	}
	if err != nil {
		return ScanMetadata{}, false, err
	}
	metadata.ModTime = fromNanos(mod)
	return metadata, true, nil
}

func (s *Store) querySessions(ctx context.Context, query string, args []any) ([]SessionSummary, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	var sessions []SessionSummary
	for rows.Next() {
		var session SessionSummary
		var created, updated int64
		var tokenUsageAvailable int
		if err := rows.Scan(&session.ID, &session.ProjectID, &session.ParentID, &session.ProjectPath, &session.Directory, &session.Title, &session.ModelProvider, &session.ModelID, &created, &updated, &session.MessageCount, &session.PartCount, &session.HeavyPartCount, &tokenUsageAvailable, &session.TokenUsage.Total, &session.TokenUsage.Input, &session.TokenUsage.Output, &session.TokenUsage.Reasoning, &session.TokenUsage.CacheRead, &session.TokenUsage.CacheWrite); err != nil {
			_ = rows.Close()
			return nil, err
		}
		session.CreatedAt = fromMillis(created)
		session.UpdatedAt = fromMillis(updated)
		session.TokenUsage.Available = tokenUsageAvailable == 1
		sessions = append(sessions, session)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range sessions {
		tags, err := s.Tags(ctx, sessions[i].ID)
		if err != nil {
			return nil, err
		}
		sessions[i].Tags = tags
		bookmarked, err := s.IsBookmarked(ctx, sessions[i].ID)
		if err != nil {
			return nil, err
		}
		sessions[i].Bookmarked = bookmarked
	}
	return sessions, nil
}

func (s *Store) queryTimeline(ctx context.Context, query string, args ...any) ([]TimelinePart, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var parts []TimelinePart
	for rows.Next() {
		var part TimelinePart
		var kind string
		var heavy, binary, skipped int
		var created, updated int64
		if err := rows.Scan(&part.PartID, &part.SessionID, &part.MessageID, &part.Role, &part.Type, &kind, &part.ToolName, &part.Status, &part.Title, &part.SubagentName, &part.LinkedSessionID, &part.FilePath, &part.Preview, &part.IndexText, &part.RawJSON, &part.SourcePath, &part.SizeBytes, &heavy, &binary, &skipped, &created, &updated); err != nil {
			return nil, err
		}
		part.Kind = opencode.PartKind(kind)
		part.Heavy = heavy == 1
		part.Binary = binary == 1
		part.SkippedRaw = skipped == 1
		part.CreatedAt = fromMillis(created)
		part.UpdatedAt = fromMillis(updated)
		parts = append(parts, part)
	}
	return parts, rows.Err()
}

func upsertScanMetadataTx(ctx context.Context, tx *sql.Tx, path string, sizeBytes int64, modTime time.Time) error {
	if path == "" {
		return nil
	}
	_, err := tx.ExecContext(ctx, `
INSERT INTO scan_metadata (path, size_bytes, mod_time)
VALUES (?, ?, ?)
ON CONFLICT(path) DO UPDATE SET size_bytes = excluded.size_bytes, mod_time = excluded.mod_time`, path, sizeBytes, nanos(modTime))
	return err
}

func shouldCreateParent(path string) bool {
	return path != ":memory:" && !strings.HasPrefix(path, "file:")
}

func rollback(tx *sql.Tx) {
	_ = tx.Rollback()
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func millis(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return value.UnixMilli()
}

func fromMillis(value int64) time.Time {
	if value == 0 {
		return time.Time{}
	}
	return time.UnixMilli(value)
}

func nanos(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return value.UnixNano()
}

func fromNanos(value int64) time.Time {
	if value == 0 {
		return time.Time{}
	}
	return time.Unix(0, value)
}
