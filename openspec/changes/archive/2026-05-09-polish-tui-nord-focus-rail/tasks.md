## 1. Theme Foundation

- [x] 1.1 Add a small semantic Nord theme layer for TUI colors and derived Lip Gloss styles.
- [x] 1.2 Replace existing direct style color choices in `internal/tui/model.go` with semantic theme-backed styles.
- [x] 1.3 Align assistant markdown style colors in `internal/tui/markdown.go` with the Nord-inspired semantic palette.

## 2. Focus Rail Rendering

- [x] 2.1 Replace the existing `>` marker helper with focus rail helpers for focused, continuation, and unfocused row prefixes.
- [x] 2.2 Apply the focus rail to session list rows while preserving grouped-list indentation and selected-row visibility.
- [x] 2.3 Apply the focus rail to session tree rows while preserving depth indentation and tree glyph readability.
- [x] 2.4 Apply the focus rail to focused timeline text, markdown, reasoning, tool, patch, file, and linked task rows.
- [x] 2.5 Update wrapping/truncation logic where needed so rail-prefixed rows remain within terminal width.

## 3. Visual Affordances and Hierarchy

- [x] 3.1 Restyle source badges so `[opencode]`, `[pi]`, and fallback source labels are visually distinct but remain textual.
- [x] 3.2 Add consistent textual status affordances for tool rows/details, including completed, failed, running/pending, and unknown states.
- [x] 3.3 Refresh header, metadata, search prompt, warning, and footer styling to use the Nord semantic theme consistently.
- [x] 3.4 Improve detail-view section headings and warnings without changing raw/detail safety behavior or available content.

## 4. Tests and Validation

- [x] 4.1 Update TUI rendering tests that currently expect `>` focus markers to assert the new rail indicator instead.
- [x] 4.2 Add or update tests for focused multi-line timeline content to verify first-line and continuation rail alignment in plain output.
- [x] 4.3 Add or update tests for source badges, status affordances, and guard/warning text remaining understandable after ANSI stripping.
- [x] 4.4 Run `gofmt` on touched Go files.
- [x] 4.5 Run `go test ./internal/tui`.
- [x] 4.6 Run `go test ./...`.
