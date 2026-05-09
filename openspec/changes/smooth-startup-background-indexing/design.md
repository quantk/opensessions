## Context

Startup currently performs source discovery, metadata lookup, full opensession snapshot loading, OpenCode/Pi scanning, and SQLite upsert before starting the Bubble Tea program. On large local histories this creates a visible launch pause. The cost is amplified for OpenCode `opencode.db`: DB-backed synthetic source records currently use the whole database file mtime as their modification time, so a small OpenCode update can make many DB rows appear changed and force broad reclassification before the UI appears.

The application already has an application-owned SQLite index that can serve cached sessions quickly, async TUI search patterns that ignore stale results, and scan metadata used to reuse unchanged records. OpenCode and Pi source storage must remain read-only; all writable state stays in the opensession database.

## Goals / Non-Goals

**Goals:**

- Make normal startup show the TUI immediately after config resolution, schema initialization, and cached `ListSessions`.
- Run source scanning and index refresh in the background when scanning is enabled.
- Display background indexing state in the TUI without exposing scanner internals in rendering code.
- Refresh cached session lists after indexing completes while preserving useful UI context where possible.
- Avoid unnecessary OpenCode SQLite row reprocessing when `opencode.db` file mtime changes but individual rows did not.
- Preserve existing read-only guarantees for source storage and existing `--no-scan` semantics.

**Non-Goals:**

- Replacing SQLite or introducing a daemon/watch service.
- Real-time file watching or continuously tracking OpenCode writes after the startup refresh completes.
- Full-text-search engine migration; existing search behavior can remain `LIKE`-based unless separately changed.
- Perfect progress percentages for every scan phase. Coarse, truthful status is sufficient.
- Making every repository query non-blocking; user-triggered long queries can continue to use existing async command patterns.

## Decisions

### 1. Start from cached data, then refresh in the background

`cmd/opensession/main.go` should open the index, load `store.ListSessions(ctx)`, construct the TUI model, and start Bubble Tea without waiting for source scans. When `NoScan` is false, main should start a background refresh worker that reports typed status events to the TUI.

Alternatives considered:

- Keep blocking scan and add terminal logging. This explains the pause but does not make launch smooth.
- Move all startup orchestration into `internal/tui`. This would leak scanner/source details into the UI layer.

### 2. Introduce a small refresh/indexer orchestration boundary

Add a small internal orchestration layer, for example `internal/indexer` or `internal/refresh`, that owns the current scan sequence for enabled sources:

1. Discover source paths.
2. Load scan metadata.
3. Load existing indexed snapshot for reuse.
4. Scan source using metadata.
5. Upsert snapshot.
6. Emit source-level and phase-level status events.

The TUI should receive simple events/messages such as started, phase changed, completed, failed, and sessions refreshed. It should not call OpenCode/Pi scanners directly.

Alternatives considered:

- Put status callbacks inside `internal/opencode` and `internal/pi`. That gives fine-grained progress but spreads UI-oriented concerns through scanners.
- Expose scanner internals through `tui.Repository`. That makes the UI harder to test and violates the existing layering.

### 3. Use Bubble Tea messages for status and refresh

The TUI model should gain indexing status fields and handle messages for refresh progress, completion, errors, and refreshed session results. The background worker can send messages through `tea.Program.Send`, or main can wire an event channel into a model command that drains events. In either case, rendering remains driven by normal Bubble Tea messages.

On completion, the worker should request or include a fresh `ListSessions` result. The model should replace `allSessions` and `sessions` according to the current search state:

- No active search query: replace visible sessions and try to preserve the selected session ID.
- Search results visible: either rerun the current search asynchronously or mark cached search results as potentially stale until the next search. Prefer rerunning if the query is still known.
- Timeline/detail views: do not forcibly navigate away; update session list cache and status, and let opened timelines remain stable unless the user refreshes/reopens.

Alternatives considered:

- Full model reload after indexing. Simpler, but disruptive to current navigation context.
- Always rerun all visible queries/timelines after indexing. More consistent, but can create extra work and unexpected focus changes.

### 4. Represent OpenCode DB-backed freshness with row-level synthetic metadata

For synthetic paths like `opencode.db#part/<id>.json`, use row-level data to build `FileRecord` freshness instead of the whole DB file mtime alone:

- `project`, `session`, `message`: use the row `time_updated` as synthetic `ModTime`; use a stable row size when cheap/available or zero if no data payload exists.
- `part`: use `length(data)` as `SizeBytes` and row `time_updated` as synthetic `ModTime`.
- Fall back to the database file mtime only when required row metadata is unavailable.

This keeps the existing `source path + size + mod time` scan metadata contract but makes DB-backed records dirty only when their row-level fingerprint changes.

Alternatives considered:

- Hash every DB row's JSON data during discovery. Stronger correctness, but discovery becomes expensive and defeats the purpose for large `opencode.db` files.
- Keep the database mtime and rely only on background scanning. This improves perceived launch but leaves a large recurring refresh cost.
- Store a new opaque fingerprint column. Useful later, but the row-level `ModTime` approach can fit existing metadata with less migration complexity.

### 5. Keep first-run behavior simple and honest

When the local index is empty, the TUI should still open immediately and show an empty/loading state. First-run indexing may still take time, but the user sees status and can quit normally.

Alternatives considered:

- Block only on first run. This avoids empty UI but preserves the worst startup experience for new users.

## Risks / Trade-offs

- Background upsert may briefly contend with UI reads in the opensession SQLite database → keep expensive source scanning outside write transactions, keep status visible during write phases, and avoid forcing synchronous UI reloads while writes are active.
- Cached data can be stale while indexing runs → explicitly label status as showing cached results until refresh completes.
- Row-level freshness relies on OpenCode `time_updated` correctness → use size plus updated time for parts, and fall back safely when row metadata is missing.
- Bubble Tea message delivery from a goroutine can race with program shutdown → use context cancellation and tolerate send failures/no-ops after quit.
- Progress counts can be misleading if too detailed → report phase/source/counts only when known; otherwise use indeterminate wording.
- Session IDs from multiple sources may collide in existing schema assumptions → preserve existing source-aware behavior and avoid expanding scope unless tests expose a regression.

## Migration Plan

1. Existing databases keep working with current schema.
2. New OpenCode DB-backed scan metadata will be written with row-level synthetic modification times as records are refreshed.
3. On the first run after the change, rows whose old metadata used DB file mtime may be treated as changed once and rewritten with row-level metadata.
4. Subsequent runs should reuse unchanged DB-backed rows even if `opencode.db` mtime changes.
5. Rollback is safe: older code may treat row-level metadata as different from DB mtime and rescan, but source storage remains read-only and local tags/bookmarks stay in the opensession database.

## Open Questions

- Should the status footer include exact counts for discovered/changed records, or only source/phase text for the initial version?
- Should a completed background refresh automatically rerun an active session-list search, or should it only mark results as stale until the user searches again?
- Is a future explicit manual refresh key desirable, or should this change stay limited to startup refresh?
