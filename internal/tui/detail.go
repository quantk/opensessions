package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

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
