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
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type databaseSources struct {
	dbPath   string
	info     fs.FileInfo
	projects map[string]FileRecord
	sessions map[string]FileRecord
	messages map[string]FileRecord
	parts    map[string]FileRecord
}

func newDatabaseSources() databaseSources {
	return databaseSources{
		projects: map[string]FileRecord{},
		sessions: map[string]FileRecord{},
		messages: map[string]FileRecord{},
		parts:    map[string]FileRecord{},
	}
}

func discoverDatabaseSources(root string) (databaseSources, error) {
	sources := newDatabaseSources()
	dbPath := databasePathForStorageRoot(root)
	info, err := os.Stat(dbPath)
	if errors.Is(err, os.ErrNotExist) {
		return sources, nil
	}
	if err != nil {
		return sources, fmt.Errorf("stat opencode database: %w", err)
	}
	if info.IsDir() {
		return sources, nil
	}

	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(dbPath)+"?mode=ro")
	if err != nil {
		return sources, fmt.Errorf("open opencode database read-only: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	sources.dbPath = dbPath
	sources.info = info
	ctx := context.Background()
	if err := discoverDatabaseIDs(ctx, db, dbPath, info, "project", sources.projects); err != nil {
		return sources, err
	}
	if err := discoverDatabaseIDs(ctx, db, dbPath, info, "session", sources.sessions); err != nil {
		return sources, err
	}
	if err := discoverDatabaseIDs(ctx, db, dbPath, info, "message", sources.messages); err != nil {
		return sources, err
	}
	if err := discoverDatabasePartIDs(ctx, db, dbPath, info, sources.parts); err != nil {
		return sources, err
	}
	return sources, nil
}

func discoverDatabaseIDs(ctx context.Context, db *sql.DB, dbPath string, info fs.FileInfo, table string, out map[string]FileRecord) error {
	rows, err := db.QueryContext(ctx, `SELECT id FROM `+table)
	if err != nil {
		return fmt.Errorf("discover database %s IDs: %w", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("discover database %s row: %w", table, err)
		}
		out[id] = FileRecord{Path: dbSourcePath(dbPath, table, id), SizeBytes: info.Size(), ModTime: info.ModTime()}
	}
	return rows.Err()
}

func discoverDatabasePartIDs(ctx context.Context, db *sql.DB, dbPath string, info fs.FileInfo, out map[string]FileRecord) error {
	rows, err := db.QueryContext(ctx, `SELECT id, length(data) FROM part`)
	if err != nil {
		return fmt.Errorf("discover database part IDs: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var size int64
		if err := rows.Scan(&id, &size); err != nil {
			return fmt.Errorf("discover database part row: %w", err)
		}
		out[id] = FileRecord{Path: dbSourcePath(dbPath, "part", id), SizeBytes: size, ModTime: info.ModTime()}
	}
	return rows.Err()
}

func scanDatabaseForStorageRoot(root string, options scanOptions) ([]Project, []Session, []Message, []Part, error) {
	sources := options.dbSources
	if sources.dbPath == "" {
		return nil, nil, nil, nil, nil
	}
	dbPath := sources.dbPath

	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(dbPath)+"?mode=ro")
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("open opencode database read-only: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	ctx := context.Background()
	projects, err := scanDatabaseProjects(ctx, db, sources, options)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	sessions, err := scanDatabaseSessions(ctx, db, sources, options)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	messages, err := scanDatabaseMessages(ctx, db, sources, options)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	parts, err := scanDatabaseParts(ctx, db, sources, options)
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

func scanDatabaseProjects(ctx context.Context, db *sql.DB, sources databaseSources, options scanOptions) ([]Project, error) {
	var projects []Project
	changedIDs := reusableDatabaseProjects(sources.projects, options, &projects)
	if len(changedIDs) == 0 {
		return projects, nil
	}
	err := queryDatabaseRows(ctx, db, `SELECT id, worktree, coalesce(vcs, ''), time_created, time_updated FROM project`, changedIDs, len(changedIDs) == len(sources.projects), func(rows *sql.Rows) error {
		for rows.Next() {
			var project Project
			var created, updated int64
			if err := rows.Scan(&project.ID, &project.Worktree, &project.VCS, &created, &updated); err != nil {
				return fmt.Errorf("scan database project row: %w", err)
			}
			project.CreatedAt = unixMilli(created)
			project.UpdatedAt = unixMilli(updated)
			project.Source = sources.projects[project.ID]
			projects = append(projects, project)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("scan database projects: %w", err)
	}
	return projects, nil
}

func scanDatabaseSessions(ctx context.Context, db *sql.DB, sources databaseSources, options scanOptions) ([]Session, error) {
	query := `SELECT id, project_id, directory, title, slug, version, time_created, time_updated, '' FROM session`
	if ok, err := databaseColumnExists(ctx, db, "session", "parent_id"); err != nil {
		return nil, err
	} else if ok {
		query = `SELECT id, project_id, directory, title, slug, version, time_created, time_updated, coalesce(parent_id, '') FROM session`
	}
	var sessions []Session
	changedIDs := reusableDatabaseSessions(sources.sessions, options, &sessions)
	if len(changedIDs) == 0 {
		return sessions, nil
	}
	err := queryDatabaseRows(ctx, db, query, changedIDs, len(changedIDs) == len(sources.sessions), func(rows *sql.Rows) error {
		for rows.Next() {
			var session Session
			var created, updated int64
			if err := rows.Scan(&session.ID, &session.ProjectID, &session.Directory, &session.Title, &session.Slug, &session.Version, &created, &updated, &session.ParentID); err != nil {
				return fmt.Errorf("scan database session row: %w", err)
			}
			session.ProjectPath = session.Directory
			session.CreatedAt = unixMilli(created)
			session.UpdatedAt = unixMilli(updated)
			session.Source = sources.sessions[session.ID]
			sessions = append(sessions, session)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("scan database sessions: %w", err)
	}
	return sessions, nil
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

func scanDatabaseMessages(ctx context.Context, db *sql.DB, sources databaseSources, options scanOptions) ([]Message, error) {
	var messages []Message
	changedIDs := reusableDatabaseMessages(sources.messages, options, &messages)
	if len(changedIDs) == 0 {
		return messages, nil
	}
	err := queryDatabaseRows(ctx, db, `SELECT id, session_id, time_created, time_updated, data FROM message`, changedIDs, len(changedIDs) == len(sources.messages), func(rows *sql.Rows) error {
		for rows.Next() {
			var id, sessionID, raw string
			var created, updated int64
			if err := rows.Scan(&id, &sessionID, &created, &updated, &raw); err != nil {
				return fmt.Errorf("scan database message row: %w", err)
			}
			var data map[string]any
			if err := readJSONString(raw, &data); err != nil {
				return fmt.Errorf("parse database message %s: %w", id, err)
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
				Source:        sources.messages[id],
			}
			if strings.EqualFold(message.Role, "assistant") {
				message.TokenUsage = parseTokenUsageMap(mapValue(data, "tokens"))
			}
			messages = append(messages, message)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("scan database messages: %w", err)
	}
	return messages, nil
}

func scanDatabaseParts(ctx context.Context, db *sql.DB, sources databaseSources, options scanOptions) ([]Part, error) {
	var parts []Part
	changedIDs := reusableDatabaseParts(sources.parts, options, &parts)
	if len(changedIDs) == 0 {
		return parts, nil
	}
	err := queryDatabaseRows(ctx, db, `SELECT id, session_id, message_id, time_created, time_updated, data FROM part`, changedIDs, len(changedIDs) == len(sources.parts), func(rows *sql.Rows) error {
		for rows.Next() {
			var id, sessionID, messageID, raw string
			var created, updated int64
			if err := rows.Scan(&id, &sessionID, &messageID, &created, &updated, &raw); err != nil {
				return fmt.Errorf("scan database part row: %w", err)
			}
			source := sources.parts[id]
			part, err := classifyPart(source.Path, staticFileInfo{name: id + ".json", size: source.SizeBytes, modTime: source.ModTime}, []byte(raw))
			if err != nil {
				return fmt.Errorf("parse database part %s: %w", id, err)
			}
			part.ID = id
			part.SessionID = sessionID
			part.MessageID = messageID
			part.CreatedAt = firstTime(unixMilli(created), part.CreatedAt)
			part.UpdatedAt = firstTime(unixMilli(updated), part.UpdatedAt)
			part.Source = source
			parts = append(parts, part)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("scan database parts: %w", err)
	}
	return parts, nil
}

func reusableDatabaseProjects(sources map[string]FileRecord, options scanOptions, projects *[]Project) []string {
	var changed []string
	for _, id := range sortedDatabaseSourceIDs(sources) {
		source := sources[id]
		if options.unchanged(source) {
			if project, ok := options.existing.projects[source.Path]; ok {
				*projects = append(*projects, project)
				continue
			}
		}
		changed = append(changed, id)
	}
	return changed
}

func reusableDatabaseSessions(sources map[string]FileRecord, options scanOptions, sessions *[]Session) []string {
	var changed []string
	for _, id := range sortedDatabaseSourceIDs(sources) {
		source := sources[id]
		if options.unchanged(source) {
			if session, ok := options.existing.sessions[source.Path]; ok {
				*sessions = append(*sessions, session)
				continue
			}
		}
		changed = append(changed, id)
	}
	return changed
}

func reusableDatabaseMessages(sources map[string]FileRecord, options scanOptions, messages *[]Message) []string {
	var changed []string
	for _, id := range sortedDatabaseSourceIDs(sources) {
		source := sources[id]
		if options.unchanged(source) {
			if message, ok := options.existing.messages[source.Path]; ok {
				*messages = append(*messages, message)
				continue
			}
		}
		changed = append(changed, id)
	}
	return changed
}

func reusableDatabaseParts(sources map[string]FileRecord, options scanOptions, parts *[]Part) []string {
	var changed []string
	for _, id := range sortedDatabaseSourceIDs(sources) {
		source := sources[id]
		if options.unchanged(source) {
			if part, ok := options.existing.parts[source.Path]; ok {
				*parts = append(*parts, part)
				continue
			}
		}
		changed = append(changed, id)
	}
	return changed
}

func sortedDatabaseSourceIDs(sources map[string]FileRecord) []string {
	ids := make([]string, 0, len(sources))
	for id := range sources {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func queryDatabaseRows(ctx context.Context, db *sql.DB, baseQuery string, ids []string, all bool, scan func(*sql.Rows) error) error {
	if all {
		rows, err := db.QueryContext(ctx, baseQuery)
		if err != nil {
			return err
		}
		defer rows.Close()
		return scan(rows)
	}
	const chunkSize = 500
	for start := 0; start < len(ids); start += chunkSize {
		end := start + chunkSize
		if end > len(ids) {
			end = len(ids)
		}
		chunk := ids[start:end]
		args := make([]any, len(chunk))
		for i, id := range chunk {
			args[i] = id
		}
		rows, err := db.QueryContext(ctx, baseQuery+` WHERE id IN (`+placeholders(len(chunk))+`)`, args...)
		if err != nil {
			return err
		}
		if err := scan(rows); err != nil {
			_ = rows.Close()
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}
	}
	return nil
}

func placeholders(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", count), ",")
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
