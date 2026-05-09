## Context

`opensession` currently scans OpenCode storage into an application-owned SQLite index and renders a flat session list plus per-session timelines. The scanner model (`Snapshot`, `Project`, `Session`, `Message`, `Part`) is OpenCode-named but already behaves like a normalized local agent history model.

Pi stores sessions under `~/.pi/agent/sessions/--<cwd>--/*.jsonl`. Each file has a session header followed by entries linked by `id`/`parentId`, so a single session can contain branches, compactions, labels, model changes, messages, tool calls, and extension data. Pi content is append-only JSONL rather than OpenCode's separate project/session/message/part files.

The change should keep both source stores read-only and continue writing only to the opensession SQLite database.

## Goals / Non-Goals

**Goals:**
- Make source identity explicit so OpenCode, Pi, and future local agent sources can coexist safely.
- Add Pi session scanning with default Pi session root discovery and optional override.
- Preserve existing OpenCode behavior while showing OpenCode and Pi sessions together in list/search/grouped views.
- Provide full Pi session viewing, including branch/tree navigation and timeline projection for the selected branch.
- Normalize Pi text, reasoning, tool calls, tool results, bash executions, compactions, branch summaries, labels, model changes, and session names into searchable/browsable records.
- Keep heavy/binary/raw guardrails and responsive rendering/search behavior.

**Non-Goals:**
- Live tailing currently running Pi sessions.
- Editing, deleting, renaming, compacting, forking, or resuming Pi sessions from opensession.
- Replacing Pi's own `/tree`, `/resume`, `/export`, or sharing workflows.
- Adding non-Pi sources in this change, beyond the source-aware architecture needed to support them later.
- Displaying monetary cost metadata.

## Decisions

### Decision: Add source identity to normalized indexed records

Add a `source_kind` concept to indexed projects, sessions, messages/entries, parts, searchable documents, scan metadata, tags, and bookmarks where needed to distinguish `opencode`, `pi`, and future sources. Persisted IDs should be stable and collision-resistant; Pi-derived IDs should be namespaced from the Pi session UUID and entry/block identifiers.

Alternatives considered:
- Prefix only string IDs with `pi:`. This avoids some collisions but makes SQL filtering, UI badges, migrations, and future source-specific behavior harder.
- Separate source-specific tables. This preserves fidelity but duplicates list/search/timeline logic and makes combined browsing harder.

### Decision: Introduce a source adapter boundary before indexing

Keep source-specific parsing in source packages and feed the index a normalized snapshot/record model. The OpenCode scanner remains responsible for OpenCode JSON/DB details; the Pi scanner handles Pi JSONL entries and tree relationships.

Alternatives considered:
- Teach the OpenCode scanner to parse Pi. This is fast initially but makes the current `internal/opencode` package a misleading dumping ground.
- Rewrite the whole model before adding Pi. This is cleaner but increases risk and delays visible functionality. A compatibility layer can be introduced first, with package renaming/refactoring done incrementally.

### Decision: Model Pi entries separately from renderable parts

Store Pi session entry metadata (`source_entry_id`, `parent_entry_id`, append order, entry type, role, labels/model changes where applicable) separately from renderable/searchable parts. Parts remain the units rendered in timelines and opened in details.

Alternatives considered:
- Flatten Pi directly into messages/parts only. This loses tree structure and makes full branch browsing difficult.
- Store raw Pi files only and compute trees on demand. This keeps the schema small but makes search, tags, and responsive navigation harder.

### Decision: Default Pi timeline is a branch projection, not the full append log

Opening a Pi session displays the latest leaf branch by default. A tree/branch navigator lets the user inspect alternate branches and switch the visible timeline to any branch path. The raw append log is not the primary timeline because it can interleave mutually exclusive branches.

Alternatives considered:
- Show all appended entries in file order. This is simple but semantically misleading for branched sessions.
- Create separate pseudo-sessions per branch. This makes branches visible in the top-level list but fragments one Pi session and complicates tags/bookmarks.

### Decision: Search scopes follow existing opensession mental model

Session-list search searches indexed safe content for all enabled top-level sessions, including all safe Pi branches. Timeline search searches the currently visible session timeline/branch. Tree navigator search, if present, searches entries within the open Pi session tree.

Alternatives considered:
- Timeline search across all Pi branches. This can surface matches that are not visible in the current timeline, which is surprising unless paired with tree navigation.
- Only latest-branch indexing. This loses useful searchability for alternate Pi branches.

### Decision: Store bounded Pi raw payloads in the index

Pi parts originate from blocks inside JSONL lines rather than standalone part files. For safe bounded content, store raw JSON for the entry/block in the index. For unsafe/heavy content, store previews and metadata only and show guards in raw/detail views.

Alternatives considered:
- Store file offsets/line numbers and reread JSONL slices. This preserves source fidelity but requires additional offset tracking and careful handling of edited/rewritten files.
- Store complete raw JSONL entries for everything. This risks indexing large or unsafe payloads and conflicts with existing heavy payload guardrails.

## Risks / Trade-offs

- **Schema migration touches core tables** → Add backward-compatible columns/tables with defaults, preserve existing OpenCode rows, and cover migration with store tests.
- **Pi branches increase UI complexity** → Keep default timeline behavior familiar and isolate tree navigation behind an explicit action.
- **Mixed-source IDs/tags/bookmarks can collide** → Use explicit source identity and namespaced external IDs before inserting records.
- **Large Pi tool results can degrade indexing/rendering** → Reuse current safe text, raw JSON, and heavy payload guards for Pi blocks.
- **Partially written or malformed JSONL files can break scans** → Treat unreadable files/lines as scan errors or bounded warnings according to existing error handling, and do not mutate source files.
- **Renaming normalized packages may create broad churn** → Prefer minimal source adapter and schema changes first, then consider package renaming after tests protect behavior.

## Migration Plan

1. Add source-aware schema fields/tables with default `opencode` values for existing rows.
2. Update OpenCode indexing queries to populate source identity without changing existing user-visible OpenCode behavior.
3. Add Pi scanner/indexing behind the same startup scan path and configuration resolution.
4. Add TUI source badges, combined grouping/search, and Pi tree navigation.
5. Add fixtures and tests for OpenCode regression, Pi parsing, branch projections, source-aware search, tags/bookmarks, and raw/detail guards.

Rollback is straightforward for runtime behavior: disable Pi scanning/source selection and continue using existing OpenCode rows. Schema additions should remain backward-compatible and ignored by older logic where possible.

## Open Questions

- Should the first UI include an explicit source filter (`all`, `opencode`, `pi`), or is a visible badge sufficient for the initial combined list?
- Which key should open Pi tree navigation without conflicting with existing timeline actions?
- Should malformed trailing JSONL lines be skipped with a warning by default, or should the entire file fail scanning?
