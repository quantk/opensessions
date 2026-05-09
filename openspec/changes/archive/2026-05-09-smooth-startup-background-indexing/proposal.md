## Why

opensession currently performs source discovery, scan, and index upsert before starting the Bubble Tea program, so users can see a long blank pause at launch. This is especially painful when OpenCode's `opencode.db` changed, because DB-backed records can be treated as dirty through the database file mtime and many rows may be re-parsed before the UI appears.

## What Changes

- Start the TUI immediately using cached sessions from the opensession index when available.
- Move normal source scanning and index refresh work into a background job after TUI startup.
- Surface indexing status, progress, completion, and failure states in the TUI while keeping navigation/search responsive.
- Refresh visible session data after background indexing completes without discarding local UI context unnecessarily.
- Improve OpenCode SQLite source freshness so unchanged database rows can be reused even when the `opencode.db` file mtime changes.
- Preserve read-only access to OpenCode/Pi source storage and keep writable state in the opensession SQLite database.
- Keep `--no-scan` behavior as an explicit way to skip background scanning entirely.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `opencode-session-tui`: Startup, indexing, and status requirements change so the TUI can open immediately from cached data while indexing continues in the background, and OpenCode database-backed indexing avoids unnecessary full reprocessing of unchanged rows.

## Impact

- `cmd/opensession/main.go`: Startup sequencing changes from blocking pre-TUI scan to cached load plus background refresh wiring.
- `internal/tui`: Model/repository integration gains indexing status messages and session refresh behavior.
- `internal/index`: May need scan/index metadata extensions for background status and DB row-level freshness.
- `internal/opencode`: SQLite-backed source discovery/scanning needs row-level freshness metadata rather than relying only on whole-database mtime.
- Tests: Add coverage for immediate TUI startup, background status transitions, refreshed session lists, `--no-scan`, and unchanged OpenCode DB row reuse.
