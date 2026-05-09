package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	termansi "github.com/charmbracelet/x/ansi"
	"github.com/quantick/opensession/internal/index"
	"github.com/quantick/opensession/internal/indexer"
	"github.com/quantick/opensession/internal/opencode"
)

func TestIndexingStatusTransitionsAndRefreshesSessions(t *testing.T) {
	repo := newFakeRepo(t)
	model := NewModelWithIndexEvents(repo, repo.sessions, nil, true)
	model, _ = updateModel(t, model, indexEventMsg{event: indexer.Event{Kind: indexer.EventStarted, Phase: "starting"}, ok: true})
	if !model.indexingActive || !strings.Contains(model.indexingStatus, "refreshing") {
		t.Fatalf("indexing did not start: active=%v status=%q", model.indexingActive, model.indexingStatus)
	}
	model, _ = updateModel(t, model, indexEventMsg{event: indexer.Event{Kind: indexer.EventPhase, Source: "opencode", Phase: "writing index", Current: 2, Total: 3}, ok: true})
	if !strings.Contains(model.indexingStatus, "opencode writing index 2/3") {
		t.Fatalf("phase status = %q", model.indexingStatus)
	}
	selectedID := model.selectedSessionID()
	refreshed := []index.SessionSummary{
		{ID: "ses_new", Title: "New", UpdatedAt: time.Now()},
		repo.sessions[0],
		repo.sessions[1],
	}
	model, _ = updateModel(t, model, indexEventMsg{event: indexer.Event{Kind: indexer.EventSessions, Sessions: refreshed}, ok: true})
	if len(model.allSessions) != 3 || len(model.sessions) != 3 {
		t.Fatalf("sessions not refreshed: visible=%d all=%d", len(model.sessions), len(model.allSessions))
	}
	if got := model.selectedSessionID(); got != selectedID {
		t.Fatalf("selected session = %q, want preserved %q", got, selectedID)
	}
	model, _ = updateModel(t, model, indexEventMsg{event: indexer.Event{Kind: indexer.EventComplete, Sessions: refreshed}, ok: true})
	if model.indexingActive || !model.indexingDone || !strings.Contains(model.indexingStatus, "up to date") {
		t.Fatalf("complete status active=%v done=%v status=%q", model.indexingActive, model.indexingDone, model.indexingStatus)
	}
}

func TestIndexingFailureAndEmptyCacheDisplay(t *testing.T) {
	repo := newFakeRepo(t)
	model := NewModelWithIndexEvents(repo, nil, nil, true)
	model, _ = updateModel(t, model, indexEventMsg{event: indexer.Event{Kind: indexer.EventStarted}, ok: true})
	plain := termansi.Strip(model.View())
	if !strings.Contains(plain, "No cached sessions yet") || !strings.Contains(plain, "Indexing is running") {
		t.Fatalf("empty indexing view missing status:\n%s", plain)
	}
	model, _ = updateModel(t, model, indexEventMsg{event: indexer.Event{Kind: indexer.EventFailed, Err: fmt.Errorf("boom")}, ok: true})
	plain = termansi.Strip(model.View())
	if !strings.Contains(plain, "Index: refresh failed: boom") {
		t.Fatalf("failure status missing:\n%s", plain)
	}
}

func TestNoScanIndexingStatus(t *testing.T) {
	repo := newFakeRepo(t)
	model := NewModelWithIndexingDisabled(repo, repo.sessions)
	plain := termansi.Strip(model.View())
	if !strings.Contains(plain, "Index: disabled (--no-scan)") {
		t.Fatalf("no-scan status missing:\n%s", plain)
	}
}

func TestIndexRefreshDoesNotResetTimelineView(t *testing.T) {
	repo := newFakeRepo(t)
	model := NewModelWithIndexEvents(repo, repo.sessions, nil, true)
	model = sendKey(t, model, "l")
	if model.mode != ViewTimeline {
		t.Fatalf("mode = %v, want timeline", model.mode)
	}
	model, _ = updateModel(t, model, indexEventMsg{event: indexer.Event{Kind: indexer.EventSessions, Sessions: append(repo.sessions, index.SessionSummary{ID: "ses_new", Title: "New"})}, ok: true})
	if model.mode != ViewTimeline || model.currentSession.ID == "" || len(model.timeline) == 0 {
		t.Fatalf("refresh reset timeline: mode=%v current=%q timeline=%d", model.mode, model.currentSession.ID, len(model.timeline))
	}
}

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

func TestAsyncSessionSearchStartsCommandAndIgnoresStaleResult(t *testing.T) {
	repo := newFakeRepo(t)
	model := NewModel(repo, repo.sessions)
	model, cmd1 := startSearchCommand(t, model, "project")
	if cmd1 == nil || !model.searchLoading || repo.lastSessionSearch != "" {
		t.Fatalf("session search should start loading without calling repository immediately: loading=%v last=%q cmdNil=%v", model.searchLoading, repo.lastSessionSearch, cmd1 == nil)
	}
	model = sendKey(t, model, "j")
	if model.selectedSession != 1 {
		t.Fatalf("input should remain responsive while search is loading, selected=%d", model.selectedSession)
	}
	model, cmd2 := startSearchCommand(t, model, "global")
	if cmd2 == nil || !model.searchLoading {
		t.Fatalf("second session search did not start loading")
	}
	stale := sessionSearchResultMsg{requestID: model.searchRequest - 1, query: "project", sessions: []index.SessionSummary{{ID: "stale", Title: "Stale"}}}
	updated, _ := updateModel(t, model, stale)
	model = updated
	if len(model.sessions) == 1 && model.sessions[0].ID == "stale" {
		t.Fatalf("stale session search result was applied")
	}
	if !model.searchLoading {
		t.Fatalf("stale result should not clear loading for newer request")
	}
	model = runCmd(t, model, cmd2)
	if repo.lastSessionSearch != "global" || len(model.sessions) != 1 || model.sessions[0].ID != "ses_global" || model.searchLoading {
		t.Fatalf("latest session search not applied: last=%q sessions=%#v loading=%v", repo.lastSessionSearch, model.sessions, model.searchLoading)
	}
	_ = cmd1
}

func TestAsyncTimelineSearchIgnoresStaleAndViewChangedResults(t *testing.T) {
	repo := newFakeRepo(t)
	model := NewModel(repo, repo.sessions)
	model = sendKey(t, model, "l")
	model, cmd1 := startSearchCommand(t, model, "README")
	if cmd1 == nil || !model.searchLoading || repo.lastTimelineSearch != "" {
		t.Fatalf("timeline search should start loading without immediate repository call")
	}
	model, cmd2 := startSearchCommand(t, model, "open docs")
	stale := timelineSearchResultMsg{requestID: model.searchRequest - 1, sessionID: model.currentSession.ID, query: "README", parts: []index.TimelinePart{{PartID: "stale"}}}
	updated, _ := updateModel(t, model, stale)
	model = updated
	if len(model.timeline) == 1 && model.timeline[0].PartID == "stale" {
		t.Fatalf("stale timeline result was applied")
	}
	model = runCmd(t, model, cmd2)
	if repo.lastTimelineSearch != "open docs" || len(model.timeline) != 1 || model.timeline[0].PartID != "prt_text" {
		t.Fatalf("latest timeline search not applied: last=%q timeline=%#v", repo.lastTimelineSearch, model.timeline)
	}

	model, cmd3 := startSearchCommand(t, model, "README")
	model = sendKey(t, model, "h")
	result := cmd3().(timelineSearchResultMsg)
	updated, _ = updateModel(t, model, result)
	model = updated
	if model.mode != ViewSessions || model.searchLoading {
		t.Fatalf("view-changed timeline result should be ignored: mode=%v loading=%v", model.mode, model.searchLoading)
	}
	_ = cmd1
}

func TestSessionTokenSortHotkeyOrdersByTokenTotal(t *testing.T) {
	base := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	sessions := []index.SessionSummary{
		{ID: "recent-low", ProjectPath: "/tmp/a", Title: "Recent low", UpdatedAt: base.Add(2 * time.Hour), TokenUsage: opencode.TokenUsage{Available: true, Total: 100_000}},
		{ID: "old-high", ProjectPath: "/tmp/b", Title: "Old high", UpdatedAt: base, TokenUsage: opencode.TokenUsage{Available: true, Total: 2_500_000}},
		{ID: "unavailable", ProjectPath: "/tmp/c", Title: "Unavailable", UpdatedAt: base.Add(time.Hour)},
	}
	model := NewModel(newFakeRepo(t), sessions)

	rows := model.sessionRows()
	if rows[0].session.ID != "recent-low" {
		t.Fatalf("default session sort should be recent first: %#v", sessionRowIDs(rows))
	}

	model = sendKey(t, model, "t")
	rows = model.sessionRows()
	if model.sessionSortMode != SessionSortTokens || rows[0].session.ID != "old-high" || rows[1].session.ID != "recent-low" || rows[2].session.ID != "unavailable" {
		t.Fatalf("token sort order = %#v, mode=%v", sessionRowIDs(rows), model.sessionSortMode)
	}
	plain := plainView(model.View())
	if !strings.Contains(plain, "tokens sort") || !strings.Contains(plain, "t recent sort") {
		t.Fatalf("token sort mode/help missing:\n%s", model.View())
	}

	model = sendKey(t, model, "t")
	rows = model.sessionRows()
	if model.sessionSortMode != SessionSortRecent || rows[0].session.ID != "recent-low" {
		t.Fatalf("recent sort order = %#v, mode=%v", sessionRowIDs(rows), model.sessionSortMode)
	}
}

func TestGroupedSessionTokenSortOrdersGroupsByLargestTokenTotal(t *testing.T) {
	base := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	sessions := []index.SessionSummary{
		{ID: "alpha-low", ProjectID: "alpha", ProjectPath: "/tmp/alpha", Title: "Alpha low", UpdatedAt: base.Add(3 * time.Hour), TokenUsage: opencode.TokenUsage{Available: true, Total: 100_000}},
		{ID: "beta-high", ProjectID: "beta", ProjectPath: "/tmp/beta", Title: "Beta high", UpdatedAt: base, TokenUsage: opencode.TokenUsage{Available: true, Total: 2_500_000}},
		{ID: "alpha-mid", ProjectID: "alpha", ProjectPath: "/tmp/alpha", Title: "Alpha mid", UpdatedAt: base.Add(2 * time.Hour), TokenUsage: opencode.TokenUsage{Available: true, Total: 500_000}},
	}
	model := NewModel(newFakeRepo(t), sessions)
	model = sendKey(t, model, "v")
	model = sendKey(t, model, "t")

	rows := model.sessionRows()
	if labels := sessionHeaderLabels(rows); !reflect.DeepEqual(labels, []string{"/tmp/beta", "/tmp/alpha"}) {
		t.Fatalf("token-sorted group headers = %#v", labels)
	}
	if got := groupSessionIDs(rows, "/tmp/alpha"); !reflect.DeepEqual(got, []string{"alpha-mid", "alpha-low"}) {
		t.Fatalf("token-sorted alpha sessions = %#v", got)
	}
}

func TestSessionListModeTogglePreservesSelectedSession(t *testing.T) {
	repo := newFakeRepo(t)
	repo.sessions = sessionListModeTestSessions()
	model := NewModel(repo, repo.sessions)
	model.selectedSession = 3
	selectedID := model.selectedSessionID()

	model = sendKey(t, model, "v")
	if model.sessionListMode != SessionListGrouped {
		t.Fatalf("session list mode = %v, want grouped", model.sessionListMode)
	}
	if model.selectedSessionID() != selectedID {
		t.Fatalf("selected session after grouped toggle = %q, want %q", model.selectedSessionID(), selectedID)
	}
	if row, ok := model.selectedSessionRow(model.sessionRows()); !ok || row.session.ID != selectedID {
		t.Fatalf("selected grouped row = %#v, ok = %v, want session %q", row, ok, selectedID)
	}

	model = sendKey(t, model, "v")
	if model.sessionListMode != SessionListFlat {
		t.Fatalf("session list mode = %v, want flat", model.sessionListMode)
	}
	if model.selectedSessionID() != selectedID {
		t.Fatalf("selected session after flat toggle = %q, want %q", model.selectedSessionID(), selectedID)
	}
}

func TestMixedSourceRowsRenderBadgesAndGroupByProjectPath(t *testing.T) {
	base := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	sessions := []index.SessionSummary{
		{SourceKind: "opencode", ID: "oc", ProjectID: "oc-proj", ProjectPath: "/tmp/shared", Title: "OpenCode", UpdatedAt: base.Add(time.Hour)},
		{SourceKind: "pi", ID: "pi:s", ProjectID: "pi-project", ProjectPath: "/tmp/shared", Title: "Pi", UpdatedAt: base.Add(2 * time.Hour)},
	}
	model := NewModel(newFakeRepo(t), sessions)
	view := plainView(model.View())
	if !strings.Contains(view, "[pi] Pi") || !strings.Contains(view, "[opencode] OpenCode") || !strings.Contains(view, "Source: opencode") {
		t.Fatalf("source badges missing:\n%s", view)
	}
	model = sendKey(t, model, "v")
	rows := model.sessionRows()
	if got := sessionHeaderLabels(rows); !reflect.DeepEqual(got, []string{"/tmp/shared"}) {
		t.Fatalf("mixed source headers = %#v", got)
	}
	if got := groupSessionIDs(rows, "/tmp/shared"); !reflect.DeepEqual(got, []string{"pi:s", "oc"}) {
		t.Fatalf("mixed source group ids = %#v", got)
	}
}

func TestGroupedSessionRowsOrderByVisibleActivity(t *testing.T) {
	rows := groupedSessionRows(sessionListModeTestSessions(), SessionSortRecent)
	wantHeaders := []string{"/tmp/beta", "Global sessions", "/tmp/alpha"}
	if got := sessionHeaderLabels(rows); !reflect.DeepEqual(got, wantHeaders) {
		t.Fatalf("header labels = %#v, want %#v", got, wantHeaders)
	}

	wantAlphaSessions := []string{"alpha-recent", "alpha-match-old"}
	if got := groupSessionIDs(rows, "/tmp/alpha"); !reflect.DeepEqual(got, wantAlphaSessions) {
		t.Fatalf("alpha sessions = %#v, want %#v", got, wantAlphaSessions)
	}
}

func TestGroupedSearchResultsStayGroupedByMatchingActivity(t *testing.T) {
	repo := newFakeRepo(t)
	repo.sessions = sessionListModeTestSessions()
	model := NewModel(repo, repo.sessions)
	model.selectedSession = 1
	selectedID := model.selectedSessionID()
	model = sendKey(t, model, "v")
	model = search(t, model, "match")

	if model.sessionListMode != SessionListGrouped {
		t.Fatalf("session list mode = %v, want grouped", model.sessionListMode)
	}
	if model.selectedSessionID() != selectedID {
		t.Fatalf("selected session after grouped search = %q, want %q", model.selectedSessionID(), selectedID)
	}
	wantHeaders := []string{"/tmp/beta", "Global sessions", "/tmp/alpha"}
	if got := sessionHeaderLabels(model.sessionRows()); !reflect.DeepEqual(got, wantHeaders) {
		t.Fatalf("search header labels = %#v, want %#v", got, wantHeaders)
	}
	if got := groupSessionIDs(model.sessionRows(), "/tmp/alpha"); !reflect.DeepEqual(got, []string{"alpha-match-old"}) {
		t.Fatalf("alpha search sessions = %#v, want only matching alpha session", got)
	}
	if strings.Contains(model.View(), "Alpha Recent") {
		t.Fatalf("grouped search should not render unmatched sessions:\n%s", model.View())
	}

	model = search(t, model, "")
	if model.selectedSessionID() != selectedID {
		t.Fatalf("selected session after clearing search = %q, want %q", model.selectedSessionID(), selectedID)
	}
}

func TestGroupedSessionNavigationSkipsHeadersAndKeepsSelectionVisible(t *testing.T) {
	repo := newFakeRepo(t)
	repo.sessions = sessionListModeTestSessions()
	model := NewModel(repo, repo.sessions)
	model, _ = updateModel(t, model, tea.WindowSizeMsg{Width: 80, Height: 5})
	model = sendKey(t, model, "v")

	if model.selectedSessionID() != "beta-match" {
		t.Fatalf("initial selected session = %q, want beta-match", model.selectedSessionID())
	}
	model = sendKey(t, model, "j")
	if model.selectedSessionID() != "global-match" {
		t.Fatalf("selected session after j = %q, want global-match", model.selectedSessionID())
	}
	requireSelectedSessionVisible(t, model)

	model = sendKey(t, model, "j")
	if model.selectedSessionID() != "alpha-recent" {
		t.Fatalf("selected session after second j = %q, want alpha-recent", model.selectedSessionID())
	}
	requireSelectedSessionVisible(t, model)
}

func TestModelFiltersChildSessionsFromFlatAndGroupedLists(t *testing.T) {
	repo := newFakeRepo(t)
	repo.sessions = append(repo.sessions, index.SessionSummary{ID: "ses_child", ParentID: "ses_project", ProjectID: "proj", ProjectPath: "/tmp/project", Title: "Child session"})
	model := NewModel(repo, repo.sessions)

	if containsSessionID(model.sessions, "ses_child") || strings.Contains(plainView(model.View()), "Child session") {
		t.Fatalf("flat session list should hide child sessions:\n%s", model.View())
	}
	model = sendKey(t, model, "v")
	if containsSessionID(model.sessions, "ses_child") || strings.Contains(plainView(model.View()), "Child session") {
		t.Fatalf("grouped session list should hide child sessions:\n%s", model.View())
	}
	model = search(t, model, "child")
	if containsSessionID(model.sessions, "ses_child") || strings.Contains(plainView(model.View()), "Child session") {
		t.Fatalf("session search should hide child sessions:\n%s", model.View())
	}
}

func TestSessionListKeepsTokenUsageVisibleWithLongTitle(t *testing.T) {
	repo := newFakeRepo(t)
	repo.sessions[0].Title = strings.Repeat("very long session title ", 8)
	model := NewModel(repo, repo.sessions)
	model, _ = updateModel(t, model, tea.WindowSizeMsg{Width: 72, Height: 12})

	plain := plainView(model.View())
	if !strings.Contains(plain, "0.00M") {
		t.Fatalf("token usage should stay visible when title is long:\n%s", model.View())
	}
	if strings.Contains(plain, "321 tok") || strings.Contains(plain, "2m") || strings.Contains(plain, "3p") {
		t.Fatalf("session list token badge should only show token total in millions without suffix:\n%s", model.View())
	}
}

func TestModelRendersSessionTokenUsage(t *testing.T) {
	repo := newFakeRepo(t)
	model := NewModel(repo, repo.sessions)

	view := model.View()
	if !strings.Contains(view, "0.00M") || strings.Contains(plainView(view), "0.00M tok") || !strings.Contains(view, "Tokens: total 0.00M") || !strings.Contains(view, "cache read 0.00M") {
		t.Fatalf("session token usage missing from list/preview:\n%s", view)
	}
	if strings.Contains(strings.ToLower(view), "cost") {
		t.Fatalf("cost should not render with token usage:\n%s", view)
	}

	model = sendKey(t, model, "l")
	view = model.View()
	if !strings.Contains(view, "Tokens: 0.00M") || strings.Contains(plainView(view), "0.00M tok") {
		t.Fatalf("session token usage missing from detail header:\n%s", view)
	}
	if strings.Contains(strings.ToLower(view), "cost") {
		t.Fatalf("cost should not render in detail header:\n%s", view)
	}

	model = sendKey(t, model, "h")
	model = sendKey(t, model, "j")
	view = model.View()
	if !strings.Contains(view, "Tokens: unavailable") {
		t.Fatalf("unavailable token usage missing from preview:\n%s", view)
	}
	if strings.Contains(view, "0 tok") {
		t.Fatalf("unavailable token usage should not render zero total:\n%s", view)
	}
}

func TestPiTreeNavigationSwitchesBranchAndBacksOut(t *testing.T) {
	repo := newFakeRepo(t)
	repo.sessions = []index.SessionSummary{{SourceKind: "pi", ID: "pi:s", ProjectID: "pi-proj", ProjectPath: "/tmp/pi", Title: "Pi session"}}
	repo.timelines["pi:s"] = []index.TimelinePart{{SourceKind: "pi", PartID: "prt_latest", MessageID: "pi:s:b", SessionID: "pi:s", Role: "user", Kind: opencode.PartKindText, Preview: "branch B latest"}}
	repo.trees["pi:s"] = []index.SessionTreeEntry{
		{SourceKind: "pi", ID: "pi:s:u", SessionID: "pi:s", Role: "user", AppendOrder: 1},
		{SourceKind: "pi", ID: "pi:s:a", SessionID: "pi:s", ParentID: "pi:s:u", Role: "assistant", AppendOrder: 2},
		{SourceKind: "pi", ID: "pi:s:branch-a", SessionID: "pi:s", ParentID: "pi:s:a", Role: "user", Label: "alt", AppendOrder: 3},
		{SourceKind: "pi", ID: "pi:s:branch-b", SessionID: "pi:s", ParentID: "pi:s:a", Role: "user", Label: "chosen", AppendOrder: 4},
	}
	repo.branchTimelines["pi:s:branch-a"] = []index.TimelinePart{{SourceKind: "pi", PartID: "prt_a", MessageID: "pi:s:branch-a", SessionID: "pi:s", Role: "user", Kind: opencode.PartKindText, Preview: "branch A selected"}}

	model := NewModel(repo, repo.sessions)
	model = sendKey(t, model, "l")
	if model.mode != ViewTimeline || model.activeEntryID != "pi:s:branch-b" {
		t.Fatalf("Pi timeline did not default to latest branch:\n%s", model.View())
	}
	model = sendKey(t, model, "b")
	if model.mode != ViewSessionTree || !strings.Contains(plainView(model.View()), "chosen") {
		t.Fatalf("tree view missing label:\n%s", model.View())
	}
	model = sendKey(t, model, "k")
	model = sendKey(t, model, "enter")
	if model.mode != ViewTimeline || model.activeEntryID != "pi:s:branch-a" || !strings.Contains(plainView(model.View()), "branch A selected") {
		t.Fatalf("branch switch failed: active=%q\n%s", model.activeEntryID, model.View())
	}
	model = sendKey(t, model, "b")
	model = sendKey(t, model, "h")
	if model.mode != ViewTimeline || model.activeEntryID != "pi:s:branch-a" {
		t.Fatalf("tree back changed context: mode=%v active=%q", model.mode, model.activeEntryID)
	}
}

func TestPiToolDetailUsesStoredRawJSONDespiteLargeSourceFile(t *testing.T) {
	repo := newFakeRepo(t)
	repo.sessions = []index.SessionSummary{{SourceKind: "pi", ID: "pi:tool", ProjectPath: "/tmp/pi", Title: "Pi tools"}}
	repo.timelines["pi:tool"] = []index.TimelinePart{{SourceKind: "pi", PartID: "pi_tool_small", SessionID: "pi:tool", MessageID: "pi:m1", Role: "assistant", Kind: opencode.PartKindTool, ToolName: "bash", Preview: "go test", RawJSON: `{"type":"toolCall","name":"bash","arguments":{"command":"go test ./..."}}`, SizeBytes: MaxRawDisplayBytes + 1}}
	repo.rawParts["pi_tool_small"] = index.RawPart{SourceKind: "pi", PartID: "pi_tool_small", SessionID: "pi:tool", MessageID: "pi:m1", Role: "assistant", Kind: opencode.PartKindTool, ToolName: "bash", Preview: "go test", RawJSON: `{"type":"toolCall","name":"bash","arguments":{"command":"go test ./..."}}`, SizeBytes: MaxRawDisplayBytes + 1}

	model := NewModel(repo, repo.sessions)
	model = sendKey(t, model, "l")
	model = sendKey(t, model, "enter")
	view := plainView(model.View())
	if strings.Contains(view, "too large or unsafe") {
		t.Fatalf("stored Pi tool raw JSON should not be blocked by source file size:\n%s", model.View())
	}
	if !strings.Contains(view, "Tool Detail") && !strings.Contains(view, "go test") {
		t.Fatalf("Pi tool detail missing:\n%s", model.View())
	}
}

func TestPiMessageDetailRawAndReasoningGuardrails(t *testing.T) {
	repo := newFakeRepo(t)
	repo.sessions = []index.SessionSummary{{SourceKind: "pi", ID: "pi:detail", ProjectPath: "/tmp/pi", Title: "Pi detail"}}
	repo.timelines["pi:detail"] = []index.TimelinePart{
		{SourceKind: "pi", PartID: "pi_text", SessionID: "pi:detail", MessageID: "pi:m1", Role: "assistant", Kind: opencode.PartKindText, Preview: "# Pi detail", IndexText: "# Pi detail", RawJSON: `{"type":"text","text":"# Pi detail\n\nBody"}`},
		{SourceKind: "pi", PartID: "pi_reasoning", SessionID: "pi:detail", MessageID: "pi:m2", Role: "assistant", Kind: opencode.PartKindReasoning, Preview: "hidden pi reasoning", IndexText: "hidden pi reasoning", RawJSON: `{"type":"thinking","text":"hidden pi reasoning"}`},
		{SourceKind: "pi", PartID: "pi_tool", SessionID: "pi:detail", MessageID: "pi:m3", Role: "assistant", Kind: opencode.PartKindTool, ToolName: "bash", Preview: "large", Heavy: true, SkippedRaw: true},
	}
	repo.rawParts["pi_tool"] = index.RawPart{SourceKind: "pi", PartID: "pi_tool", Role: "assistant", Kind: opencode.PartKindTool, ToolName: "bash", Preview: "large", Heavy: true, SkippedRaw: true, SizeBytes: 1024 * 1024}
	model := NewModel(repo, repo.sessions)
	model = sendKey(t, model, "l")
	if strings.Contains(plainView(model.View()), "hidden pi reasoning") {
		t.Fatalf("Pi reasoning should be hidden by default:\n%s", model.View())
	}
	model = sendKey(t, model, "enter")
	if model.mode != ViewRawPart || !model.messageDetail.active || !strings.Contains(plainView(model.View()), "Pi detail") {
		t.Fatalf("Pi text detail failed:\n%s", model.View())
	}
	model = sendKey(t, model, "h")
	model = sendKey(t, model, "r")
	if !strings.Contains(plainView(model.View()), "hidden pi reasoning") {
		t.Fatalf("Pi reasoning toggle failed:\n%s", model.View())
	}
	model = sendKey(t, model, "j")
	model = sendKey(t, model, "j")
	model = sendKey(t, model, "enter")
	if !strings.Contains(plainView(model.View()), "too large or unsafe") {
		t.Fatalf("Pi heavy guard missing:\n%s", model.View())
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

func TestTimelineRepaintNavigationAndTogglesUseCachedRendering(t *testing.T) {
	repo := newFakeRepo(t)
	var parts []index.TimelinePart
	for i := 0; i < 8; i++ {
		text := fmt.Sprintf("# Item %02d\n\nUse `cached rendering` for part %02d.", i, i)
		parts = append(parts, index.TimelinePart{
			PartID:    fmt.Sprintf("prt_cache_%02d", i),
			SessionID: "ses_project",
			MessageID: fmt.Sprintf("msg_cache_%02d", i),
			Role:      "assistant",
			Kind:      opencode.PartKindText,
			Preview:   text,
			IndexText: text,
			RawJSON:   fmt.Sprintf(`{"type":"text","text":%q}`, text),
		})
	}
	repo.timelines["ses_project"] = parts

	var rawDecodes int
	var markdownRenders int
	partTextFromRawJSONHook = func() { rawDecodes++ }
	assistantMarkdownRowsHook = func() { markdownRenders++ }
	t.Cleanup(func() {
		partTextFromRawJSONHook = nil
		assistantMarkdownRowsHook = nil
	})

	model := NewModel(repo, repo.sessions)
	model, _ = updateModel(t, model, tea.WindowSizeMsg{Width: 80, Height: 9})
	model = sendKey(t, model, "l")
	if rawDecodes != len(parts) {
		t.Fatalf("raw text should be decoded once on load, got %d want %d", rawDecodes, len(parts))
	}
	_ = model.View()
	initialMarkdownRenders := markdownRenders
	if initialMarkdownRenders != len(parts) {
		t.Fatalf("initial markdown render count = %d, want %d", initialMarkdownRenders, len(parts))
	}
	_ = model.View()
	model = sendKey(t, model, "j")
	_ = model.View()
	if rawDecodes != len(parts) || markdownRenders != initialMarkdownRenders {
		t.Fatalf("repaint/navigation rebuilt cached content: raw=%d markdown=%d", rawDecodes, markdownRenders)
	}
	model = sendKey(t, model, "r")
	_ = model.View()
	model = sendKey(t, model, "m")
	_ = model.View()
	model = sendKey(t, model, "m")
	_ = model.View()
	if rawDecodes != len(parts) || markdownRenders != initialMarkdownRenders {
		t.Fatalf("reasoning/markdown toggles should reuse text and same-width markdown cache: raw=%d markdown=%d", rawDecodes, markdownRenders)
	}
	model, _ = updateModel(t, model, tea.WindowSizeMsg{Width: 90, Height: 9})
	_ = model.View()
	if rawDecodes != len(parts) || markdownRenders != initialMarkdownRenders+len(parts) {
		t.Fatalf("resize should reuse raw text and render markdown once for new width: raw=%d markdown=%d", rawDecodes, markdownRenders)
	}
	_ = model.View()
	if markdownRenders != initialMarkdownRenders+len(parts) {
		t.Fatalf("second repaint after resize rerendered markdown: %d", markdownRenders)
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

func TestFocusedMultilineTimelineUsesRailContinuation(t *testing.T) {
	repo := newFakeRepo(t)
	text := "first focused line\nsecond focused line\nthird focused line"
	repo.timelines["ses_project"] = []index.TimelinePart{
		{PartID: "prt_multiline", SessionID: "ses_project", MessageID: "msg_user", Role: "user", Kind: opencode.PartKindText, Preview: text, IndexText: text, RawJSON: fmt.Sprintf(`{"type":"text","text":%q}`, text)},
	}
	model := NewModel(repo, repo.sessions)
	model, _ = updateModel(t, model, tea.WindowSizeMsg{Width: 80, Height: 12})
	model = sendKey(t, model, "l")

	plain := plainView(model.View())
	if !strings.Contains(plain, "▌ first focused line") || !strings.Contains(plain, "│ second focused line") || !strings.Contains(plain, "│ third focused line") {
		t.Fatalf("focused multiline message missing rail continuation:\n%s", model.View())
	}
	if strings.Contains(plain, "> first focused line") {
		t.Fatalf("focused multiline message still uses prompt marker:\n%s", model.View())
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
	plain := plainView(model.View())
	if !strings.Contains(plain, "▌ $ bash") || strings.Contains(plain, "> $ bash") || !strings.Contains(plain, "✓") || strings.Contains(plain, "✓ completed") {
		t.Fatalf("focused tool row missing compact rail/status affordance:\n%s", model.View())
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

func TestLinkedTaskOpensChildTimelineAndBackRestoresParent(t *testing.T) {
	repo := newFakeRepo(t)
	repo.sessions = append(repo.sessions, index.SessionSummary{ID: "ses_child", ParentID: "ses_project", ProjectID: "proj", ProjectPath: "/tmp/project", Title: "Child subagent"})
	repo.timelines["ses_project"] = []index.TimelinePart{
		{PartID: "prt_text", SessionID: "ses_project", MessageID: "msg_user", Role: "user", Kind: opencode.PartKindText, Preview: "run a subagent", IndexText: "run a subagent"},
		{PartID: "prt_task", SessionID: "ses_project", MessageID: "msg_assistant", Role: "assistant", Kind: opencode.PartKindTool, ToolName: "task", Status: "completed", Title: "Research dependency", SubagentName: "explore", Preview: "subagent finished", LinkedSessionID: "ses_child"},
	}
	repo.timelines["ses_child"] = []index.TimelinePart{
		{PartID: "prt_child", SessionID: "ses_child", MessageID: "msg_child", Role: "assistant", Kind: opencode.PartKindText, Preview: "child transcript", IndexText: "child transcript"},
	}

	model := NewModel(repo, repo.sessions)
	model = sendKey(t, model, "l")
	model = sendKey(t, model, "j")
	if plain := plainView(model.View()); !strings.Contains(plain, "▌ ↪ subagent:explore Research dependency") || strings.Contains(plain, "> ↪ subagent:explore") {
		t.Fatalf("linked task row missing compact rail/subagent affordance:\n%s", model.View())
	}
	model = sendKey(t, model, "enter")
	view := plainView(model.View())
	if model.mode != ViewTimeline || model.currentSession.ID != "ses_child" || !strings.Contains(view, "child transcript") {
		t.Fatalf("linked task did not open child timeline: mode=%v current=%q\n%s", model.mode, model.currentSession.ID, model.View())
	}
	if !strings.Contains(view, "Nested under: Project session via Research dependency") {
		t.Fatalf("child timeline missing nested context:\n%s", model.View())
	}

	model = sendKey(t, model, "h")
	if model.mode != ViewTimeline || model.currentSession.ID != "ses_project" || model.selectedPart != 1 {
		t.Fatalf("back did not restore parent task context: mode=%v current=%q selected=%d", model.mode, model.currentSession.ID, model.selectedPart)
	}
	if !strings.Contains(plainView(model.View()), "Research dependency") {
		t.Fatalf("restored parent task row not visible:\n%s", model.View())
	}
}

func TestUnlinkedTaskAndOrdinaryToolStillOpenDetails(t *testing.T) {
	repo := newFakeRepo(t)
	taskRaw := `{"type":"tool","tool":"task","state":{"status":"completed","title":"No child"}}`
	toolRaw := `{"type":"tool","tool":"bash","state":{"status":"completed","input":{"command":"go test ./internal/tui"}}}`
	repo.timelines["ses_project"] = []index.TimelinePart{
		{PartID: "prt_text", SessionID: "ses_project", MessageID: "msg_user", Role: "user", Kind: opencode.PartKindText, Preview: "run tools", IndexText: "run tools"},
		{PartID: "prt_task_unlinked", SessionID: "ses_project", MessageID: "msg_assistant", Role: "assistant", Kind: opencode.PartKindTool, ToolName: "task", Status: "completed", Title: "No child", RawJSON: taskRaw, SizeBytes: int64(len(taskRaw))},
		{PartID: "prt_bash", SessionID: "ses_project", MessageID: "msg_assistant", Role: "assistant", Kind: opencode.PartKindTool, ToolName: "bash", Status: "completed", Preview: "go test", RawJSON: toolRaw, SizeBytes: int64(len(toolRaw))},
	}
	repo.rawParts["prt_task_unlinked"] = index.RawPart{PartID: "prt_task_unlinked", Kind: opencode.PartKindTool, ToolName: "task", Status: "completed", Title: "No child", RawJSON: taskRaw, SizeBytes: int64(len(taskRaw))}
	repo.rawParts["prt_bash"] = index.RawPart{PartID: "prt_bash", Kind: opencode.PartKindTool, ToolName: "bash", Status: "completed", RawJSON: toolRaw, SizeBytes: int64(len(toolRaw))}

	model := NewModel(repo, repo.sessions)
	model = sendKey(t, model, "l")
	model = sendKey(t, model, "j")
	model = sendKey(t, model, "enter")
	if model.mode != ViewRawPart || model.rawPart.PartID != "prt_task_unlinked" {
		t.Fatalf("unlinked task should open detail, got mode=%v raw=%q", model.mode, model.rawPart.PartID)
	}
	model = sendKey(t, model, "h")
	model = sendKey(t, model, "j")
	model = sendKey(t, model, "enter")
	if model.mode != ViewRawPart || model.rawPart.PartID != "prt_bash" {
		t.Fatalf("ordinary tool should open detail, got mode=%v raw=%q", model.mode, model.rawPart.PartID)
	}
}

func TestChildTimelineSearchAndBrowsingStayBounded(t *testing.T) {
	repo := newFakeRepo(t)
	repo.sessions = append(repo.sessions, index.SessionSummary{ID: "ses_child", ParentID: "ses_project", ProjectID: "proj", ProjectPath: "/tmp/project", Title: "Child subagent"})
	repo.timelines["ses_project"] = []index.TimelinePart{
		{PartID: "prt_task", SessionID: "ses_project", MessageID: "msg_assistant", Role: "assistant", Kind: opencode.PartKindTool, ToolName: "task", Status: "completed", Title: "Open child", SubagentName: "explore", LinkedSessionID: "ses_child"},
	}
	repo.timelines["ses_child"] = nil
	for i := 0; i < 30; i++ {
		repo.timelines["ses_child"] = append(repo.timelines["ses_child"], index.TimelinePart{
			PartID:    fmt.Sprintf("prt_child_%02d", i),
			SessionID: "ses_child",
			MessageID: fmt.Sprintf("msg_child_%02d", i),
			Role:      "assistant",
			Kind:      opencode.PartKindText,
			Preview:   fmt.Sprintf("child item %02d", i),
			IndexText: fmt.Sprintf("child item %02d", i),
		})
	}

	model := NewModel(repo, repo.sessions)
	model, _ = updateModel(t, model, tea.WindowSizeMsg{Width: 80, Height: 9})
	model = sendKey(t, model, "l")
	model = sendKey(t, model, "enter")
	if strings.Contains(plainView(model.View()), "child item 20") {
		t.Fatalf("child timeline render should be bounded:\n%s", model.View())
	}
	for i := 0; i < 8; i++ {
		model = sendKey(t, model, "j")
	}
	plain := plainView(model.View())
	if model.timelineScroll == 0 || strings.Contains(plain, "child item 00") || !strings.Contains(plain, "child item 08") {
		t.Fatalf("child timeline browsing did not stay bounded and scrollable:\n%s", model.View())
	}
	model = search(t, model, "child item 05")
	if repo.lastTimelineSearchSession != "ses_child" || len(model.timeline) != 1 || model.timeline[0].PartID != "prt_child_05" {
		t.Fatalf("child timeline search = session:%q timeline:%#v", repo.lastTimelineSearchSession, model.timeline)
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
	structuredOutputRaw := `{"type":"tool","tool":"read","state":{"status":"completed","input":{"path":"README.md"},"output":[{"type":"text","text":"first line"},{"type":"text","text":"second line"}]}}`
	longOutputTail := "long-read-output-tail"
	longOutput := strings.Repeat("long detail output line\n", 300) + longOutputTail
	longLineOutputTail := "long-line-output-tail"
	longLineOutput := strings.Repeat("0123456789", 80) + longLineOutputTail
	longOutputRaw := fmt.Sprintf(`{"type":"tool","tool":"read","state":{"status":"completed","input":{"path":"LONG.md"},"output":[{"type":"text","text":%q}]}}`, longOutput)
	longLineOutputRaw := fmt.Sprintf(`{"type":"tool","tool":"read","state":{"status":"completed","input":{"path":"LONG-LINE.md"},"output":[{"type":"text","text":%q}]}}`, longLineOutput)
	patchRaw := `{"type":"patch","title":"Update README","path":"README.md","diff":"@@ -1 +1\n-old\n+new"}`
	fileRaw := `{"type":"file","mime":"text/plain","filename":"README.md","source":{"type":"file","path":"README.md","text":{"value":"hello docs","start":1,"end":2}}}`
	repo.rawParts["prt_unknown"] = index.RawPart{PartID: "prt_unknown", Kind: opencode.PartKindTool, ToolName: "custom_lookup", Status: "failed", RawJSON: unknownRaw, SizeBytes: int64(len(unknownRaw))}
	repo.rawParts["prt_structured_output"] = index.RawPart{PartID: "prt_structured_output", Kind: opencode.PartKindTool, ToolName: "read", Status: "completed", RawJSON: structuredOutputRaw, SizeBytes: int64(len(structuredOutputRaw))}
	repo.rawParts["prt_long_read_output"] = index.RawPart{PartID: "prt_long_read_output", Kind: opencode.PartKindTool, ToolName: "read", Status: "completed", RawJSON: longOutputRaw, SizeBytes: int64(len(longOutputRaw))}
	repo.rawParts["prt_long_line_read_output"] = index.RawPart{PartID: "prt_long_line_read_output", Kind: opencode.PartKindTool, ToolName: "read", Status: "completed", RawJSON: longLineOutputRaw, SizeBytes: int64(len(longLineOutputRaw))}
	repo.rawParts["prt_patch"] = index.RawPart{PartID: "prt_patch", Kind: opencode.PartKindPatch, Title: "Update README", RawJSON: patchRaw, SizeBytes: int64(len(patchRaw))}
	repo.rawParts["prt_safe_file"] = index.RawPart{PartID: "prt_safe_file", Kind: opencode.PartKindFile, FilePath: "README.md", RawJSON: fileRaw, SizeBytes: int64(len(fileRaw))}

	model := NewModel(repo, repo.sessions).openRawPart("prt_unknown")
	view := model.View()
	if !strings.Contains(view, "Tool Detail") || !strings.Contains(view, "custom_lookup") || !strings.Contains(view, "query: needle") || !strings.Contains(view, "error") {
		t.Fatalf("generic tool detail missing expected fields:\n%s", view)
	}

	model = model.openRawPart("prt_structured_output")
	view = model.View()
	if !strings.Contains(view, "File Tool Detail") || !strings.Contains(view, "first line") || !strings.Contains(view, "second line") || strings.Contains(view, `\"type\": \"text\"`) {
		t.Fatalf("structured tool output was not rendered as readable text:\n%s", view)
	}
	model = sendKey(t, model, "R")
	if !strings.Contains(model.View(), `"type": "text"`) {
		t.Fatalf("raw toggle should still show stored structured output JSON:\n%s", model.View())
	}

	model = model.openRawPart("prt_long_read_output")
	model, _ = updateModel(t, model, tea.WindowSizeMsg{Width: 80, Height: 12})
	model = sendKey(t, model, "G")
	if !strings.Contains(plainView(model.View()), longOutputTail) {
		t.Fatalf("long read output detail should scroll to tail:\n%s", model.View())
	}
	model = model.openRawPart("prt_long_line_read_output")
	model, _ = updateModel(t, model, tea.WindowSizeMsg{Width: 80, Height: 12})
	model = sendKey(t, model, "G")
	if !strings.Contains(plainView(model.View()), longLineOutputTail) {
		t.Fatalf("long single-line read output detail should wrap and scroll to tail:\n%s", model.View())
	}

	model = model.openRawPart("prt_patch")
	model, _ = updateModel(t, model, tea.WindowSizeMsg{Width: 100, Height: 28})
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

func TestTimelineHidesLowSignalReadLifecycleParts(t *testing.T) {
	repo := newFakeRepo(t)
	repo.timelines["ses_project"] = []index.TimelinePart{
		{PartID: "prt_text", SessionID: "ses_project", MessageID: "msg_user", Role: "user", Kind: opencode.PartKindText, Preview: "hello", IndexText: "hello"},
		{PartID: "prt_read_started", SessionID: "ses_project", MessageID: "msg_assistant", Role: "assistant", Kind: opencode.PartKindTool, ToolName: "read", Status: "started", Preview: "read - started"},
		{PartID: "prt_read_completed", SessionID: "ses_project", MessageID: "msg_assistant", Role: "assistant", Kind: opencode.PartKindTool, ToolName: "read", Status: "completed", Preview: "read - completed"},
		{PartID: "prt_bash_completed", SessionID: "ses_project", MessageID: "msg_assistant", Role: "assistant", Kind: opencode.PartKindTool, ToolName: "bash", Status: "completed", Preview: "go test ./..."},
	}

	model := NewModel(repo, repo.sessions)
	model = sendKey(t, model, "l")
	plain := plainView(model.View())
	if strings.Contains(plain, "read - started") || strings.Contains(plain, "read - completed") || strings.Contains(plain, "[tool] read") || strings.Contains(plain, "◧ read") {
		t.Fatalf("low-signal read lifecycle parts should not render:\n%s", model.View())
	}
	if !strings.Contains(plain, "$ bash") || !strings.Contains(plain, "go test ./...") {
		t.Fatalf("useful tool row should still render:\n%s", model.View())
	}
}

func TestTimelineToolRowsUseCompactGlyphsAndStatusSymbols(t *testing.T) {
	repo := newFakeRepo(t)
	repo.timelines["ses_project"] = []index.TimelinePart{
		{PartID: "prt_text", SessionID: "ses_project", MessageID: "msg_user", Role: "user", Kind: opencode.PartKindText, Preview: "run tools", IndexText: "run tools"},
		{PartID: "prt_bash", SessionID: "ses_project", MessageID: "msg_assistant", Role: "assistant", Kind: opencode.PartKindTool, ToolName: "bash", Status: "completed", Preview: "go test ./..."},
		{PartID: "prt_grep", SessionID: "ses_project", MessageID: "msg_assistant", Role: "assistant", Kind: opencode.PartKindTool, ToolName: "grep", Status: "failed", Preview: "needle"},
		{PartID: "prt_edit", SessionID: "ses_project", MessageID: "msg_assistant", Role: "assistant", Kind: opencode.PartKindTool, ToolName: "edit", Status: "started", FilePath: "internal/tui/model.go"},
		{PartID: "prt_custom", SessionID: "ses_project", MessageID: "msg_assistant", Role: "assistant", Kind: opencode.PartKindTool, ToolName: "custom_lookup", Status: "mystery", Preview: "query users", Heavy: true, SkippedRaw: true},
		{PartID: "prt_patch", SessionID: "ses_project", MessageID: "msg_assistant", Role: "assistant", Kind: opencode.PartKindPatch, Title: "Update README", FilePath: "README.md"},
		{PartID: "prt_file", SessionID: "ses_project", MessageID: "msg_assistant", Role: "assistant", Kind: opencode.PartKindFile, FilePath: "README.md", Preview: "docs"},
	}

	model := NewModel(repo, repo.sessions)
	model, _ = updateModel(t, model, tea.WindowSizeMsg{Width: 100, Height: 24})
	model = sendKey(t, model, "l")
	plain := plainView(model.View())

	for _, want := range []string{
		"$ bash - ✓ - go test ./...",
		"⌕ grep - ✗ - needle",
		"✎ edit - … - internal/tui/model.go",
		"◆ custom_lookup - ? - query users - heavy",
		"✎ patch - Update README - README.md",
		"◧ file - README.md - docs",
	} {
		if !strings.Contains(plain, want) {
			t.Fatalf("compact timeline row %q missing:\n%s", want, model.View())
		}
	}
	for _, forbidden := range []string{"[tool]", "[patch]", "[file]", "✓ completed", "✗ failed", "… started", "? mystery"} {
		if strings.Contains(plain, forbidden) {
			t.Fatalf("timeline row contains forbidden compact formatting text %q:\n%s", forbidden, model.View())
		}
	}
}

func TestCompactToolRowUsesRawInputSummary(t *testing.T) {
	tests := []struct {
		name string
		part index.TimelinePart
		want string
	}{
		{
			name: "pi bash command",
			part: index.TimelinePart{Kind: opencode.PartKindTool, ToolName: "bash", Status: "failed", Title: "bash", Preview: `bash - {"command":"go test ./internal/tui","timeout":120}`, RawJSON: `{"type":"toolCall","name":"bash","arguments":{"command":"go test ./internal/tui","timeout":120}}`},
			want: `$ bash - ✗ - go test ./internal/tui`,
		},
		{
			name: "pi read path",
			part: index.TimelinePart{Kind: opencode.PartKindTool, ToolName: "read", Status: "completed", Title: "read", Preview: `read - {"limit":70,"offset":1380,"path":"internal/tui/model_test.go"}`, RawJSON: `{"type":"toolCall","name":"read","arguments":{"limit":70,"offset":1380,"path":"internal/tui/model_test.go"}}`},
			want: `◧ read - ✓ - internal/tui/model_test.go`,
		},
		{
			name: "pi edit path",
			part: index.TimelinePart{Kind: opencode.PartKindTool, ToolName: "edit", Status: "completed", Title: "edit", Preview: `edit - {"path":"internal/tui/model.go"}`, RawJSON: `{"type":"toolCall","name":"edit","arguments":{"path":"internal/tui/model.go","edits":[{"oldText":"x","newText":"y"}]}}`},
			want: `✎ edit - ✓ - internal/tui/model.go`,
		},
		{
			name: "pi edit count fallback",
			part: index.TimelinePart{Kind: opencode.PartKindTool, ToolName: "edit", Status: "completed", Title: "edit", Preview: `edit - {"edits":[{"oldText":"x"}]}`, RawJSON: `{"type":"toolCall","name":"edit","arguments":{"edits":[{"oldText":"x","newText":"y"}]}}`},
			want: `✎ edit - ✓ - 1 edit`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compactPart(tt.part)
			if got != tt.want {
				t.Fatalf("compactPart() = %q, want %q", got, tt.want)
			}
			if strings.Contains(got, "{\"") {
				t.Fatalf("compact row should prefer readable input summary over JSON: %q", got)
			}
		})
	}
}

func TestCompactToolRowStripsRedundantToolName(t *testing.T) {
	tests := []struct {
		name string
		part index.TimelinePart
		want string
	}{
		{
			name: "preview prefix",
			part: index.TimelinePart{Kind: opencode.PartKindTool, ToolName: "bash", Status: "completed", Preview: `bash - {"command":"go test ./internal/tui"}`},
			want: `$ bash - ✓ - {"command":"go test ./internal/tui"}`,
		},
		{
			name: "title equals tool",
			part: index.TimelinePart{Kind: opencode.PartKindTool, ToolName: "edit", Status: "completed", Title: "edit", Preview: `{"edits":[{"oldText":"x"}]}`},
			want: `✎ edit - ✓ - {"edits":[{"oldText":"x"}]}`,
		},
		{
			name: "descriptive title preserved",
			part: index.TimelinePart{Kind: opencode.PartKindTool, ToolName: "bash", Status: "completed", Title: "Run tests", Preview: `go test ./...`},
			want: `$ bash - ✓ - Run tests - go test ./...`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compactPart(tt.part)
			if got != tt.want {
				t.Fatalf("compactPart() = %q, want %q", got, tt.want)
			}
			if strings.Contains(got, "$ bash - ✓ - bash -") || strings.Contains(got, "✎ edit - ✓ - edit -") {
				t.Fatalf("compact row duplicated tool name: %q", got)
			}
		})
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

func TestAssistantMarkdownRendersByDefaultAndTogglesSource(t *testing.T) {
	repo := newFakeRepo(t)
	markdown := "# Plan\n\nUse `go test ./internal/tui`.\n\n- Keep it small"
	repo.timelines["ses_project"] = []index.TimelinePart{
		{PartID: "prt_markdown", SessionID: "ses_project", MessageID: "msg_assistant", Role: "assistant", Kind: opencode.PartKindText, Preview: markdown, IndexText: markdown, RawJSON: fmt.Sprintf(`{"type":"text","text":%q}`, markdown)},
	}

	model := NewModel(repo, repo.sessions)
	model = sendKey(t, model, "l")
	view := model.View()
	plain := plainView(view)
	if !model.renderMarkdown || !strings.Contains(view, "m source md") {
		t.Fatalf("assistant markdown should render by default with source toggle:\n%s", view)
	}
	if strings.Contains(plain, "`go test ./internal/tui`") || !strings.Contains(plain, "go test ./internal/tui") {
		t.Fatalf("assistant markdown did not render inline code by default:\n%s", view)
	}

	model = sendKey(t, model, "m")
	view = model.View()
	plain = plainView(view)
	if model.renderMarkdown || !strings.Contains(view, "m render md") {
		t.Fatalf("assistant markdown source toggle missing:\n%s", view)
	}
	if !strings.Contains(plain, "`go test ./internal/tui`") {
		t.Fatalf("assistant markdown source was not shown after toggle:\n%s", view)
	}
}

func TestUserMarkdownSyntaxStaysSourceText(t *testing.T) {
	repo := newFakeRepo(t)
	markdown := "Use `go test` and **keep source markers**."
	repo.timelines["ses_project"] = []index.TimelinePart{
		{PartID: "prt_user_md", SessionID: "ses_project", MessageID: "msg_user", Role: "user", Kind: opencode.PartKindText, Preview: markdown, IndexText: markdown, RawJSON: fmt.Sprintf(`{"type":"text","text":%q}`, markdown)},
	}

	model := NewModel(repo, repo.sessions)
	model = sendKey(t, model, "l")
	plain := plainView(model.View())
	if !strings.Contains(plain, "`go test`") || !strings.Contains(plain, "**keep source markers**") {
		t.Fatalf("user markdown should remain source text:\n%s", model.View())
	}
}

func TestAssistantMarkdownCodeBlocksInlineCodeAndUnknownFence(t *testing.T) {
	source := "Inline `value`.\n\n```go\nfmt.Println(\"hi\")\n```\n\n```definitelyunknown\nplain fallback\n```"
	rows := assistantMarkdownRows(source, 80)
	rendered := strings.Join(rows, "\n")
	plain := plainView(rendered)

	if strings.Contains(plain, "```") || strings.Contains(plain, "`value`") {
		t.Fatalf("assistant markdown should render code markers away:\n%s", rendered)
	}
	if !strings.Contains(plain, "fmt.Println") || !strings.Contains(plain, "plain fallback") || !strings.Contains(plain, "value") {
		t.Fatalf("assistant markdown code content missing:\n%s", rendered)
	}
	if !strings.Contains(rendered, "\x1b[") {
		t.Fatalf("assistant markdown code should include ANSI styling:\n%s", rendered)
	}
}

func TestAssistantMarkdownCodeFenceTimelineDoesNotPanic(t *testing.T) {
	repo := newFakeRepo(t)
	markdown := "```go\n// comment\nfmt.Println(\"hi\")\n```"
	repo.timelines["ses_project"] = []index.TimelinePart{
		{PartID: "prt_code", SessionID: "ses_project", MessageID: "msg_assistant", Role: "assistant", Kind: opencode.PartKindText, Preview: markdown, IndexText: markdown, RawJSON: fmt.Sprintf(`{"type":"text","text":%q}`, markdown)},
	}

	model := NewModel(repo, repo.sessions)
	model = sendKey(t, model, "l")
	view := model.View()
	plain := plainView(view)
	if !strings.Contains(plain, "fmt.Println") || !strings.Contains(plain, "comment") {
		t.Fatalf("assistant code fence content missing from timeline:\n%s", view)
	}
}

func TestAssistantMarkdownLongScrollingAndBoundedRendering(t *testing.T) {
	repo := newFakeRepo(t)
	var lines []string
	for i := 0; i < 30; i++ {
		lines = append(lines, fmt.Sprintf("- long item %02d", i))
	}
	markdown := strings.Join(lines, "\n")
	repo.timelines["ses_project"] = []index.TimelinePart{
		{PartID: "prt_long_md", SessionID: "ses_project", MessageID: "msg_assistant", Role: "assistant", Kind: opencode.PartKindText, Preview: markdown, IndexText: markdown, RawJSON: fmt.Sprintf(`{"type":"text","text":%q}`, markdown)},
		{PartID: "prt_after", SessionID: "ses_project", MessageID: "msg_after", Role: "assistant", Kind: opencode.PartKindText, Preview: "after long", IndexText: "after long"},
	}

	model := NewModel(repo, repo.sessions)
	model, _ = updateModel(t, model, tea.WindowSizeMsg{Width: 80, Height: 9})
	model = sendKey(t, model, "l")
	if strings.Contains(plainView(model.View()), "long item 20") {
		t.Fatalf("long assistant markdown render should be bounded:\n%s", model.View())
	}
	for i := 0; i < 8; i++ {
		model = sendKey(t, model, "j")
	}
	if model.selectedPart != 0 {
		t.Fatalf("focus moved away from long assistant markdown too early: selected=%d", model.selectedPart)
	}
	if model.timelineScroll < 8 {
		t.Fatalf("j did not keep scrolling inside long assistant markdown: scroll=%d", model.timelineScroll)
	}
	plain := plainView(model.View())
	if strings.Contains(plain, "long item 00") || !strings.Contains(plain, "long item 08") {
		t.Fatalf("long assistant markdown viewport did not advance:\n%s", model.View())
	}
}

func TestTimelineSearchUsesSourceMarkdownText(t *testing.T) {
	repo := newFakeRepo(t)
	markdown := "Use `needle` in rendered assistant markdown."
	repo.timelines["ses_project"] = []index.TimelinePart{
		{PartID: "prt_source_match", SessionID: "ses_project", MessageID: "msg_assistant", Role: "assistant", Kind: opencode.PartKindText, Preview: markdown, IndexText: markdown, RawJSON: fmt.Sprintf(`{"type":"text","text":%q}`, markdown)},
	}

	model := NewModel(repo, repo.sessions)
	model = sendKey(t, model, "l")
	if strings.Contains(plainView(model.View()), "`needle`") {
		t.Fatalf("precondition failed: rendered markdown should hide source backticks:\n%s", model.View())
	}
	model = search(t, model, "`needle`")
	if repo.lastTimelineSearch != "`needle`" {
		t.Fatalf("timeline search query = %q", repo.lastTimelineSearch)
	}
	if len(model.timeline) != 1 || model.timeline[0].PartID != "prt_source_match" {
		t.Fatalf("timeline search should match source/index markdown text, got %#v", model.timeline)
	}
}

func TestUserMessageDetailShowsMoreThanTimelinePreview(t *testing.T) {
	repo := newFakeRepo(t)
	tail := "detail-only-user-tail"
	text := strings.Repeat("A", maxTranscriptRunes+200) + "\nUse `source marker` " + tail
	raw := fmt.Sprintf(`{"type":"text","text":%q}`, text)
	part := index.TimelinePart{PartID: "prt_long_user_detail", SessionID: "ses_project", MessageID: "msg_user", Role: "user", Kind: opencode.PartKindText, Preview: text, IndexText: text, RawJSON: raw, SizeBytes: int64(len(raw))}
	repo.timelines["ses_project"] = []index.TimelinePart{part}

	preview := displayPartText(part)
	if !strings.HasSuffix(preview, "...") || strings.Contains(preview, tail) {
		t.Fatalf("timeline preview should remain bounded and omit tail: len=%d containsTail=%v", len([]rune(preview)), strings.Contains(preview, tail))
	}

	model := NewModel(repo, repo.sessions)
	model, _ = updateModel(t, model, tea.WindowSizeMsg{Width: 80, Height: 10})
	model = sendKey(t, model, "l")
	model = sendKey(t, model, "enter")
	model = sendKey(t, model, "G")
	view := plainView(model.View())
	if model.mode != ViewRawPart || !model.messageDetail.active || !strings.Contains(view, "Message Detail (source)") {
		t.Fatalf("user text did not open message detail: mode=%v active=%v\n%s", model.mode, model.messageDetail.active, model.View())
	}
	if !strings.Contains(view, tail) || !strings.Contains(view, "`source marker`") {
		t.Fatalf("message detail should show source tail beyond timeline preview:\n%s", model.View())
	}
}

func TestAssistantMessageDetailPreservesMarkdownMode(t *testing.T) {
	repo := newFakeRepo(t)
	markdown := "# Plan\n\nUse `go test ./internal/tui`."
	raw := fmt.Sprintf(`{"type":"text","text":%q}`, markdown)
	repo.timelines["ses_project"] = []index.TimelinePart{
		{PartID: "prt_assistant_detail", SessionID: "ses_project", MessageID: "msg_assistant", Role: "assistant", Kind: opencode.PartKindText, Preview: markdown, IndexText: markdown, RawJSON: raw, SizeBytes: int64(len(raw))},
	}

	model := NewModel(repo, repo.sessions)
	model = sendKey(t, model, "l")
	model = sendKey(t, model, "enter")
	plain := plainView(model.View())
	if model.mode != ViewRawPart || !model.messageDetail.renderMarkdown || !strings.Contains(plain, "Message Detail (markdown)") {
		t.Fatalf("assistant detail should preserve rendered markdown mode: mode=%v render=%v\n%s", model.mode, model.messageDetail.renderMarkdown, model.View())
	}
	if strings.Contains(plain, "`go test ./internal/tui`") || !strings.Contains(plain, "go test ./internal/tui") {
		t.Fatalf("assistant detail should render markdown by default:\n%s", model.View())
	}

	model = sendKey(t, model, "h")
	model = sendKey(t, model, "m")
	model = sendKey(t, model, "enter")
	plain = plainView(model.View())
	if model.messageDetail.renderMarkdown || !strings.Contains(plain, "Message Detail (source)") || !strings.Contains(plain, "`go test ./internal/tui`") {
		t.Fatalf("assistant detail should preserve source markdown mode:\n%s", model.View())
	}
}

func TestMessageDetailTruncationMarker(t *testing.T) {
	repo := newFakeRepo(t)
	text := strings.Repeat("A", MaxMessageDetailRunes) + "TAIL_AFTER_CAP"
	raw := fmt.Sprintf(`{"type":"text","text":%q}`, text)
	repo.timelines["ses_project"] = []index.TimelinePart{
		{PartID: "prt_truncated_detail", SessionID: "ses_project", MessageID: "msg_user", Role: "user", Kind: opencode.PartKindText, Preview: "long", IndexText: "long", RawJSON: raw, SizeBytes: int64(len(raw))},
	}

	model := NewModel(repo, repo.sessions)
	model, _ = updateModel(t, model, tea.WindowSizeMsg{Width: 80, Height: 10})
	model = sendKey(t, model, "l")
	model = sendKey(t, model, "enter")
	model = sendKey(t, model, "G")
	view := plainView(model.View())
	if !strings.Contains(view, messageDetailTruncationMarker()) {
		t.Fatalf("message detail truncation marker missing:\n%s", model.View())
	}
	if strings.Contains(view, "TAIL_AFTER_CAP") {
		t.Fatalf("message detail rendered content beyond cap:\n%s", model.View())
	}
}

func TestMessageDetailGuardShowsIndexedPreviewFallback(t *testing.T) {
	tests := []struct {
		name string
		part index.TimelinePart
	}{
		{
			name: "binary raw",
			part: index.TimelinePart{PartID: "prt_binary_detail", SessionID: "ses_project", MessageID: "msg_user", Role: "user", Kind: opencode.PartKindText, Binary: true, Preview: "indexed fallback preview", IndexText: "indexed fallback index", RawJSON: `{"type":"text","text":"unsafe full content"}`, SizeBytes: 44},
		},
		{
			name: "too large source",
			part: index.TimelinePart{PartID: "prt_large_detail", SessionID: "ses_project", MessageID: "msg_user", Role: "user", Kind: opencode.PartKindText, Preview: "large fallback preview", IndexText: "large fallback index", SourcePath: "/missing/source.json", SizeBytes: MaxRawDisplayBytes + 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo(t)
			repo.timelines["ses_project"] = []index.TimelinePart{tt.part}

			model := NewModel(repo, repo.sessions)
			model = sendKey(t, model, "l")
			model = sendKey(t, model, "enter")
			plain := plainView(model.View())
			if model.mode != ViewRawPart || !model.messageDetail.active || !strings.Contains(plain, "Message Detail") {
				t.Fatalf("guarded part did not open message detail: mode=%v active=%v\n%s", model.mode, model.messageDetail.active, model.View())
			}
			if !strings.Contains(plain, "too large") || !strings.Contains(plain, "Indexed Preview") || !strings.Contains(plain, firstNonEmpty(tt.part.IndexText, tt.part.Preview)) {
				t.Fatalf("guarded detail missing guard/fallback:\n%s", model.View())
			}
			if strings.Contains(plain, "unsafe full content") || !strings.Contains(plain, "raw unavailable") {
				t.Fatalf("guarded detail rendered unsafe raw or allowed raw toggle:\n%s", model.View())
			}
		})
	}
}

func TestReasoningMessageDetailRespectsVisibility(t *testing.T) {
	repo := newFakeRepo(t)
	reasoning := "visible reasoning detail"
	raw := fmt.Sprintf(`{"type":"reasoning","text":%q}`, reasoning)
	repo.timelines["ses_project"] = []index.TimelinePart{
		{PartID: "prt_reasoning_detail", SessionID: "ses_project", MessageID: "msg_assistant", Role: "assistant", Kind: opencode.PartKindReasoning, Preview: reasoning, IndexText: reasoning, RawJSON: raw, SizeBytes: int64(len(raw))},
	}

	model := NewModel(repo, repo.sessions)
	model = sendKey(t, model, "l")
	if strings.Contains(plainView(model.View()), reasoning) {
		t.Fatalf("reasoning should be hidden before toggle:\n%s", model.View())
	}
	model = sendKey(t, model, "enter")
	if model.mode != ViewTimeline {
		t.Fatalf("hidden reasoning should not open, mode=%v", model.mode)
	}

	model = sendKey(t, model, "r")
	model = sendKey(t, model, "enter")
	view := plainView(model.View())
	if model.mode != ViewRawPart || !model.messageDetail.active || !strings.Contains(view, reasoning) {
		t.Fatalf("visible reasoning did not open bounded message detail: mode=%v active=%v\n%s", model.mode, model.messageDetail.active, model.View())
	}
}

func TestMessageDetailRawToggleAndSearchUseSourceText(t *testing.T) {
	repo := newFakeRepo(t)
	markdown := "Use `needle` in source markdown."
	raw := fmt.Sprintf(`{"type":"text","text":%q}`, markdown)
	repo.timelines["ses_project"] = []index.TimelinePart{
		{PartID: "prt_raw_search_detail", SessionID: "ses_project", MessageID: "msg_assistant", Role: "assistant", Kind: opencode.PartKindText, Preview: markdown, IndexText: markdown, RawJSON: raw, SizeBytes: int64(len(raw))},
	}

	model := NewModel(repo, repo.sessions)
	model = sendKey(t, model, "l")
	model = sendKey(t, model, "enter")
	if strings.Contains(plainView(model.View()), "`needle`") {
		t.Fatalf("precondition failed: rendered markdown should hide source backticks:\n%s", model.View())
	}
	model = search(t, model, "`needle`")
	plain := plainView(model.View())
	if strings.Contains(plain, "No matches") || !strings.Contains(plain, "needle") {
		t.Fatalf("message detail search should match source/capped text:\n%s", model.View())
	}

	model = sendKey(t, model, "R")
	plain = plainView(model.View())
	if !model.rawMode || !strings.Contains(plain, "Raw JSON") || !strings.Contains(plain, `"text": "Use `) {
		t.Fatalf("message detail raw toggle should show guarded-safe raw JSON:\n%s", model.View())
	}
}

func TestTimelineStillOpensPatchAndFileDetails(t *testing.T) {
	repo := newFakeRepo(t)
	patchRaw := `{"type":"patch","title":"Update README","path":"README.md","diff":"@@ -1 +1\n-old\n+new"}`
	fileRaw := `{"type":"file","mime":"text/plain","filename":"README.md","source":{"type":"file","path":"README.md","text":{"value":"hello docs"}}}`
	repo.timelines["ses_project"] = []index.TimelinePart{
		{PartID: "prt_patch_timeline", SessionID: "ses_project", MessageID: "msg_assistant", Role: "assistant", Kind: opencode.PartKindPatch, Title: "Update README", RawJSON: patchRaw, SizeBytes: int64(len(patchRaw))},
		{PartID: "prt_file_timeline", SessionID: "ses_project", MessageID: "msg_assistant", Role: "assistant", Kind: opencode.PartKindFile, FilePath: "README.md", RawJSON: fileRaw, SizeBytes: int64(len(fileRaw))},
	}
	repo.rawParts["prt_patch_timeline"] = index.RawPart{PartID: "prt_patch_timeline", Kind: opencode.PartKindPatch, Title: "Update README", RawJSON: patchRaw, SizeBytes: int64(len(patchRaw))}
	repo.rawParts["prt_file_timeline"] = index.RawPart{PartID: "prt_file_timeline", Kind: opencode.PartKindFile, FilePath: "README.md", RawJSON: fileRaw, SizeBytes: int64(len(fileRaw))}

	model := NewModel(repo, repo.sessions)
	model = sendKey(t, model, "l")
	model = sendKey(t, model, "enter")
	if model.mode != ViewRawPart || model.messageDetail.active || !strings.Contains(plainView(model.View()), "Patch Detail") {
		t.Fatalf("patch timeline row should still open patch detail:\n%s", model.View())
	}
	model = sendKey(t, model, "h")
	model = sendKey(t, model, "j")
	model = sendKey(t, model, "enter")
	if model.mode != ViewRawPart || model.messageDetail.active || !strings.Contains(plainView(model.View()), "File Detail") || !strings.Contains(plainView(model.View()), "hello docs") {
		t.Fatalf("file timeline row should still open file detail:\n%s", model.View())
	}
}

func sendKey(t *testing.T, model Model, key string) Model {
	t.Helper()
	updated, cmd := updateModel(t, model, keyMsg(key))
	return runCmd(t, updated, cmd)
}

func runCmd(t *testing.T, model Model, cmd tea.Cmd) Model {
	t.Helper()
	if cmd == nil {
		return model
	}
	msg := cmd()
	if msg == nil {
		return model
	}
	updated, next := updateModel(t, model, msg)
	if next != nil {
		return runCmd(t, updated, next)
	}
	return updated
}

func plainView(view string) string {
	return termansi.Strip(view)
}

func search(t *testing.T, model Model, query string) Model {
	t.Helper()
	model = sendKey(t, model, "/")
	for _, r := range query {
		model = sendKey(t, model, string(r))
	}
	return sendKey(t, model, "enter")
}

func startSearchCommand(t *testing.T, model Model, query string) (Model, tea.Cmd) {
	t.Helper()
	model = sendKey(t, model, "/")
	for _, r := range query {
		model = sendKey(t, model, string(r))
	}
	return updateModel(t, model, keyMsg("enter"))
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

func sessionListModeTestSessions() []index.SessionSummary {
	base := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	return []index.SessionSummary{
		{ID: "beta-match", ProjectID: "beta", ProjectPath: "/tmp/beta", Title: "Beta Match", UpdatedAt: base.Add(4 * time.Hour)},
		{ID: "global-match", ProjectID: "global", ProjectPath: "Global", Title: "Global Match", UpdatedAt: base.Add(2 * time.Hour)},
		{ID: "alpha-recent", ProjectID: "alpha", ProjectPath: "/tmp/alpha", Title: "Alpha Recent", UpdatedAt: base.Add(time.Hour)},
		{ID: "alpha-match-old", ProjectID: "alpha", ProjectPath: "/tmp/alpha", Title: "Alpha Match Old", UpdatedAt: base},
	}
}

func sessionRowIDs(rows []sessionListRow) []string {
	var ids []string
	for _, row := range rows {
		if row.kind == sessionListRowSession {
			ids = append(ids, row.session.ID)
		}
	}
	return ids
}

func sessionHeaderLabels(rows []sessionListRow) []string {
	var labels []string
	for _, row := range rows {
		if row.kind == sessionListRowHeader {
			labels = append(labels, row.label)
		}
	}
	return labels
}

func containsSessionID(sessions []index.SessionSummary, id string) bool {
	for _, session := range sessions {
		if session.ID == id {
			return true
		}
	}
	return false
}

func groupSessionIDs(rows []sessionListRow, label string) []string {
	var ids []string
	inGroup := false
	for _, row := range rows {
		if row.kind == sessionListRowHeader {
			if inGroup {
				break
			}
			inGroup = row.label == label
			continue
		}
		if inGroup {
			ids = append(ids, row.session.ID)
		}
	}
	return ids
}

func requireSelectedSessionVisible(t *testing.T, model Model) {
	t.Helper()
	selected, ok := model.selectedSessionSummary()
	if !ok {
		t.Fatal("selected session missing")
	}
	title := firstNonEmpty(selected.Title, selected.ID)
	if !strings.Contains(model.View(), title) {
		t.Fatalf("selected session %q is not visible:\n%s", title, model.View())
	}
}

type fakeRepo struct {
	sessions                  []index.SessionSummary
	timelines                 map[string][]index.TimelinePart
	trees                     map[string][]index.SessionTreeEntry
	branchTimelines           map[string][]index.TimelinePart
	rawParts                  map[string]index.RawPart
	lastSessionSearch         string
	lastTimelineSearchSession string
	lastTimelineSearch        string
	timelineLoads             int
}

func newFakeRepo(t *testing.T) *fakeRepo {
	t.Helper()
	heavyPath := filepath.Join(t.TempDir(), "heavy.json")
	return &fakeRepo{
		sessions: []index.SessionSummary{
			{ID: "ses_project", ProjectID: "proj", ProjectPath: "/tmp/project", Title: "Project session", MessageCount: 2, PartCount: 3, HeavyPartCount: 1, TokenUsage: opencode.TokenUsage{Available: true, Total: 321, Input: 100, Output: 70, Reasoning: 20, CacheRead: 30, CacheWrite: 10}},
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
		trees:           map[string][]index.SessionTreeEntry{},
		branchTimelines: map[string][]index.TimelinePart{},
		rawParts: map[string]index.RawPart{
			"prt_heavy": {PartID: "prt_heavy", SourcePath: heavyPath, SizeBytes: 1024 * 1024, Heavy: true, SkippedRaw: true, Preview: "AAECAwQFBgc"},
		},
	}
}

func (f *fakeRepo) ListSessions(context.Context) ([]index.SessionSummary, error) {
	return f.sessions, nil
}

func (f *fakeRepo) Session(_ context.Context, sessionID string) (index.SessionSummary, error) {
	for _, session := range f.sessions {
		if session.ID == sessionID {
			return session, nil
		}
	}
	return index.SessionSummary{}, fmt.Errorf("session %s not found", sessionID)
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
	f.lastTimelineSearchSession = sessionID
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

func (f *fakeRepo) SessionTree(_ context.Context, sessionID string) ([]index.SessionTreeEntry, error) {
	return f.trees[sessionID], nil
}

func (f *fakeRepo) SessionBranchLeaves(_ context.Context, sessionID string) ([]index.SessionTreeEntry, error) {
	entries := f.trees[sessionID]
	children := map[string]bool{}
	for _, entry := range entries {
		if entry.ParentID != "" {
			children[entry.ParentID] = true
		}
	}
	var leaves []index.SessionTreeEntry
	for _, entry := range entries {
		if !children[entry.ID] {
			leaves = append(leaves, entry)
		}
	}
	sort.SliceStable(leaves, func(i, j int) bool { return leaves[i].AppendOrder > leaves[j].AppendOrder })
	return leaves, nil
}

func (f *fakeRepo) SessionTimelineForEntry(_ context.Context, sessionID, entryID string) ([]index.TimelinePart, error) {
	if parts, ok := f.branchTimelines[entryID]; ok {
		return parts, nil
	}
	return f.timelines[sessionID], nil
}

func (f *fakeRepo) SearchSessionTree(_ context.Context, sessionID, query string) ([]index.SessionTreeEntry, error) {
	var results []index.SessionTreeEntry
	for _, entry := range f.trees[sessionID] {
		if strings.Contains(strings.ToLower(entry.ID+" "+entry.Label+" "+entry.Role+" "+entry.EntryType), strings.ToLower(query)) {
			results = append(results, entry)
		}
	}
	return results, nil
}

func (f *fakeRepo) RawPart(_ context.Context, partID string) (index.RawPart, error) {
	return f.rawParts[partID], nil
}
