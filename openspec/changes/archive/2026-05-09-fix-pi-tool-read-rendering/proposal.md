## Why

Pi sessions currently can show a single read operation as separate `started` and `completed` tool rows, unlike OpenCode sessions where a tool call is presented as one stateful tool part. Tool detail views for Pi read output can also expose raw content-block JSON instead of readable file/output text, and Pi incremental indexing can leave older sessions with no timeline parts after a newer session in the same project is refreshed.

## What Changes

- Normalize Pi tool call/result pairs into one indexed tool timeline part whenever they share a tool call ID, with final status and output preview/search text carried on the call part.
- Keep Pi tool result messages from rendering as separate timeline rows when they are associated with an indexed tool call, including cases where output is too large for raw storage.
- Render structured tool outputs such as Pi content block arrays as readable text in pretty detail views instead of JSON dumps.
- Ensure stale Pi incremental scan metadata does not preserve old split tool rows after this parser behavior changes.
- Fix Pi incremental indexing so refreshing one Pi session or project cannot delete messages/parts for unchanged sibling Pi sessions.
- Preserve existing safety limits for heavy, unsafe, binary-looking, or very large tool output.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `pi-session-source`: Pi tool call/result normalization and Pi incremental indexing behavior for parser-version changes and unchanged sibling session preservation.
- `opencode-session-tui`: Pretty tool detail output rendering for structured tool outputs across sources.

## Impact

- Affected code: `internal/pi/scanner.go`, Pi scanner tests, `internal/tui/detail.go`, TUI detail tests, and local index dirty-session, cascade deletion, scan metadata/versioning, or invalidation paths in `internal/index`/startup scanning.
- Data safety: source session files remain read-only; only the opensession SQLite index may be refreshed.
- User impact: timelines become less noisy, read tool details become easier to read, and previously indexed Pi sessions keep their timeline content after incremental scans without changing navigation keys or raw JSON toggles.
