## Why

Tool, patch, and file part details currently default to formatted raw JSON, which is useful for debugging but slow to read when the user wants to understand what happened in a session. A pretty, quick-reading detail view will make tool activity much easier to inspect while preserving raw JSON as an explicit fallback.

## What Changes

- Add pretty default detail rendering for opened tool, patch, and file parts.
- Add tool-specific layouts for common tool shapes where the available raw part data is safe to load.
- Keep unknown or irregular tools readable through a generic summary layout.
- Add a detail-view hotkey to toggle raw JSON when safe raw content is available.
- Keep the timeline compact card rendering unchanged.
- Keep existing heavy, binary, and skipped raw guards unchanged; unsafe payloads remain summary-only.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `opencode-session-tui`: Change explicit part detail behavior so safe tool, patch, and file parts open to quick-reading structured detail views by default, with raw JSON available only through an explicit toggle.

## Impact

- Affects `internal/tui` detail rendering, detail-view state, keyboard handling, and focused model tests.
- May add tolerant parsing helpers for safe raw part JSON in the TUI layer.
- Does not change OpenCode storage access, scanner/index safety rules, database schema, or normal timeline rendering.
