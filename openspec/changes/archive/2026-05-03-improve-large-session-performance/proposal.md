## Why

Large OpenCode histories currently make opensession slow in two visible places: startup blocks while the entire storage/index is rescanned, and large session timelines lag because the TUI rebuilds full transcript output during normal navigation. The existing spec already requires responsive large-session rendering, but real-world sessions with thousands of parts and heavy tool payloads show that startup, search, and timeline rendering need explicit bounded and incremental behavior.

## What Changes

- Make startup avoid full unchanged-storage reprocessing by using existing scan metadata for incremental refreshes.
- Keep OpenCode storage read-only while reducing duplicate file/database scan work and unnecessary index rewrites.
- Bound normal timeline rendering and navigation to visible content plus small required context instead of rebuilding every transcript row for each repaint or keypress.
- Cache or precompute expensive display inputs such as safe text extraction and markdown-rendered rows without moving TUI presentation state into the scanner.
- Make large session and transcript search responsive enough for interactive use, avoiding full UI freezes where feasible.
- Preserve existing safety behavior for heavy/binary raw parts and the explicit raw-detail opt-in model.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `opencode-session-tui`: Clarify and strengthen performance requirements for incremental startup scans, bounded timeline rendering, and responsive large-session search/navigation.

## Impact

- `cmd/opensession`: startup flow may need to open the TUI using the existing index while refresh work is bounded or incremental.
- `internal/opencode`: scanner behavior may need metadata-aware skipping, heavy-payload short-circuiting, and duplicate source avoidance.
- `internal/index`: store APIs may need batch metadata reads, changed-record upserts, better search/indexing strategy, and fewer N+1 queries.
- `internal/tui`: timeline rendering, navigation, markdown rendering, raw detail rendering, and search handling may need caching, bounded row generation, and async/cancellable commands.
- Tests and fixtures: add performance-oriented regression coverage for large sessions, unchanged scans, heavy parts, and search/navigation behavior.
