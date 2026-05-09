## 1. Rendering Helpers

- [x] 1.1 Add centralized helper functions for tool-like row glyph selection, compact status-symbol mapping, and row field assembly in `internal/tui`.
- [x] 1.2 Ensure status-symbol mapping emits `✓`, `✗`, `…`, or `?` without including raw status words in timeline rows.
- [x] 1.3 Ensure low-signal read lifecycle filtering still uses raw part metadata and continues to hide trivial read lifecycle rows.

## 2. Timeline Row Formatting

- [x] 2.1 Update tool rows to render glyph-led labels such as `$ bash`, `⌕ grep`, `◧ read`, `✎ edit`, or `◆ custom_tool` instead of `[tool] <name>`.
- [x] 2.2 Update linked task/subagent rows to render a glyph-led subagent label while preserving linked child-session affordances.
- [x] 2.3 Update patch and file rows to use compact glyph-led labels instead of `[patch]` and `[file]`.
- [x] 2.4 Apply outcome styling for status symbols where practical while keeping plain-text output stable.

## 3. Tests and Validation

- [x] 3.1 Update existing TUI tests that assert bracketed tool/patch/file labels or textual statuses.
- [x] 3.2 Add or adjust tests covering completed, failed, running/started, and unknown status symbols in plain timeline output.
- [x] 3.3 Add or adjust tests confirming compact rows preserve tool metadata, heavy flags, and linked task navigation behavior.
- [x] 3.4 Run `gofmt` on touched Go files and `go test ./internal/tui`.
