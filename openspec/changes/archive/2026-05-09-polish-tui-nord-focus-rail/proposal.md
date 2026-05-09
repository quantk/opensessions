## Why

The current TUI is functional but visually flat: focus is indicated with a plain `>` marker, colors are ad hoc, and session/timeline/detail views do not share a cohesive visual language. A Nord-inspired visual pass can make opensession easier to scan and more pleasant to use without changing storage, indexing, or navigation behavior.

## What Changes

- Introduce a cohesive Nord-inspired semantic color scheme for TUI elements, including title, mode, dim metadata, source badges, role labels, tool statuses, warnings, and selection.
- Replace the `>` focus marker with a left focus rail that uses a colored first-line indicator and optional continuation rail for multi-line focused content.
- Improve visual hierarchy in headers, footers, session rows, timeline rows, and detail views using consistent separators, spacing, badges, and status icons.
- Preserve existing keyboard navigation, search behavior, markdown rendering, raw/detail safeguards, and read-only source storage behavior.
- Keep rendering bounded and responsive, including on narrow terminals and long timelines.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `opencode-session-tui`: Add requirements for a cohesive visual theme, focus rail indicator, and status/source/role visual affordances in the existing terminal browsing experience.

## Impact

- Affected code: `internal/tui/model.go`, `internal/tui/detail.go`, `internal/tui/markdown.go`, and TUI rendering tests in `internal/tui/model_test.go`.
- No storage schema changes.
- No source scanner/indexer behavior changes.
- No new runtime dependencies expected; changes should use existing Bubble Tea/Lip Gloss/Glamour rendering stack.
