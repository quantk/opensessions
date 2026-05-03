package opencode

import "time"

type PartKind string

const (
	PartKindUnknown    PartKind = "unknown"
	PartKindText       PartKind = "text"
	PartKindReasoning  PartKind = "reasoning"
	PartKindTool       PartKind = "tool"
	PartKindPatch      PartKind = "patch"
	PartKindFile       PartKind = "file"
	PartKindStepStart  PartKind = "step-start"
	PartKindStepFinish PartKind = "step-finish"
)

type Snapshot struct {
	Root     string
	Projects []Project
	Sessions []Session
}

type Project struct {
	ID        string
	Worktree  string
	VCS       string
	CreatedAt time.Time
	UpdatedAt time.Time
	Source    FileRecord
}

type Session struct {
	ID             string
	ProjectID      string
	ProjectPath    string
	Directory      string
	Title          string
	Slug           string
	Version        string
	ModelProvider  string
	ModelID        string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	MessageCount   int
	PartCount      int
	HeavyPartCount int
	TokenUsage     TokenUsage
	Messages       []Message
	Source         FileRecord
}

type Message struct {
	ID            string
	SessionID     string
	Role          string
	Agent         string
	SummaryTitle  string
	ModelProvider string
	ModelID       string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	TokenUsage    TokenUsage
	Parts         []Part
	Source        FileRecord
}

type TokenUsage struct {
	Available  bool
	Total      int64
	Input      int64
	Output     int64
	Reasoning  int64
	CacheRead  int64
	CacheWrite int64
}

type Part struct {
	ID         string
	SessionID  string
	MessageID  string
	Type       string
	Kind       PartKind
	ToolName   string
	Status     string
	Title      string
	FilePath   string
	MIME       string
	Filename   string
	Preview    string
	IndexText  string
	RawJSON    string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	SizeBytes  int64
	Heavy      bool
	Binary     bool
	SkippedRaw bool
	Source     FileRecord
}

type SessionDiff struct {
	ID        string
	SessionID string
	Path      string
	Source    FileRecord
}

type LocalSummary struct {
	SessionID string
	Title     string
	Tags      []string
	Bookmark  bool
}

type FileRecord struct {
	Path      string
	SizeBytes int64
	ModTime   time.Time
}
