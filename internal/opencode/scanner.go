package opencode

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	DefaultHeavyFileBytes = int64(1 << 20)
	DefaultHeavyTextBytes = 64 * 1024
	previewRunes          = 240
)

func Scan(root string) (Snapshot, error) {
	return scan(root, scanOptions{})
}

func ScanWithMetadata(root string, metadata map[string]FileRecord, existing Snapshot) (Snapshot, error) {
	return scan(root, scanOptions{metadata: metadata, existing: reusableRecordsFromSnapshot(existing)})
}

func DiscoverSourcePaths(root string) ([]string, error) {
	root = filepath.Clean(root)
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("stat storage root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("storage root is not a directory: %s", root)
	}
	dbSources, err := discoverDatabaseSources(root)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0)
	appendJSONPaths := func(kind, dir string, dbIDs map[string]FileRecord) error {
		return walkJSON(filepath.Join(root, dir), func(path string, info fs.FileInfo) error {
			id := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
			if _, duplicate := dbIDs[id]; duplicate {
				return nil
			}
			paths = append(paths, path)
			_ = kind
			return nil
		})
	}
	if err := appendJSONPaths("project", "project", dbSources.projects); err != nil {
		return nil, err
	}
	if err := appendJSONPaths("session", "session", dbSources.sessions); err != nil {
		return nil, err
	}
	if err := appendJSONPaths("message", "message", dbSources.messages); err != nil {
		return nil, err
	}
	if err := appendJSONPaths("part", "part", dbSources.parts); err != nil {
		return nil, err
	}
	for _, sources := range []map[string]FileRecord{dbSources.projects, dbSources.sessions, dbSources.messages, dbSources.parts} {
		for _, source := range sources {
			paths = append(paths, source.Path)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func scan(root string, options scanOptions) (Snapshot, error) {
	root = filepath.Clean(root)
	info, err := os.Stat(root)
	if err != nil {
		return Snapshot{}, fmt.Errorf("stat storage root: %w", err)
	}
	if !info.IsDir() {
		return Snapshot{}, fmt.Errorf("storage root is not a directory: %s", root)
	}
	dbSources, err := discoverDatabaseSources(root)
	if err != nil {
		return Snapshot{}, err
	}
	options.dbSources = dbSources

	projects, err := scanProjects(root, options)
	if err != nil {
		return Snapshot{}, err
	}
	sessions, err := scanSessions(root, options)
	if err != nil {
		return Snapshot{}, err
	}
	messages, err := scanMessages(root, options)
	if err != nil {
		return Snapshot{}, err
	}
	parts, err := scanParts(root, options)
	if err != nil {
		return Snapshot{}, err
	}
	dbProjects, dbSessions, dbMessages, dbParts, err := scanDatabaseForStorageRoot(root, options)
	if err != nil {
		return Snapshot{}, err
	}
	projects = append(projects, dbProjects...)
	sessions = append(sessions, dbSessions...)
	messages = append(messages, dbMessages...)
	parts = append(parts, dbParts...)
	projects = dedupeProjects(projects)
	sessions = dedupeSessions(sessions)
	messages = dedupeMessages(messages)
	parts = dedupeParts(parts)

	assembled := assembleSessions(projects, sessions, messages, parts)
	return Snapshot{Root: root, Projects: projects, Sessions: assembled}, nil
}

type scanOptions struct {
	metadata  map[string]FileRecord
	existing  reusableRecords
	dbSources databaseSources
}

type reusableRecords struct {
	projects map[string]Project
	sessions map[string]Session
	messages map[string]Message
	parts    map[string]Part
}

func reusableRecordsFromSnapshot(snapshot Snapshot) reusableRecords {
	records := reusableRecords{
		projects: map[string]Project{},
		sessions: map[string]Session{},
		messages: map[string]Message{},
		parts:    map[string]Part{},
	}
	for _, project := range snapshot.Projects {
		if project.Source.Path != "" {
			records.projects[project.Source.Path] = project
		}
	}
	for _, session := range snapshot.Sessions {
		if session.Source.Path != "" {
			sessionRecord := session
			sessionRecord.Messages = nil
			records.sessions[session.Source.Path] = sessionRecord
		}
		for _, message := range session.Messages {
			if message.Source.Path != "" {
				messageRecord := message
				messageRecord.Parts = nil
				records.messages[message.Source.Path] = messageRecord
			}
			for _, part := range message.Parts {
				if part.Source.Path != "" {
					records.parts[part.Source.Path] = part
				}
			}
		}
	}
	return records
}

func (o scanOptions) unchanged(source FileRecord) bool {
	if source.Path == "" || o.metadata == nil {
		return false
	}
	metadata, ok := o.metadata[source.Path]
	return ok && metadata.SizeBytes == source.SizeBytes && metadata.ModTime.Equal(source.ModTime)
}

func scanProjects(root string, options scanOptions) ([]Project, error) {
	var projects []Project
	err := walkJSON(filepath.Join(root, "project"), func(path string, info fs.FileInfo) error {
		id := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if _, duplicate := options.dbSources.projects[id]; duplicate {
			return nil
		}
		source := fileRecord(path, info)
		if options.unchanged(source) {
			if project, ok := options.existing.projects[source.Path]; ok {
				projects = append(projects, project)
				return nil
			}
		}
		var raw rawProject
		if err := readJSON(path, &raw); err != nil {
			return err
		}
		if raw.ID == "" {
			raw.ID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		}
		projects = append(projects, Project{
			ID:        raw.ID,
			Worktree:  raw.Worktree,
			VCS:       raw.VCS,
			CreatedAt: unixMilli(raw.Time.Created),
			UpdatedAt: unixMilli(raw.Time.Updated),
			Source:    source,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(projects, func(i, j int) bool { return projects[i].ID < projects[j].ID })
	return projects, nil
}

func scanSessions(root string, options scanOptions) ([]Session, error) {
	var sessions []Session
	sessionRoot := filepath.Join(root, "session")
	err := walkJSON(sessionRoot, func(path string, info fs.FileInfo) error {
		id := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if _, duplicate := options.dbSources.sessions[id]; duplicate {
			return nil
		}
		source := fileRecord(path, info)
		if options.unchanged(source) {
			if session, ok := options.existing.sessions[source.Path]; ok {
				sessions = append(sessions, session)
				return nil
			}
		}
		var raw rawSession
		if err := readJSON(path, &raw); err != nil {
			return err
		}
		if raw.ID == "" {
			raw.ID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		}
		if raw.ProjectID == "" {
			raw.ProjectID = filepath.Base(filepath.Dir(path))
		}
		sessions = append(sessions, Session{
			ID:          raw.ID,
			ProjectID:   raw.ProjectID,
			ParentID:    raw.ParentID,
			Directory:   raw.Directory,
			Title:       raw.Title,
			Slug:        raw.Slug,
			Version:     raw.Version,
			CreatedAt:   unixMilli(raw.Time.Created),
			UpdatedAt:   unixMilli(raw.Time.Updated),
			Source:      source,
			ProjectPath: raw.Directory,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return sessions, nil
}

func scanMessages(root string, options scanOptions) ([]Message, error) {
	var messages []Message
	err := walkJSON(filepath.Join(root, "message"), func(path string, info fs.FileInfo) error {
		id := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if _, duplicate := options.dbSources.messages[id]; duplicate {
			return nil
		}
		source := fileRecord(path, info)
		if options.unchanged(source) {
			if message, ok := options.existing.messages[source.Path]; ok {
				messages = append(messages, message)
				return nil
			}
		}
		var raw rawMessage
		if err := readJSON(path, &raw); err != nil {
			return err
		}
		if raw.ID == "" {
			raw.ID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		}
		if raw.SessionID == "" {
			raw.SessionID = filepath.Base(filepath.Dir(path))
		}
		message := Message{
			ID:            raw.ID,
			SessionID:     raw.SessionID,
			Role:          raw.Role,
			Agent:         raw.Agent,
			SummaryTitle:  raw.Summary.Title,
			ModelProvider: raw.Model.ProviderID,
			ModelID:       raw.Model.ModelID,
			CreatedAt:     unixMilli(raw.Time.Created),
			UpdatedAt:     unixMilli(raw.Time.Updated),
			Source:        source,
		}
		if strings.EqualFold(message.Role, "assistant") {
			message.TokenUsage = parseRawTokenUsage(raw.Tokens)
		}
		messages = append(messages, message)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return messages, nil
}

func scanParts(root string, options scanOptions) ([]Part, error) {
	var parts []Part
	err := walkJSON(filepath.Join(root, "part"), func(path string, info fs.FileInfo) error {
		id := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if _, duplicate := options.dbSources.parts[id]; duplicate {
			return nil
		}
		source := fileRecord(path, info)
		if options.unchanged(source) {
			if part, ok := options.existing.parts[source.Path]; ok {
				parts = append(parts, part)
				return nil
			}
		}
		if info.Size() > DefaultHeavyFileBytes {
			part, err := classifyHeavyPartFile(path, info)
			if err != nil {
				return err
			}
			parts = append(parts, part)
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		part, err := classifyPart(path, info, raw)
		if err != nil {
			return err
		}
		parts = append(parts, part)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return parts, nil
}

func assembleSessions(projects []Project, sessions []Session, messages []Message, parts []Part) []Session {
	projectsByID := make(map[string]Project, len(projects))
	for _, project := range projects {
		projectsByID[project.ID] = project
	}

	partsByMessage := make(map[string][]Part)
	for _, part := range parts {
		partsByMessage[part.MessageID] = append(partsByMessage[part.MessageID], part)
	}
	for messageID := range partsByMessage {
		sortParts(partsByMessage[messageID])
	}

	messagesBySession := make(map[string][]Message)
	for _, message := range messages {
		message.Parts = partsByMessage[message.ID]
		messagesBySession[message.SessionID] = append(messagesBySession[message.SessionID], message)
	}
	for sessionID := range messagesBySession {
		sortMessages(messagesBySession[sessionID])
	}

	assembled := make([]Session, 0, len(sessions))
	for _, session := range sessions {
		storedProvider := session.ModelProvider
		storedModelID := session.ModelID
		storedUsage := session.TokenUsage
		session.ModelProvider = ""
		session.ModelID = ""
		session.MessageCount = 0
		session.PartCount = 0
		session.HeavyPartCount = 0
		session.TokenUsage = TokenUsage{}
		if project, ok := projectsByID[session.ProjectID]; ok && project.Worktree != "" {
			session.ProjectPath = project.Worktree
		} else if session.ProjectID == "global" && session.ProjectPath == "" {
			session.ProjectPath = "Global"
		}
		session.Messages = messagesBySession[session.ID]
		for _, message := range session.Messages {
			if session.ModelProvider == "" {
				session.ModelProvider = message.ModelProvider
			}
			if session.ModelID == "" {
				session.ModelID = message.ModelID
			}
			if message.TokenUsage.Available {
				session.TokenUsage.Available = true
				session.TokenUsage.Total += message.TokenUsage.Total
				session.TokenUsage.Input += message.TokenUsage.Input
				session.TokenUsage.Output += message.TokenUsage.Output
				session.TokenUsage.Reasoning += message.TokenUsage.Reasoning
				session.TokenUsage.CacheRead += message.TokenUsage.CacheRead
				session.TokenUsage.CacheWrite += message.TokenUsage.CacheWrite
			}
			session.PartCount += len(message.Parts)
			for _, part := range message.Parts {
				if part.Heavy {
					session.HeavyPartCount++
				}
			}
		}
		session.MessageCount = len(session.Messages)
		if session.ModelProvider == "" {
			session.ModelProvider = storedProvider
		}
		if session.ModelID == "" {
			session.ModelID = storedModelID
		}
		if !session.TokenUsage.Available && storedUsage.Available {
			session.TokenUsage = storedUsage
		}
		assembled = append(assembled, session)
	}
	sort.SliceStable(assembled, func(i, j int) bool {
		if !assembled[i].UpdatedAt.Equal(assembled[j].UpdatedAt) {
			return assembled[i].UpdatedAt.After(assembled[j].UpdatedAt)
		}
		return assembled[i].ID < assembled[j].ID
	})
	return assembled
}

func walkJSON(root string, fn func(path string, info fs.FileInfo) error) error {
	if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return fmt.Errorf("stat %s: %w", root, err)
	}
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if err := fn(path, info); err != nil {
			return err
		}
		return nil
	})
}

func readJSON(path string, out any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func sortMessages(messages []Message) {
	sort.SliceStable(messages, func(i, j int) bool {
		if !messages[i].CreatedAt.Equal(messages[j].CreatedAt) {
			return messages[i].CreatedAt.Before(messages[j].CreatedAt)
		}
		return messages[i].ID < messages[j].ID
	})
}

func sortParts(parts []Part) {
	sort.SliceStable(parts, func(i, j int) bool {
		if !parts[i].CreatedAt.Equal(parts[j].CreatedAt) {
			return parts[i].CreatedAt.Before(parts[j].CreatedAt)
		}
		return parts[i].ID < parts[j].ID
	})
}

func dedupeProjects(projects []Project) []Project {
	byID := make(map[string]Project, len(projects))
	order := make([]string, 0, len(projects))
	for _, project := range projects {
		if project.ID == "" {
			continue
		}
		if _, ok := byID[project.ID]; !ok {
			order = append(order, project.ID)
		}
		byID[project.ID] = project
	}
	out := make([]Project, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func dedupeSessions(sessions []Session) []Session {
	byID := make(map[string]Session, len(sessions))
	order := make([]string, 0, len(sessions))
	for _, session := range sessions {
		if session.ID == "" {
			continue
		}
		if _, ok := byID[session.ID]; !ok {
			order = append(order, session.ID)
		}
		byID[session.ID] = session
	}
	out := make([]Session, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	return out
}

func dedupeMessages(messages []Message) []Message {
	byID := make(map[string]Message, len(messages))
	order := make([]string, 0, len(messages))
	for _, message := range messages {
		if message.ID == "" {
			continue
		}
		if _, ok := byID[message.ID]; !ok {
			order = append(order, message.ID)
		}
		byID[message.ID] = message
	}
	out := make([]Message, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	return out
}

func dedupeParts(parts []Part) []Part {
	byID := make(map[string]Part, len(parts))
	order := make([]string, 0, len(parts))
	for _, part := range parts {
		if part.ID == "" {
			continue
		}
		if _, ok := byID[part.ID]; !ok {
			order = append(order, part.ID)
		}
		byID[part.ID] = part
	}
	out := make([]Part, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	return out
}

func fileRecord(path string, info fs.FileInfo) FileRecord {
	return FileRecord{Path: path, SizeBytes: info.Size(), ModTime: info.ModTime()}
}

func unixMilli(ms int64) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms)
}

type rawTime struct {
	Created int64 `json:"created"`
	Updated int64 `json:"updated"`
	Start   int64 `json:"start"`
	End     int64 `json:"end"`
}

type rawProject struct {
	ID       string  `json:"id"`
	Worktree string  `json:"worktree"`
	VCS      string  `json:"vcs"`
	Time     rawTime `json:"time"`
}

type rawSession struct {
	ID        string  `json:"id"`
	Slug      string  `json:"slug"`
	Version   string  `json:"version"`
	ProjectID string  `json:"projectID"`
	ParentID  string  `json:"parentID"`
	Directory string  `json:"directory"`
	Title     string  `json:"title"`
	Time      rawTime `json:"time"`
}

type rawMessage struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionID"`
	Role      string `json:"role"`
	Agent     string `json:"agent"`
	Time      rawTime
	Tokens    json.RawMessage `json:"tokens"`
	Summary   struct {
		Title string `json:"title"`
	} `json:"summary"`
	Model struct {
		ProviderID string `json:"providerID"`
		ModelID    string `json:"modelID"`
	} `json:"model"`
}

func parseRawTokenUsage(raw json.RawMessage) TokenUsage {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return TokenUsage{}
	}
	var data map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&data); err != nil {
		return TokenUsage{}
	}
	return parseTokenUsageMap(data)
}

func parseTokenUsageMap(data map[string]any) TokenUsage {
	if data == nil {
		return TokenUsage{}
	}
	var usage TokenUsage
	var sawValue bool
	totalSet := setTokenValue(&usage.Total, data, "total")
	sawValue = sawValue || totalSet
	sawValue = setTokenValue(&usage.Input, data, "input") || sawValue
	sawValue = setTokenValue(&usage.Output, data, "output") || sawValue
	sawValue = setTokenValue(&usage.Reasoning, data, "reasoning") || sawValue

	cache := mapValue(data, "cache")
	cacheReadSet := setTokenValue(&usage.CacheRead, cache, "read")
	cacheWriteSet := setTokenValue(&usage.CacheWrite, cache, "write")
	if !cacheReadSet {
		cacheReadSet = setTokenValue(&usage.CacheRead, data, "cacheRead", "cache_read")
	}
	if !cacheWriteSet {
		cacheWriteSet = setTokenValue(&usage.CacheWrite, data, "cacheWrite", "cache_write")
	}
	sawValue = cacheReadSet || cacheWriteSet || sawValue

	if !sawValue {
		return TokenUsage{}
	}
	usage.Available = true
	if !totalSet {
		usage.Total = usage.Input + usage.Output + usage.Reasoning + usage.CacheRead + usage.CacheWrite
	}
	return usage
}

func setTokenValue(target *int64, data map[string]any, keys ...string) bool {
	for _, key := range keys {
		value, ok := data[key]
		if !ok {
			continue
		}
		parsed, ok := tokenInt(value)
		if !ok {
			continue
		}
		*target = parsed
		return true
	}
	return false
}

func tokenInt(value any) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return nonNegativeTokenInt(int64(typed))
	case int64:
		return nonNegativeTokenInt(typed)
	case float64:
		return nonNegativeTokenInt(int64(typed))
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			return 0, false
		}
		return nonNegativeTokenInt(parsed)
	default:
		return 0, false
	}
}

func nonNegativeTokenInt(value int64) (int64, bool) {
	if value < 0 {
		return 0, false
	}
	return value, true
}
