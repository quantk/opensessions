## 1. Source-Aware Index Foundation

- [x] 1.1 Add source-kind constants and normalized source identity helpers for OpenCode, Pi, and future sources.
- [x] 1.2 Add backward-compatible SQLite schema migrations for source identity on projects, sessions, messages or entries, parts, searchable documents, scan metadata, tags, and bookmarks as needed.
- [x] 1.3 Add an indexed entry/tree table or equivalent metadata model for source entry IDs, parent entry IDs, append order, entry type, role, labels, and branch navigation data.
- [x] 1.4 Update existing OpenCode upsert/query code to populate `opencode` source identity without changing current OpenCode behavior.
- [x] 1.5 Add store migration tests proving existing pre-source databases are treated as OpenCode records and tags/bookmarks remain associated correctly.

## 2. Source Configuration and Discovery

- [x] 2.1 Extend configuration resolution with enabled source selection and Pi session root override support.
- [x] 2.2 Add default Pi session root discovery for `~/.pi/agent/sessions` or the XDG/home-equivalent Pi session location used by Pi.
- [x] 2.3 Update CLI flags and environment-variable handling for selecting sources and overriding the Pi sessions root.
- [x] 2.4 Ensure missing optional Pi roots do not fail startup when other enabled sources are available.
- [x] 2.5 Add config tests for defaults, overrides, source selection, and missing optional roots.

## 3. Pi JSONL Scanner

- [x] 3.1 Add Pi scanner package and test fixtures for linear sessions, branched sessions, named sessions, compactions, branch summaries, labels, tool calls/results, bash executions, and custom entries.
- [x] 3.2 Parse Pi session headers and entries from JSONL while preserving source file path, append order, entry ID, parent entry ID, timestamp, and entry type.
- [x] 3.3 Normalize Pi session metadata including session ID, cwd/project key, title from session info, model/provider metadata, token usage where available, created time, and updated time.
- [x] 3.4 Normalize Pi user and assistant text blocks into safe text parts with previews, index text, bounded raw JSON, role metadata, and markdown eligibility.
- [x] 3.5 Normalize Pi assistant thinking blocks into reasoning parts hidden by default.
- [x] 3.6 Normalize Pi tool call blocks and tool result messages into tool parts with tool name, arguments/output previews, status or error state, and safe searchable text.
- [x] 3.7 Normalize Pi bash execution messages into tool-like parts with command, output preview, exit/cancel/truncation metadata, and guarded raw data.
- [x] 3.8 Normalize Pi compaction and branch summary entries into summary parts on the relevant branch path.
- [x] 3.9 Normalize Pi labels, model changes, thinking-level changes, custom messages, and unsupported entries into entry metadata or safe placeholder parts without failing scans.
- [x] 3.10 Apply existing heavy, binary-looking, unsafe text, and raw JSON safety limits to Pi content.

## 4. Pi Indexing and Incremental Scan Integration

- [x] 4.1 Add Pi source discovery to the startup scan flow while preserving OpenCode scan behavior and `--no-scan` semantics.
- [x] 4.2 Upsert Pi sessions, entries, parts, search documents, and scan metadata into the source-aware index.
- [x] 4.3 Implement unchanged-file reuse for Pi JSONL files based on path, size, and modification time.
- [x] 4.4 Refresh changed Pi files by replacing stale indexed Pi entries, parts, and search documents for that source session without affecting other sources.
- [x] 4.5 Add store/scanner tests for first scan, unchanged reuse, changed refresh, malformed or unsupported entries, and read-only source file behavior.

## 5. Repository Queries and Search Semantics

- [x] 5.1 Update session list and search queries to return top-level sessions from all enabled sources with source kind metadata and stable ordering.
- [x] 5.2 Update grouped session query/model logic to group by normalized project or working directory across sources while preserving source badges.
- [x] 5.3 Add repository APIs for Pi session tree retrieval, branch leaf listing, selected branch path timelines, and tree-entry search.
- [x] 5.4 Update timeline queries so OpenCode sessions remain chronological and Pi sessions return the selected branch projection.
- [x] 5.5 Update in-session search so Pi timeline search applies to the current branch path, while session-list search indexes safe content from all Pi branches.
- [x] 5.6 Add query tests for ID collisions across sources, source-aware tags/bookmarks, all-source search, Pi branch timeline search, and grouped ordering.

## 6. Combined Session List TUI

- [x] 6.1 Add source kind fields to TUI session summaries and render source badges in flat list rows, grouped list rows, session previews, and timeline headers.
- [x] 6.2 Update session preview rendering to include source metadata without regressing existing OpenCode metadata, token usage, tags, or bookmarks.
- [x] 6.3 Update flat and grouped list selection preservation when switching modes or applying searches across mixed sources.
- [x] 6.4 Add optional source filter UI only if needed by the final source-selection design; otherwise document that combined list uses badges only.
- [x] 6.5 Add TUI tests for mixed OpenCode/Pi session rows, source badges, grouped project behavior, and search result display.

## 7. Pi Timeline and Tree TUI

- [x] 7.1 Add Pi timeline state for active branch leaf or selected tree entry while preserving existing OpenCode timeline state.
- [x] 7.2 Display Pi sessions using the latest leaf branch by default and indicate branch projection context in the timeline header.
- [x] 7.3 Add keyboard-accessible Pi tree or branch navigator reachable from a Pi timeline.
- [x] 7.4 Render branch points, leaves, labels, entry summaries, and selected branch context in the Pi tree navigator using bounded visible rows.
- [x] 7.5 Allow opening a tree entry or branch leaf to switch the Pi timeline to that branch path.
- [x] 7.6 Ensure standard back behavior returns from tree navigation to the prior Pi timeline without changing branch selection unless the user selected a branch.
- [x] 7.7 Add TUI tests for latest-branch default, branch switching, labels, back behavior, and branch path rendering.

## 8. Pi Detail, Raw, and Rendering Guardrails

- [x] 8.1 Reuse existing message detail behavior for safe Pi user and assistant text parts, including assistant markdown rendering/source toggles.
- [x] 8.2 Reuse or extend pretty tool detail renderers for Pi tool call, tool result, and bash execution parts.
- [x] 8.3 Ensure explicit raw JSON toggle displays bounded stored Pi raw payloads only when available and safe.
- [x] 8.4 Display guard messages for skipped, heavy, binary-looking, malformed, or unavailable Pi raw/detail payloads.
- [x] 8.5 Ensure reasoning parts from Pi remain hidden by default and become openable only after the explicit reasoning toggle.
- [x] 8.6 Add detail/raw rendering tests for safe text, markdown, tool call/result, bash output, heavy payload guards, and reasoning visibility.

## 9. Documentation and Validation

- [x] 9.1 Update README usage, storage path, keyboard, and data-safety sections for Pi and multi-source browsing.
- [x] 9.2 Document new CLI flags and environment variables for source selection and Pi session root override.
- [x] 9.3 Add or update OpenSpec-driven tests to cover every new and modified scenario where practical.
- [x] 9.4 Run `gofmt` on touched Go files.
- [x] 9.5 Run `go test ./...` and fix regressions.
- [x] 9.6 Manually smoke-test mixed OpenCode/Pi browsing with `CGO_ENABLED=0 go run ./cmd/opensession` using test or local session roots.
