## Context

opensession currently starts by synchronously scanning OpenCode storage, upserting a complete normalized snapshot into the application SQLite database, listing sessions, and only then starting the Bubble Tea program. The scanner reads JSON-backed storage and, when present, the sibling `opencode.db`, then deduplicates records after both sources have already been parsed.

The local index already contains `scan_metadata` keyed by source path, size, and modification time, but startup does not use it to skip unchanged records. As a result, unchanged projects, sessions, messages, parts, searchable documents, and metadata are rewritten on every launch.

The TUI timeline currently materializes every rendered transcript row for the selected session on normal rendering and navigation, then slices the visible window. Assistant markdown rendering constructs a Glamour renderer and parses markdown repeatedly, and text display extracts content by decoding `raw_json` during the render path. This conflicts with the existing responsive large-session requirement and becomes visible on sessions with hundreds or thousands of parts.

Empirical observations from this workspace show the shape of the problem: the current OpenCode database has over sixteen thousand parts, the largest real part payload is over one hundred MiB, and the largest indexed sessions contain hundreds to more than one thousand timeline parts. Heavy raw payloads are already guarded for display, so performance work should avoid processing them eagerly rather than weakening safety behavior.

## Goals / Non-Goals

**Goals:**

- Make startup fast for unchanged or mostly unchanged OpenCode storage.
- Keep OpenCode storage strictly read-only.
- Avoid duplicate parse work when both JSON storage and `opencode.db` expose the same records.
- Keep normal list, timeline, and navigation rendering bounded to visible content plus small context needed for focus and scroll calculations.
- Keep assistant markdown rendering readable while avoiding repeated full-session markdown work.
- Make session-list search and in-session search responsive enough that the TUI does not appear hung on large indexes.
- Preserve the current explicit raw-detail flow and heavy/binary safety guards.
- Add tests that catch regressions in changed-scan behavior and bounded TUI rendering behavior.

**Non-Goals:**

- Mutating, compacting, or deleting OpenCode storage.
- Building live tailing or background watch mode for actively running sessions.
- Indexing full heavy raw payloads or making skipped binary-like content searchable by default.
- Replacing Bubble Tea or redesigning the full UI interaction model.
- Persisting ANSI-rendered markdown in SQLite; rendered output is width- and style-dependent TUI state.

## Decisions

### Use existing scan metadata for incremental refresh

Startup should compare source path, size, and modification time before parsing and upserting records. Unchanged files or database-derived records should be reused from the opensession index rather than reparsed and rewritten. Changed records should still refresh dependent summaries such as session counts, token usage, heavy part counts, searchable documents, and source metadata.

Alternatives considered: continuing full rescans is simplest but does not address startup. A separate cache file duplicates SQLite state and complicates consistency. The existing `scan_metadata` table is already the right persistence boundary and keeps local writable state inside the application database.

### Prefer one source of truth per record before parsing

When both file-backed JSON and `opencode.db` are available, the scanner should avoid doing full parse/classification work for duplicate records. The selection policy can prefer database-backed records when they are current and complete, because they are already queryable in one place, but it must remain tolerant of installations where only JSON storage exists or where one source is missing fields.

Alternatives considered: parse both and dedupe after parsing preserves current behavior but doubles work on mixed storage. Disabling one source entirely risks missing records in partially migrated storage. A source-aware discovery phase allows the implementation to keep compatibility without paying duplicate heavy-payload costs.

### Short-circuit heavy raw part processing

Part classification should identify obviously heavy payloads using source size and shallow metadata before reading or walking full raw objects where possible. For heavy parts, the scanner should preserve IDs, linkage, kind/tool/status/title, source path, size, and safety flags needed for navigation while avoiding full raw indexing and raw JSON persistence.

Alternatives considered: keep full JSON parse for every part and rely on `SkippedRaw` afterward, but that still pays the startup cost. Dropping heavy parts entirely would make timelines inaccurate. The desired middle path is metadata-rich, payload-light records.

### Keep expensive TUI rendering in width-aware caches

Timeline rendering should cache derived row/layout data by the inputs that affect it: session/timeline identity, query result identity, width, reasoning visibility, markdown mode, and relevant part content. Assistant markdown rows should be cached by part ID/content identity and width. Text extraction from safe raw JSON should happen once when timeline data is loaded or when a part-derived display cache is built, not on every `View()` call.

Alternatives considered: store rendered rows in SQLite, but rendered markdown is terminal-width and style dependent. Render only visible parts with no cache would reduce work, but scroll/focus calculations still need stable row offsets. A row-offset cache gives fast navigation while keeping presentation state in the TUI.

### Represent timeline layout as offsets plus visible rows

The timeline should maintain enough layout metadata to answer scroll bounds, focus visibility, and part-to-row mapping without rebuilding every row for every keypress. Normal rendering should produce rows only for the visible window and nearby context. Focus changes should update styling for visible rows rather than invalidating the whole transcript layout.

Alternatives considered: keep the current full `transcriptRows()` approach and only memoize its output; this is a useful intermediate step but can still create large memory spikes. A more explicit offset model better matches the requirement that rendering be bounded to visible content.

### Make searches non-blocking or indexed enough for interaction

Session and timeline search should avoid UI freezes. The implementation can combine improved SQL indexing/FTS with asynchronous Bubble Tea commands and cancellation/debounce for user-triggered searches. The repository boundary should remain the place where search details live; TUI code should handle loading/search state rather than building SQL behavior directly.

Alternatives considered: keep synchronous `LIKE '%query%'` searches, which is small but blocks the update loop and scans large content. Full FTS gives better search complexity but may require schema migration and driver validation. Async commands are useful either way because even indexed searches can be noticeable on large local databases.

### Batch local session metadata reads

Session listing should fetch tags and bookmarks in batch or through joins rather than per-session queries. This is not the largest issue, but it is a low-risk startup/list responsiveness improvement and keeps large session lists from causing N+1 query behavior.

Alternatives considered: leave N+1 queries because local SQLite is fast enough for small datasets. The code path runs at startup and after session search, so batching is preferable and simple to test.

## Risks / Trade-offs

- [Risk] Incremental scan can leave stale records when OpenCode files are deleted or source precedence changes. -> Mitigation: include deletion reconciliation for missing source paths and keep a full rescan fallback path.
- [Risk] Source precedence between JSON storage and `opencode.db` can hide records if one source is incomplete. -> Mitigation: discover IDs before parsing, prefer complete/current records, and add fixtures for file-only, db-only, and mixed duplicate cases.
- [Risk] Heavy-part shallow parsing may miss useful metadata embedded deep in raw JSON. -> Mitigation: define the minimal metadata contract for heavy parts and preserve raw explicit-open behavior when safe and available.
- [Risk] TUI caches can become stale after search, resize, markdown toggle, reasoning toggle, or nested session navigation. -> Mitigation: centralize cache invalidation around those state transitions and add model tests for each invalidating input.
- [Risk] FTS or additional SQLite indexes can increase database size and migration complexity. -> Mitigation: keep migrations additive and preserve existing `searchable_documents` semantics during transition.
- [Risk] Async search/loading introduces loading states and cancellation edge cases. -> Mitigation: keep repository APIs synchronous internally if useful, but wrap TUI calls in commands with request IDs so stale results are ignored.

## Migration Plan

The change should be additive. Existing opensession SQLite databases should continue to open. New indexes, FTS tables, cache metadata, or schema columns should be created through non-destructive schema initialization. Existing rows can be backfilled on the next scan or first search migration without touching OpenCode storage.

Rollback should be safe because OpenCode storage remains read-only. If a new local index structure causes problems, users can run with `--no-scan` against the previous database state where possible or delete the application-owned SQLite database to rebuild it.

## Open Questions

- Should startup block on a bounded incremental scan, or should the TUI open immediately from the existing index and show refresh status while scanning in the background?
- Should search move directly to SQLite FTS, or should the first implementation add async commands and query/index fixes before adopting FTS?
- What performance regression thresholds should tests assert without becoming flaky across developer machines?
