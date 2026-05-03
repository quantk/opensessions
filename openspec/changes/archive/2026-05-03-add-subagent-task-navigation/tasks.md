## 1. Scanner Metadata

- [x] 1.1 Add parent session id and linked child session id fields to scanner/index-facing models.
- [x] 1.2 Parse file-backed session `parentID` metadata and database-backed `session.parent_id` metadata.
- [x] 1.3 Extract linked child session ids from `task` tool part metadata when present.
- [x] 1.4 Add scanner fixture coverage for file-backed child sessions, database-backed child sessions, and linked task parts.

## 2. Index Persistence And Queries

- [x] 2.1 Add non-destructive SQLite schema support for session parent ids and part linked session ids.
- [x] 2.2 Persist parent session ids and linked child session ids during snapshot upsert.
- [x] 2.3 Update top-level session list and search queries to omit sessions with explicit parent ids.
- [x] 2.4 Expose linked child session metadata through timeline part queries without changing raw part guards.
- [x] 2.5 Add index tests for hidden child sessions, persisted linked task rows, and child timeline lookup.

## 3. Timeline Navigation

- [x] 3.1 Add TUI state needed to push and restore parent timeline context when opening a child session.
- [x] 3.2 Make `l`/`Enter` on a linked `task` row open the child session timeline instead of part detail.
- [x] 3.3 Keep unlinked task rows and ordinary tool, patch, and file rows opening detail views as before.
- [x] 3.4 Render a compact subagent affordance on linked task timeline rows and nested context in child timeline headers.
- [x] 3.5 Make `h`/`Esc` from a child timeline restore the parent timeline with the task row selected or visible.

## 4. Behavior Coverage

- [x] 4.1 Add TUI tests confirming child sessions are hidden from flat and grouped session lists.
- [x] 4.2 Add TUI tests confirming linked task rows open child timelines and back navigation restores parent context.
- [x] 4.3 Add TUI tests confirming unlinked task rows and ordinary tool rows still open detail views.
- [x] 4.4 Verify child timeline search and browsing remain read-only and bounded like ordinary timelines.

## 5. Verification

- [x] 5.1 Run `gofmt` on touched Go files.
- [x] 5.2 Run focused scanner, index, and TUI tests for subagent task navigation behavior.
- [x] 5.3 Run `go test ./...`.
