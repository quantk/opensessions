## Why

Long user and assistant messages are intentionally bounded in the session timeline to keep navigation responsive, but today there is no way to inspect a selected text message beyond that short timeline preview. Users need an explicit detail view for text parts that exposes substantially more content while preserving the existing safety and rendering constraints.

## What Changes

- Allow focused user and assistant text parts to open with `Enter` or `l` from the session timeline.
- Add a scrollable text message detail view that uses a larger bounded content limit than the timeline preview and clearly indicates when content is still truncated.
- Preserve the current assistant markdown/source display choice when opening assistant text details, and keep user text rendered as source text.
- Keep heavy, unsafe, binary-looking, or overly large content guarded so the TUI does not attempt unbounded loading or markdown rendering.
- Keep existing tool, patch, file, linked task, reasoning visibility, raw JSON, and timeline preview behavior intact.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `opencode-session-tui`: extend timeline open behavior and detail viewing requirements so safe text parts can be inspected in a bounded scrollable detail view.

## Impact

- Affects `internal/tui` timeline open handling, detail rendering, scrolling/search behavior, markdown/source toggling, and tests.
- May read safe text part raw JSON or source payloads through existing read-only scanner/index paths, but must not mutate OpenCode storage.
- No CLI flags, database schema changes, external APIs, or new runtime dependencies are expected.
