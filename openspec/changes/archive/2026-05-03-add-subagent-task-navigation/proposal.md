## Why

OpenCode subagent runs are stored as child sessions, but opensession currently treats them like ordinary top-level sessions. This makes the main session list noisy and disconnects subagent transcripts from the task tool calls that created them.

## What Changes

- Hide child/subagent sessions from the default flat and grouped session lists while keeping them indexed and readable.
- Read OpenCode parent-child session relationships from file-backed `parentID` metadata and database-backed `session.parent_id` metadata.
- Link `task` tool parts to their spawned child sessions when OpenCode exposes a child session id in task metadata.
- Make `l`/`Enter` on a linked `task` timeline row open the child session timeline instead of the task part detail view.
- Preserve access to ordinary tool, patch, file, and unlinked task part details through the existing timeline open behavior.
- Return from a child session timeline to the exact parent timeline context and selected task row.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `opencode-session-tui`: Session browsing and timeline navigation must understand OpenCode child/subagent sessions and expose them through parent task rows rather than as ordinary top-level list entries.

## Impact

- `internal/opencode`: extend tolerant session and task part scanning/classification to preserve parent session ids and linked child session ids.
- `internal/index`: persist parent-child session metadata and expose top-level session listing plus child timeline lookup/navigation data.
- `internal/tui`: adjust session list rows, task timeline row rendering, open behavior, and back navigation stack for child session timelines.
- Tests: add scanner, index, and TUI coverage for file-backed and database-backed child sessions, linked task rows, hidden top-level child sessions, and parent-return navigation.
