## 1. Renderer Setup

- [x] 1.1 Add the terminal markdown rendering dependency and update module files.
- [x] 1.2 Define a calm TUI-oriented markdown style for assistant timeline rendering, including restrained headings, inline code styling, lists, tables, blockquotes, and fenced code highlighting.
- [x] 1.3 Add an assistant markdown rendering helper that accepts source text and width, returns ANSI-safe display rows, and falls back to source wrapping if rendering fails.

## 2. Timeline Integration

- [x] 2.1 Add model state for assistant markdown display mode, defaulting to rendered markdown for opened timelines.
- [x] 2.2 Wire `m` in the timeline view to toggle assistant text parts between rendered markdown and original markdown source.
- [x] 2.3 Update timeline footer/help text to show the current assistant markdown toggle action.
- [x] 2.4 Route assistant text parts through markdown rendering when rendered mode is active, while keeping user text parts, reasoning parts, tool cards, patch cards, and file cards on their existing render paths.
- [x] 2.5 Ensure rendered markdown rows use ANSI-aware width, truncation, and padding so syntax highlighting does not break layout.

## 3. Behavior Coverage

- [x] 3.1 Add tests that assistant markdown renders formatted output by default and source markdown after the `m` toggle.
- [x] 3.2 Add tests that user messages containing markdown syntax remain source text.
- [x] 3.3 Add tests for fenced code blocks, inline code, and unsupported code fence language fallback.
- [x] 3.4 Add or update tests covering long assistant markdown scrolling/focus behavior and bounded rendering.
- [x] 3.5 Add or update tests confirming session timeline search still uses source/index text rather than rendered ANSI output.

## 4. Verification

- [x] 4.1 Run `gofmt` on touched Go files.
- [x] 4.2 Run focused TUI tests with `go test ./internal/tui`.
- [x] 4.3 Run the full test suite with `go test ./...`.
