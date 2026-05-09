package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/quantick/opensession/internal/index"
	"github.com/quantick/opensession/internal/opencode"
)

const maxDetailPreviewRunes = 4000

type toolDetail struct {
	Name        string
	Status      string
	Title       string
	Description string
	Input       map[string]any
	Output      any
	Metadata    map[string]any
}

type messageDetailState struct {
	active         bool
	content        string
	fallback       string
	guard          string
	truncated      bool
	renderMarkdown bool
}

func buildMessageDetail(part index.TimelinePart, renderMarkdown bool) messageDetailState {
	detail := messageDetailState{
		active:         true,
		fallback:       messageDetailFallback(part),
		renderMarkdown: renderMarkdown && strings.EqualFold(strings.TrimSpace(part.Role), "assistant"),
	}
	text, ok, guard := loadMessageDetailText(part)
	if !ok {
		detail.guard = guard
		return detail
	}
	if !safeMessageText(text) {
		detail.guard = "Message text is unsafe or binary-looking; showing indexed preview when available."
		return detail
	}
	detail.content, detail.truncated = capMessageDetailText(text)
	detail.fallback = ""
	return detail
}

func loadMessageDetailText(part index.TimelinePart) (string, bool, string) {
	if part.Binary {
		return "", false, "Message part is too large or unsafe to display normally."
	}
	if part.RawJSON != "" {
		if len(part.RawJSON) > MaxRawDisplayBytes {
			return "", false, "Message raw payload is too large to load for detail display."
		}
		return messageTextFromRawPayload([]byte(part.RawJSON))
	}
	if part.SourcePath == "" {
		return "", false, "Message source text is unavailable."
	}
	if part.SizeBytes > MaxRawDisplayBytes {
		return "", false, "Message source payload is too large to load for detail display."
	}
	if info, err := os.Stat(part.SourcePath); err == nil && info.Size() > MaxRawDisplayBytes {
		return "", false, "Message source payload is too large to load for detail display."
	}
	content, err := os.ReadFile(part.SourcePath)
	if err != nil {
		return "", false, fmt.Sprintf("Message source could not be loaded: %v", err)
	}
	if len(content) > MaxRawDisplayBytes || bytes.ContainsRune(content, '\x00') || !utf8.Valid(content) {
		return "", false, "Message source payload is too large or unsafe to display normally."
	}
	return messageTextFromRawPayload(content)
}

func messageTextFromRawPayload(content []byte) (string, bool, string) {
	var data map[string]any
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()
	if err := decoder.Decode(&data); err != nil {
		return "", false, fmt.Sprintf("Message raw payload could not be parsed: %v", err)
	}
	value, ok := data["text"].(string)
	if !ok {
		return "", false, "Message raw payload does not contain source text."
	}
	return value, true, ""
}

func capMessageDetailText(text string) (string, bool) {
	runes := []rune(text)
	if len(runes) <= MaxMessageDetailRunes {
		return text, false
	}
	return string(runes[:MaxMessageDetailRunes]), true
}

func messageDetailFallback(part index.TimelinePart) string {
	return firstNonEmpty(part.IndexText, part.Preview)
}

func (d messageDetailState) sourceContent() string {
	if d.guard != "" {
		lines := []string{d.guard}
		if d.fallback != "" {
			lines = append(lines, "", "Indexed Preview (may be incomplete)")
			lines = append(lines, splitLines(d.fallback)...)
		}
		return strings.Join(lines, "\n")
	}
	content := d.content
	if content == "" {
		content = "[empty message]"
	}
	if d.truncated {
		content = strings.TrimRight(content, "\n") + "\n\n" + messageDetailTruncationMarker()
	}
	return content
}

func messageDetailTruncationMarker() string {
	return fmt.Sprintf("[message truncated at %d KiB]", MaxMessageDetailRunes/1024)
}

func safeMessageText(text string) bool {
	lower := strings.ToLower(text)
	return utf8.ValidString(text) && !strings.ContainsRune(text, '\x00') && !(strings.Contains(lower, "data:") && strings.Contains(lower, "base64"))
}

func rawPartFromTimelinePart(part index.TimelinePart) index.RawPart {
	return index.RawPart{
		SourceKind: part.SourceKind,
		PartID:     part.PartID,
		SessionID:  part.SessionID,
		MessageID:  part.MessageID,
		Role:       part.Role,
		Type:       part.Type,
		Kind:       part.Kind,
		ToolName:   part.ToolName,
		Status:     part.Status,
		Title:      part.Title,
		FilePath:   part.FilePath,
		SourcePath: part.SourcePath,
		SizeBytes:  part.SizeBytes,
		Heavy:      part.Heavy,
		Binary:     part.Binary,
		SkippedRaw: part.SkippedRaw,
		Preview:    part.Preview,
		IndexText:  part.IndexText,
		RawJSON:    part.RawJSON,
	}
}

func parseDetailPayload(content []byte) map[string]any {
	var data map[string]any
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()
	if err := decoder.Decode(&data); err != nil {
		return nil
	}
	return data
}

func renderPrettyPartDetail(raw index.RawPart, data map[string]any) string {
	switch raw.Kind {
	case opencode.PartKindTool:
		tool := extractToolDetail(raw, data)
		switch strings.ToLower(tool.Name) {
		case "bash":
			return renderBashToolDetail(raw, tool)
		case "grep", "glob":
			return renderSearchToolDetail(raw, tool)
		case "read", "write", "edit", "apply_patch":
			return renderFileToolDetail(raw, tool)
		default:
			return renderGenericToolDetail(raw, tool)
		}
	case opencode.PartKindPatch:
		return renderPatchDetail(raw, data)
	case opencode.PartKindFile:
		return renderFileDetail(raw, data)
	default:
		return strings.Join(baseDetailLines("Part Detail", raw), "\n")
	}
}

func extractToolDetail(raw index.RawPart, data map[string]any) toolDetail {
	state := mapValue(data, "state")
	input := firstMap(mapValue(state, "input"), mapValue(data, "input"))
	metadata := firstMap(mapValue(state, "metadata"), mapValue(data, "metadata"))
	output := firstValue(valueAt(state, "output"), valueAt(state, "result"), valueAt(data, "output"), valueAt(data, "result"), valueAt(state, "error"), valueAt(data, "error"))

	return toolDetail{
		Name:        firstNonEmpty(raw.ToolName, stringValue(data, "tool"), stringValue(data, "name")),
		Status:      firstNonEmpty(raw.Status, stringValue(state, "status"), stringValue(data, "status")),
		Title:       firstNonEmpty(raw.Title, stringValue(state, "title"), stringValue(data, "title")),
		Description: firstNonEmpty(stringValue(state, "description"), stringValue(input, "description"), stringValue(data, "description")),
		Input:       input,
		Output:      output,
		Metadata:    metadata,
	}
}

func renderBashToolDetail(raw index.RawPart, tool toolDetail) string {
	lines := baseToolLines("Tool Detail", raw, tool)
	lines = appendField(lines, "Description", tool.Description)
	lines = appendField(lines, "Workdir", stringValue(tool.Input, "workdir"))
	lines = appendTextSection(lines, "Command", firstNonEmpty(stringValue(tool.Input, "command"), raw.Preview))
	lines = appendTextSection(lines, "Output", safeDetailText(tool.Output))
	lines = appendMapSection(lines, "Metadata", tool.Metadata, nil)
	return strings.Join(lines, "\n")
}

func renderSearchToolDetail(raw index.RawPart, tool toolDetail) string {
	lines := baseToolLines("Search Detail", raw, tool)
	lines = appendMapSection(lines, "Search Input", tool.Input, []string{"pattern", "query", "path", "include", "glob", "root", "limit"})
	lines = appendTextSection(lines, "Results", safeDetailText(tool.Output))
	lines = appendMapSection(lines, "Metadata", tool.Metadata, nil)
	return strings.Join(lines, "\n")
}

func renderFileToolDetail(raw index.RawPart, tool toolDetail) string {
	lines := baseToolLines("File Tool Detail", raw, tool)
	lines = appendMapSection(lines, "Target", tool.Input, []string{"filePath", "filepath", "path", "file", "offset", "limit", "start", "end"})
	lines = appendTextSection(lines, "Patch", firstNonEmpty(safeDetailText(valueAt(tool.Input, "patchText")), safeDetailText(valueAt(tool.Input, "patch")), safeDetailText(valueAt(tool.Input, "diff"))))
	lines = appendTextSection(lines, "Content Preview", firstNonEmpty(safeDetailText(valueAt(tool.Input, "content")), safeDetailText(valueAt(tool.Input, "text")), safeDetailText(valueAt(tool.Input, "old_string")), safeDetailText(valueAt(tool.Input, "new_string"))))
	lines = appendTextSection(lines, "Output", safeDetailText(tool.Output))
	lines = appendMapSection(lines, "Metadata", tool.Metadata, nil)
	return strings.Join(lines, "\n")
}

func renderGenericToolDetail(raw index.RawPart, tool toolDetail) string {
	lines := baseToolLines("Tool Detail", raw, tool)
	lines = appendField(lines, "Description", tool.Description)
	lines = appendMapSection(lines, "Input", tool.Input, []string{"command", "description", "path", "file", "pattern", "query"})
	lines = appendTextSection(lines, "Output", safeDetailText(tool.Output))
	lines = appendMapSection(lines, "Metadata", tool.Metadata, nil)
	return strings.Join(lines, "\n")
}

func renderPatchDetail(raw index.RawPart, data map[string]any) string {
	metadata := firstMap(mapValue(data, "metadata"), mapValue(data, "state"))
	lines := baseDetailLines("Patch Detail", raw)
	lines = appendField(lines, "Title", firstNonEmpty(raw.Title, stringValue(data, "title")))
	lines = appendField(lines, "Path", firstNonEmpty(raw.FilePath, stringValue(data, "path"), stringValue(data, "file")))
	lines = appendField(lines, "Summary", firstNonEmpty(raw.Preview, raw.IndexText))
	lines = appendTextSection(lines, "Diff", firstNonEmpty(safeDetailText(valueAt(data, "diff")), safeDetailText(valueAt(data, "patch")), safeDetailText(valueAt(data, "content")), safeDetailText(valueAt(metadata, "diff"))))
	lines = appendMapSection(lines, "Metadata", metadata, []string{"path", "file", "title"})
	return strings.Join(lines, "\n")
}

func renderFileDetail(raw index.RawPart, data map[string]any) string {
	source := mapValue(data, "source")
	text := mapValue(source, "text")
	lines := baseDetailLines("File Detail", raw)
	lines = appendField(lines, "Title", firstNonEmpty(raw.Title, stringValue(data, "title")))
	lines = appendField(lines, "Path", firstNonEmpty(raw.FilePath, stringValue(source, "path"), stringValue(data, "path")))
	lines = appendField(lines, "Filename", stringValue(data, "filename"))
	lines = appendField(lines, "MIME", stringValue(data, "mime"))
	lines = appendField(lines, "Summary", firstNonEmpty(raw.Preview, raw.IndexText))
	lines = appendMapSection(lines, "Source", source, []string{"type", "path"})
	lines = appendTextSection(lines, "Text Preview", firstNonEmpty(safeDetailText(valueAt(text, "value")), safeDetailText(valueAt(data, "text")), safeDetailText(valueAt(data, "content"))))
	return strings.Join(lines, "\n")
}

func baseToolLines(title string, raw index.RawPart, tool toolDetail) []string {
	lines := baseDetailLines(title, raw)
	lines = appendField(lines, "Tool", firstNonEmpty(tool.Name, "tool"))
	lines = appendField(lines, "Status", tool.Status)
	lines = appendField(lines, "Title", tool.Title)
	lines = appendField(lines, "Summary", firstNonEmpty(raw.Preview, raw.IndexText))
	return lines
}

func baseDetailLines(title string, raw index.RawPart) []string {
	lines := []string{title}
	lines = appendField(lines, "Kind", firstNonEmpty(string(raw.Kind), raw.Type, "part"))
	lines = appendField(lines, "Part ID", raw.PartID)
	return lines
}

func appendField(lines []string, label, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return lines
	}
	return append(lines, fmt.Sprintf("%s: %s", label, singleLinePreview(value)))
}

func appendTextSection(lines []string, title, text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return lines
	}
	lines = append(lines, "", title)
	for _, line := range splitLines(text) {
		lines = append(lines, "  "+line)
	}
	return lines
}

func appendMapSection(lines []string, title string, data map[string]any, preferred []string) []string {
	if len(data) == 0 {
		return lines
	}
	keys := orderedKeys(data, preferred)
	section := []string{"", title}
	for _, key := range keys {
		value := safeDetailText(data[key])
		if value == "" {
			continue
		}
		if strings.Contains(value, "\n") {
			section = append(section, "  "+key+":")
			for _, line := range splitLines(value) {
				section = append(section, "    "+line)
			}
			continue
		}
		section = append(section, fmt.Sprintf("  %s: %s", key, singleLinePreview(value)))
	}
	if len(section) == 2 {
		return lines
	}
	return append(lines, section...)
}

func orderedKeys(data map[string]any, preferred []string) []string {
	seen := map[string]bool{}
	var keys []string
	for _, key := range preferred {
		if _, ok := data[key]; ok {
			keys = append(keys, key)
			seen[key] = true
		}
	}
	var rest []string
	for key := range data {
		if !seen[key] {
			rest = append(rest, key)
		}
	}
	sort.Strings(rest)
	return append(keys, rest...)
}

func safeDetailText(value any) string {
	text := strings.TrimSpace(detailValueString(value))
	if text == "" {
		return ""
	}
	lower := strings.ToLower(text)
	if strings.Contains(lower, "data:") && strings.Contains(lower, "base64") {
		return ""
	}
	if strings.ContainsRune(text, '\x00') {
		return ""
	}
	return truncateRunes(text, maxDetailPreviewRunes)
}

func singleLinePreview(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.Join(strings.Fields(strings.ReplaceAll(value, "\n", " ")), " ")
	return truncateRunes(value, 240)
}

func detailValueString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case json.Number:
		return typed.String()
	case bool:
		return fmt.Sprintf("%t", typed)
	case float64:
		return fmt.Sprintf("%g", typed)
	case map[string]any, []any:
		if raw, err := json.MarshalIndent(typed, "", "  "); err == nil {
			return string(raw)
		}
		return fmt.Sprint(typed)
	default:
		return fmt.Sprint(typed)
	}
}

func mapValue(data map[string]any, key string) map[string]any {
	if data == nil {
		return nil
	}
	value, ok := data[key].(map[string]any)
	if !ok {
		return nil
	}
	return value
}

func stringValue(data map[string]any, key string) string {
	return safeDetailText(valueAt(data, key))
}

func valueAt(data map[string]any, key string) any {
	if data == nil {
		return nil
	}
	return data[key]
}

func firstValue(values ...any) any {
	for _, value := range values {
		if safeDetailText(value) != "" {
			return value
		}
	}
	return nil
}

func firstMap(values ...map[string]any) map[string]any {
	for _, value := range values {
		if len(value) > 0 {
			return value
		}
	}
	return nil
}
