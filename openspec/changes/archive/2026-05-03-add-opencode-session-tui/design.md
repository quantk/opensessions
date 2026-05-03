## Context

The repository is currently an empty project scaffold with OpenSpec configuration. The tool being proposed is a new local Go application, tentatively `opensession`, for inspecting OpenCode LLM chat sessions.

Current OpenCode storage is file-based and split by concern:

```text
~/.local/share/opencode/storage
├── project/*.json
├── session/{global,<project-id>}/*.json
├── message/<session-id>/*.json
├── part/<message-id>/*.json
├── session_diff/*.json
└── todo/*.json
```

Empirical storage observations from this machine:

- 44 sessions, 1295 message files, 5450 part files.
- Median session part data is about 60 KiB, p95 is about 2.3 MiB.
- One session is about 157 MiB because an `apply_patch` tool part stored a large binary `before`/`diff` payload in metadata.
- The main performance risk is heavy tool artifacts, not ordinary chat text.

## Goals / Non-Goals

**Goals:**

- Provide a fast local TUI for browsing OpenCode sessions grouped by project.
- Support read-only OpenCode storage ingestion from the user's local data directory.
- Provide session preview, session timeline, collapsed tool/reasoning blocks, and detail views.
- Provide context-sensitive `/` search for session lists and session detail views.
- Provide vim-first keyboard navigation from the first MVP.
- Store index data, tags, and bookmarks in an application-owned SQLite database.
- Avoid reading, indexing, or rendering large raw tool artifacts unless the user explicitly opens them.
- Provide a reproducible Nix flakes devShell with Go 1.25, gopls, SQLite tooling, and direnv integration while allowing ancillary quality tools to come from the host environment.
- Use TDD for scanner, index/search, and core TUI state-machine behavior.

**Non-Goals:**

- Mutating OpenCode storage in the MVP.
- Deleting, archiving, renaming, or compacting OpenCode sessions in the MVP.
- Live tailing currently running sessions in the MVP.
- Rendering every raw field from large tool parts in normal timeline views.
- Building a general-purpose log viewer unrelated to OpenCode storage.

## Decisions

### Use Go with Bubble Tea for the TUI

The application will be a Go command-line tool using Bubble Tea, Bubbles, and Lip Gloss for the interface. This matches the requested stack and provides a pragmatic model for list/detail panes, keyboard-driven state transitions, and viewport rendering.

Alternatives considered: `tview/tcell` would provide more widgets out of the box, and raw `tcell` would provide maximum rendering control. Bubble Tea is preferred because the MVP needs a focused keyboard workflow more than a large widget toolkit.

### Treat OpenCode storage as read-only source data

The scanner will read OpenCode storage files but never write to them. Session metadata, messages, and parts are source-of-truth inputs; local tags, bookmarks, and index state live in the application's own SQLite database.

This keeps the MVP safe even if OpenCode changes its storage shape or writes files while the tool is running.

### Use an application-owned SQLite database

The local database will store normalized projects, sessions, messages, parts, searchable documents, tags, bookmarks, and scan metadata. The default location should follow XDG state conventions, for example `$XDG_STATE_HOME/opensession/opensession.sqlite` or `~/.local/state/opensession/opensession.sqlite`, with an override flag or environment variable.

SQLite is preferred because the tool is local-first, single-user, offline, and benefits from indexed search without requiring a daemon.

### Use safe indexing for heavy parts

The scanner must classify part files before indexing raw content. Small text fields from `text`, `reasoning`, safe tool summaries, file paths, tool names, statuses, and titles can be indexed. Large fields such as tool `state.metadata`, binary-looking `before`/`after`, large diffs, snapshots, or raw outputs must be skipped or represented as metadata with size/type indicators.

Full raw viewing is explicit opt-in from a detail view and is separate from normal list/timeline rendering.

### Search is context-sensitive

The `/` key enters search mode for the active view:

- On start/session-list views, search filters sessions by title, project path, model/provider, tool summaries, file paths, tags, and safe indexed chat text.
- In a session detail view, search matches only the current session timeline and safe indexed content.
- In a raw part view, search applies to the loaded raw content if it was explicitly opened.

This keeps `/` vim-like and predictable without requiring separate global and local search keys in the MVP.

### Vim-first navigation is the primary interaction model

The first MVP will support `h`, `j`, `k`, `l`, `/`, `Enter`, `Esc`, and `q`. Mouse support is optional and non-blocking.

```text
Projects/Sessions        Timeline              Part Detail
      │                     │                     │
      ├── l / Enter ───────▶│── l / Enter ───────▶│
      │◀──── h / Esc ───────┤◀──── h / Esc ───────┤
      │                     │                     │
      j/k select            j/k select            j/k scroll
      / search              / search              / search in part
```

### Prefer incremental scans

The scanner should use path, size, and modification time to detect changed files. The first implementation can run a scan at startup, but the schema should support incremental refreshes so future live/offline refresh modes do not require a redesign.

### Keep rendering virtualized and lazy

The TUI should render only visible rows and only load detail content when needed. Session lists should come from SQLite summaries. Session timelines should use indexed rows and bounded previews, not concatenated full transcripts.

### Develop test-first

Implementation should follow TDD for the core behavior. Each scanner rule, index/search behavior, and TUI state transition should begin with a failing test or fixture-driven assertion before production code is added. Manual exploratory testing is still required for the terminal UI, but it does not replace automated tests for state and data behavior.

This is especially important because OpenCode storage contains irregular real-world JSON shapes and very large tool parts; tests should lock in safe parsing and skipping behavior before optimizing UI rendering.

## Risks / Trade-offs

- OpenCode storage format can change → Keep parsing tolerant, preserve unknown metadata only as optional raw references, and make scanner failures visible without crashing the UI.
- Heavy JSON files can block startup if parsed eagerly → Use file size thresholds, bounded reads where possible, and skip full raw indexing for heavy parts.
- SQLite FTS driver choice can affect portability → Start with the smallest reliable SQLite integration that supports the required search behavior under Nix; keep database access behind a small package boundary.
- Search may miss skipped heavy raw fields → Make skipped/heavy indicators visible so users understand why some raw content is not searchable by default.
- Tags/bookmarks introduce writes → Ensure writes go only to the application database and never to OpenCode storage.
- TDD can slow the first UI slice → Keep tests focused on pure scanner/index/model/update functions and use manual checks only for terminal rendering details.

## Migration Plan

This is a new project, so no application data migration is required for the first implementation. The first run creates the local SQLite database and indexes existing OpenCode storage. Rollback is deleting the application binary and its application-owned SQLite state; OpenCode storage remains unchanged.

## Open Questions

- Exact binary name: use `opensession` unless renamed during implementation.
- Exact SQLite driver: choose during implementation based on FTS support and Nix compatibility.
- Exact default index path: confirm whether XDG state or cache is preferred before implementation if this matters.
