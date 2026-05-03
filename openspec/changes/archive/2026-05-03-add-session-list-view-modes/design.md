## Context

The TUI currently models the session list as a flat `[]index.SessionSummary` plus a selected session index and scroll offset. `Store.ListSessions` and `Store.SearchSessions` already return session summaries with project metadata and updated timestamps, ordered by recency. Rendering builds list rows directly from `sessions`, and the preview panel reads the selected session from the same slice.

The existing OpenSpec requirement mentions project-grouped browsing, but the concrete user need is two switchable session-list views: a flat global recency stream and a grouped project/global view whose groups are themselves ordered by recent visible activity.

## Goals / Non-Goals

**Goals:**

- Provide a session-list view toggle between flat and grouped modes.
- Keep flat mode as a single `updated_at DESC` session stream.
- Render grouped mode as one list with non-selectable project/global headers and selectable session rows.
- Sort grouped sections by the most recent visible session in each group, including `Global sessions` as a normal activity-sorted group.
- Preserve the selected session by session ID when switching modes or rebuilding rows after search where possible.
- Keep search context-sensitive and have grouped search results group only the matched visible sessions.

**Non-Goals:**

- No project-first tree view or separate project drill-down screen.
- No persisted preference for the selected list mode.
- No SQLite schema changes or new repository API requirements.
- No changes to session timeline, raw/pretty part detail rendering, tags, bookmarks, or OpenCode storage scanning.

## Decisions

### Use an in-memory session-list mode

Add TUI state for the current session-list mode, with flat mode as the default to preserve the current startup behavior. A `v` keybinding can toggle the mode from `ViewSessions`, and the footer/header should expose the current mode and available toggle.

Alternative considered: make grouped mode the default because the original spec says grouped browsing. This would be a larger behavioral change and would make the new flat mode feel secondary, so the safer path is to preserve the current flat default while adding grouped browsing explicitly.

### Build display rows from the visible session set

Introduce a rendering/navigation projection for the session list, conceptually:

```
[]SessionSummary
      |
      v
[]sessionListRow
      |-- header row: project/global label, count, active time
      |-- session row: selectable SessionSummary
```

Flat mode produces only session rows. Grouped mode produces header rows plus session rows. Navigation should skip header rows so `j/k`, paging, `g/G`, and `Enter` continue to operate on sessions rather than groups.

Alternative considered: mutate `sessions` into grouped order and special-case headers separately. A row projection keeps the source session set intact and makes it easier to preserve selection by session ID across view-mode changes and search updates.

### Define activity from visible session timestamps

For each group, compute activity as `max(session.UpdatedAt)` over the sessions currently visible in the list. In normal browsing, visible sessions are all loaded sessions; after search, visible sessions are the matched results. Sort groups by activity descending, then by label or stable key for ties. Sort sessions inside each group by `UpdatedAt` descending, with a stable tie-breaker such as session ID.

`Global sessions` is only special in label/key derivation. It participates in the same activity sort as project groups and is not pinned above or below real projects.

Alternative considered: sort groups alphabetically or by OpenCode project metadata timestamps. Alphabetical ordering hides recency, and project metadata timestamps may not reflect user-visible session activity.

### Keep grouped search grouped

`SearchSessions` can continue returning a flat set of matching sessions. The TUI should apply the active view mode to that visible set. In grouped mode, only groups with matched sessions should appear, and their ordering should be based on the latest matched session in each group.

Alternative considered: always show search results as flat. That is simpler but violates the user's active browsing context and makes mode switching less predictable.

## Risks / Trade-offs

- Header rows can make scroll math and selected-index handling more error-prone. Mitigation: centralize row projection and navigation helpers, and test selection, paging, search, and toggle behavior.
- Preserving selection by index will select the wrong session after regrouping. Mitigation: preserve by session ID when rebuilding the view.
- Very small terminal heights may show only headers if not handled carefully. Mitigation: navigation/rendering should ensure selected session rows remain visible and tests should cover constrained heights.
- Group labels can be long project paths. Mitigation: reuse existing truncation helpers and keep full path available in the preview panel.

## Migration Plan

No data migration is required. The change is TUI-only and uses existing indexed session metadata. Rollback is removing the view-mode state/projection and returning to the current flat list rendering.

## Open Questions

- None currently. `Global sessions` participates in the same activity ordering as project groups.
