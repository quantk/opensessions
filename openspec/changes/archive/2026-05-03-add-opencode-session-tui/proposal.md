## Why

OpenCode sessions are split across many JSON metadata and part files, and long sessions can become slow or awkward to inspect when tool artifacts contain large snapshots, patches, or binary data. A dedicated local TUI can make session discovery, navigation, and contextual search fast while keeping OpenCode storage safe and read-only.

## What Changes

- Add a Go command-line TUI for browsing OpenCode LLM chat sessions from local OpenCode storage.
- Add project-grouped session browsing with previews, timeline outlines, and read-only session detail views.
- Add fast contextual search: `/` searches sessions on the start/session-list views and searches within the current session in detail views.
- Add vim-first keyboard navigation, starting with `h`, `j`, `k`, `l`, `Enter`, `Esc`, and `q`.
- Add a local SQLite index for session metadata, safe searchable text, tool summaries, file paths, and local-only tags/bookmarks.
- Add safeguards for heavy OpenCode parts: index only safe text by default and expose full raw content only through explicit opt-in viewing.
- Add a Nix flakes devShell using nixos-unstable with Go 1.25, gopls, SQLite tools, and direnv integration; ancillary quality tools can come from the host environment.
- Use a TDD implementation approach: write failing tests before production code for scanner, index, and core TUI state behavior.

## Capabilities

### New Capabilities

- `opencode-session-tui`: Local read-only OpenCode session browser with project grouping, fast contextual search, vim-first navigation, SQLite indexing, safe handling of heavy parts, and project devShell setup.

### Modified Capabilities

- None.

## Impact

- Adds a new Go module/application in this repository.
- Adds Bubble Tea-based TUI dependencies and SQLite indexing dependencies.
- Adds Nix flake and direnv integration for reproducible local development.
- Reads OpenCode storage under the user's data directory, including `project`, `session`, `message`, `part`, `session_diff`, and related metadata directories.
- Writes only project-owned files and a local application index/tag store; the MVP must not mutate OpenCode storage.
- Requires implementation work to proceed test-first for scanner, indexing/search, and TUI state transitions.
