## Context

opensession currently scans OpenCode projects, sessions, messages, and parts from JSON storage and the sibling `opencode.db`, then stores normalized session summaries in its own SQLite index. Session summaries already include counts for messages, parts, and heavy parts, but they do not include token usage.

OpenCode assistant message data already contains token usage metadata under `tokens`, including input, output, reasoning, total, and cache token counts. Some `step-finish` parts also contain usage metadata, but session-level display can be derived from assistant messages without depending on timeline parts.

OpenCode storage remains read-only. All derived aggregates belong in the opensession SQLite index.

## Goals / Non-Goals

**Goals:**

- Show session-level token usage while browsing sessions.
- Preserve the distinction between missing usage and zero token counts.
- Include total, input, output, reasoning, cache read, and cache write counts.
- Keep the initial UI compact by showing only session-level usage.
- Support both file-backed OpenCode storage and `opencode.db` scanning paths.

**Non-Goals:**

- Display monetary cost or estimated pricing.
- Add per-message, per-step, or per-part usage rendering.
- Add sorting or filtering by token usage.
- Re-tokenize transcript text when OpenCode metadata is unavailable.

## Decisions

### Use assistant messages as the session usage source

Session usage SHALL be aggregated from assistant messages that contain `tokens` metadata.

Alternative considered: aggregate `step-finish` parts. Step-finish parts can carry similar metadata, but they are timeline implementation details and may be missing for some assistant messages. Message-level aggregation better matches the user-facing concept of a model response and avoids double-counting the same OpenCode usage record.

### Store aggregates on session summaries

The scanner should parse token usage into message models, aggregate it while assembling sessions, and persist aggregate columns on the `sessions` table. `SessionSummary` then exposes the same aggregate fields to the TUI.

Alternative considered: compute usage dynamically from stored raw JSON. Dynamic computation would avoid schema changes, but it would push JSON parsing into query/render paths and make list rendering depend on part/message raw payload availability.

### Represent availability explicitly

The data model should include an explicit boolean-style availability field, such as `HasTokenUsage`, alongside numeric token counts. If no assistant message in a session has `tokens` metadata, the UI displays usage as unavailable rather than displaying zero.

Alternative considered: infer availability from `total == 0`. That fails for valid zero-valued components and makes missing data indistinguishable from real zero usage.

### Prefer OpenCode total when present, otherwise derive total

For each assistant message, `tokens.total` should be used when present. If it is absent but component fields are present, total should be derived from input, output, reasoning, cache read, and cache write counts. The UI still shows the component breakdown so cache-heavy sessions are not reduced to a misleading single number.

Alternative considered: only show total when OpenCode provides `tokens.total`. That would hide useful usage for messages where components are available but total is omitted.

### Keep cost out of the model for this change

OpenCode message data can include `cost`, but this change excludes it from parsing, storage, and rendering. Cost units and pricing semantics can be explored separately without coupling them to token usage display.

## Risks / Trade-offs

- OpenCode token metadata shape changes -> Keep parsing tolerant of missing fields and default to unavailable when no usable `tokens` object exists.
- Existing opensession indexes lack new columns -> Add non-destructive schema migration columns and backfill from OpenCode storage on the next scan.
- Total token semantics differ across providers -> Preserve component breakdown and document that `total` follows OpenCode metadata when available.
- Session list can become visually noisy -> Render only a compact total in list rows and reserve breakdown for the preview/detail header.

## Migration Plan

- Add nullable or defaulted token aggregate columns plus an availability column to the opensession `sessions` table.
- Extend scanner and database scanner parsing so a normal scan populates aggregates from existing OpenCode data.
- Existing databases are upgraded in place through the current schema initialization path.
- Rollback is safe because extra columns in the local opensession index can be ignored by older code or removed by recreating the local index; OpenCode storage is not modified.

## Open Questions

- None for V1. Per-message usage, usage-based sorting, and monetary cost are intentionally deferred.
