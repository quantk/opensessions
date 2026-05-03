## Why

Users need a quick way to understand how token-heavy each OpenCode session was without opening raw OpenCode data or estimating from transcript length. OpenCode already records token usage in assistant message metadata, so opensession can surface this existing information directly in the session-level browsing experience.

## What Changes

- Add session-level token usage summaries derived from OpenCode assistant message `tokens` metadata.
- Display aggregate token usage in the session list, session preview, and session detail header.
- Include total, input, output, reasoning, cache read, and cache write token counts where available.
- Treat missing usage as unavailable rather than zero.
- Exclude monetary cost display from this change.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `opencode-session-tui`: Add requirements for displaying session-level token usage summaries when OpenCode usage metadata is available.

## Impact

- `internal/opencode`: Parse token usage from OpenCode message data and aggregate it per session.
- `internal/index`: Persist token usage aggregates in the opensession SQLite index and expose them through session summaries.
- `internal/tui`: Render compact and detailed session-level token usage without adding cost display.
- Tests and fixtures: Add coverage for token usage parsing, persistence, and rendering, including unavailable usage.
