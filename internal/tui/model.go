package tui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/quantick/opensession/internal/index"
	"github.com/quantick/opensession/internal/opencode"
)

const (
	MaxRawDisplayBytes = 256 * 1024
	maxTranscriptRunes = 2000
)

type ViewMode int

const (
	ViewSessions ViewMode = iota
	ViewTimeline
	ViewRawPart
)

type Repository interface {
	ListSessions(context.Context) ([]index.SessionSummary, error)
	SearchSessions(context.Context, string) ([]index.SessionSummary, error)
	SessionTimeline(context.Context, string) ([]index.TimelinePart, error)
	SearchSession(context.Context, string, string) ([]index.TimelinePart, error)
	RawPart(context.Context, string) (index.RawPart, error)
}

type Model struct {
	repo Repository

	mode            ViewMode
	sessions        []index.SessionSummary
	allSessions     []index.SessionSummary
	timeline        []index.TimelinePart
	allTimeline     []index.TimelinePart
	currentSession  index.SessionSummary
	selectedSession int
	selectedPart    int
	sessionScroll   int
	timelineScroll  int
	rawScroll       int

	searchMode     bool
	searchQuery    string
	rawSearchQuery string
	showReasoning  bool

	rawPart    index.RawPart
	rawContent string
	rawGuard   string
	lastErr    error

	width  int
	height int
}

var (
	titleStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	modeStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("183"))
	accentStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("75"))
	selectedStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("57"))
	activeToolStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("58"))
	userStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	assistantStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("111"))
	toolStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	dimStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	warnStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

func NewModel(repo Repository, sessions []index.SessionSummary) Model {
	return Model{
		repo:        repo,
		mode:        ViewSessions,
		sessions:    append([]index.SessionSummary(nil), sessions...),
		allSessions: append([]index.SessionSummary(nil), sessions...),
		width:       100,
		height:      28,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typed.Width
		m.height = typed.Height
		m.sessionScroll = clamp(m.sessionScroll, 0, max(0, len(m.sessions)-1))
		m.timelineScroll = clamp(m.timelineScroll, 0, m.maxTimelineScroll())
		if m.mode == ViewTimeline {
			m.normalizeTimelineFocus()
		}
		m.rawScroll = clamp(m.rawScroll, 0, m.maxRawScroll())
		return m, nil
	case tea.KeyMsg:
		if m.searchMode {
			return m.updateSearch(typed), nil
		}
		return m.updateKey(typed)
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
	case "tab", "shift+tab":
		return m, nil
	case "l", "enter":
		return m.openSelected(), nil
	case "h", "esc":
		return m.back(), nil
	case "r":
		if m.mode == ViewTimeline {
			m.showReasoning = !m.showReasoning
			m.normalizeTimelineFocus()
		}
		return m, nil
	default:
		return m, nil
	}
}

func (m Model) updateSearch(msg tea.KeyMsg) Model {
	switch msg.String() {
	case "esc":
		m.searchMode = false
		m.searchQuery = ""
		return m
	case "enter":
		m.searchMode = false
		return m.applySearch()
	case "backspace":
		if len(m.searchQuery) > 0 {
			runes := []rune(m.searchQuery)
			m.searchQuery = string(runes[:len(runes)-1])
		}
		return m
	default:
		if msg.Type == tea.KeyRunes {
			m.searchQuery += string(msg.Runes)
		}
		return m
	}
}

func (m Model) applySearch() Model {
	query := strings.TrimSpace(m.searchQuery)
	ctx := context.Background()
	switch m.mode {
	case ViewSessions:
		if query == "" {
			m.sessions = append([]index.SessionSummary(nil), m.allSessions...)
			m.selectedSession = clamp(m.selectedSession, 0, len(m.sessions)-1)
		} else if sessions, err := m.repo.SearchSessions(ctx, query); err != nil {
			m.lastErr = err
		} else {
			m.sessions = sessions
			m.selectedSession = clamp(m.selectedSession, 0, len(m.sessions)-1)
			m.sessionScroll = 0
			m.lastErr = nil
		}
	case ViewTimeline:
		if query == "" {
			m.timeline = append([]index.TimelinePart(nil), m.allTimeline...)
			m.timelineScroll = 0
			m.selectedPart = m.firstFocusablePartInViewport()
		} else if parts, err := m.repo.SearchSession(ctx, m.currentSession.ID, query); err != nil {
			m.lastErr = err
		} else {
			m.timeline = parts
			m.timelineScroll = 0
			m.selectedPart = m.firstFocusablePartInViewport()
			m.lastErr = nil
		}
	case ViewRawPart:
		m.rawSearchQuery = query
		m.rawScroll = 0
	}
	return m
}

func (m *Model) move(delta int) {
	switch m.mode {
	case ViewSessions:
		m.selectedSession = clamp(m.selectedSession+delta, 0, len(m.sessions)-1)
		m.sessionScroll = visibleStart(m.selectedSession, m.sessionListRows(), len(m.sessions))
	case ViewTimeline:
		m.moveTimelineFocus(delta)
	case ViewRawPart:
		m.rawScroll = clamp(m.rawScroll+delta, 0, m.maxRawScroll())
	}
}

func (m *Model) page(delta int) {
	amount := max(1, m.bodyHeight()-4)
	switch m.mode {
	case ViewSessions:
		m.selectedSession = clamp(m.selectedSession+delta*amount, 0, len(m.sessions)-1)
		m.sessionScroll = visibleStart(m.selectedSession, m.sessionListRows(), len(m.sessions))
	case ViewTimeline:
		m.timelineScroll = clamp(m.timelineScroll+delta*amount, 0, m.maxTimelineScroll())
		m.selectedPart = m.firstFocusablePartInViewport()
	case ViewRawPart:
		m.rawScroll = clamp(m.rawScroll+delta*amount, 0, m.maxRawScroll())
	}
}

func (m *Model) jump(bottom bool) {
	switch m.mode {
	case ViewSessions:
		if bottom {
			m.selectedSession = max(0, len(m.sessions)-1)
		} else {
			m.selectedSession = 0
		}
		m.sessionScroll = visibleStart(m.selectedSession, m.sessionListRows(), len(m.sessions))
	case ViewTimeline:
		if bottom {
			m.timelineScroll = m.maxTimelineScroll()
			m.selectedPart = m.lastFocusablePart()
		} else {
			m.timelineScroll = 0
			m.selectedPart = m.firstFocusablePart()
		}
		m.ensureFocusedPartVisible()
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
		if len(m.sessions) == 0 {
			return m
		}
		session := m.sessions[m.selectedSession]
		parts, err := m.repo.SessionTimeline(context.Background(), session.ID)
		if err != nil {
			m.lastErr = err
			return m
		}
		m.currentSession = session
		m.timeline = parts
		m.allTimeline = append([]index.TimelinePart(nil), parts...)
		m.timelineScroll = 0
		m.showReasoning = false
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
		return m.openRawPart(m.timeline[partIndex].PartID)
	case ViewRawPart:
		return m
	default:
		return m
	}
}

func (m Model) openRawPart(partID string) Model {
	raw, err := m.repo.RawPart(context.Background(), partID)
	if err != nil {
		m.lastErr = err
		return m
	}
	m.mode = ViewRawPart
	m.rawPart = raw
	m.rawContent = ""
	m.rawGuard = ""
	m.rawScroll = 0
	if raw.Heavy || raw.Binary || raw.SkippedRaw || raw.SizeBytes > MaxRawDisplayBytes {
		m.rawGuard = "Raw part is too large or unsafe to display normally."
		return m
	}
	if raw.RawJSON != "" {
		m.rawContent = formatRawContent([]byte(raw.RawJSON))
		return m
	}
	content, err := os.ReadFile(raw.SourcePath)
	if err != nil {
		m.rawGuard = fmt.Sprintf("Raw part could not be loaded: %v", err)
		return m
	}
	if len(content) > MaxRawDisplayBytes {
		m.rawGuard = "Raw part is too large or unsafe to display normally."
		return m
	}
	m.rawContent = formatRawContent(content)
	return m
}

func (m Model) back() Model {
	switch m.mode {
	case ViewTimeline:
		m.mode = ViewSessions
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

	height := m.height - len(sections) - 1
	if height < 1 {
		height = 1
	}
	switch m.mode {
	case ViewSessions:
		sections = append(sections, m.renderSessions(height))
	case ViewTimeline:
		sections = append(sections, m.renderTimeline(height))
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
		detail = fmt.Sprintf("%d sessions", len(m.sessions))
	case ViewTimeline:
		detail = firstNonEmpty(m.currentSession.Title, m.currentSession.ID)
	case ViewRawPart:
		detail = firstNonEmpty(m.rawPart.ToolName, m.rawPart.Title, m.rawPart.PartID)
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

func (m Model) renderFooter() string {
	var help string
	switch m.mode {
	case ViewSessions:
		help = "j/k move  l/Enter open  / search  q quit"
	case ViewTimeline:
		help = "j/k move focus  l/Enter details  r reasoning  h back  / search  q quit"
	case ViewRawPart:
		help = "j/k scroll  pgup/pgdown page  h back  / filter  q quit"
	default:
		help = "q quit"
	}
	return dimStyle.Render(truncatePlain(help, m.safeWidth()))
}

func (m Model) renderSessions(height int) string {
	width := m.safeWidth()
	if len(m.sessions) == 0 {
		return fitBlock([]string{accentStyle.Render("Sessions"), "No sessions found."}, height, width)
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
	lines := []string{accentStyle.Render(padPlain(fmt.Sprintf("Sessions %d", len(m.sessions)), width))}
	visible := max(1, height-1)
	start := visibleStart(m.selectedSession, visible, len(m.sessions))
	end := min(len(m.sessions), start+visible)
	for i := start; i < end; i++ {
		session := m.sessions[i]
		title := firstNonEmpty(session.Title, session.ID)
		if session.Bookmarked {
			title = "* " + title
		}
		label := marker(i == m.selectedSession) + " " + title
		line := withTail(label, countLabel(session), width)
		if i == m.selectedSession {
			line = selectedStyle.Render(padPlain(line, width))
		}
		lines = append(lines, line)
	}
	return lines
}

func (m Model) sessionPreviewLines(height, width int) []string {
	if len(m.sessions) == 0 {
		return []string{"No selection."}
	}
	selected := m.sessions[m.selectedSession]
	lines := []string{accentStyle.Render("Session")}
	lines = appendWrapped(lines, firstNonEmpty(selected.Title, selected.ID), width, titleStyle)
	lines = append(lines,
		dimStyle.Render(truncatePlain("Project: "+groupName(selected), width)),
		dimStyle.Render(truncatePlain("Model: "+firstNonEmpty(selected.ModelProvider, "unknown")+"/"+firstNonEmpty(selected.ModelID, "unknown"), width)),
		dimStyle.Render(truncatePlain("Updated: "+formatTime(selected.UpdatedAt), width)),
	)
	lines = append(lines, fmt.Sprintf("Messages: %d  Parts: %d  Heavy: %d", selected.MessageCount, selected.PartCount, selected.HeavyPartCount))
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
	header := []string{
		titleStyle.Render(truncatePlain(firstNonEmpty(m.currentSession.Title, m.currentSession.ID), width)),
		dimStyle.Render(truncatePlain(fmt.Sprintf("Project: %s  Model: %s/%s  Reasoning: %s  Scroll: %s", groupName(m.currentSession), firstNonEmpty(m.currentSession.ModelProvider, "unknown"), firstNonEmpty(m.currentSession.ModelID, "unknown"), onOff(m.showReasoning), m.timelineScrollLabel()), width)),
	}
	contentHeight := max(1, height-len(header))
	rows := m.transcriptRows(width)
	maxScroll := max(0, len(rows)-contentHeight)
	start := clamp(m.timelineScroll, 0, maxScroll)
	window := rows[start:min(len(rows), start+contentHeight)]
	lines := append([]string{}, header...)
	for _, row := range window {
		lines = append(lines, row.text)
	}
	return fitBlock(lines, height, width)
}

func (m Model) transcriptRows(width int) []transcriptLine {
	if len(m.timeline) == 0 {
		return []transcriptLine{{text: "No timeline parts.", partIndex: -1}}
	}
	var rows []transcriptLine
	currentMessage := ""
	for i, part := range m.timeline {
		partRows := m.partRows(part, i, width)
		if len(partRows) == 0 {
			continue
		}
		messageID := part.MessageID
		if messageID == "" {
			messageID = fmt.Sprintf("%s-%d", part.Role, i)
		}
		if messageID != currentMessage {
			if len(rows) > 0 {
				rows = append(rows, transcriptLine{text: "", partIndex: -1})
			}
			rows = append(rows, roleHeader(part, width))
			currentMessage = messageID
		}
		rows = append(rows, partRows...)
	}
	if len(rows) == 0 {
		return []transcriptLine{{text: "No visible timeline parts.", partIndex: -1}}
	}
	return rows
}

func roleHeader(part index.TimelinePart, width int) transcriptLine {
	role := strings.ToUpper(firstNonEmpty(part.Role, "message"))
	if !part.CreatedAt.IsZero() {
		role += " " + formatTime(part.CreatedAt)
	}
	return transcriptLine{text: roleStyle(part.Role).Bold(true).Render(truncatePlain(role, width)), partIndex: -1}
}

func (m Model) partRows(part index.TimelinePart, partIndex, width int) []transcriptLine {
	switch part.Kind {
	case opencode.PartKindText:
		return bodyTextRows(displayPartText(part), part.Role, width, partIndex, partIndex == m.selectedPart)
	case opencode.PartKindReasoning:
		if !m.showReasoning {
			return []transcriptLine{{text: dimStyle.Render(truncatePlain("  [reasoning hidden] r to show", width)), partIndex: -1}}
		}
		return bodyTextRows(displayPartText(part), part.Role, width, partIndex, partIndex == m.selectedPart)
	case opencode.PartKindTool, opencode.PartKindPatch, opencode.PartKindFile:
		prefix := "  "
		style := toolStyle
		if partIndex == m.selectedPart {
			prefix = "> "
			style = activeToolStyle
		}
		line := padPlain(truncatePlain(prefix+compactPart(part), width), width)
		return []transcriptLine{{text: style.Render(line), partIndex: partIndex}}
	case opencode.PartKindStepStart, opencode.PartKindStepFinish:
		return nil
	default:
		line := truncatePlain("  "+compactPart(part), width)
		return []transcriptLine{{text: dimStyle.Render(line), partIndex: partIndex}}
	}
}

func bodyTextRows(text, role string, width int, partIndex int, focused bool) []transcriptLine {
	if text == "" {
		text = "[empty message]"
	}
	wrapped := wrapText(text, max(12, width-2))
	rows := make([]transcriptLine, 0, len(wrapped))
	style := roleStyle(role)
	if focused {
		style = style.Bold(true)
	}
	for _, line := range wrapped {
		prefix := "  "
		if focused {
			prefix = "> "
		}
		rows = append(rows, transcriptLine{text: style.Render(truncatePlain(prefix+line, width)), partIndex: partIndex})
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

func partTextFromRawJSON(raw string) string {
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
		fields := nonEmpty([]string{part.Status, part.Title, shortPath(part.FilePath), part.Preview})
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
	lines := []string{
		titleStyle.Render(truncatePlain("Part Detail", width)),
		dimStyle.Render(truncatePlain(fmt.Sprintf("%s  %s  %d bytes", kind, summary, m.rawPart.SizeBytes), width)),
		dimStyle.Render(truncatePlain("Source: "+firstNonEmpty(m.rawPart.SourcePath, "unknown"), width)),
	}
	if m.rawGuard != "" {
		lines = append(lines, warnStyle.Render(truncatePlain(m.rawGuard, width)))
		return fitBlock(lines, height, width)
	}

	content := m.rawDisplayContent()
	contentLines := splitLines(content)
	contentHeight := max(1, height-len(lines)-1)
	maxScroll := max(0, len(contentLines)-contentHeight)
	start := clamp(m.rawScroll, 0, maxScroll)
	lines = append(lines, accentStyle.Render("Raw JSON"))
	for _, line := range contentLines[start:min(len(contentLines), start+contentHeight)] {
		lines = append(lines, truncatePlain(line, width))
	}
	return fitBlock(lines, height, width)
}

func (m Model) rawDisplayContent() string {
	content := m.rawContent
	if m.rawSearchQuery != "" {
		content = matchingLines(content, m.rawSearchQuery)
	}
	return content
}

func (m Model) bodyHeight() int {
	reserved := 2
	if m.searchMode {
		reserved++
	}
	if m.lastErr != nil {
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

func (m Model) timelineContentHeight() int {
	return max(1, m.bodyHeight()-2)
}

func (m Model) maxTimelineScroll() int {
	return max(0, len(m.transcriptRows(m.safeWidth()))-m.timelineContentHeight())
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
	return max(0, len(splitLines(m.rawDisplayContent()))-m.rawContentHeight())
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

	rows := m.transcriptRows(m.safeWidth())
	start, end := focusedRowRange(rows, m.selectedPart)
	visible := m.timelineContentHeight()
	m.timelineScroll = clamp(m.timelineScroll, 0, max(0, len(rows)-visible))
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
	rows := m.transcriptRows(m.safeWidth())
	start, end := focusedRowRange(rows, m.selectedPart)
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

func focusedRowRange(rows []transcriptLine, partIndex int) (int, int) {
	start := -1
	end := -1
	for i, row := range rows {
		if row.partIndex != partIndex {
			continue
		}
		if start < 0 {
			start = i
		}
		end = i
	}
	return start, end
}

func (m Model) firstFocusablePartInViewport() int {
	rows := m.transcriptRows(m.safeWidth())
	visible := m.timelineContentHeight()
	maxScroll := max(0, len(rows)-visible)
	start := clamp(m.timelineScroll, 0, maxScroll)
	for _, row := range rows[start:min(len(rows), start+visible)] {
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
	rows := m.transcriptRows(m.safeWidth())
	visible := m.timelineContentHeight()
	maxScroll := max(0, len(rows)-visible)
	start := clamp(m.timelineScroll, 0, maxScroll)
	for _, row := range rows[start:min(len(rows), start+visible)] {
		if row.partIndex == partIndex {
			return true
		}
	}
	return false
}

func isOpenablePart(part index.TimelinePart) bool {
	switch part.Kind {
	case opencode.PartKindTool, opencode.PartKindPatch, opencode.PartKindFile:
		return true
	default:
		return false
	}
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
	return firstNonEmpty(session.ProjectPath, session.ProjectID, "Unknown project")
}

func marker(selected bool) string {
	if selected {
		return ">"
	}
	return " "
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
	if lipgloss.Width(value) >= width {
		return value
	}
	return value + strings.Repeat(" ", width-lipgloss.Width(value))
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
	return fmt.Sprintf("%dm %dp", session.MessageCount, session.PartCount)
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
