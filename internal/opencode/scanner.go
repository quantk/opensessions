package opencode

import (
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
	root = filepath.Clean(root)
	info, err := os.Stat(root)
	if err != nil {
		return Snapshot{}, fmt.Errorf("stat storage root: %w", err)
	}
	if !info.IsDir() {
		return Snapshot{}, fmt.Errorf("storage root is not a directory: %s", root)
	}

	projects, err := scanProjects(root)
	if err != nil {
		return Snapshot{}, err
	}
	sessions, err := scanSessions(root)
	if err != nil {
		return Snapshot{}, err
	}
	messages, err := scanMessages(root)
	if err != nil {
		return Snapshot{}, err
	}
	parts, err := scanParts(root)
	if err != nil {
		return Snapshot{}, err
	}
	dbProjects, dbSessions, dbMessages, dbParts, err := scanDatabaseForStorageRoot(root)
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

func scanProjects(root string) ([]Project, error) {
	var projects []Project
	err := walkJSON(filepath.Join(root, "project"), func(path string, info fs.FileInfo) error {
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
			Source:    fileRecord(path, info),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(projects, func(i, j int) bool { return projects[i].ID < projects[j].ID })
	return projects, nil
}

func scanSessions(root string) ([]Session, error) {
	var sessions []Session
	sessionRoot := filepath.Join(root, "session")
	err := walkJSON(sessionRoot, func(path string, info fs.FileInfo) error {
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
			Directory:   raw.Directory,
			Title:       raw.Title,
			Slug:        raw.Slug,
			Version:     raw.Version,
			CreatedAt:   unixMilli(raw.Time.Created),
			UpdatedAt:   unixMilli(raw.Time.Updated),
			Source:      fileRecord(path, info),
			ProjectPath: raw.Directory,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return sessions, nil
}

func scanMessages(root string) ([]Message, error) {
	var messages []Message
	err := walkJSON(filepath.Join(root, "message"), func(path string, info fs.FileInfo) error {
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
		messages = append(messages, Message{
			ID:            raw.ID,
			SessionID:     raw.SessionID,
			Role:          raw.Role,
			Agent:         raw.Agent,
			SummaryTitle:  raw.Summary.Title,
			ModelProvider: raw.Model.ProviderID,
			ModelID:       raw.Model.ModelID,
			CreatedAt:     unixMilli(raw.Time.Created),
			UpdatedAt:     unixMilli(raw.Time.Updated),
			Source:        fileRecord(path, info),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return messages, nil
}

func scanParts(root string) ([]Part, error) {
	var parts []Part
	err := walkJSON(filepath.Join(root, "part"), func(path string, info fs.FileInfo) error {
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
			session.PartCount += len(message.Parts)
			for _, part := range message.Parts {
				if part.Heavy {
					session.HeavyPartCount++
				}
			}
		}
		session.MessageCount = len(session.Messages)
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
	Summary   struct {
		Title string `json:"title"`
	} `json:"summary"`
	Model struct {
		ProviderID string `json:"providerID"`
		ModelID    string `json:"modelID"`
	} `json:"model"`
}
