package opencode

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

func scanDatabaseForStorageRoot(root string) ([]Project, []Session, []Message, []Part, error) {
	dbPath := databasePathForStorageRoot(root)
	info, err := os.Stat(dbPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil, nil, nil, nil
	}
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("stat opencode database: %w", err)
	}
	if info.IsDir() {
		return nil, nil, nil, nil, nil
	}

	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(dbPath)+"?mode=ro")
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("open opencode database read-only: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	ctx := context.Background()
	projects, err := scanDatabaseProjects(ctx, db, dbPath, info)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	sessions, err := scanDatabaseSessions(ctx, db, dbPath, info)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	messages, err := scanDatabaseMessages(ctx, db, dbPath, info)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	parts, err := scanDatabaseParts(ctx, db, dbPath, info)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return projects, sessions, messages, parts, nil
}

func databasePathForStorageRoot(root string) string {
	if filepath.Base(root) == "storage" {
		return filepath.Join(filepath.Dir(root), "opencode.db")
	}
	return filepath.Join(root, "opencode.db")
}

func scanDatabaseProjects(ctx context.Context, db *sql.DB, dbPath string, info fs.FileInfo) ([]Project, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, worktree, coalesce(vcs, ''), time_created, time_updated FROM project`)
	if err != nil {
		return nil, fmt.Errorf("scan database projects: %w", err)
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var project Project
		var created, updated int64
		if err := rows.Scan(&project.ID, &project.Worktree, &project.VCS, &created, &updated); err != nil {
			return nil, fmt.Errorf("scan database project row: %w", err)
		}
		project.CreatedAt = unixMilli(created)
		project.UpdatedAt = unixMilli(updated)
		project.Source = FileRecord{Path: dbSourcePath(dbPath, "project", project.ID), SizeBytes: info.Size(), ModTime: info.ModTime()}
		projects = append(projects, project)
	}
	return projects, rows.Err()
}

func scanDatabaseSessions(ctx context.Context, db *sql.DB, dbPath string, info fs.FileInfo) ([]Session, error) {
	query := `SELECT id, project_id, directory, title, slug, version, time_created, time_updated, '' FROM session`
	if ok, err := databaseColumnExists(ctx, db, "session", "parent_id"); err != nil {
		return nil, err
	} else if ok {
		query = `SELECT id, project_id, directory, title, slug, version, time_created, time_updated, coalesce(parent_id, '') FROM session`
	}
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("scan database sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var session Session
		var created, updated int64
		if err := rows.Scan(&session.ID, &session.ProjectID, &session.Directory, &session.Title, &session.Slug, &session.Version, &created, &updated, &session.ParentID); err != nil {
			return nil, fmt.Errorf("scan database session row: %w", err)
		}
		session.ProjectPath = session.Directory
		session.CreatedAt = unixMilli(created)
		session.UpdatedAt = unixMilli(updated)
		session.Source = FileRecord{Path: dbSourcePath(dbPath, "session", session.ID), SizeBytes: info.Size(), ModTime: info.ModTime()}
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

func databaseColumnExists(ctx context.Context, db *sql.DB, table, column string) (bool, error) {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(`+table+`)`)
	if err != nil {
		return false, fmt.Errorf("inspect database table %s: %w", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull int
		var defaultValue any
		var primaryKey int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return false, fmt.Errorf("inspect database table %s: %w", table, err)
		}
		if name == column {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("inspect database table %s: %w", table, err)
	}
	return false, nil
}

func scanDatabaseMessages(ctx context.Context, db *sql.DB, dbPath string, info fs.FileInfo) ([]Message, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, session_id, time_created, time_updated, data FROM message`)
	if err != nil {
		return nil, fmt.Errorf("scan database messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var id, sessionID, raw string
		var created, updated int64
		if err := rows.Scan(&id, &sessionID, &created, &updated, &raw); err != nil {
			return nil, fmt.Errorf("scan database message row: %w", err)
		}
		var data map[string]any
		if err := readJSONString(raw, &data); err != nil {
			return nil, fmt.Errorf("parse database message %s: %w", id, err)
		}
		model := mapValue(data, "model")
		summary := mapValue(data, "summary")
		message := Message{
			ID:            id,
			SessionID:     sessionID,
			Role:          stringValue(data, "role"),
			Agent:         stringValue(data, "agent"),
			SummaryTitle:  stringValue(summary, "title"),
			ModelProvider: stringValue(model, "providerID"),
			ModelID:       stringValue(model, "modelID"),
			CreatedAt:     firstTime(unixMilli(created), unixMilli(timeValue(data, "created"))),
			UpdatedAt:     firstTime(unixMilli(updated), unixMilli(timeValue(data, "updated"))),
			Source:        FileRecord{Path: dbSourcePath(dbPath, "message", id), SizeBytes: info.Size(), ModTime: info.ModTime()},
		}
		if strings.EqualFold(message.Role, "assistant") {
			message.TokenUsage = parseTokenUsageMap(mapValue(data, "tokens"))
		}
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

func scanDatabaseParts(ctx context.Context, db *sql.DB, dbPath string, info fs.FileInfo) ([]Part, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, session_id, message_id, time_created, time_updated, data FROM part`)
	if err != nil {
		return nil, fmt.Errorf("scan database parts: %w", err)
	}
	defer rows.Close()

	var parts []Part
	for rows.Next() {
		var id, sessionID, messageID, raw string
		var created, updated int64
		if err := rows.Scan(&id, &sessionID, &messageID, &created, &updated, &raw); err != nil {
			return nil, fmt.Errorf("scan database part row: %w", err)
		}
		sourcePath := dbSourcePath(dbPath, "part", id)
		part, err := classifyPart(sourcePath, staticFileInfo{name: id + ".json", size: int64(len(raw)), modTime: info.ModTime()}, []byte(raw))
		if err != nil {
			return nil, fmt.Errorf("parse database part %s: %w", id, err)
		}
		part.ID = id
		part.SessionID = sessionID
		part.MessageID = messageID
		part.CreatedAt = firstTime(unixMilli(created), part.CreatedAt)
		part.UpdatedAt = firstTime(unixMilli(updated), part.UpdatedAt)
		part.Source = FileRecord{Path: sourcePath, SizeBytes: int64(len(raw)), ModTime: info.ModTime()}
		parts = append(parts, part)
	}
	return parts, rows.Err()
}

func readJSONString(value string, out any) error {
	return json.Unmarshal([]byte(value), out)
}

func dbSourcePath(dbPath, kind, id string) string {
	return dbPath + "#" + kind + "/" + id + ".json"
}

func firstTime(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value
		}
	}
	return time.Time{}
}

type staticFileInfo struct {
	name    string
	size    int64
	modTime time.Time
}

func (s staticFileInfo) Name() string       { return s.name }
func (s staticFileInfo) Size() int64        { return s.size }
func (s staticFileInfo) Mode() fs.FileMode  { return 0o444 }
func (s staticFileInfo) ModTime() time.Time { return s.modTime }
func (s staticFileInfo) IsDir() bool        { return false }
func (s staticFileInfo) Sys() any           { return nil }
