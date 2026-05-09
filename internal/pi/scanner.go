package pi

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/quantick/opensession/internal/opencode"
	"github.com/quantick/opensession/internal/source"
)

const (
	previewRunes       = 240
	maxPiRawBytes      = 256 * 1024
	maxPiIndexTextByte = opencode.DefaultHeavyTextBytes
)

type scanOptions struct {
	metadata map[string]opencode.FileRecord
	existing map[string]opencode.Session
}

func DiscoverSourcePaths(root string) ([]string, error) {
	root = filepath.Clean(root)
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("stat Pi sessions root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("Pi sessions root is not a directory: %s", root)
	}
	var paths []string
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".jsonl") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

func Scan(root string) (opencode.Snapshot, error) {
	return scan(root, scanOptions{})
}

func ScanWithMetadata(root string, metadata map[string]opencode.FileRecord, existing opencode.Snapshot) (opencode.Snapshot, error) {
	byPath := map[string]opencode.Session{}
	for _, session := range existing.Sessions {
		if source.NormalizeKind(session.SourceKind) == source.KindPi && session.Source.Path != "" {
			byPath[session.Source.Path] = session
		}
	}
	return scan(root, scanOptions{metadata: metadata, existing: byPath})
}

func scan(root string, options scanOptions) (opencode.Snapshot, error) {
	paths, err := DiscoverSourcePaths(root)
	if err != nil {
		return opencode.Snapshot{}, err
	}
	projectsByID := map[string]opencode.Project{}
	var sessions []opencode.Session
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return opencode.Snapshot{}, err
		}
		sourceRecord := fileRecord(path, info)
		if unchanged(sourceRecord, options.metadata) {
			if session, ok := options.existing[sourceRecord.Path]; ok {
				sessions = append(sessions, session)
				if session.ProjectID != "" {
					projectsByID[session.ProjectID] = opencode.Project{SourceKind: string(source.KindPi), ID: session.ProjectID, Worktree: session.ProjectPath, CreatedAt: session.CreatedAt, UpdatedAt: session.UpdatedAt, Source: sourceRecord}
				}
				continue
			}
		}
		session, project, err := scanFile(path, sourceRecord)
		if err != nil {
			return opencode.Snapshot{}, err
		}
		if session.ID == "" {
			continue
		}
		sessions = append(sessions, session)
		projectsByID[project.ID] = project
	}
	projects := make([]opencode.Project, 0, len(projectsByID))
	for _, project := range projectsByID {
		projects = append(projects, project)
	}
	sort.Slice(projects, func(i, j int) bool { return projects[i].ID < projects[j].ID })
	sort.SliceStable(sessions, func(i, j int) bool {
		if !sessions[i].UpdatedAt.Equal(sessions[j].UpdatedAt) {
			return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
		}
		return sessions[i].ID < sessions[j].ID
	})
	return opencode.Snapshot{Root: filepath.Clean(root), Projects: projects, Sessions: sessions}, nil
}

func scanFile(path string, sourceRecord opencode.FileRecord) (opencode.Session, opencode.Project, error) {
	file, err := os.Open(path)
	if err != nil {
		return opencode.Session{}, opencode.Project{}, fmt.Errorf("open Pi session %s: %w", path, err)
	}
	defer file.Close()

	var header piHeader
	var messages []opencode.Message
	labels := map[string]string{}
	latestTitle := ""
	latestProvider := ""
	latestModel := ""
	var usage opencode.TokenUsage
	created := time.Time{}
	updated := time.Time{}
	sequence := 0

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), maxPiRawBytes)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var entry map[string]any
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.UseNumber()
		if err := decoder.Decode(&entry); err != nil {
			return opencode.Session{}, opencode.Project{}, fmt.Errorf("parse Pi session %s line %d: %w", path, lineNo, err)
		}
		entryType := stringValue(entry, "type")
		if entryType == "session" {
			header = piHeader{ID: stringValue(entry, "id"), CWD: stringValue(entry, "cwd"), Timestamp: parseTime(stringValue(entry, "timestamp"))}
			created = minNonZeroTime(created, header.Timestamp)
			updated = maxTime(updated, header.Timestamp)
			continue
		}
		sequence++
		entryID := stringValue(entry, "id")
		if entryID == "" {
			entryID = fmt.Sprintf("line-%d", lineNo)
		}
		entryTime := parseTime(stringValue(entry, "timestamp"))
		created = minNonZeroTime(created, entryTime)
		updated = maxTime(updated, entryTime)

		if entryType == "label" {
			if target := stringValue(entry, "targetId"); target != "" {
				labels[target] = stringValue(entry, "label")
			}
		}
		if entryType == "session_info" && stringValue(entry, "name") != "" {
			latestTitle = stringValue(entry, "name")
		}
		message := buildEntryMessage(header.ID, entry, entryType, sequence, sourceRecord, line)
		if message.ID == "" {
			continue
		}
		if message.ModelProvider != "" {
			latestProvider = message.ModelProvider
		}
		if message.ModelID != "" {
			latestModel = message.ModelID
		}
		if message.TokenUsage.Available {
			usage.Available = true
			usage.Total += message.TokenUsage.Total
			usage.Input += message.TokenUsage.Input
			usage.Output += message.TokenUsage.Output
			usage.Reasoning += message.TokenUsage.Reasoning
			usage.CacheRead += message.TokenUsage.CacheRead
			usage.CacheWrite += message.TokenUsage.CacheWrite
		}
		messages = append(messages, message)
	}
	if err := scanner.Err(); err != nil {
		return opencode.Session{}, opencode.Project{}, fmt.Errorf("read Pi session %s: %w", path, err)
	}
	if header.ID == "" {
		header.ID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	if header.CWD == "" {
		header.CWD = filepath.Dir(path)
	}
	sessionID := source.NamespacedID(source.KindPi, header.ID)
	projectID := source.NamespacedID(source.KindPi, "project", shortHash(header.CWD))
	for i := range messages {
		messages[i].SessionID = sessionID
		if messages[i].ParentID != "" {
			messages[i].ParentID = source.NamespacedID(source.KindPi, header.ID, messages[i].ParentID)
		}
		if label := labels[strings.TrimPrefix(messages[i].ID, string(source.KindPi)+":"+header.ID+":")]; label != "" {
			messages[i].Label = label
		}
		for j := range messages[i].Parts {
			messages[i].Parts[j].SessionID = sessionID
			messages[i].Parts[j].MessageID = messages[i].ID
		}
	}
	mergeToolResults(messages)
	messageCount, partCount, heavyCount := counts(messages)
	if latestTitle == "" {
		latestTitle = firstUserText(messages)
	}
	session := opencode.Session{
		SourceKind:     string(source.KindPi),
		ID:             sessionID,
		ProjectID:      projectID,
		ProjectPath:    header.CWD,
		Directory:      header.CWD,
		Title:          latestTitle,
		ModelProvider:  latestProvider,
		ModelID:        latestModel,
		CreatedAt:      created,
		UpdatedAt:      maxTime(updated, sourceRecord.ModTime),
		MessageCount:   messageCount,
		PartCount:      partCount,
		HeavyPartCount: heavyCount,
		TokenUsage:     usage,
		Messages:       messages,
		Source:         sourceRecord,
	}
	project := opencode.Project{SourceKind: string(source.KindPi), ID: projectID, Worktree: header.CWD, CreatedAt: created, UpdatedAt: session.UpdatedAt, Source: sourceRecord}
	return session, project, nil
}

type piHeader struct {
	ID        string
	CWD       string
	Timestamp time.Time
}

func buildEntryMessage(sessionID string, entry map[string]any, entryType string, sequence int, sourceRecord opencode.FileRecord, raw []byte) opencode.Message {
	entryID := stringValue(entry, "id")
	if entryID == "" {
		entryID = fmt.Sprintf("entry-%d", sequence)
	}
	created := parseTime(stringValue(entry, "timestamp"))
	message := opencode.Message{
		SourceKind:  string(source.KindPi),
		ID:          source.NamespacedID(source.KindPi, sessionID, entryID),
		ParentID:    stringValue(entry, "parentId"),
		EntryType:   entryType,
		AppendOrder: sequence,
		CreatedAt:   created,
		UpdatedAt:   created,
		Source:      sourceRecord,
	}
	if msg, ok := mapValue(entry, "message"); ok {
		message.Role = stringValue(msg, "role")
		message.ModelProvider = firstNonEmpty(stringValue(msg, "provider"), stringValue(msg, "api"))
		message.ModelID = stringValue(msg, "model")
		message.TokenUsage = tokenUsage(mapValueOrNil(msg, "usage"))
		message.Parts = partsFromAgentMessage(sessionID, entryID, message.Role, msg, sourceRecord, raw)
		return message
	}
	message.Role = "event"
	switch entryType {
	case "model_change":
		message.ModelProvider = stringValue(entry, "provider")
		message.ModelID = stringValue(entry, "modelId")
		message.Parts = []opencode.Part{eventPart(sessionID, entryID, "model_change", opencode.PartKindTool, "model", firstNonEmpty(message.ModelProvider+"/"+message.ModelID, "model change"), sourceRecord, entry)}
	case "thinking_level_change":
		message.Parts = []opencode.Part{eventPart(sessionID, entryID, "thinking_level_change", opencode.PartKindTool, "thinking", "thinking level "+stringValue(entry, "thinkingLevel"), sourceRecord, entry)}
	case "compaction":
		message.Parts = []opencode.Part{eventPart(sessionID, entryID, "compaction", opencode.PartKindText, "", stringValue(entry, "summary"), sourceRecord, entry)}
	case "branch_summary":
		message.Parts = []opencode.Part{eventPart(sessionID, entryID, "branch_summary", opencode.PartKindText, "", stringValue(entry, "summary"), sourceRecord, entry)}
	case "custom_message":
		if display, _ := entry["display"].(bool); display {
			message.Parts = contentParts(sessionID, entryID, "custom", entry["content"], sourceRecord, raw)
		}
	case "session_info", "label", "custom":
		// Tree metadata only.
	default:
		message.Parts = []opencode.Part{eventPart(sessionID, entryID, entryType, opencode.PartKindUnknown, "", entryType, sourceRecord, entry)}
	}
	return message
}

func partsFromAgentMessage(sessionID, entryID, role string, msg map[string]any, sourceRecord opencode.FileRecord, raw []byte) []opencode.Part {
	switch role {
	case "user":
		return contentParts(sessionID, entryID, role, msg["content"], sourceRecord, raw)
	case "assistant":
		return assistantParts(sessionID, entryID, msg["content"], sourceRecord, raw)
	case "toolResult":
		part := eventPart(sessionID, entryID, "tool_result", opencode.PartKindTool, stringValue(msg, "toolName"), textFromContent(msg["content"]), sourceRecord, msg)
		if isError, _ := msg["isError"].(bool); isError {
			part.Status = "error"
		} else {
			part.Status = "completed"
		}
		part.ToolName = firstNonEmpty(part.ToolName, "toolResult")
		return []opencode.Part{part}
	case "bashExecution":
		command := stringValue(msg, "command")
		output := stringValue(msg, "output")
		part := eventPart(sessionID, entryID, "bash_execution", opencode.PartKindTool, "bash", command+"\n"+output, sourceRecord, msg)
		part.Title = command
		part.Status = "completed"
		if cancelled, _ := msg["cancelled"].(bool); cancelled {
			part.Status = "cancelled"
		}
		return []opencode.Part{part}
	default:
		return contentParts(sessionID, entryID, role, msg["content"], sourceRecord, raw)
	}
}

func assistantParts(sessionID, entryID string, content any, sourceRecord opencode.FileRecord, raw []byte) []opencode.Part {
	blocks, ok := content.([]any)
	if !ok {
		return contentParts(sessionID, entryID, "assistant", content, sourceRecord, raw)
	}
	var parts []opencode.Part
	for i, blockAny := range blocks {
		block, ok := blockAny.(map[string]any)
		if !ok {
			continue
		}
		typeName := stringValue(block, "type")
		switch typeName {
		case "text":
			parts = append(parts, textPart(sessionID, entryID, i, "assistant", "text", stringValue(block, "text"), opencode.PartKindText, sourceRecord, block))
		case "thinking", "reasoning":
			parts = append(parts, textPart(sessionID, entryID, i, "assistant", typeName, firstNonEmpty(stringValue(block, "thinking"), stringValue(block, "text"), "[reasoning metadata]"), opencode.PartKindReasoning, sourceRecord, block))
		case "toolCall":
			part := textPart(sessionID, entryID, i, "assistant", "tool", jsonSummary(block["arguments"]), opencode.PartKindTool, sourceRecord, block)
			part.ToolName = stringValue(block, "name")
			part.Status = "started"
			part.Title = part.ToolName
			parts = append(parts, part)
		case "image":
			part := textPart(sessionID, entryID, i, "assistant", "file", stringValue(block, "mimeType"), opencode.PartKindFile, sourceRecord, block)
			part.MIME = stringValue(block, "mimeType")
			parts = append(parts, part)
		}
	}
	return parts
}

func contentParts(sessionID, entryID, role string, content any, sourceRecord opencode.FileRecord, raw []byte) []opencode.Part {
	if text, ok := content.(string); ok {
		return []opencode.Part{textPart(sessionID, entryID, 0, role, "text", text, opencode.PartKindText, sourceRecord, map[string]any{"type": "text", "text": text})}
	}
	blocks, ok := content.([]any)
	if !ok {
		if content == nil {
			return nil
		}
		return []opencode.Part{textPart(sessionID, entryID, 0, role, "text", jsonSummary(content), opencode.PartKindText, sourceRecord, content)}
	}
	var parts []opencode.Part
	for i, blockAny := range blocks {
		block, ok := blockAny.(map[string]any)
		if !ok {
			continue
		}
		switch stringValue(block, "type") {
		case "text", "":
			parts = append(parts, textPart(sessionID, entryID, i, role, "text", stringValue(block, "text"), opencode.PartKindText, sourceRecord, block))
		case "image":
			part := textPart(sessionID, entryID, i, role, "file", stringValue(block, "mimeType"), opencode.PartKindFile, sourceRecord, block)
			part.MIME = stringValue(block, "mimeType")
			parts = append(parts, part)
		}
	}
	return parts
}

func eventPart(sessionID, entryID, typeName string, kind opencode.PartKind, toolName, text string, sourceRecord opencode.FileRecord, raw any) opencode.Part {
	part := textPart(sessionID, entryID, 0, "event", typeName, text, kind, sourceRecord, raw)
	part.ToolName = toolName
	part.Title = bounded(strings.TrimSpace(firstLine(text)), previewRunes)
	return part
}

func textPart(sessionID, entryID string, index int, role, typeName, text string, kind opencode.PartKind, sourceRecord opencode.FileRecord, raw any) opencode.Part {
	text = strings.TrimSpace(text)
	part := opencode.Part{SourceKind: string(source.KindPi), ID: source.NamespacedID(source.KindPi, sessionID, entryID, fmt.Sprintf("part-%03d", index)), Type: typeName, Kind: kind, Preview: bounded(text, previewRunes), IndexText: text, CreatedAt: sourceRecord.ModTime, UpdatedAt: sourceRecord.ModTime, Source: sourceRecord, SizeBytes: sourceRecord.SizeBytes}
	if part.Preview == "" {
		part.Preview = firstNonEmpty(typeName, string(kind))
	}
	if !safeText(text) || len(text) > maxPiIndexTextByte {
		part.Heavy = true
		part.SkippedRaw = true
		part.IndexText = bounded(text, previewRunes)
	}
	if rawJSON, ok := boundedRaw(raw); ok && !part.Heavy {
		part.RawJSON = rawJSON
	}
	if kind == opencode.PartKindFile && strings.Contains(strings.ToLower(text), "base64") {
		part.Binary = true
		part.SkippedRaw = true
		part.RawJSON = ""
	}
	return part
}

func boundedRaw(value any) (string, bool) {
	data, err := json.Marshal(value)
	if err != nil || len(data) > maxPiRawBytes || bytes.ContainsRune(data, '\x00') || !utf8.Valid(data) {
		return "", false
	}
	return string(data), true
}

func textFromContent(content any) string {
	if text, ok := content.(string); ok {
		return text
	}
	blocks, ok := content.([]any)
	if !ok {
		return jsonSummary(content)
	}
	var texts []string
	for _, blockAny := range blocks {
		if block, ok := blockAny.(map[string]any); ok {
			texts = append(texts, stringValue(block, "text"))
		}
	}
	return strings.Join(nonEmpty(texts), "\n")
}

func fileRecord(path string, info fs.FileInfo) opencode.FileRecord {
	return opencode.FileRecord{Path: filepath.Clean(path), SizeBytes: info.Size(), ModTime: info.ModTime()}
}

func unchanged(sourceRecord opencode.FileRecord, metadata map[string]opencode.FileRecord) bool {
	if sourceRecord.Path == "" || metadata == nil {
		return false
	}
	stored, ok := metadata[sourceRecord.Path]
	return ok && stored.SizeBytes == sourceRecord.SizeBytes && stored.ModTime.Equal(sourceRecord.ModTime)
}

type toolCallRef struct {
	message int
	part    int
	data    map[string]any
}

func mergeToolResults(messages []opencode.Message) {
	calls := map[string]toolCallRef{}
	for messageIndex := range messages {
		for partIndex := range messages[messageIndex].Parts {
			part := &messages[messageIndex].Parts[partIndex]
			if part.Kind != opencode.PartKindTool || part.RawJSON == "" {
				continue
			}
			data := rawJSONMap(part.RawJSON)
			if stringValue(data, "type") != "toolCall" {
				continue
			}
			callID := stringValue(data, "id")
			if callID != "" {
				calls[callID] = toolCallRef{message: messageIndex, part: partIndex, data: data}
			}
		}
	}
	if len(calls) == 0 {
		return
	}
	for messageIndex := range messages {
		kept := messages[messageIndex].Parts[:0]
		for _, part := range messages[messageIndex].Parts {
			if part.Type != "tool_result" || part.RawJSON == "" {
				kept = append(kept, part)
				continue
			}
			result := rawJSONMap(part.RawJSON)
			callID := stringValue(result, "toolCallId")
			call, ok := calls[callID]
			if !ok {
				kept = append(kept, part)
				continue
			}
			callPart := &messages[call.message].Parts[call.part]
			callPart.Status = firstNonEmpty(part.Status, "completed")
			callPart.IndexText = normalizeSpace(strings.Join(nonEmpty([]string{callPart.IndexText, part.IndexText, part.Preview}), " "))
			callPart.UpdatedAt = maxTime(callPart.UpdatedAt, part.UpdatedAt)
			if callPart.Preview == "" || callPart.Preview == "{}" {
				callPart.Preview = part.Preview
			}
			if mergedRaw, ok := mergedToolRaw(call.data, result, callPart); ok {
				callPart.RawJSON = mergedRaw
			}
		}
		messages[messageIndex].Parts = kept
	}
}

func rawJSONMap(raw string) map[string]any {
	if raw == "" {
		return nil
	}
	var data map[string]any
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&data); err != nil {
		return nil
	}
	return data
}

func mergedToolRaw(call, result map[string]any, part *opencode.Part) (string, bool) {
	if call == nil || result == nil {
		return "", false
	}
	merged := map[string]any{
		"type":       "tool",
		"tool":       firstNonEmpty(stringValue(call, "name"), part.ToolName),
		"status":     firstNonEmpty(part.Status, "completed"),
		"toolCallId": stringValue(result, "toolCallId"),
		"input":      call["arguments"],
		"output":     result["content"],
		"isError":    result["isError"],
	}
	return boundedRaw(merged)
}

func normalizeSpace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func tokenUsage(data map[string]any) opencode.TokenUsage {
	if data == nil {
		return opencode.TokenUsage{}
	}
	usage := opencode.TokenUsage{Available: true}
	usage.Input = int64Value(data, "input")
	usage.Output = int64Value(data, "output")
	usage.Reasoning = int64Value(data, "reasoning")
	usage.CacheRead = int64Value(data, "cacheRead")
	usage.CacheWrite = int64Value(data, "cacheWrite")
	if cache, ok := mapValue(data, "cache"); ok {
		usage.CacheRead += int64Value(cache, "read")
		usage.CacheWrite += int64Value(cache, "write")
	}
	usage.Total = int64Value(data, "totalTokens")
	if usage.Total == 0 {
		usage.Total = usage.Input + usage.Output + usage.Reasoning + usage.CacheRead + usage.CacheWrite
	}
	return usage
}

func counts(messages []opencode.Message) (int, int, int) {
	parts := 0
	heavy := 0
	for _, message := range messages {
		parts += len(message.Parts)
		for _, part := range message.Parts {
			if part.Heavy {
				heavy++
			}
		}
	}
	return len(messages), parts, heavy
}

func firstUserText(messages []opencode.Message) string {
	for _, message := range messages {
		if message.Role != "user" {
			continue
		}
		for _, part := range message.Parts {
			if part.Kind == opencode.PartKindText && part.Preview != "" {
				return part.Preview
			}
		}
	}
	return ""
}

func shortHash(value string) string {
	sum := sha1.Sum([]byte(value))
	return hex.EncodeToString(sum[:])[:12]
}

func parseTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return t
	}
	return time.Time{}
}

func minNonZeroTime(a, b time.Time) time.Time {
	if a.IsZero() {
		return b
	}
	if b.IsZero() || a.Before(b) {
		return a
	}
	return b
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func stringValue(data map[string]any, key string) string {
	value, ok := data[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	default:
		return fmt.Sprint(typed)
	}
}

func int64Value(data map[string]any, key string) int64 {
	value, ok := data[key]
	if !ok || value == nil {
		return 0
	}
	switch typed := value.(type) {
	case json.Number:
		v, _ := typed.Int64()
		return v
	case float64:
		return int64(typed)
	case int64:
		return typed
	case int:
		return int64(typed)
	default:
		return 0
	}
}

func mapValue(data map[string]any, key string) (map[string]any, bool) {
	value, ok := data[key]
	if !ok {
		return nil, false
	}
	out, ok := value.(map[string]any)
	return out, ok
}

func mapValueOrNil(data map[string]any, key string) map[string]any {
	out, _ := mapValue(data, key)
	return out
}

func jsonSummary(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}

func bounded(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes]) + "…"
}

func safeText(text string) bool {
	lower := strings.ToLower(text)
	return utf8.ValidString(text) && !strings.ContainsRune(text, '\x00') && !(strings.Contains(lower, "data:") && strings.Contains(lower, "base64"))
}

func firstLine(text string) string {
	if idx := strings.IndexByte(text, '\n'); idx >= 0 {
		return text[:idx]
	}
	return text
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func nonEmpty(values []string) []string {
	out := values[:0]
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}
