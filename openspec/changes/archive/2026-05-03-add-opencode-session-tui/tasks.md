## 1. Project Setup

- [x] 1.1 Add Nix flake devShell using nixos-unstable with Go 1.25, gopls, SQLite tools, and direnv integration; use host-provided ancillary tools.
- [x] 1.2 Initialize the Go module and add a minimal `opensession` CLI entry point.
- [x] 1.3 Add Bubble Tea, Bubbles, Lip Gloss, SQLite, and test dependencies needed for the MVP.
- [x] 1.4 Add basic project layout for storage scanning, indexing, TUI views, and command/config handling.

## 2. OpenCode Storage Scanner

- [x] 2.0 Write failing scanner tests/fixtures before implementing scanner behavior for each supported OpenCode file type.
- [x] 2.1 Implement storage root discovery for the default OpenCode data path plus an override flag or environment variable.
- [x] 2.2 Define tolerant Go models for project, session, message, part, session diff, and local summary records.
- [x] 2.3 Implement read-only scanning for `project`, `session`, `message`, and `part` directories without writing to OpenCode storage.
- [x] 2.4 Implement chronological session assembly from session metadata, message metadata, and part files.
- [x] 2.5 Implement safe part classification for text, reasoning, tool, patch, file, step-start, and step-finish parts.
- [x] 2.6 Implement heavy/binary part detection that records summaries and sizes without indexing full raw payloads by default.
- [x] 2.7 Add scanner fixtures and tests covering ordinary text parts, reasoning parts, tool summaries, file paths, and a heavy tool artifact.

## 3. SQLite Index And Local State

- [x] 3.0 Write failing database/search tests before implementing schema, indexing, search scopes, tags, and bookmarks.
- [x] 3.1 Create SQLite schema for projects, sessions, messages, parts, searchable documents, scan metadata, tags, and bookmarks.
- [x] 3.2 Implement database initialization at the application-owned XDG state path with an override option.
- [x] 3.3 Implement idempotent upserts for scanned projects, sessions, messages, parts, and safe searchable documents.
- [x] 3.4 Implement incremental scan metadata using file path, size, and modification time.
- [x] 3.5 Implement session-list search over titles, project paths, file paths, tool summaries, tags, and safe indexed chat text.
- [x] 3.6 Implement current-session search over that session's indexed timeline content.
- [x] 3.7 Implement local-only tag and bookmark persistence without mutating OpenCode storage.
- [x] 3.8 Add database tests for initialization, upserts, search scopes, tags, bookmarks, and skipped heavy payloads.

## 4. Bubble Tea TUI

- [x] 4.0 Write failing tests for core TUI model update/state transitions before implementing each navigation and search behavior.
- [x] 4.1 Implement start/session-list view grouped by project with global sessions in a distinct group.
- [x] 4.2 Implement session preview panel with title, project path, model/provider, timestamps, message counts, part counts, and heavy-part indicators.
- [x] 4.3 Implement session timeline view with user messages, assistant messages, tool events, file references, and bounded previews.
- [x] 4.4 Hide reasoning parts by default and add an explicit toggle or detail action to reveal selected reasoning content.
- [x] 4.5 Implement vim-first navigation with `h`, `j`, `k`, `l`, `/`, `Enter`, `Esc`, and `q`.
- [x] 4.6 Implement context-sensitive `/` search for session lists and current session detail views.
- [x] 4.7 Implement raw part detail view as explicit opt-in, with guards for content that is too large or unsafe to display normally.
- [x] 4.8 Keep rendering bounded to visible rows and avoid concatenating full raw transcripts for normal navigation.

## 5. Verification And Documentation

- [x] 5.1 Add README usage notes for storage root selection, index location, vim navigation, search behavior, tags, and raw part safeguards.
- [x] 5.2 Add automated tests for scanner, index, and core TUI state transitions.
- [x] 5.3 Run Go formatting and linting through the devShell tooling.
- [x] 5.4 Run the test suite and fix failures.
- [x] 5.5 Manually run the TUI against the local OpenCode storage and verify session browsing, contextual search, timeline rendering, and read-only behavior.
