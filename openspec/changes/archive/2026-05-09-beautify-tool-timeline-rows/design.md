## Context

The session timeline currently renders tool-like parts through compact string builders in `internal/tui/model.go`. Tool, patch, file, and linked task rows are displayed as one-line rail rows, with labels such as `[tool] bash` and status text from `statusAffordance`, for example `✓ completed`. This change is purely presentational: it improves scanability while preserving the existing timeline data model, row focus behavior, and `Enter`/`l` detail or child-session navigation behavior.

## Goals / Non-Goals

**Goals:**

- Replace bracketed textual labels for tool-like timeline rows with compact glyph-led labels.
- Display tool statuses as symbols only in the timeline (`✓`, `✗`, `…`, `?`) while optionally styling them by outcome.
- Keep rows single-line and dense, matching the selected Variant A direction.
- Preserve existing visible metadata: tool name, title, target path, preview, linked session, and heavy flags.
- Keep tests focused on plain-text output so styling does not make assertions brittle.

**Non-Goals:**

- No multi-line inline tool cards in this change.
- No changes to indexing, storage, raw/pretty detail views, or child-session navigation.
- No new external rendering dependencies.
- No emoji-based icons that can create ambiguous terminal cell widths.

## Decisions

1. **Use plain terminal-safe glyphs instead of bracket labels.**

   Tool-like rows will start with a compact glyph and label, such as `$ bash`, `⌕ grep`, `◧ read`, `✎ edit`, `↪ subagent`, or `◆ custom_tool`. This keeps each row readable in one line while avoiding noisy bracket tags. Alternatives considered were full words (`tool bash`) and emoji; full words are less visually distinctive, while emoji can render at inconsistent widths.

2. **Split status display into symbol and style.**

   Timeline rows will use a status symbol without the raw status word: `✓` for completed/success/ok, `✗` for failed/error/cancelled, `…` for running/pending/started/active, and `?` for unknown statuses. Styling can use the existing success/error/warn/dim palette or nearby styles, but the plain-text content remains stable for tests and non-color terminals. Detailed views may continue showing full status text where useful.

3. **Keep compact row generation centralized.**

   The existing `compactPart` path should be refactored or supplemented with small helpers for tool glyph selection, status symbol selection, and field assembly. This avoids scattering formatting rules across the timeline renderer and keeps the behavior easy to test.

4. **Preserve low-signal lifecycle filtering.**

   Existing filtering for low-signal read lifecycle rows should continue to work. If helper functions change how status text is derived, the filtering logic should still use raw metadata and preview comparisons rather than styled output.

## Risks / Trade-offs

- **Glyph support varies by terminal** → Use conservative non-emoji Unicode symbols and keep labels/tool names present so rows remain understandable.
- **Removing status words may hide detail** → Detailed part views still expose the full status; timeline rows prioritize scanability.
- **Color assertions can become brittle** → Tests should strip ANSI and assert plain glyph/label/status-symbol output.
- **Search/index behavior could accidentally change if preview helpers are reused incorrectly** → Limit the change to TUI rendering helpers and avoid scanner/index changes.
