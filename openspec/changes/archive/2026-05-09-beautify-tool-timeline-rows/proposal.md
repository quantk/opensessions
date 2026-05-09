## Why

Tool, patch, file, and subagent rows in the timeline currently use bracketed textual labels such as `[tool]`, which makes high-frequency tool activity feel noisy and visually unfinished. A compact, icon-led row format will make tool calls easier to scan without changing navigation or detail behavior.

## What Changes

- Replace bracketed timeline labels for tool-like parts with compact glyph-led labels, for example `$ bash`, `⌕ grep`, `◧ read`, `✎ edit`, `↪ subagent`, or `◆ custom_tool`.
- Render tool status as a compact symbol only (`✓`, `✗`, `…`, `?`) instead of including status words such as `completed` in the timeline row.
- Use existing theme colors/styles to visually distinguish status outcomes where terminal styling is available.
- Preserve existing information density: tool name, title, file path/target, preview text, heavy flags, and linked child-session affordances remain visible when available.
- Preserve all existing open/navigation behavior for tool, patch, file, and linked subagent rows.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `opencode-session-tui`: Timeline tool-like part rows use compact icon/status affordances instead of bracketed textual labels and status words.

## Impact

- Affected code: `internal/tui` timeline row rendering and related tests.
- Affected UI: session timeline rows for tool, patch, file, and subagent/task parts.
- No storage, indexing, command-line, or dependency changes are expected.
