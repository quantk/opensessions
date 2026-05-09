package tui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/quantick/opensession/internal/index"
	"github.com/quantick/opensession/internal/indexer"
	"github.com/quantick/opensession/internal/opencode"
	"github.com/quantick/opensession/internal/source"
)

const (
	MaxRawDisplayBytes    = 256 * 1024
	MaxMessageDetailRunes = 128 * 1024
	maxTranscriptRunes    = 2000
)

type ViewMode int

const (
	ViewSessions ViewMode = iota
	ViewTimeline
	ViewSessionTree
	ViewRawPart
)

type SessionListMode int

const (
	SessionListFlat SessionListMode = iota
	SessionListGrouped
)

type sessionListRowKind int

const (
	sessionListRowSession sessionListRowKind = iota
	sessionListRowHeader
)

type sessionListRow struct {
	kind         sessionListRowKind
	session      index.SessionSummary
	sessionIndex int
	key          string
	label        string
	count        int
	activeAt     time.Time
}

type sessionListGroup struct {
	key      string
	label    string
	activeAt time.Time
	rows     []sessionListRow
}

type timelineContext struct {
	session        index.SessionSummary
	timeline       []index.TimelinePart
	allTimeline    []index.TimelinePart
	selectedPart   int
	timelineScroll int
	showReasoning  bool
	renderMarkdown bool
}

type timelineDisplayCache struct {
	textByPartID map[string]string
	markdownRows map[markdownCacheKey][]string
	layoutKey    timelineLayoutKey
	layout       *timelineLayout
}

type markdownCacheKey struct {
	partID  string
	content string
	width   int
}

type timelineLayoutKey struct {
	revision       int
	width          int
	showReasoning  bool
	renderMarkdown bool
}

type timelineLayout struct {
	rows   []timelineLayoutRow
	ranges []timelineRowRange
}

type timelineLayoutRowKind int

const (
	timelineLayoutRowEmpty timelineLayoutRowKind = iota
	timelineLayoutRowSpacer
	timelineLayoutRowHeader
	timelineLayoutRowPart
)

type timelineLayoutRow struct {
	kind      timelineLayoutRowKind
	partIndex int
	rowIndex  int
}

type timelineRowRange struct {
	start int
	end   int
}

type sessionSearchResultMsg struct {
	requestID int
	query     string
	sessions  []index.SessionSummary
	err       error
}

type timelineSearchResultMsg struct {
	requestID int
	sessionID string
	query     string
	parts     []index.TimelinePart
	err       error
}

type treeSearchResultMsg struct {
	requestID int
	sessionID string
	query     string
	entries   []index.SessionTreeEntry
	err       error
}

type indexEventMsg struct {
	event indexer.Event
	ok    bool
}

type Repository interface {
	ListSessions(context.Context) ([]index.SessionSummary, error)
	Session(context.Context, string) (index.SessionSummary, error)
	SearchSessions(context.Context, string) ([]index.SessionSummary, error)
	SessionTimeline(context.Context, string) ([]index.TimelinePart, error)
	SearchSession(context.Context, string, string) ([]index.TimelinePart, error)
	SessionTree(context.Context, string) ([]index.SessionTreeEntry, error)
	SessionBranchLeaves(context.Context, string) ([]index.SessionTreeEntry, error)
	SessionTimelineForEntry(context.Context, string, string) ([]index.TimelinePart, error)
	SearchSessionTree(context.Context, string, string) ([]index.SessionTreeEntry, error)
	RawPart(context.Context, string) (index.RawPart, error)
}

type Model struct {
	repo Repository

	mode            ViewMode
	sessionListMode SessionListMode
	sessions        []index.SessionSummary
	allSessions     []index.SessionSummary
	timeline        []index.TimelinePart
	allTimeline     []index.TimelinePart
	treeEntries     []index.SessionTreeEntry
	allTreeEntries  []index.SessionTreeEntry
	timelineStack   []timelineContext
	currentSession  index.SessionSummary
	selectedSession int
	selectedPart    int
	selectedTree    int
	sessionScroll   int
	timelineScroll  int
	treeScroll      int
	rawScroll       int
	activeEntryID   string

	searchMode     bool
	searchLoading  bool
	searchRequest  int
	searchQuery    string
	rawSearchQuery string
	showReasoning  bool
	renderMarkdown bool

	rawPart       index.RawPart
	rawContent    string
	rawData       map[string]any
	rawGuard      string
	rawMode       bool
	messageDetail messageDetailState
	lastErr       error

	indexEvents      <-chan indexer.Event
	indexingEnabled  bool
	indexingActive   bool
	indexingDone     bool
	indexingStatus   string
	indexingErr      string
	indexingSessions int

	width  int
	height int

	timelineRevision int
	timelineCache    *timelineDisplayCache
}

var partTextFromRawJSONHook func()

func NewModel(repo Repository, sessions []index.SessionSummary) Model {
	return newModel(repo, sessions, nil, false, "")
}

func NewModelWithIndexEvents(repo Repository, sessions []index.SessionSummary, events <-chan indexer.Event, indexingEnabled bool) Model {
	return newModel(repo, sessions, events, indexingEnabled, "")
}

func NewModelWithIndexingDisabled(repo Repository, sessions []index.SessionSummary) Model {
	return newModel(repo, sessions, nil, false, "Index: disabled (--no-scan)")
}

func newModel(repo Repository, sessions []index.SessionSummary, events <-chan indexer.Event, indexingEnabled bool, status string) Model {
	sessions = topLevelSessions(sessions)
	if status == "" && indexingEnabled {
		status = "Index: waiting to refresh cached sessions"
	}
	return Model{
		repo:             repo,
		mode:             ViewSessions,
		sessionListMode:  SessionListFlat,
		sessions:         append([]index.SessionSummary(nil), sessions...),
		allSessions:      append([]index.SessionSummary(nil), sessions...),
		renderMarkdown:   true,
		indexEvents:      events,
		indexingEnabled:  indexingEnabled,
		indexingActive:   indexingEnabled && events != nil,
		indexingStatus:   status,
		indexingSessions: len(sessions),
		width:            100,
		height:           28,
		timelineCache:    newTimelineDisplayCache(),
	}
}

func newTimelineDisplayCache() *timelineDisplayCache {
	return &timelineDisplayCache{
		textByPartID: map[string]string{},
		markdownRows: map[markdownCacheKey][]string{},
	}
}

func (m *Model) cancelPendingSearch() {
	m.searchRequest++
	m.searchLoading = false
}

func (m *Model) resetTimelineDisplayCache() {
	m.timelineRevision++
	m.timelineCache = newTimelineDisplayCache()
	m.precomputeTimelineText()
}

func (m *Model) invalidateTimelineLayout() {
	if m.timelineCache == nil {
		m.timelineCache = newTimelineDisplayCache()
		return
	}
	m.timelineCache.layout = nil
}

func (m *Model) precomputeTimelineText() {
	if m.timelineCache == nil {
		m.timelineCache = newTimelineDisplayCache()
	}
	for _, part := range m.timeline {
		m.cachedDisplayText(part)
	}
}

func (m Model) Init() tea.Cmd {
	return m.waitIndexEventCmd()
}

func (m Model) waitIndexEventCmd() tea.Cmd {
	if m.indexEvents == nil {
		return nil
	}
	return func() tea.Msg {
		event, ok := <-m.indexEvents
		return indexEventMsg{event: event, ok: ok}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typed.Width
		m.height = typed.Height
		m.invalidateTimelineLayout()
		m.normalizeSessionSelection()
		m.timelineScroll = clamp(m.timelineScroll, 0, m.maxTimelineScroll())
		if m.mode == ViewTimeline {
			m.normalizeTimelineFocus()
		}
		m.rawScroll = clamp(m.rawScroll, 0, m.maxRawScroll())
		return m, nil
	case tea.KeyMsg:
		if m.searchMode {
			return m.updateSearch(typed)
		}
		return m.updateKey(typed)
	case sessionSearchResultMsg:
		return m.applySessionSearchResult(typed), nil
	case timelineSearchResultMsg:
		return m.applyTimelineSearchResult(typed), nil
	case treeSearchResultMsg:
		return m.applyTreeSearchResult(typed), nil
	case indexEventMsg:
		m = m.applyIndexEvent(typed)
		if typed.ok {
			return m, m.waitIndexEventCmd()
		}
		return m, nil
	default:
		return m, nil
	}
}

func (m Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "/":
		m.searchMode = true
		m.searchQuery = ""
		return m, nil
	case "j", "down":
		m.move(1)
		return m, nil
	case "k", "up":
		m.move(-1)
		return m, nil
	case "pgdown", "ctrl+d":
		m.page(1)
		return m, nil
	case "pgup", "ctrl+u":
		m.page(-1)
		return m, nil
	case "g", "home":
		m.jump(false)
		return m, nil
	case "G", "end":
		m.jump(true)
		return m, nil
	case "v":
		if m.mode == ViewSessions {
			m.toggleSessionListMode()
		}
		return m, nil
	case "tab", "shift+tab":
		return m, nil
	case "b":
		if m.mode == ViewTimeline && isPiSource(m.currentSession.SourceKind) {
			return m.openSessionTree(), nil
		}
		return m, nil
	case "l", "enter":
		return m.openSelected(), nil
	case "h", "esc":
		return m.back(), nil
	case "r":
		if m.mode == ViewTimeline {
			m.showReasoning = !m.showReasoning
			m.invalidateTimelineLayout()
			m.normalizeTimelineFocus()
		}
		return m, nil
	case "m":
		if m.mode == ViewTimeline {
			m.renderMarkdown = !m.renderMarkdown
			m.invalidateTimelineLayout()
			m.timelineScroll = clamp(m.timelineScroll, 0, m.maxTimelineScroll())
			m.normalizeTimelineFocus()
		} else if m.mode == ViewRawPart && m.messageDetail.active && isAssistantRole(m.rawPart.Role) && !m.rawMode {
			m.messageDetail.renderMarkdown = !m.messageDetail.renderMarkdown
			m.rawScroll = clamp(m.rawScroll, 0, m.maxRawScroll())
		}
		return m, nil
	case "R":
		if m.mode == ViewRawPart && m.rawGuard == "" && m.rawContent != "" {
			m.rawMode = !m.rawMode
			m.rawScroll = clamp(m.rawScroll, 0, m.maxRawScroll())
		}
		return m, nil
	default:
		return m, nil
	}
}

func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searchMode = false
		m.searchQuery = ""
		return m, nil
	case "enter":
		m.searchMode = false
		return m.applySearch()
	case "backspace":
		if len(m.searchQuery) > 0 {
			runes := []rune(m.searchQuery)
			m.searchQuery = string(runes[:len(runes)-1])
		}
		return m, nil
	default:
		if msg.Type == tea.KeyRunes {
			m.searchQuery += string(msg.Runes)
		}
		return m, nil
	}
}

func (m Model) applySearch() (Model, tea.Cmd) {
	query := strings.TrimSpace(m.searchQuery)
	m.searchRequest++
	requestID := m.searchRequest
	switch m.mode {
	case ViewSessions:
		selectedID := m.selectedSessionID()
		fallback := m.selectedSession
		if query == "" {
			m.sessions = append([]index.SessionSummary(nil), m.allSessions...)
			m.selectSessionByID(selectedID, fallback)
			m.lastErr = nil
			m.searchLoading = false
			return m, nil
		}
		m.searchLoading = true
		return m, func() tea.Msg {
			sessions, err := m.repo.SearchSessions(context.Background(), query)
			return sessionSearchResultMsg{requestID: requestID, query: query, sessions: sessions, err: err}
		}
	case ViewTimeline:
		if query == "" {
			m.timeline = append([]index.TimelinePart(nil), m.allTimeline...)
			m.resetTimelineDisplayCache()
			m.timelineScroll = 0
			m.selectedPart = m.firstFocusablePartInViewport()
			m.lastErr = nil
			m.searchLoading = false
			return m, nil
		}
		sessionID := m.currentSession.ID
		m.searchLoading = true
		return m, func() tea.Msg {
			parts, err := m.repo.SearchSession(context.Background(), sessionID, query)
			return timelineSearchResultMsg{requestID: requestID, sessionID: sessionID, query: query, parts: parts, err: err}
		}
	case ViewSessionTree:
		if query == "" {
			m.treeEntries = append([]index.SessionTreeEntry(nil), m.allTreeEntries...)
			m.treeScroll = 0
			m.selectedTree = 0
			m.lastErr = nil
			m.searchLoading = false
			return m, nil
		}
		sessionID := m.currentSession.ID
		m.searchLoading = true
		return m, func() tea.Msg {
			entries, err := m.repo.SearchSessionTree(context.Background(), sessionID, query)
			return treeSearchResultMsg{requestID: requestID, sessionID: sessionID, query: query, entries: entries, err: err}
		}
	case ViewRawPart:
		m.rawSearchQuery = query
		m.rawScroll = 0
	}
	return m, nil
}

func (m Model) applySessionSearchResult(msg sessionSearchResultMsg) Model {
	if msg.requestID != m.searchRequest || m.mode != ViewSessions {
		return m
	}
	m.searchLoading = false
	if msg.err != nil {
		m.lastErr = msg.err
		return m
	}
	selectedID := m.selectedSessionID()
	fallback := m.selectedSession
	m.sessions = topLevelSessions(msg.sessions)
	m.selectSessionByID(selectedID, fallback)
	m.lastErr = nil
	return m
}

func (m Model) applyTreeSearchResult(msg treeSearchResultMsg) Model {
	if msg.requestID != m.searchRequest || m.mode != ViewSessionTree || msg.sessionID != m.currentSession.ID {
		return m
	}
	m.searchLoading = false
	if msg.err != nil {
		m.lastErr = msg.err
		return m
	}
	m.treeEntries = append([]index.SessionTreeEntry(nil), msg.entries...)
	m.selectedTree = clamp(m.selectedTree, 0, max(0, len(m.treeEntries)-1))
	m.treeScroll = clamp(m.treeScroll, 0, m.maxTreeScroll())
	m.lastErr = nil
	return m
}

func (m Model) applyTimelineSearchResult(msg timelineSearchResultMsg) Model {
	if msg.requestID != m.searchRequest || m.mode != ViewTimeline || msg.sessionID != m.currentSession.ID {
		return m
	}
	m.searchLoading = false
	if msg.err != nil {
		m.lastErr = msg.err
		return m
	}
	m.timeline = append([]index.TimelinePart(nil), msg.parts...)
	m.resetTimelineDisplayCache()
	m.timelineScroll = 0
	m.selectedPart = m.firstFocusablePartInViewport()
	m.lastErr = nil
	return m
}

func (m Model) applyIndexEvent(msg indexEventMsg) Model {
	if !msg.ok {
		m.indexingActive = false
		if m.indexingEnabled && m.indexingStatus == "" && !m.indexingDone && m.indexingErr == "" {
			m.indexingStatus = "Index: refresh stopped"
		}
		return m
	}
	event := msg.event
	switch event.Kind {
	case indexer.EventStarted:
		m.indexingEnabled = true
		m.indexingActive = true
		m.indexingDone = false
		m.indexingErr = ""
		m.indexingStatus = "Index: refreshing cached sessions"
	case indexer.EventPhase:
		m.indexingEnabled = true
		m.indexingActive = true
		m.indexingStatus = formatIndexEventStatus(event)
	case indexer.EventSessions:
		m.indexingSessions = len(event.Sessions)
		m.applyRefreshedSessions(event.Sessions)
	case indexer.EventComplete:
		m.indexingActive = false
		m.indexingDone = true
		m.indexingErr = ""
		if len(event.Sessions) > 0 || m.indexingSessions == 0 {
			m.indexingSessions = len(event.Sessions)
		}
		if event.Sessions != nil {
			m.applyRefreshedSessions(event.Sessions)
		}
		m.indexingStatus = fmt.Sprintf("Index: up to date (%d sessions)", m.indexingSessions)
	case indexer.EventFailed:
		m.indexingActive = false
		m.indexingDone = false
		if event.Err != nil {
			m.indexingErr = event.Err.Error()
		} else {
			m.indexingErr = "unknown indexing error"
		}
		m.indexingStatus = "Index: refresh failed"
	}
	return m
}

func (m *Model) applyRefreshedSessions(sessions []index.SessionSummary) {
	refreshed := topLevelSessions(sessions)
	selectedID := m.selectedSessionID()
	fallback := m.selectedSession
	m.allSessions = append([]index.SessionSummary(nil), refreshed...)
	if m.mode == ViewSessions && strings.TrimSpace(m.searchQuery) == "" {
		m.sessions = append([]index.SessionSummary(nil), refreshed...)
		m.selectSessionByID(selectedID, fallback)
	}
}

func formatIndexEventStatus(event indexer.Event) string {
	fields := []string{"Index:"}
	if event.Source != "" {
		fields = append(fields, sourceLabel(string(event.Source)))
	}
	phase := strings.TrimSpace(event.Phase)
	if phase == "" {
		phase = "refreshing"
	}
	fields = append(fields, phase)
	if event.Total > 0 {
		fields = append(fields, fmt.Sprintf("%d/%d", event.Current, event.Total))
	}
	return strings.Join(fields, " ")
}

func (m *Model) move(delta int) {
	switch m.mode {
	case ViewSessions:
		m.moveSessionSelection(delta)
	case ViewTimeline:
		m.moveTimelineFocus(delta)
	case ViewSessionTree:
		m.moveTreeSelection(delta)
	case ViewRawPart:
		m.rawScroll = clamp(m.rawScroll+delta, 0, m.maxRawScroll())
	}
}

func (m *Model) page(delta int) {
	amount := max(1, m.bodyHeight()-4)
	switch m.mode {
	case ViewSessions:
		m.pageSessionSelection(delta * amount)
	case ViewTimeline:
		m.timelineScroll = clamp(m.timelineScroll+delta*amount, 0, m.maxTimelineScroll())
		m.selectedPart = m.firstFocusablePartInViewport()
	case ViewSessionTree:
		m.moveTreeSelection(delta * amount)
	case ViewRawPart:
		m.rawScroll = clamp(m.rawScroll+delta*amount, 0, m.maxRawScroll())
	}
}

func (m *Model) jump(bottom bool) {
	switch m.mode {
	case ViewSessions:
		m.jumpSessionSelection(bottom)
	case ViewTimeline:
		if bottom {
			m.timelineScroll = m.maxTimelineScroll()
			m.selectedPart = m.lastFocusablePart()
		} else {
			m.timelineScroll = 0
			m.selectedPart = m.firstFocusablePart()
		}
		m.ensureFocusedPartVisible()
	case ViewSessionTree:
		if bottom {
			m.selectedTree = max(0, len(m.treeEntries)-1)
		} else {
			m.selectedTree = 0
		}
		m.ensureTreeSelectionVisible()
	case ViewRawPart:
		if bottom {
			m.rawScroll = m.maxRawScroll()
		} else {
			m.rawScroll = 0
		}
	}
}

func (m Model) openSelected() Model {
	switch m.mode {
	case ViewSessions:
		session, ok := m.selectedSessionSummary()
		if !ok {
			return m
		}
		parts, err := m.repo.SessionTimeline(context.Background(), session.ID)
		if err != nil {
			m.lastErr = err
			return m
		}
		m.cancelPendingSearch()
		m.currentSession = session
		m.timeline = parts
		m.allTimeline = append([]index.TimelinePart(nil), parts...)
		m.activeEntryID = ""
		if isPiSource(session.SourceKind) {
			if leaves, err := m.repo.SessionBranchLeaves(context.Background(), session.ID); err == nil && len(leaves) > 0 {
				m.activeEntryID = leaves[0].ID
			}
		}
		m.timelineStack = nil
		m.timelineScroll = 0
		m.showReasoning = false
		m.renderMarkdown = true
		m.resetTimelineDisplayCache()
		m.selectedPart = m.firstFocusablePartInViewport()
		m.mode = ViewTimeline
		return m
	case ViewTimeline:
		partIndex := m.selectedPart
		if !m.partVisible(partIndex) {
			partIndex = m.firstFocusablePartInViewport()
		}
		if len(m.timeline) == 0 || partIndex < 0 || partIndex >= len(m.timeline) || !isOpenablePart(m.timeline[partIndex]) {
			return m
		}
		part := m.timeline[partIndex]
		if isLinkedTaskPart(part) {
			return m.openLinkedSession(partIndex, part)
		}
		if isMessageDetailPart(part) {
			return m.openMessageDetail(part)
		}
		return m.openRawPart(part.PartID)
	case ViewSessionTree:
		if len(m.treeEntries) == 0 || m.selectedTree < 0 || m.selectedTree >= len(m.treeEntries) {
			return m
		}
		entry := m.treeEntries[m.selectedTree]
		parts, err := m.repo.SessionTimelineForEntry(context.Background(), m.currentSession.ID, entry.ID)
		if err != nil {
			m.lastErr = err
			return m
		}
		m.cancelPendingSearch()
		m.timeline = parts
		m.allTimeline = append([]index.TimelinePart(nil), parts...)
		m.activeEntryID = entry.ID
		m.timelineScroll = 0
		m.resetTimelineDisplayCache()
		m.selectedPart = m.firstFocusablePartInViewport()
		m.mode = ViewTimeline
		m.lastErr = nil
		return m
	case ViewRawPart:
		return m
	default:
		return m
	}
}

func (m Model) openSessionTree() Model {
	entries, err := m.repo.SessionTree(context.Background(), m.currentSession.ID)
	if err != nil {
		m.lastErr = err
		return m
	}
	m.cancelPendingSearch()
	m.treeEntries = entries
	m.allTreeEntries = append([]index.SessionTreeEntry(nil), entries...)
	m.selectedTree = m.treeIndexByID(m.activeEntryID)
	m.treeScroll = 0
	m.ensureTreeSelectionVisible()
	m.mode = ViewSessionTree
	m.lastErr = nil
	return m
}

func (m Model) openLinkedSession(parentPartIndex int, parentPart index.TimelinePart) Model {
	childID := strings.TrimSpace(parentPart.LinkedSessionID)
	parts, err := m.repo.SessionTimeline(context.Background(), childID)
	if err != nil {
		m.lastErr = err
		return m
	}
	child, err := m.repo.Session(context.Background(), childID)
	if err != nil {
		child = index.SessionSummary{
			ID:            childID,
			ParentID:      m.currentSession.ID,
			ProjectID:     m.currentSession.ProjectID,
			ProjectPath:   m.currentSession.ProjectPath,
			ModelProvider: m.currentSession.ModelProvider,
			ModelID:       m.currentSession.ModelID,
		}
	}
	if child.ParentID == "" {
		child.ParentID = m.currentSession.ID
	}
	m.cancelPendingSearch()
	m.timelineStack = append(m.timelineStack, timelineContext{
		session:        m.currentSession,
		timeline:       append([]index.TimelinePart(nil), m.timeline...),
		allTimeline:    append([]index.TimelinePart(nil), m.allTimeline...),
		selectedPart:   parentPartIndex,
		timelineScroll: m.timelineScroll,
		showReasoning:  m.showReasoning,
		renderMarkdown: m.renderMarkdown,
	})
	m.currentSession = child
	m.timeline = parts
	m.allTimeline = append([]index.TimelinePart(nil), parts...)
	m.timelineScroll = 0
	m.showReasoning = false
	m.renderMarkdown = true
	m.resetTimelineDisplayCache()
	m.selectedPart = m.firstFocusablePartInViewport()
	m.mode = ViewTimeline
	m.lastErr = nil
	return m
}

func (m Model) openRawPart(partID string) Model {
	raw, err := m.repo.RawPart(context.Background(), partID)
	if err != nil {
		m.lastErr = err
		return m
	}
	m.cancelPendingSearch()
	m.mode = ViewRawPart
	m.rawPart = raw
	m.rawSearchQuery = ""
	m.messageDetail = messageDetailState{}
	m.rawMode = false
	m.rawScroll = 0
	m.loadRawDisplay(raw)
	m.lastErr = nil
	return m
}

func (m Model) openMessageDetail(part index.TimelinePart) Model {
	raw := rawPartFromTimelinePart(part)
	m.cancelPendingSearch()
	m.mode = ViewRawPart
	m.rawPart = raw
	m.rawSearchQuery = ""
	m.rawMode = false
	m.rawScroll = 0
	m.messageDetail = buildMessageDetail(part, m.renderMarkdown)
	m.loadRawDisplay(raw)
	m.lastErr = nil
	return m
}

func (m *Model) loadRawDisplay(raw index.RawPart) {
	m.rawContent = ""
	m.rawData = nil
	m.rawGuard = ""
	if raw.Heavy || raw.Binary || raw.SkippedRaw {
		m.rawGuard = "Raw part is too large or unsafe to display normally."
		return
	}
	if raw.RawJSON != "" {
		if len(raw.RawJSON) > MaxRawDisplayBytes {
			m.rawGuard = "Raw part is too large or unsafe to display normally."
			return
		}
		m.rawData = parseDetailPayload([]byte(raw.RawJSON))
		m.rawContent = formatRawContent([]byte(raw.RawJSON))
		return
	}
	if raw.SizeBytes > MaxRawDisplayBytes {
		m.rawGuard = "Raw part is too large or unsafe to display normally."
		return
	}
	content, err := os.ReadFile(raw.SourcePath)
	if err != nil {
		m.rawGuard = fmt.Sprintf("Raw part could not be loaded: %v", err)
		return
	}
	if len(content) > MaxRawDisplayBytes {
		m.rawGuard = "Raw part is too large or unsafe to display normally."
		return
	}
	m.rawData = parseDetailPayload(content)
	m.rawContent = formatRawContent(content)
}

func (m Model) back() Model {
	m.cancelPendingSearch()
	switch m.mode {
	case ViewTimeline:
		if len(m.timelineStack) > 0 {
			last := m.timelineStack[len(m.timelineStack)-1]
			m.timelineStack = m.timelineStack[:len(m.timelineStack)-1]
			m.currentSession = last.session
			m.timeline = append([]index.TimelinePart(nil), last.timeline...)
			m.allTimeline = append([]index.TimelinePart(nil), last.allTimeline...)
			m.selectedPart = last.selectedPart
			m.timelineScroll = last.timelineScroll
			m.showReasoning = last.showReasoning
			m.renderMarkdown = last.renderMarkdown
			m.resetTimelineDisplayCache()
			m.mode = ViewTimeline
			m.ensureFocusedPartVisible()
			break
		}
		m.mode = ViewSessions
	case ViewSessionTree:
		m.mode = ViewTimeline
	case ViewRawPart:
		m.mode = ViewTimeline
	}
	m.searchMode = false
	m.searchQuery = ""
	return m
}

type transcriptLine struct {
	text      string
	partIndex int
}

func (m Model) View() string {
	sections := []string{m.renderHeader()}
	if m.searchMode {
		sections = append(sections, m.renderSearchPrompt())
	}
	if m.lastErr != nil {
		sections = append(sections, warnStyle.Render(truncatePlain(m.lastErr.Error(), m.safeWidth())))
	}
	if m.searchLoading {
		sections = append(sections, dimStyle.Render(truncatePlain("Searching...", m.safeWidth())))
	}
	if status := m.indexStatusLine(); status != "" {
		style := dimStyle
		if m.indexingErr != "" {
			style = warnStyle
		}
		sections = append(sections, style.Render(truncatePlain(status, m.safeWidth())))
	}

	height := m.height - len(sections) - 1
	if height < 1 {
		height = 1
	}
	switch m.mode {
	case ViewSessions:
		sections = append(sections, m.renderSessions(height))
	case ViewTimeline:
		sections = append(sections, m.renderTimeline(height))
	case ViewSessionTree:
		sections = append(sections, m.renderSessionTree(height))
	case ViewRawPart:
		sections = append(sections, m.renderRawPart(height))
	}
	sections = append(sections, m.renderFooter())
	return strings.Join(sections, "\n")
}

func (m Model) renderHeader() string {
	width := m.safeWidth()
	detail := ""
	switch m.mode {
	case ViewSessions:
		detail = fmt.Sprintf("%d sessions - %s mode", len(m.sessions), m.sessionListModeLabel())
	case ViewTimeline:
		detail = firstNonEmpty(m.currentSession.Title, m.currentSession.ID)
	case ViewSessionTree:
		detail = firstNonEmpty(m.currentSession.Title, m.currentSession.ID)
	case ViewRawPart:
		detail = firstNonEmpty(m.rawPart.ToolName, m.rawPart.Title, m.rawPart.FilePath, m.rawPart.PartID)
	}
	plainDetail := truncatePlain(detail, max(0, width-24))
	line := titleStyle.Render("opensession") + " " + modeStyle.Render(m.modeLabel())
	if plainDetail != "" {
		line += " " + dimStyle.Render(plainDetail)
	}
	return line
}

func (m Model) renderSearchPrompt() string {
	query := m.searchQuery
	if m.mode == ViewRawPart && m.rawSearchQuery != "" && query == "" {
		query = m.rawSearchQuery
	}
	return accentStyle.Render("/") + truncatePlain(query, max(1, m.safeWidth()-1))
}

func (m Model) indexStatusLine() string {
	if m.indexingErr != "" {
		return "Index: refresh failed: " + m.indexingErr
	}
	if m.indexingStatus != "" {
		return m.indexingStatus
	}
	return ""
}

func (m Model) renderFooter() string {
	var help string
	switch m.mode {
	case ViewSessions:
		help = fmt.Sprintf("j/k move  l/Enter open  v %s view  / search  q quit", m.nextSessionListModeLabel())
	case ViewTimeline:
		branch := ""
		if isPiSource(m.currentSession.SourceKind) {
			branch = "  b tree"
		}
		help = fmt.Sprintf("j/k move focus  l/Enter open%s  r reasoning  %s  h back  / search  q quit", branch, m.markdownToggleHelp())
	case ViewSessionTree:
		help = "j/k move  l/Enter view branch  h back  / search  q quit"
	case ViewRawPart:
		toggle := "R raw JSON"
		if m.rawMode {
			toggle = "R " + m.rawPrimaryModeLabel()
		}
		if m.rawGuard != "" || m.rawContent == "" {
			toggle = "raw unavailable"
		}
		fields := []string{"j/k scroll", "pgup/pgdown page", toggle}
		if m.messageDetail.active && isAssistantRole(m.rawPart.Role) && !m.rawMode {
			fields = append(fields, m.markdownToggleHelp())
		}
		fields = append(fields, "h back", "/ filter", "q quit")
		help = strings.Join(fields, "  ")
	default:
		help = "q quit"
	}
	return dimStyle.Render(truncatePlain(help, m.safeWidth()))
}

func (m Model) renderSessions(height int) string {
	width := m.safeWidth()
	if len(m.sessions) == 0 {
		lines := []string{accentStyle.Render("Sessions")}
		if m.indexingActive {
			lines = append(lines, "No cached sessions yet.", "Indexing is running in the background.")
		} else {
			lines = append(lines, "No sessions found.")
		}
		return fitBlock(lines, height, width)
	}
	if width >= 86 && height >= 8 {
		return m.renderSessionsWide(height, width)
	}
	return fitBlock(m.sessionListLines(height, width), height, width)
}

func (m Model) renderSessionsWide(height, width int) string {
	leftWidth := width * 42 / 100
	if leftWidth < 34 {
		leftWidth = 34
	}
	if leftWidth > 54 {
		leftWidth = 54
	}
	rightWidth := width - leftWidth - 2
	if rightWidth < 24 {
		rightWidth = 24
		leftWidth = max(20, width-rightWidth-2)
	}
	left := fitLines(m.sessionListLines(height, leftWidth), height, leftWidth)
	right := fitLines(m.sessionPreviewLines(height, rightWidth), height, rightWidth)
	return joinColumns(left, right, leftWidth, rightWidth, 2, height)
}

func (m Model) sessionListLines(height, width int) []string {
	rows := m.sessionRows()
	lines := []string{accentStyle.Render(padPlain(fmt.Sprintf("Sessions %d (%s)", len(m.sessions), m.sessionListModeLabel()), width))}
	visible := max(1, height-1)
	start, end := m.sessionListWindow(rows, visible)
	selectedRow := m.selectedSessionRowIndex(rows)
	for i := start; i < end; i++ {
		row := rows[i]
		if row.kind == sessionListRowHeader {
			lines = append(lines, m.sessionHeaderLine(row, width))
			continue
		}
		session := row.session
		title := sourceBadge(session.SourceKind) + " " + firstNonEmpty(session.Title, session.ID)
		if session.Bookmarked {
			title = "* " + title
		}
		prefix := " "
		if m.sessionListMode == SessionListGrouped {
			prefix = "  "
		}
		content := withTail(prefix+title, countLabel(session), focusRailContentWidth(width))
		rowStyle := lipgloss.NewStyle()
		if i == selectedRow {
			rowStyle = selectedStyle
		}
		lines = append(lines, renderRailLine(content, width, i == selectedRow, false, rowStyle, i == selectedRow))
	}
	return lines
}

func (m Model) sessionPreviewLines(height, width int) []string {
	selected, ok := m.selectedSessionSummary()
	if !ok {
		return []string{"No selection."}
	}
	lines := []string{accentStyle.Render("Session")}
	lines = appendWrapped(lines, sourceBadge(selected.SourceKind)+" "+firstNonEmpty(selected.Title, selected.ID), width, titleStyle)
	lines = append(lines,
		dimStyle.Render(truncatePlain("Source: "+sourceLabel(selected.SourceKind), width)),
		dimStyle.Render(truncatePlain("Project: "+groupName(selected), width)),
		dimStyle.Render(truncatePlain("Model: "+firstNonEmpty(selected.ModelProvider, "unknown")+"/"+firstNonEmpty(selected.ModelID, "unknown"), width)),
		dimStyle.Render(truncatePlain("Updated: "+formatTime(selected.UpdatedAt), width)),
	)
	lines = append(lines, fmt.Sprintf("Messages: %d  Parts: %d  Heavy: %d", selected.MessageCount, selected.PartCount, selected.HeavyPartCount))
	lines = append(lines, tokenUsagePreviewLines(selected.TokenUsage, width)...)
	if len(selected.Tags) > 0 {
		lines = append(lines, truncatePlain("Tags: "+strings.Join(selected.Tags, ", "), width))
	}
	if selected.Bookmarked {
		lines = append(lines, "Bookmarked: yes")
	}
	if height-len(lines) > 2 {
		lines = append(lines, "", dimStyle.Render("Open to read the chat transcript."))
	}
	return lines
}

func (m Model) renderTimeline(height int) string {
	width := m.safeWidth()
	header := m.timelineHeader(width)
	contentHeight := max(1, height-len(header))
	layout := m.timelineLayout(width)
	maxScroll := max(0, len(layout.rows)-contentHeight)
	start := clamp(m.timelineScroll, 0, maxScroll)
	lines := append([]string{}, header...)
	for _, row := range m.visibleTimelineRows(layout, start, min(len(layout.rows), start+contentHeight), width) {
		lines = append(lines, row.text)
	}
	return fitBlock(lines, height, width)
}

func (m Model) timelineHeader(width int) []string {
	metadata := []string{
		"Source: " + sourceLabel(m.currentSession.SourceKind),
		"Project: " + groupName(m.currentSession),
		"Model: " + firstNonEmpty(m.currentSession.ModelProvider, "unknown") + "/" + firstNonEmpty(m.currentSession.ModelID, "unknown"),
	}
	if usage := compactTokenUsage(m.currentSession.TokenUsage); usage != "" {
		metadata = append(metadata, "Tokens: "+usage)
	}
	metadata = append(metadata, "Reasoning: "+onOff(m.showReasoning), "Scroll: "+m.timelineScrollLabel())
	if isPiSource(m.currentSession.SourceKind) {
		metadata = append(metadata, "Branch: "+firstNonEmpty(shortID(m.activeEntryID), "latest"))
	}
	header := []string{
		titleStyle.Render(truncatePlain(firstNonEmpty(m.currentSession.Title, m.currentSession.ID), width)),
		dimStyle.Render(truncatePlain(strings.Join(metadata, "  "), width)),
	}
	if context := m.nestedTimelineContext(); context != "" {
		header = append(header, dimStyle.Render(truncatePlain("Nested under: "+context, width)))
	}
	return header
}

func (m Model) nestedTimelineContext() string {
	if len(m.timelineStack) == 0 {
		return ""
	}
	parent := m.timelineStack[len(m.timelineStack)-1]
	label := firstNonEmpty(parent.session.Title, parent.session.ID)
	if parent.selectedPart >= 0 && parent.selectedPart < len(parent.timeline) {
		part := parent.timeline[parent.selectedPart]
		if task := firstNonEmpty(part.Title, part.Preview, part.PartID); task != "" {
			label += " via " + task
		}
	}
	return label
}

func (m Model) renderSessionTree(height int) string {
	width := m.safeWidth()
	lines := []string{
		titleStyle.Render(truncatePlain(firstNonEmpty(m.currentSession.Title, m.currentSession.ID), width)),
		dimStyle.Render(truncatePlain("Pi session tree  active "+firstNonEmpty(shortID(m.activeEntryID), "latest"), width)),
	}
	bodyHeight := max(1, height-len(lines))
	if len(m.treeEntries) == 0 {
		lines = append(lines, "No tree entries.")
		return fitBlock(lines, height, width)
	}
	start := clamp(m.treeScroll, 0, m.maxTreeScroll())
	end := min(len(m.treeEntries), start+bodyHeight)
	depths := treeDepths(m.treeEntries)
	childCounts := treeChildCounts(m.treeEntries)
	for i := start; i < end; i++ {
		entry := m.treeEntries[i]
		prefix := strings.Repeat("  ", min(depths[entry.ID], 8))
		shape := "─"
		if childCounts[entry.ID] > 1 {
			shape = "┬"
		} else if childCounts[entry.ID] == 0 {
			shape = "●"
		}
		label := firstNonEmpty(entry.Label, entry.Role, entry.EntryType, shortID(entry.ID))
		line := prefix + shape + " " + label
		if entry.ID == m.activeEntryID {
			line += "  active"
		}
		line = withTail(line, shortID(entry.ID), focusRailContentWidth(width))
		rowStyle := lipgloss.NewStyle()
		if i == m.selectedTree {
			rowStyle = selectedStyle
		}
		lines = append(lines, renderRailLine(line, width, i == m.selectedTree, false, rowStyle, i == m.selectedTree))
	}
	return fitBlock(lines, height, width)
}

func (m Model) transcriptRows(width int) []transcriptLine {
	layout := m.timelineLayout(width)
	return m.visibleTimelineRows(layout, 0, len(layout.rows), width)
}

func (m Model) timelineLayout(width int) *timelineLayout {
	if m.timelineCache == nil {
		m.timelineCache = newTimelineDisplayCache()
	}
	key := timelineLayoutKey{revision: m.timelineRevision, width: width, showReasoning: m.showReasoning, renderMarkdown: m.renderMarkdown}
	if m.timelineCache.layout != nil && m.timelineCache.layoutKey == key {
		return m.timelineCache.layout
	}
	layout := &timelineLayout{ranges: make([]timelineRowRange, len(m.timeline))}
	if len(m.timeline) == 0 {
		layout.rows = []timelineLayoutRow{{kind: timelineLayoutRowEmpty, partIndex: -1}}
		m.timelineCache.layoutKey = key
		m.timelineCache.layout = layout
		return layout
	}
	currentMessage := ""
	for i, part := range m.timeline {
		count := m.partRowCount(part, i, width)
		if count == 0 {
			layout.ranges[i] = timelineRowRange{start: -1, end: -1}
			continue
		}
		messageID := part.MessageID
		if messageID == "" {
			messageID = fmt.Sprintf("%s-%d", part.Role, i)
		}
		if messageID != currentMessage {
			if len(layout.rows) > 0 {
				layout.rows = append(layout.rows, timelineLayoutRow{kind: timelineLayoutRowSpacer, partIndex: -1})
			}
			layout.rows = append(layout.rows, timelineLayoutRow{kind: timelineLayoutRowHeader, partIndex: i})
			currentMessage = messageID
		}
		start := len(layout.rows)
		rowPartIndex := i
		if part.Kind == opencode.PartKindReasoning && !m.showReasoning {
			rowPartIndex = -1
		}
		for row := 0; row < count; row++ {
			layout.rows = append(layout.rows, timelineLayoutRow{kind: timelineLayoutRowPart, partIndex: rowPartIndex, rowIndex: row})
		}
		if rowPartIndex >= 0 {
			layout.ranges[i] = timelineRowRange{start: start, end: len(layout.rows) - 1}
		} else {
			layout.ranges[i] = timelineRowRange{start: -1, end: -1}
		}
	}
	if len(layout.rows) == 0 {
		layout.rows = []timelineLayoutRow{{kind: timelineLayoutRowEmpty, partIndex: -1}}
	}
	m.timelineCache.layoutKey = key
	m.timelineCache.layout = layout
	return layout
}

func (m Model) visibleTimelineRows(layout *timelineLayout, start, end, width int) []transcriptLine {
	if start >= end {
		return nil
	}
	partRows := make(map[int][]transcriptLine)
	rows := make([]transcriptLine, 0, end-start)
	for _, row := range layout.rows[start:end] {
		switch row.kind {
		case timelineLayoutRowEmpty:
			if len(m.timeline) == 0 {
				rows = append(rows, transcriptLine{text: "No timeline parts.", partIndex: -1})
			} else {
				rows = append(rows, transcriptLine{text: "No visible timeline parts.", partIndex: -1})
			}
		case timelineLayoutRowSpacer:
			rows = append(rows, transcriptLine{text: "", partIndex: -1})
		case timelineLayoutRowHeader:
			if row.partIndex >= 0 && row.partIndex < len(m.timeline) {
				rows = append(rows, roleHeader(m.timeline[row.partIndex], width))
			}
		case timelineLayoutRowPart:
			if row.partIndex < 0 || row.partIndex >= len(m.timeline) {
				rows = append(rows, transcriptLine{text: renderRailLine("[reasoning hidden] r to show", width, false, false, dimStyle, false), partIndex: -1})
				continue
			}
			rowsForPart, ok := partRows[row.partIndex]
			if !ok {
				rowsForPart = m.partRows(m.timeline[row.partIndex], row.partIndex, width)
				partRows[row.partIndex] = rowsForPart
			}
			if row.rowIndex >= 0 && row.rowIndex < len(rowsForPart) {
				rows = append(rows, rowsForPart[row.rowIndex])
			}
		}
	}
	return rows
}

func (m Model) partRowCount(part index.TimelinePart, partIndex, width int) int {
	return len(m.partRows(part, partIndex, width))
}

func roleHeader(part index.TimelinePart, width int) transcriptLine {
	role := strings.ToUpper(firstNonEmpty(part.Role, "message"))
	if !part.CreatedAt.IsZero() {
		role += " " + formatTime(part.CreatedAt)
	}
	return transcriptLine{text: roleStyle(part.Role).Bold(true).Render(truncatePlain(role, width)), partIndex: -1}
}

func (m Model) partRows(part index.TimelinePart, partIndex, width int) []transcriptLine {
	if isLowSignalToolLifecyclePart(part) {
		return nil
	}
	switch part.Kind {
	case opencode.PartKindText:
		text := m.cachedDisplayText(part)
		if m.renderMarkdown && isAssistantRole(part.Role) {
			return m.bodyMarkdownRows(part, text, width, partIndex, partIndex == m.selectedPart)
		}
		return bodyTextRows(text, part.Role, width, partIndex, partIndex == m.selectedPart)
	case opencode.PartKindReasoning:
		if !m.showReasoning {
			return []transcriptLine{{text: renderRailLine("[reasoning hidden] r to show", width, false, false, dimStyle, false), partIndex: -1}}
		}
		return bodyTextRows(m.cachedDisplayText(part), part.Role, width, partIndex, partIndex == m.selectedPart)
	case opencode.PartKindTool, opencode.PartKindPatch, opencode.PartKindFile:
		focused := partIndex == m.selectedPart
		style := toolStyle
		if focused {
			style = activeToolStyle
		}
		line := compactPart(part)
		return []transcriptLine{{text: renderRailLine(line, width, focused, false, style, focused), partIndex: partIndex}}
	case opencode.PartKindStepStart, opencode.PartKindStepFinish:
		return nil
	default:
		return []transcriptLine{{text: renderRailLine(compactPart(part), width, partIndex == m.selectedPart, false, dimStyle, false), partIndex: partIndex}}
	}
}

func (m Model) bodyMarkdownRows(part index.TimelinePart, text string, width int, partIndex int, focused bool) []transcriptLine {
	if text == "" {
		text = "[empty message]"
	}
	markdownRows := m.cachedMarkdownRows(part, text, max(12, focusRailContentWidth(width)))
	rows := make([]transcriptLine, 0, len(markdownRows))
	for i, line := range markdownRows {
		rows = append(rows, transcriptLine{text: renderRailStyledLine(line, width, focused, i > 0), partIndex: partIndex})
	}
	return rows
}

func isAssistantRole(role string) bool {
	return strings.EqualFold(strings.TrimSpace(role), "assistant")
}

func bodyTextRows(text, role string, width int, partIndex int, focused bool) []transcriptLine {
	if text == "" {
		text = "[empty message]"
	}
	wrapped := wrapText(text, max(12, focusRailContentWidth(width)))
	rows := make([]transcriptLine, 0, len(wrapped))
	style := roleStyle(role)
	if focused {
		style = style.Bold(true)
	}
	for i, line := range wrapped {
		rows = append(rows, transcriptLine{text: renderRailLine(line, width, focused, i > 0, style, false), partIndex: partIndex})
	}
	return rows
}

func roleStyle(role string) lipgloss.Style {
	switch strings.ToLower(role) {
	case "user":
		return userStyle
	case "assistant":
		return assistantStyle
	default:
		return dimStyle
	}
}

func displayPartText(part index.TimelinePart) string {
	text := partTextFromRawJSON(part.RawJSON)
	if text == "" {
		text = firstNonEmpty(part.Preview, part.IndexText)
	}
	return truncateRunes(text, maxTranscriptRunes)
}

func (m Model) cachedDisplayText(part index.TimelinePart) string {
	if m.timelineCache == nil {
		m.timelineCache = newTimelineDisplayCache()
	}
	key := firstNonEmpty(part.PartID, part.SourcePath, part.MessageID+":"+part.Type)
	if key == "" {
		return displayPartText(part)
	}
	if text, ok := m.timelineCache.textByPartID[key]; ok {
		return text
	}
	text := displayPartText(part)
	m.timelineCache.textByPartID[key] = text
	return text
}

func (m Model) cachedMarkdownRows(part index.TimelinePart, text string, width int) []string {
	if m.timelineCache == nil {
		m.timelineCache = newTimelineDisplayCache()
	}
	key := markdownCacheKey{partID: firstNonEmpty(part.PartID, part.SourcePath), content: text, width: width}
	if key.partID == "" {
		return assistantMarkdownRows(text, width)
	}
	if rows, ok := m.timelineCache.markdownRows[key]; ok {
		return rows
	}
	rows := assistantMarkdownRows(text, width)
	m.timelineCache.markdownRows[key] = rows
	return rows
}

func partTextFromRawJSON(raw string) string {
	if partTextFromRawJSONHook != nil {
		partTextFromRawJSONHook()
	}
	if raw == "" {
		return ""
	}
	var data struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return ""
	}
	return data.Text
}

func compactPart(part index.TimelinePart) string {
	flags := partFlags(part)
	switch part.Kind {
	case opencode.PartKindTool:
		if isLinkedTaskPart(part) {
			label := "[subagent]"
			if part.SubagentName != "" {
				label = "[subagent:" + part.SubagentName + "]"
			}
			fields := nonEmpty([]string{statusAffordance(part.Status), toolDisplayPreview(part), "opens " + part.LinkedSessionID})
			return strings.Join(append([]string{label + " " + firstNonEmpty(part.Title, part.LinkedSessionID)}, append(fields, flags...)...), " - ")
		}
		fields := nonEmpty([]string{statusAffordance(part.Status), part.Title, shortPath(part.FilePath), toolDisplayPreview(part)})
		return strings.Join(append([]string{"[tool] " + firstNonEmpty(part.ToolName, "tool")}, append(fields, flags...)...), " - ")
	case opencode.PartKindPatch:
		fields := nonEmpty([]string{part.Title, shortPath(part.FilePath), part.Preview})
		return strings.Join(append([]string{"[patch]"}, append(fields, flags...)...), " - ")
	case opencode.PartKindFile:
		fields := nonEmpty([]string{shortPath(part.FilePath), part.Preview})
		return strings.Join(append([]string{"[file]"}, append(fields, flags...)...), " - ")
	case opencode.PartKindStepStart, opencode.PartKindStepFinish:
		return strings.Join(nonEmpty([]string{"[" + string(part.Kind) + "]", part.Preview}), " ")
	default:
		return strings.Join(nonEmpty([]string{"[" + firstNonEmpty(string(part.Kind), "part") + "]", part.Preview}), " ")
	}
}

func isLowSignalToolLifecyclePart(part index.TimelinePart) bool {
	if part.Kind != opencode.PartKindTool || isLinkedTaskPart(part) {
		return false
	}
	tool := strings.ToLower(strings.TrimSpace(part.ToolName))
	if tool != "read" {
		return false
	}
	status := strings.ToLower(strings.TrimSpace(part.Status))
	if status != "started" && status != "completed" {
		return false
	}
	return strings.TrimSpace(part.Title) == "" && strings.TrimSpace(part.FilePath) == "" && toolDisplayPreview(part) == "" && len(partFlags(part)) == 0
}

func toolDisplayPreview(part index.TimelinePart) string {
	preview := strings.TrimSpace(part.Preview)
	if preview == "" {
		return ""
	}
	candidates := [][]string{
		{part.ToolName, part.Status},
		{part.ToolName, statusAffordance(part.Status)},
		{part.ToolName, part.Status, part.Title},
		{part.ToolName, statusAffordance(part.Status), part.Title},
	}
	for _, candidate := range candidates {
		if strings.EqualFold(preview, strings.Join(nonEmpty(candidate), " - ")) {
			return ""
		}
	}
	return preview
}

func statusAffordance(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return ""
	}
	lower := strings.ToLower(status)
	switch {
	case strings.Contains(lower, "complete"), strings.Contains(lower, "success"), lower == "ok":
		return "✓ " + status
	case strings.Contains(lower, "fail"), strings.Contains(lower, "error"), strings.Contains(lower, "cancel"):
		return "✗ " + status
	case strings.Contains(lower, "run"), strings.Contains(lower, "pend"), strings.Contains(lower, "start"), strings.Contains(lower, "active"):
		return "… " + status
	default:
		return "? " + status
	}
}

func partFlags(part index.TimelinePart) []string {
	var flags []string
	if part.Heavy || part.Binary || part.SkippedRaw {
		flags = append(flags, "heavy")
	}
	return flags
}

func (m Model) renderRawPart(height int) string {
	width := m.safeWidth()
	kind := firstNonEmpty(string(m.rawPart.Kind), m.rawPart.Type, "part")
	summary := firstNonEmpty(m.rawPart.ToolName, m.rawPart.Title, m.rawPart.FilePath, m.rawPart.PartID)
	title := "Part Detail (" + m.rawModeLabel() + ")"
	if m.messageDetail.active {
		title = "Message Detail (" + m.rawModeLabel() + ")"
	}
	lines := []string{
		titleStyle.Render(truncatePlain(title, width)),
		dimStyle.Render(truncatePlain(fmt.Sprintf("%s  %s  %d bytes", kind, summary, m.rawPart.SizeBytes), width)),
		dimStyle.Render(truncatePlain("Source: "+firstNonEmpty(m.rawPart.SourcePath, "unknown"), width)),
	}
	if m.rawGuard != "" && !m.messageDetail.active {
		lines = append(lines, warnStyle.Render(truncatePlain(m.rawGuard, width)))
		return fitBlock(lines, height, width)
	}

	contentLines := m.rawDisplayLines()
	contentHeight := max(1, height-len(lines)-1)
	maxScroll := max(0, len(contentLines)-contentHeight)
	start := clamp(m.rawScroll, 0, maxScroll)
	lines = append(lines, accentStyle.Render(m.rawContentHeading()))
	for _, line := range contentLines[start:min(len(contentLines), start+contentHeight)] {
		if m.messageDetail.active && !m.rawMode && m.messageDetail.renderMarkdown && isAssistantRole(m.rawPart.Role) && m.messageDetail.guard == "" {
			lines = append(lines, truncateStyled(line, width))
			continue
		}
		if !m.messageDetail.active && !m.rawMode {
			lines = append(lines, renderDetailContentLine(line, width))
			continue
		}
		lines = append(lines, truncatePlain(line, width))
	}
	return fitBlock(lines, height, width)
}

func renderDetailContentLine(line string, width int) string {
	line = truncatePlain(line, width)
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return line
	}
	if !strings.HasPrefix(line, " ") && !strings.Contains(trimmed, ":") {
		return detailHeadingStyle.Render(line)
	}
	if idx := strings.Index(line, ":"); idx > 0 && !strings.HasPrefix(line, "    ") {
		label := line[:idx+1]
		return detailLabelStyle.Render(label) + line[idx+1:]
	}
	return line
}

func (m Model) rawDisplayLines() []string {
	content := m.rawDisplayContent()
	if m.messageDetail.active || m.rawMode {
		return splitLines(content)
	}
	return wrapText(content, m.safeWidth())
}

func (m Model) rawDisplayContent() string {
	if m.messageDetail.active && !m.rawMode {
		content := m.messageDetail.sourceContent()
		if m.rawSearchQuery != "" {
			content = matchingLines(content, m.rawSearchQuery)
		}
		contentWidth := max(12, m.safeWidth())
		if m.messageDetail.renderMarkdown && isAssistantRole(m.rawPart.Role) && m.messageDetail.guard == "" {
			part := index.TimelinePart{PartID: m.rawPart.PartID, SourcePath: m.rawPart.SourcePath}
			return strings.Join(m.cachedMarkdownRows(part, content, contentWidth), "\n")
		}
		return strings.Join(wrapText(content, contentWidth), "\n")
	}
	content := renderPrettyPartDetail(m.rawPart, m.rawData)
	if m.rawMode {
		content = m.rawContent
	}
	if m.rawSearchQuery != "" {
		content = matchingLines(content, m.rawSearchQuery)
	}
	return content
}

func (m Model) rawModeLabel() string {
	if m.rawMode {
		return "raw"
	}
	return m.rawPrimaryModeLabel()
}

func (m Model) rawPrimaryModeLabel() string {
	if m.messageDetail.active {
		if m.messageDetail.renderMarkdown && isAssistantRole(m.rawPart.Role) {
			return "markdown"
		}
		return "source"
	}
	return "pretty"
}

func (m Model) rawContentHeading() string {
	if m.rawMode {
		return "Raw JSON"
	}
	if m.messageDetail.active {
		if m.rawSearchQuery != "" {
			return "Message Detail Matches"
		}
		if m.messageDetail.renderMarkdown && isAssistantRole(m.rawPart.Role) {
			return "Rendered Message"
		}
		return "Source Message"
	}
	return "Pretty Detail"
}

func (m Model) bodyHeight() int {
	reserved := 2
	if m.searchMode {
		reserved++
	}
	if m.lastErr != nil {
		reserved++
	}
	if m.searchLoading {
		reserved++
	}
	return max(1, m.height-reserved)
}

func (m Model) safeWidth() int {
	if m.width < 20 {
		return 20
	}
	return max(20, m.width-1)
}

func (m Model) sessionListRows() int {
	return max(1, m.bodyHeight()-1)
}

func (m *Model) toggleSessionListMode() {
	selectedID := m.selectedSessionID()
	if m.sessionListMode == SessionListGrouped {
		m.sessionListMode = SessionListFlat
	} else {
		m.sessionListMode = SessionListGrouped
	}
	m.selectSessionByID(selectedID, m.selectedSession)
}

func (m Model) sessionListModeLabel() string {
	if m.sessionListMode == SessionListGrouped {
		return "grouped"
	}
	return "flat"
}

func (m Model) nextSessionListModeLabel() string {
	if m.sessionListMode == SessionListGrouped {
		return "flat"
	}
	return "grouped"
}

func (m Model) markdownToggleHelp() string {
	renderMarkdown := m.renderMarkdown
	if m.mode == ViewRawPart && m.messageDetail.active {
		renderMarkdown = m.messageDetail.renderMarkdown
	}
	if renderMarkdown {
		return "m source md"
	}
	return "m render md"
}

func (m Model) sessionRows() []sessionListRow {
	if m.sessionListMode == SessionListGrouped {
		return groupedSessionRows(m.sessions)
	}
	rows := make([]sessionListRow, 0, len(m.sessions))
	for i, session := range m.sessions {
		rows = append(rows, sessionListRow{kind: sessionListRowSession, session: session, sessionIndex: i})
	}
	return rows
}

func groupedSessionRows(sessions []index.SessionSummary) []sessionListRow {
	groupsByKey := map[string]*sessionListGroup{}
	for i, session := range sessions {
		key := sessionGroupKey(session)
		group := groupsByKey[key]
		if group == nil {
			group = &sessionListGroup{key: key, label: groupName(session)}
			groupsByKey[key] = group
		}
		if session.UpdatedAt.After(group.activeAt) || group.activeAt.IsZero() {
			group.activeAt = session.UpdatedAt
		}
		group.rows = append(group.rows, sessionListRow{kind: sessionListRowSession, session: session, sessionIndex: i})
	}

	groups := make([]sessionListGroup, 0, len(groupsByKey))
	for _, group := range groupsByKey {
		sort.SliceStable(group.rows, func(i, j int) bool {
			left := group.rows[i]
			right := group.rows[j]
			if !left.session.UpdatedAt.Equal(right.session.UpdatedAt) {
				return left.session.UpdatedAt.After(right.session.UpdatedAt)
			}
			if left.session.ID != right.session.ID {
				return left.session.ID < right.session.ID
			}
			return left.sessionIndex < right.sessionIndex
		})
		groups = append(groups, *group)
	}
	sort.SliceStable(groups, func(i, j int) bool {
		if !groups[i].activeAt.Equal(groups[j].activeAt) {
			return groups[i].activeAt.After(groups[j].activeAt)
		}
		if groups[i].label != groups[j].label {
			return groups[i].label < groups[j].label
		}
		return groups[i].key < groups[j].key
	})

	rows := make([]sessionListRow, 0, len(sessions)+len(groups))
	for _, group := range groups {
		rows = append(rows, sessionListRow{kind: sessionListRowHeader, key: group.key, label: group.label, count: len(group.rows), activeAt: group.activeAt})
		rows = append(rows, group.rows...)
	}
	return rows
}

func sessionGroupKey(session index.SessionSummary) string {
	if session.ProjectID == "global" {
		return "global"
	}
	return "project:" + firstNonEmpty(session.ProjectPath, session.Directory, session.ProjectID, "unknown")
}

func (m Model) sessionHeaderLine(row sessionListRow, width int) string {
	count := fmt.Sprintf("%d sessions", row.count)
	if row.count == 1 {
		count = "1 session"
	}
	line := withTail("  "+row.label, count+"  active "+formatTime(row.activeAt), width)
	return dimStyle.Render(padPlain(line, width))
}

func (m Model) sessionListWindow(rows []sessionListRow, visible int) (int, int) {
	if len(rows) == 0 {
		return 0, 0
	}
	visible = max(1, visible)
	start := clamp(m.sessionScroll, 0, max(0, len(rows)-visible))
	selectedRow := m.selectedSessionRowIndex(rows)
	if selectedRow >= 0 {
		if selectedRow < start {
			start = selectedRow
		}
		if selectedRow >= start+visible {
			start = selectedRow - visible + 1
		}
	}
	start = clamp(start, 0, max(0, len(rows)-visible))
	return start, min(len(rows), start+visible)
}

func (m Model) selectedSessionID() string {
	if m.selectedSession < 0 || m.selectedSession >= len(m.sessions) {
		return ""
	}
	return m.sessions[m.selectedSession].ID
}

func (m *Model) selectSessionByID(sessionID string, fallback int) {
	if len(m.sessions) == 0 {
		m.selectedSession = 0
		m.sessionScroll = 0
		return
	}
	if sessionID != "" {
		for i, session := range m.sessions {
			if session.ID == sessionID {
				m.selectedSession = i
				m.ensureSelectedSessionVisible()
				return
			}
		}
	}
	m.selectedSession = clamp(fallback, 0, len(m.sessions)-1)
	m.ensureSelectedSessionVisible()
}

func (m *Model) normalizeSessionSelection() {
	if len(m.sessions) == 0 {
		m.selectedSession = 0
		m.sessionScroll = 0
		return
	}
	m.selectedSession = clamp(m.selectedSession, 0, len(m.sessions)-1)
	m.ensureSelectedSessionVisible()
}

func (m Model) selectedSessionSummary() (index.SessionSummary, bool) {
	rows := m.sessionRows()
	if row, ok := m.selectedSessionRow(rows); ok {
		return row.session, true
	}
	if len(m.sessions) == 0 {
		return index.SessionSummary{}, false
	}
	return m.sessions[clamp(m.selectedSession, 0, len(m.sessions)-1)], true
}

func (m Model) selectedSessionRow(rows []sessionListRow) (sessionListRow, bool) {
	rowIndex := m.selectedSessionRowIndex(rows)
	if rowIndex < 0 {
		return sessionListRow{}, false
	}
	return rows[rowIndex], true
}

func (m Model) selectedSessionRowIndex(rows []sessionListRow) int {
	if m.selectedSession < 0 || m.selectedSession >= len(m.sessions) {
		return -1
	}
	for i, row := range rows {
		if row.kind == sessionListRowSession && row.sessionIndex == m.selectedSession {
			return i
		}
	}
	return -1
}

func (m *Model) ensureSelectedSessionVisible() {
	rows := m.sessionRows()
	if len(rows) == 0 {
		m.sessionScroll = 0
		return
	}
	rowIndex := m.selectedSessionRowIndex(rows)
	if rowIndex < 0 {
		rowIndex = firstSelectableRowIndex(rows)
		if rowIndex < 0 {
			m.sessionScroll = 0
			return
		}
		m.selectedSession = rows[rowIndex].sessionIndex
	}
	visible := m.sessionListRows()
	if rowIndex < m.sessionScroll {
		m.sessionScroll = rowIndex
	}
	if rowIndex >= m.sessionScroll+visible {
		m.sessionScroll = rowIndex - visible + 1
	}
	m.sessionScroll = clamp(m.sessionScroll, 0, max(0, len(rows)-visible))
}

func (m *Model) moveSessionSelection(delta int) {
	rows := m.sessionRows()
	if len(rows) == 0 {
		m.normalizeSessionSelection()
		return
	}
	current := m.selectedSessionRowIndex(rows)
	target := -1
	if current < 0 {
		if delta < 0 {
			target = lastSelectableRowIndex(rows)
		} else {
			target = firstSelectableRowIndex(rows)
		}
	} else {
		target = nextSelectableRowIndex(rows, current, delta)
	}
	if target >= 0 {
		m.selectedSession = rows[target].sessionIndex
	}
	m.ensureSelectedSessionVisible()
}

func (m *Model) pageSessionSelection(delta int) {
	rows := m.sessionRows()
	if len(rows) == 0 {
		m.normalizeSessionSelection()
		return
	}
	current := m.selectedSessionRowIndex(rows)
	target := -1
	if current < 0 {
		if delta < 0 {
			target = lastSelectableRowIndex(rows)
		} else {
			target = firstSelectableRowIndex(rows)
		}
	} else {
		target = selectableRowAtOrNear(rows, clamp(current+delta, 0, len(rows)-1), delta)
	}
	if target >= 0 {
		m.selectedSession = rows[target].sessionIndex
	}
	m.ensureSelectedSessionVisible()
}

func (m *Model) jumpSessionSelection(bottom bool) {
	rows := m.sessionRows()
	target := firstSelectableRowIndex(rows)
	if bottom {
		target = lastSelectableRowIndex(rows)
	}
	if target >= 0 {
		m.selectedSession = rows[target].sessionIndex
	}
	m.ensureSelectedSessionVisible()
}

func firstSelectableRowIndex(rows []sessionListRow) int {
	for i, row := range rows {
		if row.kind == sessionListRowSession {
			return i
		}
	}
	return -1
}

func lastSelectableRowIndex(rows []sessionListRow) int {
	for i := len(rows) - 1; i >= 0; i-- {
		if rows[i].kind == sessionListRowSession {
			return i
		}
	}
	return -1
}

func nextSelectableRowIndex(rows []sessionListRow, current, delta int) int {
	if delta == 0 {
		return -1
	}
	step := 1
	if delta < 0 {
		step = -1
	}
	for i := current + step; i >= 0 && i < len(rows); i += step {
		if rows[i].kind == sessionListRowSession {
			return i
		}
	}
	return -1
}

func selectableRowAtOrNear(rows []sessionListRow, target, delta int) int {
	if len(rows) == 0 {
		return -1
	}
	if rows[target].kind == sessionListRowSession {
		return target
	}
	if delta < 0 {
		for i := target - 1; i >= 0; i-- {
			if rows[i].kind == sessionListRowSession {
				return i
			}
		}
		for i := target + 1; i < len(rows); i++ {
			if rows[i].kind == sessionListRowSession {
				return i
			}
		}
		return -1
	}
	for i := target + 1; i < len(rows); i++ {
		if rows[i].kind == sessionListRowSession {
			return i
		}
	}
	for i := target - 1; i >= 0; i-- {
		if rows[i].kind == sessionListRowSession {
			return i
		}
	}
	return -1
}

func (m Model) timelineContentHeight() int {
	return max(1, m.bodyHeight()-m.timelineHeaderHeight())
}

func (m Model) timelineHeaderHeight() int {
	if len(m.timelineStack) > 0 {
		return 3
	}
	return 2
}

func (m Model) maxTimelineScroll() int {
	layout := m.timelineLayout(m.safeWidth())
	return max(0, len(layout.rows)-m.timelineContentHeight())
}

func (m Model) timelineScrollLabel() string {
	maxScroll := m.maxTimelineScroll()
	if maxScroll == 0 {
		return "top"
	}
	position := clamp(m.timelineScroll, 0, maxScroll)
	if position == 0 {
		return fmt.Sprintf("top 0/%d", maxScroll)
	}
	if position == maxScroll {
		return fmt.Sprintf("bottom %d/%d", position, maxScroll)
	}
	return fmt.Sprintf("%d/%d", position, maxScroll)
}

func (m Model) rawContentHeight() int {
	return max(1, m.bodyHeight()-4)
}

func (m Model) maxRawScroll() int {
	return max(0, len(m.rawDisplayLines())-m.rawContentHeight())
}

func (m *Model) moveTimelineFocus(delta int) {
	if delta == 0 || len(m.timeline) == 0 {
		return
	}
	if m.selectedPart < 0 || m.selectedPart >= len(m.timeline) || !m.isFocusablePart(m.timeline[m.selectedPart]) {
		m.normalizeTimelineFocus()
	}
	if m.selectedPart < 0 {
		m.timelineScroll = clamp(m.timelineScroll+delta, 0, m.maxTimelineScroll())
		return
	}

	layout := m.timelineLayout(m.safeWidth())
	start, end := m.focusedRowRange(layout, m.selectedPart)
	visible := m.timelineContentHeight()
	m.timelineScroll = clamp(m.timelineScroll, 0, max(0, len(layout.rows)-visible))
	if start < 0 || end < 0 {
		m.selectedPart = m.firstFocusablePartInViewport()
		m.ensureFocusedPartVisible()
		return
	}
	if end < m.timelineScroll || start >= m.timelineScroll+visible {
		m.ensureFocusedPartVisible()
	}

	if delta > 0 && end >= m.timelineScroll+visible {
		m.timelineScroll = clamp(m.timelineScroll+1, 0, m.maxTimelineScroll())
		return
	}
	if delta < 0 && start < m.timelineScroll {
		m.timelineScroll = clamp(m.timelineScroll-1, 0, m.maxTimelineScroll())
		return
	}

	if next := m.nextFocusablePart(m.selectedPart, delta); next >= 0 {
		m.selectedPart = next
		m.ensureFocusedPartVisible()
	}
}

func (m *Model) normalizeTimelineFocus() {
	if m.selectedPart >= 0 && m.selectedPart < len(m.timeline) && m.isFocusablePart(m.timeline[m.selectedPart]) {
		m.ensureFocusedPartVisible()
		return
	}
	m.selectedPart = m.firstFocusablePartInViewport()
	if m.selectedPart < 0 {
		m.selectedPart = m.firstFocusablePart()
	}
	m.ensureFocusedPartVisible()
}

func (m *Model) ensureFocusedPartVisible() {
	if m.selectedPart < 0 {
		return
	}
	layout := m.timelineLayout(m.safeWidth())
	start, end := m.focusedRowRange(layout, m.selectedPart)
	if start < 0 || end < 0 {
		return
	}
	visible := m.timelineContentHeight()
	if start < m.timelineScroll {
		m.timelineScroll = start
	}
	if end >= m.timelineScroll+visible {
		if end-start+1 >= visible {
			m.timelineScroll = start
		} else {
			m.timelineScroll = end - visible + 1
		}
	}
	m.timelineScroll = clamp(m.timelineScroll, 0, m.maxTimelineScroll())
}

func (m Model) focusedRowRange(layout *timelineLayout, partIndex int) (int, int) {
	if partIndex < 0 || partIndex >= len(layout.ranges) {
		return -1, -1
	}
	rangeForPart := layout.ranges[partIndex]
	return rangeForPart.start, rangeForPart.end
}

func (m Model) firstFocusablePartInViewport() int {
	layout := m.timelineLayout(m.safeWidth())
	visible := m.timelineContentHeight()
	maxScroll := max(0, len(layout.rows)-visible)
	start := clamp(m.timelineScroll, 0, maxScroll)
	for _, row := range layout.rows[start:min(len(layout.rows), start+visible)] {
		if row.partIndex >= 0 && row.partIndex < len(m.timeline) && m.isFocusablePart(m.timeline[row.partIndex]) {
			return row.partIndex
		}
	}
	return -1
}

func (m Model) firstFocusablePart() int {
	for i, part := range m.timeline {
		if m.isFocusablePart(part) {
			return i
		}
	}
	return -1
}

func (m Model) lastFocusablePart() int {
	for i := len(m.timeline) - 1; i >= 0; i-- {
		if m.isFocusablePart(m.timeline[i]) {
			return i
		}
	}
	return -1
}

func (m Model) nextFocusablePart(current, delta int) int {
	if delta > 0 {
		for i := current + 1; i < len(m.timeline); i++ {
			if m.isFocusablePart(m.timeline[i]) {
				return i
			}
		}
		return -1
	}
	for i := current - 1; i >= 0; i-- {
		if m.isFocusablePart(m.timeline[i]) {
			return i
		}
	}
	return -1
}

func (m Model) partVisible(partIndex int) bool {
	if partIndex < 0 {
		return false
	}
	layout := m.timelineLayout(m.safeWidth())
	visible := m.timelineContentHeight()
	maxScroll := max(0, len(layout.rows)-visible)
	start := clamp(m.timelineScroll, 0, maxScroll)
	rangeStart, rangeEnd := m.focusedRowRange(layout, partIndex)
	if rangeStart < 0 || rangeEnd < 0 {
		return false
	}
	return rangeEnd >= start && rangeStart < start+visible
}

func isOpenablePart(part index.TimelinePart) bool {
	switch part.Kind {
	case opencode.PartKindText, opencode.PartKindReasoning, opencode.PartKindTool, opencode.PartKindPatch, opencode.PartKindFile:
		return true
	default:
		return false
	}
}

func isMessageDetailPart(part index.TimelinePart) bool {
	switch part.Kind {
	case opencode.PartKindText, opencode.PartKindReasoning:
		return true
	default:
		return false
	}
}

func isLinkedTaskPart(part index.TimelinePart) bool {
	return part.Kind == opencode.PartKindTool && strings.EqualFold(strings.TrimSpace(part.ToolName), "task") && strings.TrimSpace(part.LinkedSessionID) != ""
}

func (m Model) isFocusablePart(part index.TimelinePart) bool {
	switch part.Kind {
	case opencode.PartKindStepStart, opencode.PartKindStepFinish:
		return false
	case opencode.PartKindReasoning:
		return m.showReasoning
	default:
		return true
	}
}

func (m Model) modeLabel() string {
	switch m.mode {
	case ViewSessions:
		return "sessions"
	case ViewTimeline:
		return "chat"
	case ViewSessionTree:
		return "tree"
	case ViewRawPart:
		return "detail"
	default:
		return "unknown"
	}
}

func groupName(session index.SessionSummary) string {
	if session.ProjectID == "global" {
		return "Global sessions"
	}
	return firstNonEmpty(session.ProjectPath, session.Directory, session.ProjectID, "Unknown project")
}

func sourceBadge(sourceKind string) string {
	label := "[" + source.KindString(sourceKind) + "]"
	switch source.NormalizeKind(sourceKind) {
	case source.KindPi:
		label = "[pi]"
	case source.KindOpenCode:
		label = "[opencode]"
	}
	return sourceBadgeStyle.Render(label)
}

func sourceLabel(sourceKind string) string {
	return source.KindString(sourceKind)
}

func (m *Model) moveTreeSelection(delta int) {
	if len(m.treeEntries) == 0 {
		m.selectedTree = 0
		m.treeScroll = 0
		return
	}
	m.selectedTree = clamp(m.selectedTree+delta, 0, len(m.treeEntries)-1)
	m.ensureTreeSelectionVisible()
}

func (m *Model) ensureTreeSelectionVisible() {
	visible := max(1, m.bodyHeight()-2)
	m.treeScroll = clamp(m.treeScroll, 0, m.maxTreeScroll())
	if m.selectedTree < m.treeScroll {
		m.treeScroll = m.selectedTree
	}
	if m.selectedTree >= m.treeScroll+visible {
		m.treeScroll = m.selectedTree - visible + 1
	}
	m.treeScroll = clamp(m.treeScroll, 0, m.maxTreeScroll())
}

func (m Model) maxTreeScroll() int {
	return max(0, len(m.treeEntries)-max(1, m.bodyHeight()-2))
}

func (m Model) treeIndexByID(id string) int {
	if id != "" {
		for i, entry := range m.treeEntries {
			if entry.ID == id {
				return i
			}
		}
	}
	return 0
}

func treeDepths(entries []index.SessionTreeEntry) map[string]int {
	byID := make(map[string]index.SessionTreeEntry, len(entries))
	for _, entry := range entries {
		byID[entry.ID] = entry
	}
	depths := make(map[string]int, len(entries))
	var depth func(string, map[string]bool) int
	depth = func(id string, seen map[string]bool) int {
		if value, ok := depths[id]; ok {
			return value
		}
		entry, ok := byID[id]
		if !ok || entry.ParentID == "" || seen[id] {
			return 0
		}
		seen[id] = true
		value := depth(entry.ParentID, seen) + 1
		depths[id] = value
		return value
	}
	for _, entry := range entries {
		depth(entry.ID, map[string]bool{})
	}
	return depths
}

func treeChildCounts(entries []index.SessionTreeEntry) map[string]int {
	counts := map[string]int{}
	for _, entry := range entries {
		if entry.ParentID != "" {
			counts[entry.ParentID]++
		}
	}
	return counts
}

func shortID(id string) string {
	if idx := strings.LastIndex(id, ":"); idx >= 0 && idx+1 < len(id) {
		return id[idx+1:]
	}
	return id
}

func isPiSource(sourceKind string) bool {
	return source.NormalizeKind(sourceKind) == source.KindPi
}

func topLevelSessions(sessions []index.SessionSummary) []index.SessionSummary {
	out := make([]index.SessionSummary, 0, len(sessions))
	for _, session := range sessions {
		if strings.TrimSpace(session.ParentID) == "" {
			out = append(out, session)
		}
	}
	return out
}

const focusRailWidth = 2

func focusRailContentWidth(width int) int {
	return max(1, width-focusRailWidth)
}

func focusRailPrefix(focused, continuation bool) string {
	if !focused {
		return strings.Repeat(" ", focusRailWidth)
	}
	if continuation {
		return focusRailQuietStyle.Render("│") + " "
	}
	return focusRailStyle.Render("▌") + " "
}

func renderRailLine(content string, width int, focused, continuation bool, style lipgloss.Style, pad bool) string {
	contentWidth := focusRailContentWidth(width)
	content = truncatePlain(content, contentWidth)
	if pad {
		content = padPlain(content, contentWidth)
	}
	return truncateStyled(focusRailPrefix(focused, continuation)+style.Render(content), width)
}

func renderRailStyledLine(content string, width int, focused, continuation bool) string {
	content = truncateStyled(content, focusRailContentWidth(width))
	return truncateStyled(focusRailPrefix(focused, continuation)+content, width)
}

func onOff(value bool) string {
	if value {
		return "shown"
	}
	return "hidden"
}

func visibleStart(selected, visible, total int) int {
	if total <= visible || selected < visible {
		return 0
	}
	start := selected - visible + 1
	if start+visible > total {
		start = total - visible
	}
	if start < 0 {
		return 0
	}
	return start
}

func appendWrapped(lines []string, text string, width int, style lipgloss.Style) []string {
	for _, line := range wrapText(text, width) {
		lines = append(lines, style.Render(truncatePlain(line, width)))
	}
	return lines
}

func wrapText(text string, width int) []string {
	width = max(1, width)
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	if strings.TrimSpace(text) == "" {
		return []string{""}
	}
	var lines []string
	for _, rawLine := range strings.Split(text, "\n") {
		rawLine = strings.ReplaceAll(rawLine, "\t", "  ")
		if strings.TrimSpace(rawLine) == "" {
			lines = append(lines, "")
			continue
		}
		if isPreformattedLine(rawLine) {
			lines = append(lines, hardWrapLine(rawLine, width)...)
			continue
		}
		prefix := leadingSpaces(rawLine)
		trimmed := strings.TrimSpace(rawLine)
		wrapWidth := max(8, width-lipgloss.Width(prefix))
		words := strings.Fields(trimmed)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
		current := ""
		for _, word := range words {
			if current == "" {
				current = word
				continue
			}
			candidate := current + " " + word
			if lipgloss.Width(candidate) <= wrapWidth {
				current = candidate
				continue
			}
			lines = append(lines, truncatePlain(prefix+current, width))
			current = word
		}
		if current != "" {
			lines = append(lines, truncatePlain(prefix+current, width))
		}
	}
	return lines
}

func isPreformattedLine(line string) bool {
	return strings.HasPrefix(line, "  ") || strings.HasPrefix(strings.TrimSpace(line), "```")
}

func leadingSpaces(line string) string {
	count := 0
	for _, r := range line {
		if r != ' ' {
			break
		}
		count++
	}
	return strings.Repeat(" ", min(count, 12))
}

func hardWrapLine(line string, width int) []string {
	if lipgloss.Width(line) <= width {
		return []string{line}
	}
	var lines []string
	current := ""
	for _, r := range line {
		candidate := current + string(r)
		if current != "" && lipgloss.Width(candidate) > width {
			lines = append(lines, current)
			current = string(r)
			continue
		}
		current = candidate
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func fitBlock(lines []string, height, width int) string {
	return strings.Join(fitLines(lines, height, width), "\n")
}

func fitLines(lines []string, height, width int) []string {
	height = max(1, height)
	out := make([]string, 0, height)
	for i := 0; i < height; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		out = append(out, padStyled(line, width))
	}
	return out
}

func joinColumns(left, right []string, leftWidth, rightWidth, gap, height int) string {
	separator := strings.Repeat(" ", gap)
	lines := make([]string, 0, height)
	for i := 0; i < height; i++ {
		lines = append(lines, padStyled(left[i], leftWidth)+separator+padStyled(right[i], rightWidth))
	}
	return strings.Join(lines, "\n")
}

func withTail(label, tail string, width int) string {
	label = truncatePlain(label, width)
	tail = truncatePlain(tail, width)
	space := width - lipgloss.Width(label) - lipgloss.Width(tail)
	if space < 1 {
		return truncatePlain(label+" "+tail, width)
	}
	return label + strings.Repeat(" ", space) + tail
}

func padPlain(value string, width int) string {
	value = truncatePlain(value, width)
	if lipgloss.Width(value) >= width {
		return value
	}
	return value + strings.Repeat(" ", width-lipgloss.Width(value))
}

func padStyled(value string, width int) string {
	if styledWidth(value) >= width {
		return truncateStyled(value, width)
	}
	return value + strings.Repeat(" ", width-styledWidth(value))
}

func truncatePlain(value string, width int) string {
	if width <= 0 {
		return ""
	}
	value = strings.ReplaceAll(value, "\t", "  ")
	value = strings.ReplaceAll(value, "\n", " ")
	if lipgloss.Width(value) <= width {
		return value
	}
	if width <= 3 {
		return strings.Repeat(".", width)
	}
	limit := width - 3
	var out strings.Builder
	for _, r := range value {
		candidate := out.String() + string(r)
		if lipgloss.Width(candidate) > limit {
			break
		}
		out.WriteRune(r)
	}
	return out.String() + "..."
}

func truncateRunes(value string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes]) + "..."
}

func splitLines(content string) []string {
	content = strings.TrimRight(content, "\n")
	if content == "" {
		return []string{""}
	}
	return strings.Split(content, "\n")
}

func matchingLines(content, query string) string {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return content
	}
	var matches []string
	for _, line := range strings.Split(content, "\n") {
		if strings.Contains(strings.ToLower(line), query) {
			matches = append(matches, line)
		}
	}
	if len(matches) == 0 {
		return "No matches."
	}
	return strings.Join(matches, "\n")
}

func formatRawContent(content []byte) string {
	var formatted bytes.Buffer
	if json.Indent(&formatted, content, "", "  ") == nil {
		return formatted.String()
	}
	return string(content)
}

func countLabel(session index.SessionSummary) string {
	fields := []string{}
	if usage := compactTokenUsage(session.TokenUsage); usage != "" {
		fields = append(fields, usage)
	}
	fields = append(fields, fmt.Sprintf("%dm", session.MessageCount), fmt.Sprintf("%dp", session.PartCount))
	return strings.Join(fields, " ")
}

func compactTokenUsage(usage opencode.TokenUsage) string {
	if !usage.Available {
		return ""
	}
	return formatTokenCount(usage.Total) + " tok"
}

func tokenUsagePreviewLines(usage opencode.TokenUsage, width int) []string {
	if !usage.Available {
		return []string{dimStyle.Render(truncatePlain("Tokens: unavailable", width))}
	}
	return []string{
		truncatePlain(fmt.Sprintf("Tokens: total %s  input %s  output %s", formatTokenCount(usage.Total), formatTokenCount(usage.Input), formatTokenCount(usage.Output)), width),
		truncatePlain(fmt.Sprintf("        reasoning %s  cache read %s  cache write %s", formatTokenCount(usage.Reasoning), formatTokenCount(usage.CacheRead), formatTokenCount(usage.CacheWrite)), width),
	}
}

func formatTokenCount(value int64) string {
	if value >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(value)/1_000_000)
	}
	if value >= 1_000 {
		return fmt.Sprintf("%.1fk", float64(value)/1_000)
	}
	return fmt.Sprintf("%d", value)
}

func shortPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if lipgloss.Width(value) <= 48 {
		return value
	}
	base := filepath.Base(value)
	dir := filepath.Base(filepath.Dir(value))
	return filepath.Join(dir, base)
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return "unknown"
	}
	return value.Format("2006-01-02 15:04")
}

func nonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func clamp(value, low, high int) int {
	if high < low {
		return 0
	}
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
