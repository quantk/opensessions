## Why

The session list currently behaves as a single recency-ordered stream, but users also need to orient by project without losing recency signals. Providing both flat and project-grouped views lets users switch between "what happened most recently" and "what happened in each active project".

## What Changes

- Add a user-selectable session-list view mode for flat recency-ordered browsing.
- Add a project-grouped session-list view mode that renders project/global group headers with sessions beneath them.
- Order grouped view sections by the most recent visible session activity in each group.
- Treat `Global sessions` as a normal group for activity ordering rather than pinning it to a fixed position.
- Preserve session selection where possible when toggling between flat and grouped views.
- Keep session-list search context-sensitive and make grouped search results preserve grouping over the visible matched sessions.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `opencode-session-tui`: Refine session browsing requirements to support both flat and project-grouped list modes, with grouped sections ordered by visible activity.

## Impact

- Affects `internal/tui` session-list state, navigation, rendering, preview selection, and tests.
- Uses existing `index.SessionSummary` project and timestamp metadata; no storage schema change is expected.
- Does not change OpenCode storage access or mutate OpenCode files.
