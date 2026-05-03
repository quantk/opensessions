## Context

The current TUI renders text timeline parts by extracting source text from `TimelinePart.RawJSON` when available, falling back to indexed previews, wrapping it with `wrapText`, and then applying role-level Lip Gloss styling. This preserves bounded timeline rendering and read-only OpenCode storage access, but assistant markdown remains visible as raw source, including fenced code delimiters and inline formatting markers.

The change should improve readability for assistant responses without changing scanner behavior, search indexing, SQLite schema, raw part details, or user-message rendering. It also needs to preserve existing timeline navigation semantics where `j/k` move focus across focusable parts and scroll through long focused messages.

## Goals / Non-Goals

**Goals:**

- Render assistant text parts as formatted terminal markdown by default in the session timeline.
- Provide an explicit timeline toggle for viewing original assistant markdown source.
- Support fenced code block syntax highlighting and inline code styling using a calm TUI-oriented style.
- Keep rendering bounded to timeline rows and safe for large sessions.
- Keep user messages, reasoning visibility rules, tool/file/patch cards, raw detail views, storage scanning, and search behavior unchanged unless directly necessary for the toggle display.

**Non-Goals:**

- Persisting rendered markdown, markdown mode, or any new transcript data in SQLite.
- Rendering user messages as markdown.
- Replacing raw part detail rendering or pretty tool detail rendering with markdown.
- Adding markdown editing, copying, export, or preview outside the timeline.
- Guaranteeing perfect syntax highlighting for unknown or unsupported fenced languages.

## Decisions

### Use a Charm-compatible terminal markdown renderer

Use `github.com/charmbracelet/glamour` for assistant markdown rendering. Glamour provides GFM markdown parsing, fenced code rendering, inline code styling, table/list/heading support, and Chroma-backed syntax highlighting while fitting the existing Bubble Tea/Lip Gloss terminal UI stack.

Alternatives considered:

- Keep a custom renderer limited to fenced code blocks. This would be smaller but would not satisfy full markdown rendering for lists, headings, tables, and inline code.
- Use Goldmark plus a custom terminal renderer. This would offer more control but adds substantial implementation and maintenance cost.
- Use Chroma only for code fences. This improves the most visible code-block case but leaves markdown structure raw.

### Keep markdown rendering in the TUI layer

Render markdown only when constructing timeline transcript rows. Scanner output, index text, previews, raw JSON, and search documents remain plain/source content.

This keeps OpenCode storage read-only, avoids SQLite migrations, and preserves search semantics. The rendered representation is disposable display output and can be recomputed from source text.

Alternatives considered:

- Store rendered ANSI rows in SQLite. This would complicate width-dependent rendering and introduce stale cache risks.
- Store parsed markdown metadata in the index. This would couple scanning to a TUI concern and make future renderer changes harder.

### Add a timeline-wide assistant markdown mode toggle

Use a model-level timeline flag, defaulting to rendered markdown, and toggle it with `m` in the timeline view. The toggle applies to all assistant text parts in the open timeline so users can consistently switch between readable rendering and exact markdown source.

Alternatives considered:

- Toggle only the focused assistant part. This gives finer control but makes row counts and navigation state more surprising.
- Use `R` for source/raw mode. `R` is already used for raw JSON in part detail views, and source markdown is distinct from raw part JSON.
- Persist the mode across app launches. There is no current settings persistence layer, and a display default is sufficient for the MVP.

### Treat reasoning display as separate from assistant markdown rendering

Reasoning parts remain hidden by default and controlled by the existing `r` toggle. When reasoning is explicitly shown, it should continue to use the existing safe text rendering path unless implementation finds that reasoning content is consistently assistant markdown and the spec is expanded later.

This avoids broadening a sensitive display surface while meeting the requested assistant-answer use case.

### Use ANSI-aware row handling for rendered markdown

Rendered markdown includes ANSI escape sequences. Timeline row construction must avoid plain string truncation that can split escape sequences. Markdown rows should use ANSI-aware width, truncation, padding, or stripping helpers where truncation is needed, while existing plain rows can keep the current `wrapText` behavior.

Alternatives considered:

- Apply existing `truncatePlain` to rendered output. This risks broken terminal styling because it can cut escape sequences.
- Strip ANSI before timeline row management. This would remove syntax highlighting and defeat the feature.

## Risks / Trade-offs

- Markdown rendering can increase per-frame work -> keep rendering bounded by the existing transcript row limits where possible and consider a small width/source keyed cache only if profiling shows a problem.
- Glamour default styles may feel too document-like for an embedded transcript -> use a restrained style configuration with limited margins and subtle headings/code styling.
- Rendered markdown can change line counts compared with source markdown -> update timeline focus/scroll tests around long assistant markdown messages and toggling source mode.
- Search results are based on source/index text, not rendered ANSI output -> keep this behavior explicit; source mode remains available when users need to inspect exact markdown syntax.
- Unknown fenced languages may not highlight -> render as a normal code block without surfacing an error in the timeline.

## Migration Plan

No data migration is required. The change adds a display dependency and TUI rendering behavior only. Rollback is removing the dependency and reverting the timeline markdown rendering/toggle code.

## Open Questions

- None. The initial behavior is assistant-only rendering, rendered by default, with a timeline-wide source toggle and best-effort syntax highlighting.
