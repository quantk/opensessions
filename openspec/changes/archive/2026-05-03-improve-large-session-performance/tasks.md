## 1. Baseline And Regression Coverage

- [x] 1.1 Add scanner/index fixtures or test builders for unchanged records, changed records, mixed JSON plus `opencode.db` duplicates, and heavy part payloads.
- [x] 1.2 Add TUI model tests that fail when timeline navigation, repaint, resize, reasoning toggle, or markdown toggle rebuilds full transcript output unnecessarily.
- [x] 1.3 Add search behavior tests for non-blocking request handling and stale search result rejection using a controllable fake repository.
- [x] 1.4 Add focused tests or benchmarks for large-session timeline rendering and unchanged startup scan behavior with stable, machine-independent assertions.

## 2. Incremental Startup Scan And Indexing

- [x] 2.1 Add batch scan metadata lookup APIs so scanner/indexing code can decide changed versus unchanged records without per-file database queries.
- [x] 2.2 Refactor scanning to perform source discovery before expensive part payload parsing and choose one current source for duplicate JSON/database records.
- [x] 2.3 Implement unchanged-record reuse so unchanged OpenCode records are not reparsed and unchanged application index rows are not rewritten.
- [x] 2.4 Implement changed-record refresh so changed records update dependent session summaries, timeline rows, searchable documents, and scan metadata.
- [x] 2.5 Add stale/deleted source reconciliation or a safe full-rescan fallback so the application index does not retain removed OpenCode records indefinitely.
- [x] 2.6 Short-circuit heavy part processing where source size or shallow metadata proves the payload should be skipped for raw indexing.

## 3. Startup Session Listing Improvements

- [x] 3.1 Replace session-list tag/bookmark N+1 queries with batch queries or joins while preserving existing `SessionSummary` output.
- [x] 3.2 Ensure `ListSessions` and `SearchSessions` continue to omit child sessions from top-level browsing after query changes.
- [x] 3.3 Keep `--no-scan`, custom `--storage-root`, and custom `--db` behavior compatible with incremental scan changes.

## 4. Bounded Timeline Rendering

- [x] 4.1 Introduce timeline display/cache state that is invalidated by timeline load, search result changes, resize, reasoning toggle, markdown toggle, and nested session navigation.
- [x] 4.2 Precompute or cache safe display text for timeline parts so `raw_json` is not decoded during normal repaint or navigation.
- [x] 4.3 Cache assistant markdown render output by part/content identity and width, with invalidation for width and markdown mode changes.
- [x] 4.4 Replace full transcript row rebuilding in scroll bounds, focus visibility, and render paths with cached row offsets plus visible-window row generation.
- [x] 4.5 Preserve existing focus semantics for text, reasoning, tool, patch, file, and linked task parts after bounded rendering changes.
- [x] 4.6 Keep raw part detail rendering guarded and explicit, with any detail-view caching scoped to the opened raw part.

## 5. Responsive Search

- [x] 5.1 Fix timeline search query planning so in-session search does not scan unrelated searchable documents for each part.
- [x] 5.2 Add or migrate to indexed search structures, such as SQLite FTS or targeted indexes, for session-list and in-session safe content search.
- [x] 5.3 Move user-triggered session and timeline searches into Bubble Tea commands with loading state and request IDs.
- [x] 5.4 Ignore stale async search results when a newer query has been submitted or the user has changed views.
- [x] 5.5 Preserve source-content search semantics for rendered assistant markdown and existing raw-view filtering behavior.

## 6. Verification

- [x] 6.1 Run focused scanner and index tests for incremental scan, duplicate source handling, heavy part skipping, and batched session metadata.
- [x] 6.2 Run focused TUI tests for bounded timeline rendering, cache invalidation, focus navigation, and async search behavior.
- [x] 6.3 Run `gofmt` on touched Go files.
- [x] 6.4 Run `go test ./...`.
- [x] 6.5 Manually smoke test `CGO_ENABLED=0 go run ./cmd/opensession --no-scan` and a normal startup against a large local storage/index.
