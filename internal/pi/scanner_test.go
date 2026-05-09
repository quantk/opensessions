package pi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/quantick/opensession/internal/opencode"
	"github.com/quantick/opensession/internal/source"
)

func TestScanParsesLinearAndBranchedPiSessions(t *testing.T) {
	root := t.TempDir()
	sessionPath := filepath.Join(root, "--tmp-project--", "2026-test.jsonl")
	mustWritePiSession(t, sessionPath, []string{
		`{"type":"session","version":3,"id":"pi-session","timestamp":"2026-05-09T10:00:00Z","cwd":"/tmp/project"}`,
		`{"type":"model_change","id":"model1","parentId":null,"timestamp":"2026-05-09T10:00:01Z","provider":"openai-codex","modelId":"gpt-5.5"}`,
		`{"type":"message","id":"u1","parentId":"model1","timestamp":"2026-05-09T10:00:02Z","message":{"role":"user","content":[{"type":"text","text":"Build Pi browser"}],"timestamp":1778320802000}}`,
		`{"type":"message","id":"a1","parentId":"u1","timestamp":"2026-05-09T10:00:03Z","message":{"role":"assistant","content":[{"type":"thinking","thinking":"private thought"},{"type":"text","text":"Assistant **markdown**"},{"type":"toolCall","id":"call1","name":"read","arguments":{"path":"README.md"}}],"provider":"openai-codex","model":"gpt-5.5","usage":{"input":10,"output":5,"reasoning":2,"cacheRead":3,"cacheWrite":1,"totalTokens":21},"timestamp":1778320803000}}`,
		`{"type":"message","id":"tr1","parentId":"a1","timestamp":"2026-05-09T10:00:04Z","message":{"role":"toolResult","toolCallId":"call1","toolName":"read","content":[{"type":"text","text":"README output"}],"isError":false,"timestamp":1778320804000}}`,
		`{"type":"compaction","id":"c1","parentId":"tr1","timestamp":"2026-05-09T10:00:05Z","summary":"compact summary","firstKeptEntryId":"a1","tokensBefore":100}`,
		`{"type":"message","id":"branch-a","parentId":"c1","timestamp":"2026-05-09T10:00:06Z","message":{"role":"user","content":"branch A"}}`,
		`{"type":"message","id":"branch-b","parentId":"c1","timestamp":"2026-05-09T10:00:07Z","message":{"role":"user","content":"branch B"}}`,
		`{"type":"label","id":"label1","parentId":"branch-b","timestamp":"2026-05-09T10:00:08Z","targetId":"branch-b","label":"chosen"}`,
		`{"type":"session_info","id":"name1","parentId":"label1","timestamp":"2026-05-09T10:00:09Z","name":"Named Pi Session"}`,
		`{"type":"message","id":"bash1","parentId":"name1","timestamp":"2026-05-09T10:00:10Z","message":{"role":"bashExecution","command":"go test ./...","output":"ok","exitCode":0,"cancelled":false,"truncated":false,"timestamp":1778320810000}}`,
		`{"type":"custom_message","id":"custom1","parentId":"bash1","timestamp":"2026-05-09T10:00:11Z","customType":"demo","content":"custom visible","display":true}`,
	})

	snapshot, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(snapshot.Sessions) != 1 {
		t.Fatalf("sessions = %d", len(snapshot.Sessions))
	}
	session := snapshot.Sessions[0]
	if session.SourceKind != string(source.KindPi) || session.ID != "pi:pi-session" || session.ProjectPath != "/tmp/project" || session.Title != "Named Pi Session" {
		t.Fatalf("session metadata = %#v", session)
	}
	if !session.TokenUsage.Available || session.TokenUsage.Total != 21 || session.TokenUsage.CacheRead != 3 || session.TokenUsage.CacheWrite != 1 {
		t.Fatalf("token usage = %#v", session.TokenUsage)
	}
	if session.MessageCount != len(session.Messages) || session.PartCount == 0 {
		t.Fatalf("counts = messages:%d/%d parts:%d", session.MessageCount, len(session.Messages), session.PartCount)
	}
	branchB := findMessage(t, session, "pi:pi-session:branch-b")
	if branchB.ParentID != "pi:pi-session:c1" || branchB.Label != "chosen" {
		t.Fatalf("branch metadata = %#v", branchB)
	}
	assistant := findMessage(t, session, "pi:pi-session:a1")
	if assistant.ModelProvider != "openai-codex" || assistant.ModelID != "gpt-5.5" {
		t.Fatalf("assistant model = %#v", assistant)
	}
	readTool := findToolPart(assistant, "read")
	if findPartKind(assistant, opencode.PartKindReasoning) == nil || readTool == nil {
		t.Fatalf("assistant parts = %#v", assistant.Parts)
	}
	if readTool.Status != "completed" || !strings.Contains(readTool.IndexText, "README output") || !strings.Contains(readTool.RawJSON, "README output") {
		t.Fatalf("tool result was not merged into call: %#v", readTool)
	}
	toolResult := findMessage(t, session, "pi:pi-session:tr1")
	if len(toolResult.Parts) != 0 {
		t.Fatalf("tool result message should not render as a separate timeline part: %#v", toolResult.Parts)
	}
	bash := findMessage(t, session, "pi:pi-session:bash1")
	if part := findToolPart(bash, "bash"); part == nil || !strings.Contains(part.IndexText, "go test") {
		t.Fatalf("bash part = %#v", bash.Parts)
	}
}

func TestScanWithMetadataReusesUnchangedPiSession(t *testing.T) {
	root := t.TempDir()
	sessionPath := filepath.Join(root, "project", "session.jsonl")
	mustWritePiSession(t, sessionPath, []string{
		`{"type":"session","id":"reuse","timestamp":"2026-05-09T10:00:00Z","cwd":"/tmp/reuse"}`,
		`{"type":"message","id":"u1","parentId":null,"timestamp":"2026-05-09T10:00:01Z","message":{"role":"user","content":"new text would fail if reparsed"}}`,
	})
	info, err := os.Stat(sessionPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	record := opencode.FileRecord{Path: sessionPath, SizeBytes: info.Size(), ModTime: info.ModTime()}
	existing := opencode.Snapshot{Sessions: []opencode.Session{{SourceKind: string(source.KindPi), ID: "pi:reuse", ProjectPath: "/tmp/reuse", Title: "reused", Source: record}}}
	snapshot, err := ScanWithMetadata(root, map[string]opencode.FileRecord{sessionPath: record}, existing)
	if err != nil {
		t.Fatalf("ScanWithMetadata: %v", err)
	}
	if len(snapshot.Sessions) != 1 || snapshot.Sessions[0].Title != "reused" {
		t.Fatalf("reused snapshot = %#v", snapshot.Sessions)
	}
}

func mustWritePiSession(t *testing.T, path string, lines []string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write session: %v", err)
	}
}

func findMessage(t *testing.T, session opencode.Session, id string) opencode.Message {
	t.Helper()
	for _, message := range session.Messages {
		if message.ID == id {
			return message
		}
	}
	t.Fatalf("message %s not found", id)
	return opencode.Message{}
}

func findPartKind(message opencode.Message, kind opencode.PartKind) *opencode.Part {
	for i := range message.Parts {
		if message.Parts[i].Kind == kind {
			return &message.Parts[i]
		}
	}
	return nil
}

func findToolPart(message opencode.Message, tool string) *opencode.Part {
	for i := range message.Parts {
		if message.Parts[i].ToolName == tool {
			return &message.Parts[i]
		}
	}
	return nil
}
