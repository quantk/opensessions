package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/quantick/opensession/internal/index"
	"github.com/quantick/opensession/internal/opencode"
)

func TestModelNavigatesSessionsTimelineAndBack(t *testing.T) {
	repo := newFakeRepo(t)
	model := NewModel(repo, repo.sessions)

	model = sendKey(t, model, "j")
	if model.selectedSession != 1 {
		t.Fatalf("selected session = %d, want 1", model.selectedSession)
	}
	model = sendKey(t, model, "k")
	if model.selectedSession != 0 {
		t.Fatalf("selected session = %d, want 0", model.selectedSession)
	}

	model = sendKey(t, model, "l")
	if model.mode != ViewTimeline {
		t.Fatalf("mode = %v, want timeline", model.mode)
	}
	if repo.timelineLoads != 1 {
		t.Fatalf("timeline loads = %d, want 1", repo.timelineLoads)
	}

	view := model.View()
	if !strings.Contains(view, "[reasoning hidden]") || strings.Contains(view, "secret reasoning") {
		t.Fatalf("reasoning should be hidden by default:\n%s", view)
	}
	model = sendKey(t, model, "r")
	if !strings.Contains(model.View(), "secret reasoning") {
		t.Fatalf("reasoning toggle did not reveal content:\n%s", model.View())
	}

	model = sendKey(t, model, "h")
	if model.mode != ViewSessions {
		t.Fatalf("mode = %v, want sessions", model.mode)
	}
}

func TestModelContextSensitiveSearch(t *testing.T) {
	repo := newFakeRepo(t)
	model := NewModel(repo, repo.sessions)

	model = search(t, model, "global")
	if repo.lastSessionSearch != "global" {
		t.Fatalf("session search query = %q", repo.lastSessionSearch)
	}
	if len(model.sessions) != 1 || model.sessions[0].ID != "ses_global" {
		t.Fatalf("filtered sessions = %#v", model.sessions)
	}

	model = NewModel(repo, repo.sessions)
	model = sendKey(t, model, "l")
	model = search(t, model, "README")
	if repo.lastTimelineSearch != "README" {
		t.Fatalf("timeline search query = %q", repo.lastTimelineSearch)
	}
	if len(model.timeline) != 1 || model.timeline[0].PartID != "prt_file" {
		t.Fatalf("filtered timeline = %#v", model.timeline)
	}
}

func TestModelRawPartGuard(t *testing.T) {
	repo := newFakeRepo(t)
	model := NewModel(repo, repo.sessions)
	model = sendKey(t, model, "l")
	model = sendKey(t, model, "j")
	model = sendKey(t, model, "enter")

	if model.mode != ViewRawPart {
		t.Fatalf("mode = %v, want raw part", model.mode)
	}
	view := model.View()
	if !strings.Contains(view, "too large or unsafe") {
		t.Fatalf("raw guard missing:\n%s", view)
	}
	if strings.Contains(view, "AAECAwQFBgc") {
		t.Fatalf("raw heavy content should not render:\n%s", view)
	}
}

func TestModelRenderingIsBounded(t *testing.T) {
	repo := newFakeRepo(t)
	repo.timelines["ses_project"] = nil
	for i := 0; i < 30; i++ {
		repo.timelines["ses_project"] = append(repo.timelines["ses_project"], index.TimelinePart{
			PartID:    strings.Join([]string{"prt", string(rune('a' + i))}, "_"),
			SessionID: "ses_project",
			Role:      "assistant",
			Kind:      opencode.PartKindText,
			Preview:   strings.Join([]string{"visible item", string(rune('0' + i%10))}, " "),
		})
	}
	model := NewModel(repo, repo.sessions)
	model, _ = updateModel(t, model, tea.WindowSizeMsg{Width: 80, Height: 8})
	model = sendKey(t, model, "l")

	view := model.View()
	if !strings.Contains(view, "visible item 0") {
		t.Fatalf("first visible item missing:\n%s", view)
	}
	if strings.Contains(view, "visible item 9") {
		t.Fatalf("render should be bounded to visible rows:\n%s", view)
	}
}

func TestTimelineJKScrollsTranscript(t *testing.T) {
	repo := newFakeRepo(t)
	repo.timelines["ses_project"] = nil
	for i := 0; i < 12; i++ {
		repo.timelines["ses_project"] = append(repo.timelines["ses_project"], index.TimelinePart{
			PartID:    fmt.Sprintf("prt_%02d", i),
			SessionID: "ses_project",
			MessageID: fmt.Sprintf("msg_%02d", i),
			Role:      "assistant",
			Kind:      opencode.PartKindText,
			Preview:   fmt.Sprintf("visible item %d", i),
			IndexText: fmt.Sprintf("visible item %d", i),
		})
	}
	model := NewModel(repo, repo.sessions)
	model, _ = updateModel(t, model, tea.WindowSizeMsg{Width: 80, Height: 9})
	model = sendKey(t, model, "l")

	if model.selectedPart != 0 {
		t.Fatalf("selected part = %d, want first transcript part", model.selectedPart)
	}
	if !strings.Contains(model.View(), "visible item 0") {
		t.Fatalf("initial transcript missing first item:\n%s", model.View())
	}
	for i := 0; i < 10; i++ {
		model = sendKey(t, model, "j")
	}
	view := model.View()
	if model.selectedPart < 5 || model.timelineScroll == 0 {
		t.Fatalf("j/k did not move focus through timeline: selected=%d scroll=%d", model.selectedPart, model.timelineScroll)
	}
	if !strings.Contains(view, fmt.Sprintf("visible item %d", model.selectedPart)) || strings.Contains(view, "visible item 0") {
		t.Fatalf("timeline viewport did not move as expected:\n%s", view)
	}
}

func TestTimelineJKScrollsWithinLongFocusedMessage(t *testing.T) {
	repo := newFakeRepo(t)
	var lines []string
	for i := 0; i < 30; i++ {
		lines = append(lines, fmt.Sprintf("long line %02d", i))
	}
	text := strings.Join(lines, "\n")
	repo.timelines["ses_project"] = []index.TimelinePart{
		{PartID: "prt_long", SessionID: "ses_project", MessageID: "msg_user", Role: "user", Kind: opencode.PartKindText, Preview: text, IndexText: text, RawJSON: fmt.Sprintf(`{"type":"text","text":%q}`, text)},
		{PartID: "prt_after", SessionID: "ses_project", MessageID: "msg_assistant", Role: "assistant", Kind: opencode.PartKindText, Preview: "after long", IndexText: "after long"},
	}
	model := NewModel(repo, repo.sessions)
	model, _ = updateModel(t, model, tea.WindowSizeMsg{Width: 80, Height: 9})
	model = sendKey(t, model, "l")

	for i := 0; i < 8; i++ {
		model = sendKey(t, model, "j")
	}
	if model.selectedPart != 0 {
		t.Fatalf("focus moved away from long message too early: selected=%d", model.selectedPart)
	}
	if model.timelineScroll < 8 {
		t.Fatalf("j did not keep scrolling inside long focused message: scroll=%d", model.timelineScroll)
	}
	view := model.View()
	if strings.Contains(view, "long line 00") || !strings.Contains(view, "long line 08") {
		t.Fatalf("long focused message viewport did not advance:\n%s", view)
	}
}

func TestTimelineOpensFocusedToolDetails(t *testing.T) {
	repo := newFakeRepo(t)
	rawPath := filepath.Join(t.TempDir(), "tool.json")
	raw := []byte(`{"type":"tool","tool":"bash","state":{"status":"completed","input":{"command":"go test ./...","description":"Run tests","workdir":"/tmp/fixture"},"output":"ok"}}`)
	if err := os.WriteFile(rawPath, raw, 0o644); err != nil {
		t.Fatalf("write raw tool: %v", err)
	}
	repo.timelines["ses_project"] = []index.TimelinePart{
		{PartID: "prt_text", SessionID: "ses_project", MessageID: "msg_user", Role: "user", Kind: opencode.PartKindText, Preview: "run tests", IndexText: "run tests"},
		{PartID: "prt_tool", SessionID: "ses_project", MessageID: "msg_assistant", Role: "assistant", Kind: opencode.PartKindTool, ToolName: "bash", Status: "completed", Preview: "go test ./...", SourcePath: rawPath, SizeBytes: int64(len(raw))},
	}
	repo.rawParts["prt_tool"] = index.RawPart{PartID: "prt_tool", Kind: opencode.PartKindTool, ToolName: "bash", Status: "completed", SourcePath: rawPath, SizeBytes: int64(len(raw))}

	model := NewModel(repo, repo.sessions)
	model = sendKey(t, model, "l")
	if model.selectedPart != 0 {
		t.Fatalf("selected part = %d, want initial text focus", model.selectedPart)
	}
	model = sendKey(t, model, "j")
	if model.selectedPart != 1 {
		t.Fatalf("selected part = %d, want tool focus after j", model.selectedPart)
	}
	if !strings.Contains(model.View(), "> [tool] bash") {
		t.Fatalf("focused tool card missing:\n%s", model.View())
	}

	model = sendKey(t, model, "enter")
	view := model.View()
	if model.mode != ViewRawPart {
		t.Fatalf("mode = %v, want raw detail", model.mode)
	}
	if !strings.Contains(view, "Pretty Detail") || strings.Contains(view, "Raw JSON") {
		t.Fatalf("tool detail should default to pretty mode:\n%s", view)
	}
	if !strings.Contains(view, "Command") || !strings.Contains(view, "go test ./...") || !strings.Contains(view, "Workdir: /tmp/fixture") {
		t.Fatalf("tool detail missing structured fields:\n%s", view)
	}
}

func TestRawPartToggleShowsRawJSON(t *testing.T) {
	repo := newFakeRepo(t)
	raw := `{"type":"tool","tool":"bash","state":{"status":"completed","input":{"command":"go test ./..."},"output":"ok"}}`
	repo.rawParts["prt_tool"] = index.RawPart{PartID: "prt_tool", Kind: opencode.PartKindTool, ToolName: "bash", Status: "completed", RawJSON: raw, SizeBytes: int64(len(raw))}

	model := NewModel(repo, repo.sessions).openRawPart("prt_tool")
	if model.rawMode || !strings.Contains(model.View(), "Pretty Detail") {
		t.Fatalf("detail should start in pretty mode:\n%s", model.View())
	}

	model.rawScroll = 999
	model = sendKey(t, model, "R")
	view := model.View()
	if !model.rawMode || !strings.Contains(view, "Raw JSON") || !strings.Contains(view, `"command": "go test ./..."`) {
		t.Fatalf("raw toggle did not show raw JSON:\n%s", view)
	}
	if model.rawScroll > model.maxRawScroll() {
		t.Fatalf("raw scroll = %d, max = %d", model.rawScroll, model.maxRawScroll())
	}

	model = search(t, model, "command")
	if !strings.Contains(model.View(), `"command": "go test ./..."`) || strings.Contains(model.View(), `"output": "ok"`) {
		t.Fatalf("raw filter should apply to raw content:\n%s", model.View())
	}
	model = sendKey(t, model, "R")
	view = model.View()
	if model.rawMode || !strings.Contains(view, "Pretty Detail") || !strings.Contains(view, "Command") || strings.Contains(view, `"command"`) {
		t.Fatalf("pretty toggle should filter pretty content:\n%s", view)
	}
}

func TestPrettyDetailRendersGenericToolPatchAndFile(t *testing.T) {
	repo := newFakeRepo(t)
	unknownRaw := `{"type":"tool","tool":"custom_lookup","state":{"status":"failed","input":{"query":"needle","path":"src"},"output":{"error":"not found"},"metadata":{"duration":"1s"}}}`
	patchRaw := `{"type":"patch","title":"Update README","path":"README.md","diff":"@@ -1 +1\n-old\n+new"}`
	fileRaw := `{"type":"file","mime":"text/plain","filename":"README.md","source":{"type":"file","path":"README.md","text":{"value":"hello docs","start":1,"end":2}}}`
	repo.rawParts["prt_unknown"] = index.RawPart{PartID: "prt_unknown", Kind: opencode.PartKindTool, ToolName: "custom_lookup", Status: "failed", RawJSON: unknownRaw, SizeBytes: int64(len(unknownRaw))}
	repo.rawParts["prt_patch"] = index.RawPart{PartID: "prt_patch", Kind: opencode.PartKindPatch, Title: "Update README", RawJSON: patchRaw, SizeBytes: int64(len(patchRaw))}
	repo.rawParts["prt_safe_file"] = index.RawPart{PartID: "prt_safe_file", Kind: opencode.PartKindFile, FilePath: "README.md", RawJSON: fileRaw, SizeBytes: int64(len(fileRaw))}

	model := NewModel(repo, repo.sessions).openRawPart("prt_unknown")
	view := model.View()
	if !strings.Contains(view, "Tool Detail") || !strings.Contains(view, "custom_lookup") || !strings.Contains(view, "query: needle") || !strings.Contains(view, "error") {
		t.Fatalf("generic tool detail missing expected fields:\n%s", view)
	}

	model = model.openRawPart("prt_patch")
	view = model.View()
	if !strings.Contains(view, "Patch Detail") || !strings.Contains(view, "README.md") || !strings.Contains(view, "Diff") || !strings.Contains(view, "+new") {
		t.Fatalf("patch detail missing expected fields:\n%s", view)
	}

	model = model.openRawPart("prt_safe_file")
	view = model.View()
	if !strings.Contains(view, "File Detail") || !strings.Contains(view, "MIME: text/plain") || !strings.Contains(view, "Filename: README.md") || !strings.Contains(view, "hello docs") {
		t.Fatalf("file detail missing expected fields:\n%s", view)
	}
}

func TestTimelineHidesStepParts(t *testing.T) {
	repo := newFakeRepo(t)
	repo.timelines["ses_project"] = []index.TimelinePart{
		{PartID: "prt_text", SessionID: "ses_project", MessageID: "msg_user", Role: "user", Kind: opencode.PartKindText, Preview: "hello", IndexText: "hello"},
		{PartID: "prt_start", SessionID: "ses_project", MessageID: "msg_assistant", Role: "assistant", Kind: opencode.PartKindStepStart, Preview: "step start snapshot"},
		{PartID: "prt_finish", SessionID: "ses_project", MessageID: "msg_assistant", Role: "assistant", Kind: opencode.PartKindStepFinish, Preview: "step finish stop"},
	}

	model := NewModel(repo, repo.sessions)
	model = sendKey(t, model, "l")
	view := model.View()
	if strings.Contains(view, "step-start") || strings.Contains(view, "step-finish") || strings.Contains(view, "step start") || strings.Contains(view, "step finish") {
		t.Fatalf("step parts should not render:\n%s", view)
	}
}

func TestTimelinePreservesMessageLineBreaks(t *testing.T) {
	repo := newFakeRepo(t)
	repo.timelines["ses_project"] = []index.TimelinePart{
		{
			PartID:    "prt_text",
			SessionID: "ses_project",
			MessageID: "msg_user",
			Role:      "user",
			Kind:      opencode.PartKindText,
			Preview:   "first line second line",
			IndexText: "first line second line",
			RawJSON:   `{"type":"text","text":"first line\n\nsecond line\n  indented code"}`,
		},
	}

	model := NewModel(repo, repo.sessions)
	model = sendKey(t, model, "l")
	view := model.View()
	if !strings.Contains(view, "first line") || !strings.Contains(view, "second line") || !strings.Contains(view, "indented code") {
		t.Fatalf("formatted text missing expected lines:\n%s", view)
	}
	if strings.Contains(view, "first line second line") {
		t.Fatalf("message line breaks were collapsed:\n%s", view)
	}
}

func sendKey(t *testing.T, model Model, key string) Model {
	t.Helper()
	updated, _ := updateModel(t, model, keyMsg(key))
	return updated
}

func search(t *testing.T, model Model, query string) Model {
	t.Helper()
	model = sendKey(t, model, "/")
	for _, r := range query {
		model = sendKey(t, model, string(r))
	}
	return sendKey(t, model, "enter")
}

func updateModel(t *testing.T, model Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	updated, cmd := model.Update(msg)
	result, ok := updated.(Model)
	if !ok {
		t.Fatalf("updated model type = %T", updated)
	}
	return result, cmd
}

func keyMsg(key string) tea.KeyMsg {
	switch key {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
}

type fakeRepo struct {
	sessions           []index.SessionSummary
	timelines          map[string][]index.TimelinePart
	rawParts           map[string]index.RawPart
	lastSessionSearch  string
	lastTimelineSearch string
	timelineLoads      int
}

func newFakeRepo(t *testing.T) *fakeRepo {
	t.Helper()
	heavyPath := filepath.Join(t.TempDir(), "heavy.json")
	return &fakeRepo{
		sessions: []index.SessionSummary{
			{ID: "ses_project", ProjectID: "proj", ProjectPath: "/tmp/project", Title: "Project session", MessageCount: 2, PartCount: 3, HeavyPartCount: 1},
			{ID: "ses_global", ProjectID: "global", ProjectPath: "Global", Title: "Global session"},
		},
		timelines: map[string][]index.TimelinePart{
			"ses_project": {
				{PartID: "prt_text", SessionID: "ses_project", Role: "user", Kind: opencode.PartKindText, Preview: "open docs", IndexText: "open docs"},
				{PartID: "prt_reasoning", SessionID: "ses_project", Role: "assistant", Kind: opencode.PartKindReasoning, Preview: "secret reasoning", IndexText: "secret reasoning"},
				{PartID: "prt_heavy", SessionID: "ses_project", Role: "assistant", Kind: opencode.PartKindTool, ToolName: "apply_patch", Preview: "heavy payload", Heavy: true, SkippedRaw: true, SourcePath: heavyPath, SizeBytes: 1024 * 1024},
				{PartID: "prt_file", SessionID: "ses_project", Role: "assistant", Kind: opencode.PartKindFile, FilePath: "README.md", Preview: "README.md", IndexText: "README.md"},
			},
			"ses_global": {},
		},
		rawParts: map[string]index.RawPart{
			"prt_heavy": {PartID: "prt_heavy", SourcePath: heavyPath, SizeBytes: 1024 * 1024, Heavy: true, SkippedRaw: true, Preview: "AAECAwQFBgc"},
		},
	}
}

func (f *fakeRepo) ListSessions(context.Context) ([]index.SessionSummary, error) {
	return f.sessions, nil
}

func (f *fakeRepo) SearchSessions(_ context.Context, query string) ([]index.SessionSummary, error) {
	f.lastSessionSearch = query
	var results []index.SessionSummary
	for _, session := range f.sessions {
		if strings.Contains(strings.ToLower(session.Title+" "+session.ProjectPath), strings.ToLower(query)) {
			results = append(results, session)
		}
	}
	return results, nil
}

func (f *fakeRepo) SessionTimeline(_ context.Context, sessionID string) ([]index.TimelinePart, error) {
	f.timelineLoads++
	return f.timelines[sessionID], nil
}

func (f *fakeRepo) SearchSession(_ context.Context, sessionID, query string) ([]index.TimelinePart, error) {
	f.lastTimelineSearch = query
	var results []index.TimelinePart
	for _, part := range f.timelines[sessionID] {
		content := part.Preview + " " + part.IndexText + " " + part.FilePath
		if strings.Contains(strings.ToLower(content), strings.ToLower(query)) {
			results = append(results, part)
		}
	}
	return results, nil
}

func (f *fakeRepo) RawPart(_ context.Context, partID string) (index.RawPart, error) {
	return f.rawParts[partID], nil
}
