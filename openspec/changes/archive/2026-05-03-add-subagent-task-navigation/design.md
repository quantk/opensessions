## Context

opensession currently models OpenCode sessions as a flat set of indexed sessions. The scanner reads file-backed `session` JSON and database-backed `session` rows, assembles `messages` and `parts` by `session_id`, and the index exposes `ListSessions`, `SearchSessions`, and `SessionTimeline` without parent-child awareness.

Recent OpenCode storage includes explicit child-session metadata: file-backed sessions can contain `parentID`, and `opencode.db` has `session.parent_id` plus an index on that column. Task tool parts can also expose the spawned child session through `state.metadata.sessionId`. That gives opensession enough information to treat subagent sessions as nested transcripts instead of ordinary top-level sessions.

The design must preserve read-only OpenCode storage access and keep writable state limited to the opensession SQLite index.

## Goals / Non-Goals

**Goals:**

- Hide child/subagent sessions from the normal flat and grouped session lists.
- Preserve child sessions in the local index so they can be opened from their parent session.
- Show linked `task` tool rows as the primary entry point into spawned subagent timelines.
- Keep `l`/`Enter` behavior for ordinary tools, patch parts, file parts, and unlinked task parts.
- Return from a child timeline to the exact parent timeline context that opened it.
- Support both file-backed OpenCode storage and `opencode.db` scanning paths.

**Non-Goals:**

- Do not merge child transcript parts inline into parent timelines.
- Do not mutate, annotate, archive, or compact OpenCode storage.
- Do not add new external dependencies.
- Do not require global session-list search to surface parent sessions based on child-session content in this change.
- Do not redesign the timeline, raw detail, or project grouping visual language beyond the subagent affordance.

## Decisions

### Persist parent-child session metadata in the index

Add parent session metadata to the scanner model and local `sessions` table rather than deriving parentage only in the TUI. The file scanner should read `parentID`, and the database scanner should read `parent_id`. Existing index initialization can add non-destructive columns with `ensureColumn`, matching the current token usage migration style.

Alternative considered: infer child sessions only by matching task part metadata to session ids. That misses child sessions with parent metadata but no retained task metadata, and it does not support hiding all child sessions from top-level lists reliably.

### Link task parts to child sessions during scanning/indexing

Task tool parts should retain the child session id from `state.metadata.sessionId` when present. The index can persist this on `parts` as a nullable linked session id, or via a small relation table keyed by part id. A nullable column is likely enough because a single OpenCode task part represents one subagent invocation.

Alternative considered: parse raw JSON in the TUI when opening a task row. That would avoid an index field, but it would make task row affordances, tests, and future query behavior depend on raw JSON availability and TUI parsing.

### Treat linked task open as session navigation, not part detail

When a focused timeline part is a `task` tool with a linked child session id, `l`/`Enter` should open that child session timeline. Unlinked task rows and all other openable parts should continue to open part detail as they do today.

Alternative considered: always open task detail first and put a nested session link inside detail. That keeps raw/detail behavior primary, but it makes the subagent transcript harder to discover and adds an extra step for the main use case.

### Keep timeline navigation as a stack

Opening a child session should push the current parent timeline context: session summary, timeline data, selected part, scroll position, search state if applicable, reasoning visibility, and markdown mode as needed to return predictably. `h`/`Esc` from the child timeline should pop back to the parent timeline and reselect the task row that opened the child.

Alternative considered: replace `currentSession` with the child and return directly to the session list on `h`/`Esc`. That matches current top-level timeline behavior but loses the parent task context and makes nested exploration feel disconnected.

### Keep child sessions hidden only from top-level browsing

Child sessions should remain persisted and readable through `SessionTimeline`. The list APIs used for top-level browsing should filter out sessions with a non-empty parent id. This keeps implementation small and avoids changing raw scanner/index safety behavior.

Alternative considered: expose child sessions as collapsible rows under parent sessions in the main list. That would be discoverable, but it reintroduces list noise and duplicates the task-row entry point.

## Risks / Trade-offs

- [Risk] Older file-backed sessions or database rows might omit parent metadata while task parts still reference child sessions. → Mitigation: task rows with linked child session ids should still open the child even if parent metadata is incomplete; unlinked child sessions can remain visible as ordinary sessions if no parent relationship exists.
- [Risk] A child session can exist with `parentID` but no linked task part. → Mitigation: hide it from the top-level list because parentage is explicit; leave a future affordance for an optional parent header list of unlinked subagents outside this change.
- [Risk] `task` raw JSON can be heavy or skipped. → Mitigation: extract and persist only the linked child session id during scanning/classification before raw display guards affect UI access.
- [Risk] Navigation stack state can become inconsistent after search filtering or refresh. → Mitigation: store enough parent context to restore the opened timeline; if the original selected part is unavailable, fall back to the nearest focusable part in the restored timeline.
- [Risk] Filtering top-level lists changes visible session counts. → Mitigation: update tests and UI expectations to count top-level sessions only while keeping child sessions accessible through parent timelines.

## Migration Plan

Add nullable schema fields through the existing non-destructive SQLite schema initialization path. Existing local opensession databases can be reopened and migrated automatically. If rollback is needed, old application versions will ignore extra nullable columns, while newly indexed child sessions may reappear as ordinary sessions in older versions.

No OpenCode storage migration is required.

## Open Questions

- Should a future change make session-list search surface parent sessions when only child-session content matches?
- Should a future change show a compact child-session count in parent session list rows or preview panes?
- Should unlinked child sessions be reachable from a parent timeline header if no task row links to them?
