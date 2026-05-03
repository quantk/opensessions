## 1. Scanner Data Model

- [x] 1.1 Add token usage fields to OpenCode message and session models, including an explicit availability flag.
- [x] 1.2 Parse assistant message `tokens` metadata from file-backed OpenCode message JSON.
- [x] 1.3 Parse assistant message `tokens` metadata from `opencode.db` message data.
- [x] 1.4 Aggregate message token usage into session-level totals during session assembly, using OpenCode `tokens.total` when present and derived totals otherwise.
- [x] 1.5 Add scanner fixture coverage for available usage, unavailable usage, cache token counts, and missing `tokens.total`.

## 2. Index Persistence

- [x] 2.1 Add non-destructive `sessions` table columns for token usage availability and aggregate token counts.
- [x] 2.2 Persist session token usage aggregates during snapshot upsert.
- [x] 2.3 Expose token usage fields through `SessionSummary` queries.
- [x] 2.4 Add index tests for persistence, idempotent upsert, and unavailable usage.

## 3. TUI Rendering

- [x] 3.1 Render compact total token usage in session list rows only when usage is available.
- [x] 3.2 Render session token usage breakdown in the session preview pane.
- [x] 3.3 Render compact session token usage in the session detail header.
- [x] 3.4 Ensure monetary cost is not rendered even when source metadata contains cost.
- [x] 3.5 Add TUI tests for available usage, unavailable usage, and cache token display.

## 4. Verification

- [x] 4.1 Run `gofmt` on touched Go files.
- [x] 4.2 Run focused scanner, index, and TUI tests for token usage behavior.
- [x] 4.3 Run `go test ./...`.
