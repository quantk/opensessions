## 1. Detail Data Path

- [x] 1.1 Add a message-detail display cap distinct from the timeline preview cap and raw JSON display cap.
- [x] 1.2 Add helpers to extract text/reasoning body content from raw JSON or safe read-only source payloads without mutating OpenCode storage.
- [x] 1.3 Return guard state and indexed-preview fallback when text content is unavailable, unsafe, binary-looking, or too large to load within the detail guard.
- [x] 1.4 Add explicit truncation metadata when message detail content exceeds the message-detail cap.

## 2. TUI Open And Render Behavior

- [x] 2.1 Make text parts openable from the timeline with `Enter` or `l` while keeping hidden reasoning parts non-openable.
- [x] 2.2 Route opened text and visible reasoning parts into the existing detail-view navigation path without changing linked task, tool, patch, or file open behavior.
- [x] 2.3 Render user text details as source text with scrolling and the message-detail truncation marker when applicable.
- [x] 2.4 Render assistant text details using the current timeline markdown/source mode at open time and support detail scrolling over the bounded content.
- [x] 2.5 Keep raw JSON toggling available only when the raw payload passes existing raw display guards, and keep detail search based on safe source/capped text for text details.
- [x] 2.6 Update detail titles/help text so text parts are labeled as message detail rather than generic tool/raw detail where visible.

## 3. Tests And Verification

- [x] 3.1 Add TUI tests that opening a long user text part shows more than the timeline preview while preserving source rendering.
- [x] 3.2 Add TUI tests that opening an assistant text part preserves rendered markdown mode and source markdown mode.
- [x] 3.3 Add TUI tests for message-detail truncation markers and unsafe/too-large guard behavior.
- [x] 3.4 Add TUI tests that hidden reasoning remains non-openable and visible reasoning follows bounded detail safeguards.
- [x] 3.5 Add regression tests confirming existing linked task, tool, patch, file, raw toggle, and timeline preview behavior remain unchanged.
- [x] 3.6 Run `gofmt` on touched Go files and `go test ./...`.
