## Why

Assistant responses often contain markdown structure, fenced code blocks, and inline code that are hard to read when shown as raw markdown in the session timeline. Rendering assistant messages as calm terminal markdown improves transcript readability while preserving access to the original source text for exact inspection.

## What Changes

- Render assistant text parts as formatted markdown by default in the session timeline.
- Add syntax highlighting for fenced code blocks when the language is recognized, with graceful fallback for unknown languages.
- Style inline code, lists, headings, tables, and blockquotes in a restrained TUI-friendly style.
- Add an explicit timeline toggle to switch assistant text parts between rendered markdown and original markdown source.
- Keep user messages, tool cards, patch cards, file cards, raw part details, search indexing, and OpenCode storage access unchanged.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `opencode-session-tui`: Add assistant-only markdown rendering and a source-view toggle to session timeline behavior.

## Impact

- Affected code: `internal/tui` timeline rendering, key handling, footer/help text, and focused transcript row behavior.
- Affected specs: `openspec/specs/opencode-session-tui/spec.md` via a change-local delta spec.
- New dependency: terminal markdown rendering support, expected to use Charm-compatible markdown rendering and syntax highlighting libraries.
- Data safety: OpenCode storage remains read-only and the application SQLite index remains the source for search and raw message text.
