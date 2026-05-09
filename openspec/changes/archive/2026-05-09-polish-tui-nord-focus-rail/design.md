## Context

The TUI renderer lives primarily in `internal/tui/model.go`, with structured detail rendering in `internal/tui/detail.go` and assistant markdown styling in `internal/tui/markdown.go`. Current styling is defined as package-level Lip Gloss styles using individual numeric colors, and focus is represented by a plain `>` prefix in session rows, tree rows, timeline text rows, and tool rows.

This change is visual only. It must preserve existing navigation, search, scanner/index integration, read-only source access, raw/detail safety guards, markdown behavior, and bounded rendering guarantees.

## Goals / Non-Goals

**Goals:**

- Establish a cohesive Nord-inspired semantic theme for all TUI rendering.
- Replace `>` focus markers with a left focus rail that reads as part of the layout rather than prompt-like text.
- Improve scanability through consistent source badges, role markers, status icons, separators, and dim metadata treatment.
- Keep the renderer adaptive for narrow terminals and safe for long timelines.
- Keep tests focused on semantics and stable visual affordances rather than brittle ANSI color sequences where possible.

**Non-Goals:**

- No new storage, indexing, or scanner behavior.
- No keyboard shortcut changes.
- No mouse support.
- No configurable theme system in this change.
- No major layout redesign such as a full timeline gutter/card system unless needed to support the focus rail cleanly.
- No new runtime dependency beyond the existing Bubble Tea/Lip Gloss/Glamour stack.

## Decisions

### Use semantic theme variables over direct color literals

Define a small internal theme layer that maps semantic roles to Nord palette colors, then build Lip Gloss styles from those roles. This keeps future visual changes localized and avoids continuing the current pattern of scattered numeric colors.

Alternatives considered:

- Keep direct `lipgloss.Color("...")` usage: simplest, but perpetuates inconsistent styling.
- Add user-selectable themes: useful later, but too much scope for this visual pass.

### Use a focus rail instead of `>` markers

Replace prompt-like focus markers with a reserved left rail column. Focused first lines use a strong rail glyph such as `▌`; continuation lines for the same focused block may use a quieter `│`. Unfocused rows keep the same column width with whitespace so layout alignment remains stable.

Alternatives considered:

- Use `›`: more polished than `>`, but still reads as a textual prefix rather than a layout affordance.
- Full-row selection background only: visible, but heavy in Nord and noisy in long transcripts.
- Status-colored rail: visually rich but ambiguous; the rail should mean focus only, while status belongs in icons/text.

### Keep selection background subtle

The Nord palette works best when most of the interface remains low-contrast and accents are reserved for meaning. Focus should primarily be the rail, with optional subtle background on compact list/tree rows. Avoid strong full-row blue backgrounds except where existing active tool emphasis needs it and remains readable.

Alternatives considered:

- Bright full-width selected rows: highly visible but distracts from transcript reading.
- Rail-only everywhere: clean, but session/tree lists may benefit from a subtle background because rows are single-line selectable records.

### Preserve bounded rendering and width accounting

The rail consumes visible width. Rendering helpers should account for the rail before wrapping/truncating content and should continue using Lip Gloss width-aware operations. Multi-line text and markdown rows should retain bounded/cached rendering behavior.

Alternatives considered:

- Add the rail after truncation: easier but can overflow or clip content incorrectly.
- Re-render all markdown with full width then prefix: risks width drift and test fragility.

### Use icons only when paired with text or stable labels

Tool statuses may use glyphs such as `✓`, `✗`, and `…`, but these must supplement existing status words or summaries rather than become the only status indicator. Source badges should remain textual, e.g. `[opencode]` and `[pi]`, with improved styling.

Alternatives considered:

- Icon-only status display: compact but less accessible and harder to test.
- ASCII-only mode: robust, but loses much of the desired polished visual feel. The existing UI already uses Unicode tree glyphs, so Unicode rail/status glyphs fit the project.

## Risks / Trade-offs

- Unicode glyph rendering varies by terminal/font → Use common box/block glyphs already typical in TUIs and preserve alignment through width-aware helpers.
- Tests may become brittle if they assert exact styled output → Prefer tests that strip ANSI and check stable glyphs/labels; reserve exact checks for small helper behavior.
- Rail column reduces available content width → Subtract rail width before wrapping/truncating and preserve narrow-terminal fallbacks.
- Strong colors may hurt readability on non-Nord terminal palettes → Use conservative semantic colors and avoid relying on color alone.
- Detail/raw views may become visually inconsistent if only timeline/list are themed → Apply theme to detail headings, section labels, warnings, and footer/header consistently, but avoid over-framing content.
