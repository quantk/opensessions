package opencode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const heavyPartProbeBytes = 256 * 1024

func classifyPart(path string, info fs.FileInfo, raw []byte) (Part, error) {
	if info.Size() > DefaultHeavyFileBytes {
		return classifyHeavyPartBytes(path, info, raw), nil
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return Part{}, fmt.Errorf("parse %s: %w", path, err)
	}

	typeName := stringValue(data, "type")
	part := Part{
		ID:        firstNonEmpty(stringValue(data, "id"), strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))),
		SessionID: stringValue(data, "sessionID"),
		MessageID: stringValue(data, "messageID"),
		Type:      typeName,
		Kind:      classifyKind(typeName),
		MIME:      stringValue(data, "mime"),
		Filename:  stringValue(data, "filename"),
		CreatedAt: unixMilli(timeValue(data, "start")),
		UpdatedAt: unixMilli(timeValue(data, "end")),
		SizeBytes: info.Size(),
		Source:    fileRecord(path, info),
	}
	if part.MessageID == "" {
		part.MessageID = filepath.Base(filepath.Dir(path))
	}

	part.Binary = part.Kind == PartKindFile && containsDataURL(stringValue(data, "url"))
	part.Heavy = info.Size() > DefaultHeavyFileBytes || (part.Kind != PartKindFile && containsRawArtifact(data))
	part.SkippedRaw = part.Heavy || part.Binary

	switch part.Kind {
	case PartKindText, PartKindReasoning:
		classifyTextPart(&part, data)
	case PartKindTool:
		classifyToolPart(&part, data)
	case PartKindPatch:
		classifyPatchPart(&part, data)
	case PartKindFile:
		classifyFilePart(&part, data)
	case PartKindStepStart:
		classifyStepStartPart(&part, data)
	case PartKindStepFinish:
		classifyStepFinishPart(&part, data)
	default:
		part.Preview = "unknown part"
	}

	part.IndexText = normalizeIndexText(part.IndexText)
	part.Preview = bounded(strings.TrimSpace(part.Preview), previewRunes)
	if !part.Heavy && !part.Binary && !part.SkippedRaw {
		part.RawJSON = string(raw)
	}
	return part, nil
}

func classifyHeavyPartFile(path string, info fs.FileInfo) (Part, error) {
	file, err := os.Open(path)
	if err != nil {
		return Part{}, fmt.Errorf("read heavy part prefix %s: %w", path, err)
	}
	defer file.Close()
	prefix := make([]byte, heavyPartProbeBytes)
	n, err := file.Read(prefix)
	if err != nil && n == 0 {
		return Part{}, fmt.Errorf("read heavy part prefix %s: %w", path, err)
	}
	return classifyHeavyPartPrefix(path, info, prefix[:n]), nil
}

func classifyHeavyPartBytes(path string, info fs.FileInfo, raw []byte) Part {
	if len(raw) > heavyPartProbeBytes {
		raw = raw[:heavyPartProbeBytes]
	}
	return classifyHeavyPartPrefix(path, info, raw)
}

func classifyHeavyPartPrefix(path string, info fs.FileInfo, prefix []byte) Part {
	typeName := shallowJSONString(prefix, "type")
	part := Part{
		ID:         firstNonEmpty(shallowJSONString(prefix, "id"), strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))),
		SessionID:  shallowJSONString(prefix, "sessionID"),
		MessageID:  shallowJSONString(prefix, "messageID"),
		Type:       typeName,
		Kind:       classifyKind(typeName),
		ToolName:   shallowJSONString(prefix, "tool"),
		Status:     shallowJSONString(prefix, "status"),
		Title:      firstNonEmpty(shallowJSONString(prefix, "title"), shallowJSONString(prefix, "description")),
		MIME:       shallowJSONString(prefix, "mime"),
		Filename:   shallowJSONString(prefix, "filename"),
		CreatedAt:  unixMilli(shallowJSONInt(prefix, "start")),
		UpdatedAt:  unixMilli(shallowJSONInt(prefix, "end")),
		SizeBytes:  info.Size(),
		Heavy:      true,
		SkippedRaw: true,
		Source:     fileRecord(path, info),
	}
	if part.MessageID == "" {
		part.MessageID = filepath.Base(filepath.Dir(path))
	}
	part.Binary = part.Kind == PartKindFile && containsDataURL(shallowJSONString(prefix, "url"))
	part.SubagentName = firstNonEmpty(
		shallowJSONString(prefix, "subagent_type"),
		shallowJSONString(prefix, "subagentType"),
		shallowJSONString(prefix, "subagent"),
		shallowJSONString(prefix, "subagentName"),
		shallowJSONString(prefix, "agent_type"),
		shallowJSONString(prefix, "agentType"),
		shallowJSONString(prefix, "agent"),
		shallowJSONString(prefix, "agentName"),
	)
	part.LinkedSessionID = shallowJSONString(prefix, "sessionId")
	part.FilePath = firstNonEmpty(shallowJSONString(prefix, "path"), part.Filename)

	switch part.Kind {
	case PartKindText, PartKindReasoning:
		part.Preview = fmt.Sprintf("%s part skipped (%d bytes)", part.Kind, info.Size())
	case PartKindTool:
		fields := []string{"tool", part.ToolName, part.Status, part.Title, part.SubagentName, part.LinkedSessionID}
		for _, key := range []string{"command", "description", "workdir", "path", "file", "pattern"} {
			fields = append(fields, shallowJSONString(prefix, key))
		}
		part.IndexText = strings.Join(nonEmpty(fields), " ")
		part.Preview = firstNonEmpty(part.Title, fmt.Sprintf("%s %s (heavy raw payload skipped)", part.ToolName, part.Status), "tool part (heavy raw payload skipped)")
	case PartKindPatch:
		part.IndexText = strings.Join(nonEmpty([]string{"patch", part.Title, part.FilePath}), " ")
		part.Preview = firstNonEmpty(part.Title, "patch part (heavy raw payload skipped)")
	case PartKindFile:
		part.IndexText = strings.Join(nonEmpty([]string{"file", part.FilePath, part.Filename, part.MIME}), " ")
		part.Preview = firstNonEmpty(part.FilePath, part.Filename, part.MIME, "file part (heavy raw payload skipped)")
	case PartKindStepStart:
		part.Preview = "step start"
	case PartKindStepFinish:
		part.Preview = "step finish"
	default:
		part.Preview = "unknown part (heavy raw payload skipped)"
	}
	part.IndexText = normalizeIndexText(part.IndexText)
	part.Preview = bounded(strings.TrimSpace(part.Preview), previewRunes)
	return part
}

func shallowJSONString(prefix []byte, key string) string {
	needle := []byte(`"` + key + `"`)
	for search := prefix; len(search) > 0; {
		idx := bytes.Index(search, needle)
		if idx < 0 {
			return ""
		}
		rest := search[idx+len(needle):]
		colon := bytes.IndexByte(rest, ':')
		if colon < 0 {
			return ""
		}
		rest = bytes.TrimLeft(rest[colon+1:], " \t\r\n")
		if len(rest) == 0 {
			return ""
		}
		if rest[0] != '"' {
			search = rest[1:]
			continue
		}
		end := 1
		escaped := false
		for ; end < len(rest); end++ {
			if escaped {
				escaped = false
				continue
			}
			switch rest[end] {
			case '\\':
				escaped = true
			case '"':
				var value string
				if err := json.Unmarshal(rest[:end+1], &value); err == nil {
					return value
				}
				return ""
			}
		}
		return ""
	}
	return ""
}

func shallowJSONInt(prefix []byte, key string) int64 {
	needle := []byte(`"` + key + `"`)
	idx := bytes.Index(prefix, needle)
	if idx < 0 {
		return 0
	}
	rest := prefix[idx+len(needle):]
	colon := bytes.IndexByte(rest, ':')
	if colon < 0 {
		return 0
	}
	rest = bytes.TrimLeft(rest[colon+1:], " \t\r\n")
	end := 0
	for end < len(rest) && ((rest[end] >= '0' && rest[end] <= '9') || rest[end] == '-') {
		end++
	}
	if end == 0 {
		return 0
	}
	var value int64
	for _, b := range rest[:end] {
		if b < '0' || b > '9' {
			return 0
		}
		value = value*10 + int64(b-'0')
	}
	return value
}

func classifyKind(typeName string) PartKind {
	switch typeName {
	case "text":
		return PartKindText
	case "reasoning":
		return PartKindReasoning
	case "tool":
		return PartKindTool
	case "patch":
		return PartKindPatch
	case "file":
		return PartKindFile
	case "step-start":
		return PartKindStepStart
	case "step-finish":
		return PartKindStepFinish
	default:
		return PartKindUnknown
	}
}

func classifyTextPart(part *Part, data map[string]any) {
	text := stringValue(data, "text")
	if isSafeText(text) && len(text) <= DefaultHeavyTextBytes {
		part.IndexText = text
		part.Preview = text
		return
	}
	part.Heavy = true
	part.SkippedRaw = true
	part.Preview = fmt.Sprintf("%s part skipped (%d bytes)", part.Kind, len(text))
}

func classifyToolPart(part *Part, data map[string]any) {
	part.ToolName = stringValue(data, "tool")
	state := mapValue(data, "state")
	part.Status = stringValue(state, "status")
	part.Title = firstNonEmpty(stringValue(state, "title"), stringValue(state, "description"))
	input := mapValue(state, "input")
	metadata := mapValue(state, "metadata")
	part.SubagentName = firstNonEmpty(
		stringValue(input, "subagent_type"),
		stringValue(input, "subagentType"),
		stringValue(input, "subagent"),
		stringValue(input, "subagentName"),
		stringValue(input, "agent_type"),
		stringValue(input, "agentType"),
		stringValue(input, "agent"),
		stringValue(input, "agentName"),
		stringValue(metadata, "subagent_type"),
		stringValue(metadata, "subagentType"),
		stringValue(metadata, "subagent"),
		stringValue(metadata, "subagentName"),
		stringValue(metadata, "agent_type"),
		stringValue(metadata, "agentType"),
		stringValue(metadata, "agent"),
		stringValue(metadata, "agentName"),
	)
	part.LinkedSessionID = stringValue(metadata, "sessionId")

	fields := []string{"tool", part.ToolName, part.Status, part.Title, part.SubagentName}
	for _, key := range []string{"command", "description", "workdir", "path", "file", "pattern"} {
		fields = append(fields, stringValue(input, key))
	}
	fields = append(fields, collectPathStrings(input)...)

	part.IndexText = strings.Join(nonEmpty(fields), " ")
	if part.Heavy {
		part.Preview = firstNonEmpty(part.Title, fmt.Sprintf("%s %s (heavy raw payload skipped)", part.ToolName, part.Status))
		return
	}
	output := stringValue(state, "output")
	previewFields := []string{part.ToolName, part.Status, part.Title, output}
	part.Preview = strings.Join(nonEmpty(previewFields), " - ")
}

func classifyPatchPart(part *Part, data map[string]any) {
	fields := []string{"patch", stringValue(data, "title"), stringValue(data, "path"), stringValue(data, "file")}
	fields = append(fields, collectPathStrings(data)...)
	part.IndexText = strings.Join(nonEmpty(fields), " ")
	if part.Heavy {
		part.Preview = "patch part (heavy raw payload skipped)"
		return
	}
	part.Preview = firstNonEmpty(stringValue(data, "title"), part.IndexText, "patch part")
}

func classifyFilePart(part *Part, data map[string]any) {
	source := mapValue(data, "source")
	text := mapValue(source, "text")
	part.FilePath = firstNonEmpty(stringValue(source, "path"), stringValue(data, "path"), part.Filename)
	fields := []string{"file", part.FilePath, part.Filename, part.MIME, stringValue(text, "value")}
	part.IndexText = strings.Join(nonEmpty(fields), " ")
	part.Preview = strings.Join(nonEmpty([]string{part.FilePath, part.MIME, stringValue(text, "value")}), " - ")
}

func classifyStepStartPart(part *Part, data map[string]any) {
	snapshot := stringValue(data, "snapshot")
	part.IndexText = strings.Join(nonEmpty([]string{"step start", snapshot}), " ")
	part.Preview = firstNonEmpty("step start "+snapshot, "step start")
}

func classifyStepFinishPart(part *Part, data map[string]any) {
	reason := stringValue(data, "reason")
	part.IndexText = strings.Join(nonEmpty([]string{"step finish", reason}), " ")
	part.Preview = firstNonEmpty("step finish "+reason, "step finish")
}

func containsRawArtifact(value any) bool {
	switch typed := value.(type) {
	case string:
		return len(typed) > DefaultHeavyTextBytes || containsDataURL(typed) || strings.ContainsRune(typed, '\x00') || !utf8.ValidString(typed)
	case []any:
		for _, item := range typed {
			if containsRawArtifact(item) {
				return true
			}
		}
	case map[string]any:
		for _, item := range typed {
			if containsRawArtifact(item) {
				return true
			}
		}
	}
	return false
}

func containsDataURL(value string) bool {
	return strings.Contains(strings.ToLower(value), "data:") && strings.Contains(strings.ToLower(value), "base64")
}

func isSafeText(value string) bool {
	return utf8.ValidString(value) && !strings.ContainsRune(value, '\x00') && !containsDataURL(value)
}

func bounded(value string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes]) + "..."
}

func normalizeIndexText(value string) string {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return ""
	}
	return strings.Join(fields, " ")
}

func stringValue(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	value, ok := data[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	case float64:
		return fmt.Sprintf("%.0f", typed)
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func timeValue(data map[string]any, key string) int64 {
	timeMap := mapValue(data, "time")
	value, ok := timeMap[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case json.Number:
		parsed, _ := typed.Int64()
		return parsed
	default:
		return 0
	}
}

func mapValue(data map[string]any, key string) map[string]any {
	if data == nil {
		return nil
	}
	value, ok := data[key]
	if !ok || value == nil {
		return nil
	}
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return nil
}

func collectPathStrings(value any) []string {
	var out []string
	var walk func(any)
	walk = func(current any) {
		switch typed := current.(type) {
		case map[string]any:
			keys := make([]string, 0, len(typed))
			for key := range typed {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				lower := strings.ToLower(key)
				if lower == "path" || lower == "file" || lower == "filename" {
					out = append(out, stringValue(typed, key))
				}
				walk(typed[key])
			}
		case []any:
			for _, item := range typed {
				walk(item)
			}
		}
	}
	walk(value)
	return nonEmpty(out)
}

func nonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
