## 1. Background Refresh Orchestration

- [x] 1.1 Create an internal refresh/indexer orchestration boundary with typed status events for started, source phase, completed, refreshed sessions, and failed states.
- [x] 1.2 Move the existing OpenCode/Pi scan sequence out of `cmd/opensession/main.go` into the refresh orchestration while preserving source selection and read-only source access.
- [x] 1.3 Make the refresh orchestration load scan metadata, reuse existing snapshots, scan enabled sources, upsert snapshots, and emit truthful coarse phase status.
- [x] 1.4 Add focused tests for refresh orchestration behavior using temporary stores and fixture source roots where practical.

## 2. Immediate Startup Wiring

- [x] 2.1 Change `cmd/opensession/main.go` to open the index and load cached `ListSessions` before any source scan is started.
- [x] 2.2 Start the Bubble Tea TUI immediately with cached sessions, then launch the background refresh only when scanning is enabled.
- [x] 2.3 Preserve `--no-scan` so it skips background refresh entirely and uses only cached database contents.
- [x] 2.4 Ensure startup errors still fail early for config, database open, schema initialization, or initial cached session loading errors.

## 3. TUI Indexing Status and Refresh UX

- [x] 3.1 Add TUI model state and messages for background indexing status, completion, refreshed session lists, and non-fatal indexing errors.
- [x] 3.2 Render indexing status in the session list/footer or another consistently visible area, including an empty-cache indexing state.
- [x] 3.3 Apply refreshed session lists after background indexing completes while preserving the selected session when possible.
- [x] 3.4 Keep current timeline/detail navigation stable when a background refresh completes instead of forcing a view reset.
- [x] 3.5 Add TUI tests for status transitions, empty-cache startup display, no-scan display, refreshed session selection preservation, and failure status.

## 4. OpenCode DB Row-Level Freshness

- [x] 4.1 Update OpenCode database source discovery to build synthetic `FileRecord` metadata from row-level freshness data when available instead of the whole `opencode.db` mtime.
- [x] 4.2 Use `time_updated` as synthetic modification time for database-backed project, session, message, and part records, with safe fallback to database file mtime when row metadata is unavailable.
- [x] 4.3 Use `length(data)` as the size component for database-backed part records and preserve existing source path formats.
- [x] 4.4 Ensure database-backed scanning reuses unchanged rows when only the `opencode.db` file mtime changes.
- [x] 4.5 Add OpenCode scanner tests proving unchanged database rows are reused after database file mtime changes and changed rows are refreshed.

## 5. Responsiveness and Safety Validation

- [x] 5.1 Ensure expensive source scanning happens outside long SQLite write transactions where possible and that UI-visible status covers any write phase.
- [x] 5.2 Ensure background refresh cancellation or program shutdown does not panic or leak sends into a stopped Bubble Tea program.
- [x] 5.3 Verify OpenCode and Pi source fixtures are not modified during background refresh tests.
- [x] 5.4 Run `gofmt` on touched Go files.
- [x] 5.5 Run focused tests for `internal/opencode`, `internal/index`, `internal/tui`, and the new refresh orchestration.
- [x] 5.6 Run `go test ./...`.
